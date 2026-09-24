package voucher

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"trust-management/backend/internal/database"
	"trust-management/backend/internal/models"
	"trust-management/backend/internal/shared"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type VoucherHandler struct{}

func NewVoucherHandler() *VoucherHandler {
	return &VoucherHandler{}
}

// ─── LIST VOUCHERS ─────────────────────────────────────────────────────────────

func (h *VoucherHandler) GetVouchers(c *gin.Context) {
	var vouchers []models.Voucher
	query := database.DB.
		Preload("Ledger").
		Preload("Title").
		Preload("BankAccount").
		Preload("FromBankAccount").
		Preload("ToBankAccount").
		Preload("CreatedBy").
		Preload("ApprovedBy").
		Order("business_date desc, id desc")

	if voucherType := c.Query("type"); voucherType != "" && voucherType != "ALL" {
		query = query.Where("voucher_type = ?", strings.ToUpper(voucherType))
	}

	if fromDate := c.Query("from_date"); fromDate != "" {
		query = query.Where("business_date >= ?", fromDate)
	}
	if toDate := c.Query("to_date"); toDate != "" {
		query = query.Where("business_date <= ?", toDate)
	}

	if ledgerIDStr := c.Query("ledger_id"); ledgerIDStr != "" && ledgerIDStr != "ALL" {
		if lid, err := strconv.ParseUint(ledgerIDStr, 10, 32); err == nil {
			query = query.Where("ledger_id = ?", lid)
		}
	}

	if titleIDStr := c.Query("title_id"); titleIDStr != "" && titleIDStr != "ALL" {
		if tid, err := strconv.ParseUint(titleIDStr, 10, 32); err == nil {
			query = query.Where("title_id = ?", tid)
		}
	}

	if mode := c.Query("payment_mode"); mode != "" && mode != "ALL" {
		query = query.Where("payment_mode = ?", strings.ToUpper(mode))
	}

	if status := c.Query("status"); status != "" && status != "ALL" {
		upperStatus := strings.ToUpper(status)
		if upperStatus == "APPROVED" || upperStatus == "ISSUED" {
			query = query.Where("status IN ('APPROVED', 'ISSUED')")
		} else {
			query = query.Where("status = ?", upperStatus)
		}
	}

	if err := query.Find(&vouchers).Error; err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to fetch vouchers: "+err.Error())
		return
	}

	shared.SendSuccess(c, http.StatusOK, vouchers)
}

// ─── CREATE VOUCHER (FINANCIAL PROCESSING ENGINE) ──────────────────────────────

type CreateVoucherRequest struct {
	VoucherType       string          `json:"voucher_type" binding:"required"` // INCOME, EXPENSE, ASSET, LIABILITY, SELF_TRANSFER
	TitleID           *uint           `json:"title_id"`
	LedgerID          *uint           `json:"ledger_id"`
	BusinessDate      string          `json:"business_date"` // YYYY-MM-DD (defaults to server biz date)
	PaymentMode       string          `json:"payment_mode" binding:"required"` // CASH, BANK
	BankAccountID     *uint           `json:"bank_account_id"`
	FromBankAccountID *uint           `json:"from_bank_account_id"`
	ToBankAccountID   *uint           `json:"to_bank_account_id"`
	PayeeOrDonorName  string          `json:"payee_or_donor_name"`
	Amount            decimal.Decimal `json:"amount" binding:"required"`
	Details           string          `json:"details"`
	AttachmentPath    string          `json:"attachment_path"`
	AutoApprove       bool            `json:"auto_approve"` // If true, issue and post immediately without pending queue
}

