package donation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"trust-management/backend/internal/config"
	"trust-management/backend/internal/database"
	"trust-management/backend/internal/dto"
	"trust-management/backend/internal/models"
	"trust-management/backend/internal/shared"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type PublicDonationHandler struct {
	cfg *config.Config
}

func NewPublicDonationHandler(cfg *config.Config) *PublicDonationHandler {
	return &PublicDonationHandler{cfg: cfg}
}

// CreatePublicDonation handles donor donations from the mobile app
func (h *PublicDonationHandler) CreatePublicDonation(c *gin.Context) {
	var req dto.PublicCreateDonationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.SendAppError(c, http.StatusBadRequest, err.Error())
		return
	}

	phone := strings.TrimSpace(req.Phone)
	if phone == "" {
		shared.SendAppError(c, http.StatusBadRequest, "Phone number is required")
		return
	}

	if req.Amount.LessThanOrEqual(decimal.Zero) {
		shared.SendAppError(c, http.StatusBadRequest, "Donation amount must be greater than zero")
		return
	}

	bizDate := shared.GetCurrentBusinessDate()

	// 1. Locate designated App Donation Bank Account
	var bankAccount models.BankAccount
	if err := database.DB.Where("is_app_donation_account = ? AND is_active = ?", true, true).First(&bankAccount).Error; err != nil {
		// Fallback to first active bank account
		if err := database.DB.Where("is_active = ?", true).Order("id asc").First(&bankAccount).Error; err != nil {
			shared.SendAppError(c, http.StatusInternalServerError, "No active trust bank account configured to receive donations")
			return
		}
	}

	tx := database.DB.Begin()

	// 2. Find or create Donor by phone
	var donor models.Donor
	if err := tx.Where("phone = ?", phone).First(&donor).Error; err != nil {
		// Auto-generate donor code (e.g. DNR-00001)
		donorSeq, err := shared.NextSequenceAutoSeed(tx, "DONOR")
		if err != nil {
			tx.Rollback()
			shared.SendAppError(c, http.StatusInternalServerError, "Failed to generate donor ID")
			return
		}
		donorCode := fmt.Sprintf("DNR-%05d", donorSeq)

		donor = models.Donor{
			DonorCode:   donorCode,
			FullName:    strings.TrimSpace(req.FullName),
			Phone:       phone,
			Email:       strings.TrimSpace(req.Email),
			PANNumber:   strings.ToUpper(strings.TrimSpace(req.PANNumber)),
			AddressLine: strings.TrimSpace(req.AddressLine),
			City:        strings.TrimSpace(req.City),
			State:       strings.TrimSpace(req.State),
			Pincode:     strings.TrimSpace(req.Pincode),
			IsActive:    true,
		}
		if err := tx.Create(&donor).Error; err != nil {
			tx.Rollback()
			shared.SendAppError(c, http.StatusInternalServerError, "Failed to register donor")
			return
		}
	} else {
		// Update PAN or email if provided and wasn't set
		updated := false
		if donor.PANNumber == "" && req.PANNumber != "" {
			donor.PANNumber = strings.ToUpper(strings.TrimSpace(req.PANNumber))
			updated = true
		}
		if donor.Email == "" && req.Email != "" {
			donor.Email = strings.TrimSpace(req.Email)
			updated = true
		}
		if updated {
			_ = tx.Save(&donor)
		}
	}

	// 3. Derive Purpose
	category := strings.ToUpper(strings.TrimSpace(req.Category))
	if category == "" {
		category = "FOOD"
	}

	purpose := fmt.Sprintf("%s Donation", category)
	var schemeName string
	if req.SchemeID != nil && *req.SchemeID > 0 {
		var scheme models.Scheme
		if err := tx.First(&scheme, *req.SchemeID).Error; err == nil {
			schemeName = scheme.Name
			purpose = fmt.Sprintf("%s Sponsorship", scheme.Name)
		}
	} else if req.Reason != "" {
		purpose = fmt.Sprintf("%s - %s", category, req.Reason)
	}

	// 4. Generate Donation Number
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

	// Default created_by to 1 (Admin/System) for public app donations
	var systemUserID uint = 1
	var firstAdmin models.User
	if err := tx.Where("role = ?", models.RoleAdmin).Order("id asc").First(&firstAdmin).Error; err == nil {
		systemUserID = firstAdmin.ID
	}

	refNum := strings.TrimSpace(req.PaymentGatewayPaymentID)
	if refNum == "" {
		refNum = strings.TrimSpace(req.UPIReferenceNumber)
	}
	if refNum == "" {
		refNum = donationNumber
	}

	donation := models.Donation{
		DonationNumber:          donationNumber,
		DonorID:                 donor.ID,
		BusinessDate:            bizDate,
		Amount:                  req.Amount,
		PaymentMode:             models.PaymentModeBank,
		Purpose:                 purpose,
		Category:                category,
		Reason:                  req.Reason,
		Source:                  "MOBILE_APP",
		PaymentGatewayOrderID:   req.PaymentGatewayOrderID,
		PaymentGatewayPaymentID: req.PaymentGatewayPaymentID,
		VerificationStatus:      "VERIFIED",
		SchemeID:                req.SchemeID,
		EventType:               req.EventType,
		EventPersonName:         req.EventPersonName,
		EventDate:               eventDate,
		RelationshipToDonor:     req.RelationshipToDonor,
		BankAccountID:           &bankAccount.ID,
		ReferenceNumber:         refNum,
		Status:                  "ACTIVE",
		CreatedByID:             systemUserID,
	}

	if err := tx.Create(&donation).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to record donation: "+err.Error())
		return
	}

	// 5. Post Bank Transaction with source_channel = 'MOBILE_APP'
	bankTx := models.BankTransaction{
		BankAccountID:   bankAccount.ID,
		BusinessDate:    bizDate,
		TransactionType: "CREDIT",
		Amount:          req.Amount,
		Category:        "DONATION",
		ReferenceNumber: refNum,
		SourceType:      "DONATION",
		SourceID:        donation.ID,
		SourceChannel:   "MOBILE_APP",
		Description:     fmt.Sprintf("Mobile App Donation %s - %s (%s)", donation.DonationNumber, donor.FullName, purpose),
		CreatedByID:     systemUserID,
	}
	if err := tx.Create(&bankTx).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to record bank credit transaction")
		return
	}

	// Atomically increment current balance on the bank account
	if err := tx.Model(&bankAccount).Update("current_balance", gorm.Expr("current_balance + ?", req.Amount)).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to update bank balance")
		return
	}

	// 6. Generate Unique Voucher Code
	voucherNumber, err := shared.GenerateVoucherNumber(tx, bizDate)
	if err == nil {
		voucher := models.Voucher{
			VoucherNumber:    voucherNumber,
			VoucherType:      "DONATION_RECEIPT",
			BusinessDate:     bizDate,
			SourceType:       "DONATION",
			SourceID:         donation.ID,
			PayeeOrDonorName: donor.FullName,
			Amount:           req.Amount,
			AmountInWords:    shared.ConvertAmountToWords(req.Amount),
			CreatedByID:      systemUserID,
		}
		_ = tx.Create(&voucher)
	}

	tx.Commit()

	response := gin.H{
		"donation_id":        donation.ID,
		"donation_number":    donation.DonationNumber,
		"voucher_number":     voucherNumber,
		"donor_name":         donor.FullName,
		"donor_phone":        donor.Phone,
		"donor_pan":          donor.PANNumber,
		"amount":             req.Amount,
		"category":           category,
		"purpose":            purpose,
		"scheme_name":        schemeName,
		"business_date":      bizDate.Format("2006-01-02"),
		"bank_name":          bankAccount.BankName,
		"account_name":       bankAccount.AccountName,
		"reference_number":   refNum,
		"verification_status": "VERIFIED",
		"message":            "Donation received and verified successfully. Thank you for your support!",
	}

	shared.SendSuccess(c, http.StatusCreated, response)
}

