package donation

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"trust-management/backend/internal/database"
	"trust-management/backend/internal/dto"
	"trust-management/backend/internal/models"
	"trust-management/backend/internal/shared"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type DonationHandler struct{}

func NewDonationHandler() *DonationHandler {
	return &DonationHandler{}
}

// GetDonations fetches donations list with preloads and optional filters
func (h *DonationHandler) GetDonations(c *gin.Context) {
	var donations []models.Donation
	query := database.DB.Preload("Donor").Preload("Scheme").Preload("CreatedBy").Order("id desc")

	if paymentMode := c.Query("payment_mode"); paymentMode != "" {
		query = query.Where("payment_mode = ?", paymentMode)
	}
	if search := c.Query("search"); search != "" {
		likePattern := "%" + search + "%"
		query = query.Joins("LEFT JOIN donors ON donors.id = donations.donor_id").
			Where("donations.donation_number LIKE ? OR donors.full_name LIKE ? OR donations.purpose LIKE ?",
				likePattern, likePattern, likePattern)
	}

	if err := query.Find(&donations).Error; err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to fetch donations")
		return
	}

	shared.SendSuccess(c, http.StatusOK, donations)
}

// CreateDonation handles full donation creation, validation, cash/bank transactions, and voucher generation
func (h *DonationHandler) CreateDonation(c *gin.Context) {
	var req dto.CreateDonationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.SendAppError(c, http.StatusBadRequest, err.Error())
		return
	}

	// 1. Get current user ID from auth context
	userIDVal, exists := c.Get("userID")
	if !exists {
		shared.SendAppError(c, http.StatusUnauthorized, "User context missing")
		return
	}
	userID := userIDVal.(uint)

	// 2. Parse business date (default to IST today)
	bizDate := shared.GetCurrentBusinessDate()
	if req.BusinessDate != "" {
		if t, err := time.Parse("2006-01-02", req.BusinessDate); err == nil {
			bizDate = t
		}
	}

	// 3. Validate Payment Mode & Bank Account selection
	// (Closed-day protection is enforced centrally by middleware.ClosedDayProtectionMiddleware)
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		shared.SendAppError(c, http.StatusBadRequest, "Donation amount must be greater than zero.")
		return
	}

	paymentMode := models.PaymentMode(strings.ToUpper(req.PaymentMode))
	if paymentMode != models.PaymentModeCash && paymentMode != models.PaymentModeBank {
		shared.SendAppError(c, http.StatusBadRequest, "Invalid payment mode. Must be CASH or BANK.")
		return
	}

	var bankAccount *models.BankAccount
	if paymentMode == models.PaymentModeBank {
		if req.BankAccountID == nil || *req.BankAccountID == 0 {
			shared.SendAppError(c, http.StatusBadRequest, "Bank Account selection is required for BANK payment mode.")
			return
		}
		var acc models.BankAccount
		if err := database.DB.First(&acc, *req.BankAccountID).Error; err != nil {
			shared.SendAppError(c, http.StatusBadRequest, "Selected Bank Account not found or inactive.")
			return
		}
		bankAccount = &acc

		userRoleVal, _ := c.Get("role")
		userRole, _ := userRoleVal.(models.Role)
		if userRole != models.RoleAdmin {
			if err := shared.EnsureBankAccountOpen(database.DB, acc.ID, bizDate.Format("2006-01-02")); err != nil {
				shared.SendError(c, http.StatusConflict, "BANK_ACCOUNT_CLOSED",
					fmt.Sprintf("%s is CLOSED for %s. Ask an admin to unlock it before recording new transactions.", acc.BankName, bizDate.Format("2006-01-02")))
				return
			}
		}
	}

	// 5. Validate Cash Denominations if CASH mode and provided
	if paymentMode == models.PaymentModeCash && len(req.Denominations) > 0 {
		denomSum := shared.SumDenominations(req.Denominations)
		if !denomSum.Equal(req.Amount) {
			shared.SendAppError(c, http.StatusBadRequest, fmt.Sprintf("Denomination sum (₹%s) does not match total donation amount (₹%s)", denomSum.StringFixed(2), req.Amount.StringFixed(2)))
			return
		}
	}

	// 6. Begin Database Transaction
	tx := database.DB.Begin()

	// Generate unique donation number (e.g. DON-2026-00001) via atomic sequence counter
	donSeq, err := shared.NextSequence(tx, "DONATION")
	if err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to generate donation number")
		return
	}
	donationNumber := fmt.Sprintf("DON-%d-%05d", bizDate.Year(), donSeq)

	var eventDate *time.Time
	if req.EventDate != "" {
		if t, err := time.Parse("2006-01-02", req.EventDate); err == nil {
			eventDate = &t
		}
	}

	donation := models.Donation{
		DonationNumber:      donationNumber,
		DonorID:             req.DonorID,
		BusinessDate:        bizDate,
		Amount:              req.Amount,
		PaymentMode:         paymentMode,
		Purpose:             req.Purpose,
		SchemeID:            req.SchemeID,
		EventType:           req.EventType,
		EventPersonName:     req.EventPersonName,
		EventDate:           eventDate,
		RelationshipToDonor: req.RelationshipToDonor,
		FamilyMemberID:      req.FamilyMemberID,
		BankAccountID:       req.BankAccountID,
		ReferenceNumber:     req.ReferenceNumber,
		AttachmentPath:      req.AttachmentPath,
		Notes:               req.Notes,
		Status:              "ACTIVE",
		CreatedByID:         userID,
	}

	if err := tx.Create(&donation).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to record donation")
		return
	}

	// 7. Post Cash / Bank Ledger Entry
	if paymentMode == models.PaymentModeCash {
		// Create Cash Inflow transaction
		cashTx := models.CashTransaction{
			BusinessDate:    bizDate,
			TransactionType: "INFLOW",
			Amount:          req.Amount,
			SourceType:      "DONATION",
			SourceID:        donation.ID,
			Description:     fmt.Sprintf("Donation %s (%s)", donation.DonationNumber, req.Purpose),
			CreatedByID:     userID,
		}
		if err := tx.Create(&cashTx).Error; err != nil {
			tx.Rollback()
			shared.SendAppError(c, http.StatusInternalServerError, "Failed to record cash inflow")
			return
		}

		// Save Denomination breakdown if provided
		for _, d := range req.Denominations {
			if d.Quantity > 0 {
				denom := models.CashDenomination{
					EntityType:        "DONATION",
					EntityID:          donation.ID,
					DenominationValue: d.Value,
					Quantity:          d.Quantity,
					TotalAmount:       decimal.NewFromInt(int64(d.Value * d.Quantity)),
				}
				if err := tx.Create(&denom).Error; err != nil {
					tx.Rollback()
					shared.SendAppError(c, http.StatusInternalServerError, "Failed to save cash denomination")
					return
				}
			}
		}
	} else if paymentMode == models.PaymentModeBank {
		// Create Bank Credit transaction
		bankTx := models.BankTransaction{
			BankAccountID:   *req.BankAccountID,
			BusinessDate:    bizDate,
			TransactionType: "CREDIT",
			Amount:          req.Amount,
			Category:        "DONATION",
			ReferenceNumber: donation.DonationNumber,
			SourceType:      "DONATION",
			SourceID:        donation.ID,
			Description:     fmt.Sprintf("Bank Donation %s (%s)", donation.DonationNumber, req.Purpose),
			CreatedByID:     userID,
		}
		if err := tx.Create(&bankTx).Error; err != nil {
			tx.Rollback()
			shared.SendAppError(c, http.StatusInternalServerError, "Failed to record bank credit transaction")
			return
		}

		// Update Bank Account Current Balance atomically at the row level (a
		// precomputed literal here would be a lost-update race under concurrent
		// donations against the same account).
		if err := tx.Model(bankAccount).Update("current_balance", gorm.Expr("current_balance + ?", req.Amount)).Error; err != nil {
			tx.Rollback()
			shared.SendAppError(c, http.StatusInternalServerError, "Failed to update bank balance")
			return
		}
	}

	// 8. Generate Unique Voucher Code (e.g. ACHT/1/26-27) via atomic per-financial-year sequence counter
	voucherNumber, err := shared.GenerateVoucherNumber(tx, bizDate)
	if err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to generate voucher number")
		return
	}

	// Fetch donor name for voucher receipt
	var donor models.Donor
	tx.First(&donor, req.DonorID)

	voucher := models.Voucher{
		VoucherNumber:    voucherNumber,
		VoucherType:      "DONATION_RECEIPT",
		BusinessDate:     bizDate,
		SourceType:       "DONATION",
		SourceID:         donation.ID,
		PayeeOrDonorName: donor.FullName,
		Amount:           req.Amount,
		AmountInWords:    shared.ConvertAmountToWords(req.Amount),
		PaymentMode:      paymentMode,
		Status:           "ISSUED",
		CreatedByID:      userID,
	}
	if err := tx.Create(&voucher).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to generate receipt voucher")
		return
	}

	audit := models.AuditLog{
		UserID:     &userID,
		Action:     "DONATION_CREATED",
		EntityName: "Donation",
		EntityID:   donation.ID,
		BeforeData: shared.JSONOrNull(nil),
		AfterData:  shared.JSONOrNull(donation),
		IPAddress:  c.ClientIP(),
	}
	if err := tx.Create(&audit).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to record audit log")
		return
	}

	tx.Commit()

	// Return created donation with preloads
	database.DB.Preload("Donor").Preload("Scheme").Preload("CreatedBy").First(&donation, donation.ID)
	shared.SendSuccess(c, http.StatusCreated, gin.H{
		"donation": donation,
		"voucher":  voucher,
	})
}

// GetDonationByID returns single donation details with voucher
func (h *DonationHandler) GetDonationByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		shared.SendAppError(c, http.StatusBadRequest, "Invalid donation ID")
		return
	}

	var donation models.Donation
	if err := database.DB.Preload("Donor").Preload("Scheme").Preload("CreatedBy").First(&donation, id).Error; err != nil {
		shared.SendAppError(c, http.StatusNotFound, "Donation not found")
		return
	}

	var voucher models.Voucher
	database.DB.Where("source_type = ? AND source_id = ?", "DONATION", donation.ID).First(&voucher)

	shared.SendSuccess(c, http.StatusOK, gin.H{
		"donation": donation,
		"voucher":  voucher,
	})
}