// postVoucherFinancials records the double-entry transactions in cash_transactions and bank_transactions
func postVoucherFinancials(tx *gorm.DB, voucher *models.Voucher, actorID uint) error {
	vType := voucher.VoucherType
	payMode := voucher.PaymentMode
	bizDate := voucher.BusinessDate
	amount := voucher.Amount
	payeeName := voucher.PayeeOrDonorName

	if vType == "SELF_TRANSFER" {
		// Contra Transfer
		// 1. Source Bank / Cash
		if voucher.FromBankAccountID != nil {
			var srcBank models.BankAccount
			if err := tx.First(&srcBank, *voucher.FromBankAccountID).Error; err != nil {
				return errors.New("source bank account not found")
			}
			if srcBank.CurrentBalance.LessThan(amount) {
				return fmt.Errorf("insufficient balance in source bank %s (Current: ₹%s, Required: ₹%s)",
					srcBank.BankName, srcBank.CurrentBalance.StringFixed(2), amount.StringFixed(2))
			}

			txOut := models.BankTransaction{
				BankAccountID:   *voucher.FromBankAccountID,
				BusinessDate:    bizDate,
				TransactionType: "DEBIT",
				Amount:          amount,
				Category:        "SELF_TRANSFER_OUT",
				ReferenceNumber: voucher.VoucherNumber,
				SourceType:      "VOUCHER",
				SourceID:        voucher.ID,
				Description:     fmt.Sprintf("Self Transfer Out: %s", voucher.Details),
				CreatedByID:     actorID,
			}
			if err := tx.Create(&txOut).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.BankAccount{}).Where("id = ?", *voucher.FromBankAccountID).
				UpdateColumn("current_balance", gorm.Expr("current_balance - ?", amount)).Error; err != nil {
				return err
			}
		} else {
			// Cash withdrawal to bank: Cash Outflow
			cashOut := models.CashTransaction{
				BusinessDate:    bizDate,
				TransactionType: "OUTFLOW",
				Amount:          amount,
				SourceType:      "VOUCHER",
				SourceID:        voucher.ID,
				Description:     fmt.Sprintf("Self Transfer Cash Out: %s", voucher.Details),
				CreatedByID:     actorID,
			}
			if err := tx.Create(&cashOut).Error; err != nil {
				return err
			}
		}

		// 2. Destination Bank / Cash
		if voucher.ToBankAccountID != nil {
			txIn := models.BankTransaction{
				BankAccountID:   *voucher.ToBankAccountID,
				BusinessDate:    bizDate,
				TransactionType: "CREDIT",
				Amount:          amount,
				Category:        "SELF_TRANSFER_IN",
				ReferenceNumber: voucher.VoucherNumber,
				SourceType:      "VOUCHER",
				SourceID:        voucher.ID,
				Description:     fmt.Sprintf("Self Transfer In: %s", voucher.Details),
				CreatedByID:     actorID,
			}
			if err := tx.Create(&txIn).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.BankAccount{}).Where("id = ?", *voucher.ToBankAccountID).
				UpdateColumn("current_balance", gorm.Expr("current_balance + ?", amount)).Error; err != nil {
				return err
			}
		} else {
			// Bank to cash withdrawal: Cash Inflow
			cashIn := models.CashTransaction{
				BusinessDate:    bizDate,
				TransactionType: "INFLOW",
				Amount:          amount,
				SourceType:      "VOUCHER",
				SourceID:        voucher.ID,
				Description:     fmt.Sprintf("Self Transfer Cash In: %s", voucher.Details),
				CreatedByID:     actorID,
			}
			if err := tx.Create(&cashIn).Error; err != nil {
				return err
			}
		}
	} else if payMode == models.PaymentModeCash {
		// Cash flow: INCOME & LIABILITY (Inflow), EXPENSE & ASSET (Outflow)
		flowType := "OUTFLOW"
		if vType == "INCOME" || vType == "LIABILITY" || vType == "DONATION_RECEIPT" {
			flowType = "INFLOW"
		}
		ct := models.CashTransaction{
			BusinessDate:    bizDate,
			TransactionType: flowType,
			Amount:          amount,
			SourceType:      "VOUCHER",
			SourceID:        voucher.ID,
			Description:     fmt.Sprintf("[%s] %s - %s", vType, payeeName, voucher.Details),
			CreatedByID:     actorID,
		}
		if err := tx.Create(&ct).Error; err != nil {
			return err
		}
	} else if payMode == models.PaymentModeBank && voucher.BankAccountID != nil {
		// Bank flow: INCOME & LIABILITY (Credit), EXPENSE & ASSET (Debit)
		flowType := "DEBIT"
		balanceExpr := "current_balance - ?"
		if vType == "INCOME" || vType == "LIABILITY" || vType == "DONATION_RECEIPT" {
			flowType = "CREDIT"
			balanceExpr = "current_balance + ?"
		} else {
			// Check balance for debit
			var bAcc models.BankAccount
			if err := tx.First(&bAcc, *voucher.BankAccountID).Error; err != nil {
				return errors.New("selected bank account does not exist")
			}
			if bAcc.CurrentBalance.LessThan(amount) {
				return fmt.Errorf("insufficient bank balance in %s (Current: ₹%s, Required: ₹%s)",
					bAcc.BankName, bAcc.CurrentBalance.StringFixed(2), amount.StringFixed(2))
			}
		}

		bt := models.BankTransaction{
			BankAccountID:   *voucher.BankAccountID,
			BusinessDate:    bizDate,
			TransactionType: flowType,
			Amount:          amount,
			Category:        vType,
			ReferenceNumber: voucher.VoucherNumber,
			SourceType:      "VOUCHER",
			SourceID:        voucher.ID,
			Description:     fmt.Sprintf("[%s] %s - %s", vType, payeeName, voucher.Details),
			CreatedByID:     actorID,
		}
		if err := tx.Create(&bt).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.BankAccount{}).Where("id = ?", *voucher.BankAccountID).
			UpdateColumn("current_balance", gorm.Expr(balanceExpr, amount)).Error; err != nil {
			return err
		}
	}

	return nil
}