// GetPublicDonorHistory returns past donations for a donor given their phone number
func (h *PublicDonationHandler) GetPublicDonorHistory(c *gin.Context) {
	phone := strings.TrimSpace(c.Query("phone"))
	if phone == "" {
		shared.SendAppError(c, http.StatusBadRequest, "Phone number is required")
		return
	}

	var donor models.Donor
	if err := database.DB.Where("phone = ?", phone).First(&donor).Error; err != nil {
		// Donor doesn't exist yet, return empty list
		shared.SendSuccess(c, http.StatusOK, []dto.PublicDonationItem{})
		return
	}

	var donations []models.Donation
	if err := database.DB.Preload("Scheme").Where("donor_id = ?", donor.ID).Order("id desc").Find(&donations).Error; err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to fetch donation history")
		return
	}

	items := make([]dto.PublicDonationItem, 0, len(donations))
	for _, d := range donations {
		schemeName := ""
		if d.Scheme != nil {
			schemeName = d.Scheme.Name
		}
		items = append(items, dto.PublicDonationItem{
			ID:                      d.ID,
			DonationNumber:          d.DonationNumber,
			BusinessDate:            d.BusinessDate.Format("2006-01-02"),
			Amount:                  d.Amount,
			Category:                d.Category,
			Purpose:                 d.Purpose,
			Reason:                  d.Reason,
			SchemeName:              schemeName,
			VerificationStatus:      d.VerificationStatus,
			PaymentGatewayPaymentID: d.PaymentGatewayPaymentID,
			UPIReferenceNumber:      d.ReferenceNumber,
			CreatedAt:               d.CreatedAt,
		})
	}

	shared.SendSuccess(c, http.StatusOK, gin.H{
		"donor": gin.H{
			"id":         donor.ID,
			"full_name":  donor.FullName,
			"phone":      donor.Phone,
			"email":      donor.Email,
			"pan_number": donor.PANNumber,
		},
		"donations": items,
	})
}

// GetPublicDashboard returns trust overview & placeholder statistics for the mobile donor app
func (h *PublicDonationHandler) GetPublicDashboard(c *gin.Context) {
	var totalDonationsCount int64
	var activeSchemesCount int64
	var totalDonorsCount int64

	database.DB.Model(&models.Donation{}).Count(&totalDonationsCount)
	database.DB.Model(&models.Scheme{}).Where("is_active = ?", true).Count(&activeSchemesCount)
	database.DB.Model(&models.Donor{}).Where("is_active = ?", true).Count(&totalDonorsCount)

	dashboardData := gin.H{
		"trust_name":        "Sri Agathiyar Sanmarga Charitable Trust",
		"tagline":           "Selfless Service & Annadhanam for All",
		"tax_exemption":     "80G Tax Exemption Certificate Available",
		"active_schemes":    activeSchemesCount,
		"total_donors":      totalDonorsCount,
		"total_donations":   totalDonationsCount,
		"annadhanam_daily":  "Over 500+ healthy meals served daily to needy pilgrims and devotees",
		"upcoming_occasions": []gin.H{
			{"title": "Pournami Annadhanam", "description": "Special full-moon mass feeding sponsorship", "date": "Every Full Moon Day"},
			{"title": "Amavasya Poojai & Meals", "description": "Monthly new-moon spiritual service and food distribution", "date": "Every New Moon Day"},
			{"title": "Guru Poojai", "description": "Annual divine master celebration with Annadhanam", "date": "Annual Celebration"},
		},
		"categories": []gin.H{
			{"key": "FOOD", "title": "Annadhanam / Food", "icon": "food", "description": "Sponsor Veg & Non-Veg Breakfast, Lunch, or Dinner meals"},
			{"key": "MEDICINE", "title": "Medical Relief", "icon": "medicine", "description": "Support life-saving medicines and healthcare for needy patients"},
			{"key": "EDUCATION", "title": "Education Aid", "icon": "education", "description": "Support school fees, notebooks, and uniforms for underprivileged students"},
			{"key": "GENERAL", "title": "General Donation", "icon": "general", "description": "Support temple upkeep, utilities, and daily charitable activities"},
			{"key": "OTHER", "title": "Special Purpose", "icon": "other", "description": "Contribute towards any specific custom cause with your prayer reason"},
		},
	}

	shared.SendSuccess(c, http.StatusOK, dashboardData)
}