func (h *VoucherHandler) CreateVoucher(c *gin.Context) {
	var req CreateVoucherRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.SendAppError(c, http.StatusBadRequest, "Invalid voucher payload: "+err.Error())
		return
	}

	vType := strings.ToUpper(strings.TrimSpace(req.VoucherType))
	validTypes := map[string]bool{
		"INCOME": true, "EXPENSE": true, "ASSET": true, "LIABILITY": true, "SELF_TRANSFER": true,
	}
	if !validTypes[vType] {
		shared.SendAppError(c, http.StatusBadRequest, "Invalid voucher type. Must be INCOME, EXPENSE, ASSET, LIABILITY, or SELF_TRANSFER")
		return
	}

	if req.Amount.LessThanOrEqual(decimal.Zero) {
		shared.SendAppError(c, http.StatusBadRequest, "Voucher amount must be strictly greater than zero")
		return
	}

	payMode := models.PaymentMode(strings.ToUpper(strings.TrimSpace(req.PaymentMode)))
	if payMode != models.PaymentModeCash && payMode != models.PaymentModeBank {
		shared.SendAppError(c, http.StatusBadRequest, "Payment mode must be CASH or BANK")
		return
	}

	// Business Date
	bizDate := shared.GetCurrentBusinessDate()
	if req.BusinessDate != "" {
		if parsed, err := time.Parse("2006-01-02", req.BusinessDate); err == nil {
			bizDate = parsed
		}
	}

	// Title & Ledger validation
	var ledgerID *uint = req.LedgerID
	var titleID *uint = req.TitleID

	if titleID != nil && *titleID > 0 {
		var vt models.VoucherTitle
		if err := database.DB.First(&vt, *titleID).Error; err != nil {
			shared.SendAppError(c, http.StatusBadRequest, "Selected title does not exist")
			return
		}
		// Auto-resolve ledger if not explicitly supplied
		if ledgerID == nil || *ledgerID == 0 {
			ledgerID = &vt.LedgerID
		}
	}

	if ledgerID != nil && *ledgerID > 0 {
		var l models.Ledger
		if err := database.DB.First(&l, *ledgerID).Error; err != nil {
			shared.SendAppError(c, http.StatusBadRequest, "Selected ledger does not exist")
			return
		}
	}

	// Payment mode specific validation
	if vType == "SELF_TRANSFER" {
		if req.FromBankAccountID == nil && req.ToBankAccountID == nil {
			shared.SendAppError(c, http.StatusBadRequest, "Self transfer requires at least one source or destination bank account")
			return
		}
		if req.FromBankAccountID != nil && req.ToBankAccountID != nil && *req.FromBankAccountID == *req.ToBankAccountID {
			shared.SendAppError(c, http.StatusBadRequest, "Source and destination bank accounts cannot be the same")
			return
		}
	} else if payMode == models.PaymentModeBank {
		if req.BankAccountID == nil || *req.BankAccountID == 0 {
			shared.SendAppError(c, http.StatusBadRequest, "Please select a bank account for bank payment mode")
			return
		}
		var bAcc models.BankAccount
		if err := database.DB.First(&bAcc, *req.BankAccountID).Error; err != nil {
			shared.SendAppError(c, http.StatusBadRequest, "Selected bank account does not exist")
			return
		}
	}

	userID := c.MustGet("user_id").(uint)
	amountInWords := shared.ConvertAmountToWords(req.Amount)
	payeeName := strings.TrimSpace(req.PayeeOrDonorName)
	if payeeName == "" {
		if vType == "SELF_TRANSFER" {
			payeeName = "Internal Self Transfer"
		} else {
			payeeName = "Self / Direct"
		}
	}

	var voucher models.Voucher
	now := time.Now()

	// Default status is PENDING (approval workflow), matching employment standard.
	// If auto_approve is explicitly requested by an admin/privileged user, directly mark as APPROVED and post.
	initialStatus := "PENDING"
	var approvedByID *uint = nil
	var approvedAt *time.Time = nil
	if req.AutoApprove {
		initialStatus = "APPROVED"
		approvedByID = &userID
		approvedAt = &now
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// Generate sequential voucher number atomically
		vNo, err := shared.GenerateVoucherNumber(tx, bizDate)
		if err != nil {
			return fmt.Errorf("generating voucher number: %w", err)
		}

		voucher = models.Voucher{
			VoucherNumber:     vNo,
			VoucherType:       vType,
			LedgerID:          ledgerID,
			TitleID:           titleID,
			BusinessDate:      bizDate,
			SourceType:        "DIRECT",
			SourceID:          0,
			PayeeOrDonorName:  payeeName,
			Amount:            req.Amount,
			AmountInWords:     amountInWords,
			PaymentMode:       payMode,
			BankAccountID:     req.BankAccountID,
			FromBankAccountID: req.FromBankAccountID,
			ToBankAccountID:   req.ToBankAccountID,
			Details:           strings.TrimSpace(req.Details),
			AttachmentPath:    strings.TrimSpace(req.AttachmentPath),
			Status:            initialStatus,
			CreatedByID:       userID,
			ApprovedByID:      approvedByID,
			ApprovedAt:        approvedAt,
		}

		if err := tx.Create(&voucher).Error; err != nil {
			return err
		}

		// Only post financials to Cash and Bank if directly APPROVED
		if initialStatus == "APPROVED" {
			if err := postVoucherFinancials(tx, &voucher, userID); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to create voucher: "+err.Error())
		return
	}

	// Reload with associations
	_ = database.DB.
		Preload("Ledger").
		Preload("Title").
		Preload("BankAccount").
		Preload("FromBankAccount").
		Preload("ToBankAccount").
		First(&voucher, voucher.ID)

	shared.SendSuccess(c, http.StatusCreated, voucher)
}

// ─── APPROVE VOUCHER ──────────────────────────────────────────────────────────

func (h *VoucherHandler) ApproveVoucher(c *gin.Context) {
	id := c.Param("id")
	adminID := c.MustGet("user_id").(uint)

	var voucher models.Voucher
	if err := database.DB.
		Preload("Ledger").
		Preload("Title").
		Preload("BankAccount").
		Preload("FromBankAccount").
		Preload("ToBankAccount").
		First(&voucher, id).Error; err != nil {
		shared.SendAppError(c, http.StatusNotFound, "Voucher not found")
		return
	}

	if voucher.Status != "PENDING" {
		shared.SendAppError(c, http.StatusBadRequest, fmt.Sprintf("Voucher cannot be approved because it is already in %s status", voucher.Status))
		return
	}

	now := time.Now()
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// Execute financial postings to Cash and Bank
		if err := postVoucherFinancials(tx, &voucher, adminID); err != nil {
			return err
		}

		voucher.Status = "APPROVED"
		voucher.ApprovedByID = &adminID
		voucher.ApprovedAt = &now
		return tx.Save(&voucher).Error
	})

	if err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to approve voucher: "+err.Error())
		return
	}

	_ = database.DB.
		Preload("Ledger").
		Preload("Title").
		Preload("BankAccount").
		Preload("FromBankAccount").
		Preload("ToBankAccount").
		Preload("CreatedBy").
		Preload("ApprovedBy").
		First(&voucher, voucher.ID)

	shared.SendSuccess(c, http.StatusOK, gin.H{
		"message": "Voucher approved and financial transactions posted successfully",
		"voucher": voucher,
	})
}