// CreateRazorpayOrder creates an order on Razorpay servers and returns order details to the mobile app
func (h *PublicDonationHandler) CreateRazorpayOrder(c *gin.Context) {
	var req dto.CreateRazorpayOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.SendAppError(c, http.StatusBadRequest, err.Error())
		return
	}

	amtDecimal, err := decimal.NewFromString(strings.TrimSpace(req.Amount))
	if err != nil || amtDecimal.LessThanOrEqual(decimal.Zero) {
		shared.SendAppError(c, http.StatusBadRequest, "Donation amount must be greater than zero")
		return
	}

	// Razorpay amounts are represented in integer currency sub-units (paise for INR).
	amountInPaise := amtDecimal.Mul(decimal.NewFromInt(100)).IntPart()

	keyID := ""
	keySecret := ""
	if h.cfg != nil {
		keyID = h.cfg.RazorpayKeyID
		keySecret = h.cfg.RazorpayKeySecret
	}

	if keyID == "" || keySecret == "" {
		shared.SendAppError(c, http.StatusInternalServerError, "Razorpay API keys are not configured on server")
		return
	}

	receiptID := fmt.Sprintf("rcpt_%d", time.Now().UnixNano()/1e6)
	if len(receiptID) > 40 {
		receiptID = receiptID[:40]
	}

	orderPayload := map[string]interface{}{
		"amount":   amountInPaise,
		"currency": "INR",
		"receipt":  receiptID,
		"notes": map[string]string{
			"donor_name": strings.TrimSpace(req.DonorName),
			"phone":      strings.TrimSpace(req.Phone),
		},
	}

	jsonBytes, err := json.Marshal(orderPayload)
	if err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to encode Razorpay order request")
		return
	}

	httpReq, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, "https://api.razorpay.com/v1/orders", bytes.NewReader(jsonBytes))
	if err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to create HTTP request for Razorpay")
		return
	}

	httpReq.SetBasicAuth(keyID, keySecret)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		shared.SendAppError(c, http.StatusBadGateway, fmt.Sprintf("Failed to connect to Razorpay: %v", err))
		return
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		shared.SendAppError(c, http.StatusBadGateway, "Failed to read Razorpay response")
		return
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp struct {
			Error struct {
				Code        string `json:"code"`
				Description string `json:"description"`
			} `json:"error"`
		}
		_ = json.Unmarshal(bodyBytes, &errResp)
		msg := errResp.Error.Description
		if msg == "" {
			msg = string(bodyBytes)
		}
		shared.SendAppError(c, resp.StatusCode, fmt.Sprintf("Razorpay order creation rejected: %s", msg))
		return
	}

	var rzpOrder struct {
		ID       string `json:"id"`
		Amount   int64  `json:"amount"`
		Currency string `json:"currency"`
	}
	if err := json.Unmarshal(bodyBytes, &rzpOrder); err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to parse Razorpay order response")
		return
	}

	currency := rzpOrder.Currency
	if currency == "" {
		currency = "INR"
	}

	shared.SendSuccess(c, http.StatusOK, dto.RazorpayOrderResponse{
		RazorpayOrderID: rzpOrder.ID,
		Amount:          amountInPaise,
		Currency:        currency,
		KeyID:           keyID,
	})
}