// ─── REJECT VOUCHER ──────────────────────────────────────────────────────────

type RejectVoucherRequest struct {
	Reason string `json:"reason" binding:"required"`
}

func (h *VoucherHandler) RejectVoucher(c *gin.Context) {
	id := c.Param("id")
	adminID := c.MustGet("user_id").(uint)

	var req RejectVoucherRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.SendAppError(c, http.StatusBadRequest, "Rejection reason is mandatory")
		return
	}

	var voucher models.Voucher
	if err := database.DB.First(&voucher, id).Error; err != nil {
		shared.SendAppError(c, http.StatusNotFound, "Voucher not found")
		return
	}

	if voucher.Status != "PENDING" {
		shared.SendAppError(c, http.StatusBadRequest, fmt.Sprintf("Voucher cannot be rejected because it is already in %s status", voucher.Status))
		return
	}

	now := time.Now()
	voucher.Status = "REJECTED"
	voucher.RejectionReason = strings.TrimSpace(req.Reason)
	voucher.ApprovedByID = &adminID
	voucher.ApprovedAt = &now

	if err := database.DB.Save(&voucher).Error; err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to reject voucher: "+err.Error())
		return
	}

	shared.SendSuccess(c, http.StatusOK, gin.H{
		"message": "Voucher rejected successfully",
		"voucher": voucher,
	})
}

// ─── GET SINGLE VOUCHER (PRESERVES EXISTING RECEIPT/PRINT DESIGN) ───────────────

func (h *VoucherHandler) GetVoucherByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		shared.SendAppError(c, http.StatusBadRequest, "Invalid voucher ID")
		return
	}

	var voucher models.Voucher
	if err := database.DB.
		Preload("Ledger").
		Preload("Title").
		Preload("BankAccount").
		Preload("FromBankAccount").
		Preload("ToBankAccount").
		Preload("CreatedBy").
		Preload("ApprovedBy").
		First(&voucher, id).Error; err != nil {
		shared.SendAppError(c, http.StatusNotFound, "Voucher not found")
		return
	}

	result := gin.H{
		"id":                    voucher.ID,
		"voucher_number":        voucher.VoucherNumber,
		"voucher_type":          voucher.VoucherType,
		"business_date":         voucher.BusinessDate,
		"source_type":           voucher.SourceType,
		"source_id":             voucher.SourceID,
		"payee_or_donor_name":   voucher.PayeeOrDonorName,
		"amount":                voucher.Amount,
		"amount_in_words":       voucher.AmountInWords,
		"payment_mode":          voucher.PaymentMode,
		"status":                voucher.Status,
		"details":               voucher.Details,
		"attachment_path":       voucher.AttachmentPath,
		"ledger_id":             voucher.LedgerID,
		"ledger":                voucher.Ledger,
		"title_id":              voucher.TitleID,
		"title":                 voucher.Title,
		"bank_account_id":       voucher.BankAccountID,
		"bank_account":          voucher.BankAccount,
		"from_bank_account_id":  voucher.FromBankAccountID,
		"from_bank_account":     voucher.FromBankAccount,
		"to_bank_account_id":    voucher.ToBankAccountID,
		"to_bank_account":       voucher.ToBankAccount,
		"created_at":            voucher.CreatedAt,
		"created_by":            voucher.CreatedBy,
		"approved_by":           voucher.ApprovedBy,
		"approved_at":           voucher.ApprovedAt,
	}

	var bankAccount *models.BankAccount = voucher.BankAccount
	if bankAccount == nil && voucher.FromBankAccount != nil {
		bankAccount = voucher.FromBankAccount
	}

	if voucher.SourceType == "DONATION" {
		var donation models.Donation
		if err := database.DB.Preload("Donor").Preload("Scheme").First(&donation, voucher.SourceID).Error; err == nil {
			result["purpose"] = donation.Purpose
			result["reference_number"] = donation.ReferenceNumber
			if donation.Donor != nil {
				result["donor_phone"] = donation.Donor.Phone
				result["donor_father_name"] = donation.Donor.FatherName
			}
			if donation.Scheme != nil {
				result["food_type"] = donation.Scheme.FoodType
				result["meal_type"] = donation.Scheme.MealType
				result["category"] = donation.Scheme.Category
			}
			if donation.BankAccountID != nil && bankAccount == nil {
				var acc models.BankAccount
				if err := database.DB.First(&acc, *donation.BankAccountID).Error; err == nil {
					bankAccount = &acc
				}
			}
		}
	} else if voucher.SourceType == "EXPENSE" {
		var expense models.Expense
		if err := database.DB.First(&expense, voucher.SourceID).Error; err == nil {
			result["purpose"] = expense.Description
			result["reference_number"] = expense.ReferenceNumber
			result["category"] = expense.Category
			if expense.BankAccountID != nil && bankAccount == nil {
				var acc models.BankAccount
				if err := database.DB.First(&acc, *expense.BankAccountID).Error; err == nil {
					bankAccount = &acc
				}
			}
		}
	}

	if bankAccount == nil {
		var defaultAcc models.BankAccount
		err := database.DB.Where("is_active = ? AND qr_code_path IS NOT NULL AND qr_code_path <> ''", true).
			Order("id asc").First(&defaultAcc).Error
		if err != nil {
			err = database.DB.Where("is_active = ?", true).Order("id asc").First(&defaultAcc).Error
		}
		if err == nil {
			bankAccount = &defaultAcc
		}
	}
	result["bank_account"] = bankAccount

	shared.SendSuccess(c, http.StatusOK, result)
}

// ─── VOUCHER REPORT & FINANCIAL CALCULATIONS ───────────────────────────────────

type VoucherReportSummary struct {
	TotalIncome        decimal.Decimal `json:"total_income"`
	TotalExpense       decimal.Decimal `json:"total_expense"`
	NetCashFlow        decimal.Decimal `json:"net_cash_flow"`
	TotalAssets        decimal.Decimal `json:"total_assets"`
	TotalLiabilities   decimal.Decimal `json:"total_liabilities"`
	TotalSelfTransfers decimal.Decimal `json:"total_self_transfers"`
	VoucherCount       int64           `json:"voucher_count"`
}

type LedgerSummaryItem struct {
	LedgerID   uint            `json:"ledger_id"`
	LedgerNo   string          `json:"ledger_no"`
	LedgerName string          `json:"ledger_name"`
	TotalDebit decimal.Decimal `json:"total_debit"`
	TotalCredit decimal.Decimal `json:"total_credit"`
	NetAmount  decimal.Decimal `json:"net_amount"`
	Count      int64           `json:"count"`
}

func (h *VoucherHandler) GetVoucherReport(c *gin.Context) {
	fromDate := c.Query("from_date")
	toDate := c.Query("to_date")
	vType := c.Query("type")
	ledgerIDStr := c.Query("ledger_id")
	titleIDStr := c.Query("title_id")
	payMode := c.Query("payment_mode")
	status := c.Query("status")

	query := database.DB.Model(&models.Voucher{}).
		Preload("Ledger").
		Preload("Title").
		Preload("BankAccount").
		Preload("FromBankAccount").
		Preload("ToBankAccount").
		Order("business_date desc, id desc")

	if fromDate != "" {
		query = query.Where("business_date >= ?", fromDate)
	}
	if toDate != "" {
		query = query.Where("business_date <= ?", toDate)
	}
	if vType != "" && vType != "ALL" {
		query = query.Where("voucher_type = ?", strings.ToUpper(vType))
	}
	if ledgerIDStr != "" && ledgerIDStr != "ALL" {
		if lid, err := strconv.ParseUint(ledgerIDStr, 10, 32); err == nil {
			query = query.Where("ledger_id = ?", lid)
		}
	}
	if titleIDStr != "" && titleIDStr != "ALL" {
		if tid, err := strconv.ParseUint(titleIDStr, 10, 32); err == nil {
			query = query.Where("title_id = ?", tid)
		}
	}
	if payMode != "" && payMode != "ALL" {
		query = query.Where("payment_mode = ?", strings.ToUpper(payMode))
	}
	if status != "" && status != "ALL" {
		upperStatus := strings.ToUpper(status)
		if upperStatus == "APPROVED" || upperStatus == "ISSUED" {
			query = query.Where("status IN ('APPROVED', 'ISSUED')")
		} else {
			query = query.Where("status = ?", upperStatus)
		}
	}

	var vouchers []models.Voucher
	if err := query.Find(&vouchers).Error; err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to generate voucher report: "+err.Error())
		return
	}

	// Calculate Financial Metrics (Strict Decimal Calculations)
	summary := VoucherReportSummary{
		TotalIncome:        decimal.Zero,
		TotalExpense:       decimal.Zero,
		NetCashFlow:        decimal.Zero,
		TotalAssets:        decimal.Zero,
		TotalLiabilities:   decimal.Zero,
		TotalSelfTransfers: decimal.Zero,
		VoucherCount:       int64(len(vouchers)),
	}

	ledgerMap := make(map[uint]*LedgerSummaryItem)

	for _, v := range vouchers {
		if v.Status == "CANCELLED" || v.Status == "REJECTED" || v.Status == "PENDING" {
			continue
		}

		amt := v.Amount
		switch v.VoucherType {
		case "INCOME", "DONATION_RECEIPT":
			summary.TotalIncome = summary.TotalIncome.Add(amt)
		case "EXPENSE", "EXPENSE_VOUCHER":
			summary.TotalExpense = summary.TotalExpense.Add(amt)
		case "ASSET":
			summary.TotalAssets = summary.TotalAssets.Add(amt)
		case "LIABILITY":
			summary.TotalLiabilities = summary.TotalLiabilities.Add(amt)
		case "SELF_TRANSFER":
			summary.TotalSelfTransfers = summary.TotalSelfTransfers.Add(amt)
		}

		// Ledger grouping
		if v.LedgerID != nil && *v.LedgerID > 0 {
			lid := *v.LedgerID
			if _, exists := ledgerMap[lid]; !exists {
				lName := "Ledger #" + strconv.FormatUint(uint64(lid), 10)
				lNo := ""
				if v.Ledger != nil {
					lName = v.Ledger.LedgerName
					lNo = v.Ledger.LedgerNo
				}
				ledgerMap[lid] = &LedgerSummaryItem{
					LedgerID:    lid,
					LedgerNo:    lNo,
					LedgerName:  lName,
					TotalDebit:  decimal.Zero,
					TotalCredit: decimal.Zero,
					NetAmount:   decimal.Zero,
					Count:       0,
				}
			}
			item := ledgerMap[lid]
			item.Count++
			if v.VoucherType == "INCOME" || v.VoucherType == "DONATION_RECEIPT" || v.VoucherType == "LIABILITY" {
				item.TotalCredit = item.TotalCredit.Add(amt)
			} else {
				item.TotalDebit = item.TotalDebit.Add(amt)
			}
			item.NetAmount = item.TotalCredit.Sub(item.TotalDebit)
		}
	}

	summary.NetCashFlow = summary.TotalIncome.Sub(summary.TotalExpense)

	var ledgerSummary []LedgerSummaryItem
	for _, item := range ledgerMap {
		ledgerSummary = append(ledgerSummary, *item)
	}

	shared.SendSuccess(c, http.StatusOK, gin.H{
		"summary":        summary,
		"ledger_summary": ledgerSummary,
		"vouchers":       vouchers,
	})
}

// ─── CANCEL VOUCHER (AUDITED TRANSACTION REVERSAL) ─────────────────────────────

type CancelVoucherRequest struct {
	Reason string `json:"reason" binding:"required"`
}

func (h *VoucherHandler) CancelVoucher(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		shared.SendAppError(c, http.StatusBadRequest, "Invalid voucher ID")
		return
	}

	var req CancelVoucherRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Reason) == "" {
		shared.SendAppError(c, http.StatusBadRequest, "Cancellation reason is strictly mandatory")
		return
	}

	var voucher models.Voucher
	if err := database.DB.First(&voucher, id).Error; err != nil {
		shared.SendAppError(c, http.StatusNotFound, "Voucher not found")
		return
	}

	if voucher.Status == "CANCELLED" {
		shared.SendAppError(c, http.StatusBadRequest, "Voucher is already cancelled")
		return
	}

	err = database.DB.Transaction(func(tx *gorm.DB) error {
		// Only reverse postings if the voucher was approved/issued and had financial impact
		if voucher.Status == "APPROVED" || voucher.Status == "ISSUED" {
			// Reversal of Bank Transactions
			var btxs []models.BankTransaction
			if err := tx.Where("source_type = 'VOUCHER' AND source_id = ?", voucher.ID).Find(&btxs).Error; err == nil {
				for _, bt := range btxs {
					// Reverse account balance
					balExpr := "current_balance - ?"
					if bt.TransactionType == "DEBIT" {
						balExpr = "current_balance + ?"
					}
					if err := tx.Model(&models.BankAccount{}).Where("id = ?", bt.BankAccountID).
						UpdateColumn("current_balance", gorm.Expr(balExpr, bt.Amount)).Error; err != nil {
						return err
					}
					_ = tx.Delete(&bt)
				}
			}

			// Reversal of Cash Transactions
			if err := tx.Where("source_type = 'VOUCHER' AND source_id = ?", voucher.ID).
				Delete(&models.CashTransaction{}).Error; err != nil {
				return errors.New("failed removing cash transaction")
			}
		}

		// Update voucher status
		voucher.Status = "CANCELLED"
		voucher.RejectionReason = strings.TrimSpace(req.Reason)
		return tx.Save(&voucher).Error
	})

	if err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to cancel voucher: "+err.Error())
		return
	}

	shared.SendSuccess(c, http.StatusOK, gin.H{
		"message": "Voucher cancelled and ledger postings reversed successfully",
		"id":      id,
	})
}
