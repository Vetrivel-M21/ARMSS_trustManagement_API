package donation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"trust-management/backend/internal/config"
	"trust-management/backend/internal/database"
	"trust-management/backend/internal/dto"
	"trust-management/backend/internal/models"
	"trust-management/backend/internal/shared"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

var cashfreeHTTPClient = &http.Client{
	Timeout: 15 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     60 * time.Second,
	},
}

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

	trustHome := strings.TrimSpace(req.TrustHome)
	if trustHome == "" {
		trustHome = "OLD_AGE_HOME"
	}

	// Resolve the specific operational branch for this donation
	var targetBranch models.Branch
	switch strings.ToUpper(trustHome) {
	case "OLD_AGE_HOME", "OAH":
		if err := tx.Where("branch_code = ? OR id = ?", "OAH", 1).First(&targetBranch).Error; err != nil {
			targetBranch.ID = 1
			targetBranch.Name = "Old Age Home"
		}
	case "CHILDREN_HOME", "CH":
		if err := tx.Where("branch_code = ? OR id = ?", "CH", 2).First(&targetBranch).Error; err != nil {
			targetBranch.ID = 2
			targetBranch.Name = "Children Home"
		}
	case "ADOPTION_HOME", "CAH":
		if err := tx.Where("branch_code = ? OR id = ?", "CAH", 3).First(&targetBranch).Error; err != nil {
			targetBranch.ID = 3
			targetBranch.Name = "Children Adoption Home"
		}
	default:
		if err := tx.Where("branch_code = ?", strings.ToUpper(trustHome)).First(&targetBranch).Error; err != nil {
			targetBranch.ID = 1
			targetBranch.Name = "Old Age Home"
		}
	}

	donation := models.Donation{
		BranchID:                targetBranch.ID,
		DonationNumber:          donationNumber,
		DonorID:                 donor.ID,
		BusinessDate:            bizDate,
		Amount:                  req.Amount,
		PaymentMode:             models.PaymentModeBank,
		Purpose:                 purpose,
		Category:                category,
		Reason:                  req.Reason,
		Source:                  "MOBILE_APP",
		TrustHome:               trustHome,
		PaymentGatewayOrderID:   req.PaymentGatewayOrderID,
		PaymentGatewayPaymentID: req.PaymentGatewayPaymentID,
		VerificationStatus:      "PENDING",
		SchemeID:                req.SchemeID,
		EventType:               req.EventType,
		EventPersonName:         req.EventPersonName,
		EventDate:               eventDate,
		RelationshipToDonor:     req.RelationshipToDonor,
		BankAccountID:           &bankAccount.ID,
		ReferenceNumber:         refNum,
		AttachmentPath:          strings.TrimSpace(req.AttachmentPath),
		Status:                  "PENDING_VERIFICATION",
		CreatedByID:             systemUserID,
	}

	if err := tx.Create(&donation).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to record donation: "+err.Error())
		return
	}

	// 5. Generate Voucher code in PENDING status (Official receipt issued upon staff/admin approval)
	voucherNumber, err := shared.GenerateVoucherNumber(tx, bizDate)
	if err == nil {
		voucher := models.Voucher{
			BranchID:         targetBranch.ID,
			VoucherNumber:    voucherNumber,
			VoucherType:      "DONATION_RECEIPT",
			BusinessDate:     bizDate,
			SourceType:       "DONATION",
			SourceID:         donation.ID,
			PayeeOrDonorName: donor.FullName,
			Amount:           req.Amount,
			AmountInWords:    shared.ConvertAmountToWords(req.Amount),
			Status:           "PENDING",
			CreatedByID:      systemUserID,
		}
		_ = tx.Create(&voucher)
	}

	tx.Commit()

	response := gin.H{
		"donation_id":         donation.ID,
		"donation_number":     donation.DonationNumber,
		"voucher_number":      voucherNumber,
		"branch_id":           targetBranch.ID,
		"branch_name":         targetBranch.Name,
		"donor_name":          donor.FullName,
		"donor_phone":         donor.Phone,
		"donor_pan":           donor.PANNumber,
		"amount":              req.Amount,
		"category":            category,
		"purpose":             purpose,
		"scheme_name":         schemeName,
		"business_date":       bizDate.Format("2006-01-02"),
		"bank_name":           bankAccount.BankName,
		"account_name":        bankAccount.AccountName,
		"reference_number":    refNum,
		"verification_status": "PENDING",
		"message":             fmt.Sprintf("Donation submitted successfully! Your payment reference (%s) is pending verification by %s staff. Your official receipt will be unlocked once approved.", refNum, targetBranch.Name),
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
			AttachmentPath:          d.AttachmentPath,
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
		"trust_name":        "Nesakkaram Trust",
		"tagline":           "Service to Humanity • Elderly Care, Children Welfare & Adoption",
		"tax_exemption":     "Official Trust Receipt Issued",
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

// CreateGatewayOrder creates an online checkout order for Cashfree or Razorpay
func (h *PublicDonationHandler) CreateGatewayOrder(c *gin.Context) {
	var req dto.CreateGatewayOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.SendAppError(c, http.StatusBadRequest, err.Error())
		return
	}

	amtDecimal, err := decimal.NewFromString(strings.TrimSpace(req.Amount))
	if err != nil || amtDecimal.LessThanOrEqual(decimal.Zero) {
		shared.SendAppError(c, http.StatusBadRequest, "Donation amount must be greater than zero")
		return
	}

	gateway := strings.ToUpper(strings.TrimSpace(req.GatewayType))
	if gateway == "" {
		// Prefer Cashfree if keys are present
		if h.cfg != nil && h.cfg.CashfreeAppID != "" && h.cfg.CashfreeSecretKey != "" {
			gateway = "CASHFREE"
		} else {
			gateway = "RAZORPAY"
		}
	}

	if gateway == "CASHFREE" {
		cfAppID := ""
		cfSecret := ""
		cfEnv := "production"
		if h.cfg != nil {
			cfAppID = h.cfg.CashfreeAppID
			cfSecret = h.cfg.CashfreeSecretKey
			cfEnv = h.cfg.CashfreeEnv
		}
		if cfAppID == "" {
			cfAppID = os.Getenv("CASHFREE_APP_ID")
		}
		if cfSecret == "" {
			cfSecret = os.Getenv("CASHFREE_SECRET_KEY")
		}
		if cfAppID == "" || cfSecret == "" {
			shared.SendAppError(c, http.StatusInternalServerError, "Cashfree payment gateway credentials are not configured on server")
			return
		}

		baseURL := "https://api.cashfree.com/pg/orders"
		if cfEnv == "sandbox" || cfEnv == "test" {
			baseURL = "https://sandbox.cashfree.com/pg/orders"
		}

		orderID := fmt.Sprintf("order_%d", time.Now().UnixNano()/1e6)
		customerID := strings.TrimSpace(req.Phone)
		if customerID == "" {
			customerID = fmt.Sprintf("cust_%d", time.Now().UnixNano()/1e6)
		}
		custName := strings.TrimSpace(req.DonorName)
		if custName == "" {
			custName = "Anonymous Donor"
		}
		custPhone := strings.TrimSpace(req.Phone)
		if custPhone == "" {
			custPhone = "9999999999"
		}

		amtFloat, _ := amtDecimal.Float64()
		publicHost := "trustapi.arminfo.in"
		if c.Request.Host != "" && !strings.Contains(c.Request.Host, "127.0.0.1") && !strings.Contains(c.Request.Host, "localhost") {
			publicHost = c.Request.Host
		}

		cfPayload := map[string]interface{}{
			"order_id":       orderID,
			"order_amount":   amtFloat,
			"order_currency": "INR",
			"customer_details": map[string]string{
				"customer_id":    customerID,
				"customer_name":  custName,
				"customer_phone": custPhone,
				"customer_email": strings.TrimSpace(req.Email),
			},
			"order_meta": map[string]string{
				"return_url": fmt.Sprintf("https://%s/api/v1/public/donations/cashfree-return?order_id={order_id}", publicHost),
				"notify_url": "",
			},
		}

		jsonBytes, err := json.Marshal(cfPayload)
		if err != nil {
			shared.SendAppError(c, http.StatusInternalServerError, "Failed to encode Cashfree order payload")
			return
		}

		httpReq, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, baseURL, bytes.NewReader(jsonBytes))
		if err != nil {
			shared.SendAppError(c, http.StatusInternalServerError, "Failed to create HTTP request for Cashfree")
			return
		}

		httpReq.Header.Set("x-client-id", cfAppID)
		httpReq.Header.Set("x-client-secret", cfSecret)
		httpReq.Header.Set("x-api-version", "2023-08-01")
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := cashfreeHTTPClient.Do(httpReq)
		if err != nil {
			shared.SendAppError(c, http.StatusBadGateway, fmt.Sprintf("Failed to connect to Cashfree: %v", err))
			return
		}
		defer resp.Body.Close()

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			shared.SendAppError(c, http.StatusBadGateway, "Failed to read Cashfree response")
			return
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			shared.SendAppError(c, resp.StatusCode, fmt.Sprintf("Cashfree order creation error: %s", string(bodyBytes)))
			return
		}

		var cfResp struct {
			CFOrderID        string `json:"cf_order_id"`
			OrderID          string `json:"order_id"`
			PaymentSessionID string `json:"payment_session_id"`
			OrderStatus      string `json:"order_status"`
		}
		_ = json.Unmarshal(bodyBytes, &cfResp)

		checkoutURL := fmt.Sprintf("https://%s/api/v1/public/donations/cashfree-checkout?session_id=%s&order_id=%s&env=%s",
			publicHost, cfResp.PaymentSessionID, cfResp.OrderID, cfEnv)

		shared.SendSuccess(c, http.StatusOK, dto.GatewayOrderResponse{
			GatewayType:      "CASHFREE",
			OrderID:          cfResp.OrderID,
			PaymentSessionID: cfResp.PaymentSessionID,
			CheckoutURL:      checkoutURL,
			Amount:           amtDecimal.StringFixed(2),
			Currency:         "INR",
			AppID:            cfAppID,
			Environment:      cfEnv,
		})
		return
	}

	// Fallback to Razorpay order creation
	h.CreateRazorpayOrder(c)
}

// CashfreeCheckout renders the Cashfree JS SDK checkout wrapper
func (h *PublicDonationHandler) CashfreeCheckout(c *gin.Context) {
	sessionID := strings.TrimSpace(c.Query("session_id"))
	env := strings.TrimSpace(c.DefaultQuery("env", "production"))
	if strings.Contains(strings.ToLower(env), "sandbox") || strings.Contains(strings.ToLower(env), "test") {
		env = "sandbox"
	} else {
		env = "production"
	}

	sessJSON, _ := json.Marshal(sessionID)
	envJSON, _ := json.Marshal(env)

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Nesakkaram Charitable Trust - Gateway Payment</title>
    <script src="https://sdk.cashfree.com/js/v3/cashfree.js"></script>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            display: flex;
            align-items: center;
            justify-content: center;
            min-height: 100vh;
            margin: 0;
            background: #0f172a;
            color: #f8fafc;
        }
        .card {
            background: #1e293b;
            padding: 32px;
            border-radius: 16px;
            text-align: center;
            box-shadow: 0 10px 25px rgba(0,0,0,0.5);
            max-width: 360px;
            width: 90%%;
            border: 1px solid #334155;
        }
        .spinner {
            border: 4px solid rgba(255,255,255,0.1);
            border-left-color: #10b981;
            border-radius: 50%%;
            width: 44px;
            height: 44px;
            animation: spin 1s linear infinite;
            margin: 0 auto 20px;
        }
        @keyframes spin { to { transform: rotate(360deg); } }
        .btn {
            display: inline-block;
            background: #10b981;
            color: white;
            padding: 12px 24px;
            border-radius: 8px;
            text-decoration: none;
            font-weight: bold;
            margin-top: 16px;
            border: none;
            cursor: pointer;
        }
        .err {
            display: none;
            color: #ef4444;
            background: rgba(239, 68, 68, 0.1);
            border: 1px solid rgba(239, 68, 68, 0.2);
            padding: 10px;
            border-radius: 8px;
            margin-top: 14px;
            font-size: 13px;
            line-height: 1.4;
        }
    </style>
</head>
<body>
    <div class="card">
        <div id="spin" class="spinner"></div>
        <h3 style="margin: 0 0 8px 0; font-size: 18px;">Connecting to Secure Gateway</h3>
        <p style="margin: 0; font-size: 13px; color: #94a3b8;">Opening official Cashfree payment checkout with UPI, Cards &amp; NetBanking...</p>
        <div id="errorMsg" class="err"></div>
        <button id="payBtn" class="btn" style="display:none;" onclick="triggerPay()">Pay Online Now</button>
    </div>
    <script>
        const sessionId = %s;
        const env = %s;

        function triggerPay() {
            if (!sessionId) {
                showError("Payment session ID is missing. Please initiate the donation again from the app.");
                return;
            }
            try {
                if (typeof Cashfree === 'undefined') {
                    showError("Cashfree SDK failed to load. Please check your internet connection and refresh.");
                    return;
                }
                const cashfree = Cashfree({ mode: env === 'sandbox' || env === 'test' ? 'sandbox' : 'production' });
                cashfree.checkout({
                    paymentSessionId: sessionId,
                    redirectTarget: "_self"
                }).catch(function(err) {
                    console.error("Cashfree checkout error:", err);
                    showError(err && err.message ? err.message : "Payment checkout failed to open.");
                });
            } catch (e) {
                console.error("Checkout exception:", e);
                showError(e && e.message ? e.message : "Unable to initiate payment.");
            }
        }

        function showError(msg) {
            const errEl = document.getElementById('errorMsg');
            if (errEl) {
                errEl.innerText = msg;
                errEl.style.display = 'block';
            }
            const spin = document.getElementById('spin');
            if (spin) spin.style.display = 'none';
            const btn = document.getElementById('payBtn');
            if (btn) btn.style.display = 'inline-block';
        }

        window.onload = function() {
            triggerPay();
            setTimeout(function() {
                const btn = document.getElementById('payBtn');
                if (btn) btn.style.display = 'inline-block';
            }, 3000);
        };
    </script>
</body>
</html>`, string(sessJSON), string(envJSON))

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

// CashfreeReturn handles return from Cashfree hosted checkout
func (h *PublicDonationHandler) CashfreeReturn(c *gin.Context) {
	orderID := c.Query("order_id")

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Donation Payment Status</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            display: flex;
            align-items: center;
            justify-content: center;
            min-height: 100vh;
            margin: 0;
            background: #0f172a;
            color: #f8fafc;
        }
        .card {
            background: #1e293b;
            padding: 32px;
            border-radius: 16px;
            text-align: center;
            box-shadow: 0 10px 25px rgba(0,0,0,0.5);
            max-width: 380px;
            width: 90%%;
            border: 1px solid #334155;
        }
        .icon {
            font-size: 48px;
            margin-bottom: 16px;
        }
        h2 { margin: 0 0 10px 0; color: #10b981; font-size: 22px; }
        p { margin: 0 0 16px 0; font-size: 14px; color: #94a3b8; line-height: 1.5; }
    </style>
</head>
<body>
    <div class="card">
        <div class="icon">✅</div>
        <h2>Payment Processed</h2>
        <p>Order Reference: <strong>%s</strong></p>
        <p>Your online donation has been processed! Please return to the <strong>Nesakkaram Donor App</strong> to view your official receipt and voucher.</p>
    </div>
</body>
</html>`, orderID)

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

// VerifyAndIssueGatewayDonation completes the donation flow for payment gateway payments.
// Because the transaction is already successfully verified by the payment gateway,
// this endpoint immediately creates an ACTIVE donation, issues an official voucher and
// donation receipt, completely eliminating the need for manual admin confirmation.
func (h *PublicDonationHandler) VerifyAndIssueGatewayDonation(c *gin.Context) {
	var req dto.VerifyAndIssueGatewayDonationRequest
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

	orderID := strings.TrimSpace(req.OrderID)
	if orderID == "" {
		orderID = strings.TrimSpace(req.PaymentID)
	}
	if orderID == "" {
		shared.SendAppError(c, http.StatusBadRequest, "Order / Payment ID is required for gateway verification")
		return
	}

	gateway := strings.ToUpper(strings.TrimSpace(req.GatewayType))
	if gateway == "" {
		gateway = strings.ToUpper(strings.TrimSpace(req.GatewayName))
	}
	if gateway == "" && (strings.HasPrefix(orderID, "order_") || strings.HasPrefix(orderID, "CF_")) {
		gateway = "CASHFREE"
	}

	// 1. Verify Payment with Gateway in Real-Time
	if gateway == "CASHFREE" || strings.HasPrefix(orderID, "order_") {
		cfAppID := ""
		cfSecret := ""
		cfEnv := "production"
		if h.cfg != nil {
			cfAppID = h.cfg.CashfreeAppID
			cfSecret = h.cfg.CashfreeSecretKey
			cfEnv = h.cfg.CashfreeEnv
		}
		if cfAppID == "" {
			cfAppID = os.Getenv("CASHFREE_APP_ID")
		}
		if cfSecret == "" {
			cfSecret = os.Getenv("CASHFREE_SECRET_KEY")
		}
		if cfAppID == "" || cfSecret == "" {
			shared.SendAppError(c, http.StatusInternalServerError, "Cashfree payment gateway credentials are not configured on server")
			return
		}

		baseURL := "https://api.cashfree.com/pg/orders"
		if strings.Contains(strings.ToLower(cfEnv), "sandbox") || strings.Contains(strings.ToLower(cfEnv), "test") {
			baseURL = "https://sandbox.cashfree.com/pg/orders"
		}

		cfOrderURL := fmt.Sprintf("%s/%s", baseURL, orderID)
		httpReq, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, cfOrderURL, nil)
		if err != nil {
			shared.SendAppError(c, http.StatusInternalServerError, "Failed to create verification request")
			return
		}
		httpReq.Header.Set("x-client-id", cfAppID)
		httpReq.Header.Set("x-client-secret", cfSecret)
		httpReq.Header.Set("x-api-version", "2023-08-01")

		resp, err := cashfreeHTTPClient.Do(httpReq)
		if err != nil {
			shared.SendAppError(c, http.StatusBadGateway, fmt.Sprintf("Failed to contact payment gateway for verification: %v", err))
			return
		}
		defer resp.Body.Close()

		bodyBytes, _ := io.ReadAll(resp.Body)
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			shared.SendAppError(c, http.StatusBadRequest, fmt.Sprintf("Payment verification failed on gateway: %s", string(bodyBytes)))
			return
		}

		var cfStatus struct {
			OrderID     string  `json:"order_id"`
			OrderStatus string  `json:"order_status"` // PAID, ACTIVE, EXPIRED, TERMINATED
			OrderAmount float64 `json:"order_amount"`
		}
		if err := json.Unmarshal(bodyBytes, &cfStatus); err != nil {
			shared.SendAppError(c, http.StatusInternalServerError, "Failed to parse gateway verification response")
			return
		}

		if strings.ToUpper(cfStatus.OrderStatus) != "PAID" {
			shared.SendAppError(c, http.StatusBadRequest, fmt.Sprintf("Payment has not been completed. Gateway status is '%s'. Please complete the payment in the browser first.", cfStatus.OrderStatus))
			return
		}

		if cfStatus.OrderAmount > 0 {
			reqAmtFloat, _ := req.Amount.Float64()
			if math.Abs(cfStatus.OrderAmount-reqAmtFloat) > 0.01 {
				shared.SendAppError(c, http.StatusBadRequest, fmt.Sprintf("Payment amount mismatch. Gateway recorded ₹%.2f, but request specified ₹%.2f", cfStatus.OrderAmount, reqAmtFloat))
				return
			}
		}
	} else {
		shared.SendAppError(c, http.StatusBadRequest, "Direct receipt issuance requires real-time payment gateway verification. Please complete payment via an authorized gateway.")
		return
	}

	bizDate := shared.GetCurrentBusinessDate()

	// 2. Locate designated App Donation Bank Account
	var bankAccount models.BankAccount
	if err := database.DB.Where("is_app_donation_account = ? AND is_active = ?", true, true).First(&bankAccount).Error; err != nil {
		if err := database.DB.Where("is_active = ?", true).Order("id asc").First(&bankAccount).Error; err != nil {
			shared.SendAppError(c, http.StatusInternalServerError, "No active trust bank account configured to receive donations")
			return
		}
	}

	// 3. Prevent duplicate issuance if this order ID was already recorded
	var existingDonation models.Donation
	if err := database.DB.Where("reference_number = ?", orderID).First(&existingDonation).Error; err == nil {
		var existingVoucher models.Voucher
		_ = database.DB.Where("donation_id = ?", existingDonation.ID).First(&existingVoucher)
		var donor models.Donor
		_ = database.DB.First(&donor, existingDonation.DonorID)

		shared.SendSuccess(c, http.StatusOK, gin.H{
			"donation_id":         existingDonation.ID,
			"donation_number":     existingDonation.DonationNumber,
			"voucher_number":      existingVoucher.VoucherNumber,
			"donor_name":          donor.FullName,
			"donor_phone":         donor.Phone,
			"donor_pan":           donor.PANNumber,
			"amount":              existingDonation.Amount,
			"amount_in_words":     existingVoucher.AmountInWords,
			"category":            existingDonation.Category,
			"purpose":             existingDonation.Purpose,
			"business_date":       existingDonation.BusinessDate.Format("2006-01-02"),
			"bank_name":           bankAccount.BankName,
			"reference_number":    existingDonation.ReferenceNumber,
			"verification_status": string(existingDonation.Status),
			"is_verified":         true,
			"message":             "Official receipt already verified and issued.",
		})
		return
	}

	tx := database.DB.Begin()

	// 2. Find or create Donor by phone
	var donor models.Donor
	if err := tx.Where("phone = ?", phone).First(&donor).Error; err != nil {
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

	var systemUserID uint = 1
	var firstAdmin models.User
	if err := tx.Where("role = ?", models.RoleAdmin).Order("id asc").First(&firstAdmin).Error; err == nil {
		systemUserID = firstAdmin.ID
	}

	refNum := strings.TrimSpace(req.PaymentID)
	if refNum == "" {
		refNum = strings.TrimSpace(req.OrderID)
	}
	if refNum == "" {
		refNum = donationNumber
	}

	trustHome := strings.TrimSpace(req.TrustHome)
	if trustHome == "" {
		trustHome = "OLD_AGE_HOME"
	}

	// Resolve the operational branch
	var targetBranch models.Branch
	switch strings.ToUpper(trustHome) {
	case "OLD_AGE_HOME", "OAH":
		if err := tx.Where("branch_code = ? OR id = ?", "OAH", 1).First(&targetBranch).Error; err != nil {
			targetBranch.ID = 1
			targetBranch.Name = "Old Age Home"
		}
	case "CHILDREN_HOME", "CH":
		if err := tx.Where("branch_code = ? OR id = ?", "CH", 2).First(&targetBranch).Error; err != nil {
			targetBranch.ID = 2
			targetBranch.Name = "Children Home"
		}
	case "ADOPTION_HOME", "CAH":
		if err := tx.Where("branch_code = ? OR id = ?", "CAH", 3).First(&targetBranch).Error; err != nil {
			targetBranch.ID = 3
			targetBranch.Name = "Children Adoption Home"
		}
	default:
		if err := tx.Where("branch_code = ?", strings.ToUpper(trustHome)).First(&targetBranch).Error; err != nil {
			targetBranch.ID = 1
			targetBranch.Name = "Old Age Home"
		}
	}

	// 5. Automatically mark Donation as ACTIVE & VERIFIED
	now := time.Now()
	donation := models.Donation{
		BranchID:                targetBranch.ID,
		DonationNumber:          donationNumber,
		DonorID:                 donor.ID,
		BusinessDate:            bizDate,
		Amount:                  req.Amount,
		PaymentMode:             models.PaymentModeBank,
		Purpose:                 purpose,
		Category:                category,
		Reason:                  req.Reason,
		Source:                  "MOBILE_APP",
		TrustHome:               trustHome,
		PaymentGatewayOrderID:   req.OrderID,
		PaymentGatewayPaymentID: req.PaymentID,
		VerificationStatus:      "VERIFIED", // Automatically VERIFIED
		VerifiedByID:            &systemUserID,
		VerifiedAt:              &now,
		SchemeID:                req.SchemeID,
		EventType:               req.EventType,
		EventPersonName:         req.EventPersonName,
		EventDate:               eventDate,
		RelationshipToDonor:     req.RelationshipToDonor,
		BankAccountID:           &bankAccount.ID,
		ReferenceNumber:         refNum,
		Status:                  "ACTIVE", // Automatically ACTIVE
		CreatedByID:             systemUserID,
	}

	if err := tx.Create(&donation).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to record verified donation: "+err.Error())
		return
	}

	// 6. Automatically generate official Voucher in ISSUED status
	voucherNumber, err := shared.GenerateVoucherNumber(tx, bizDate)
	if err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to generate voucher number: "+err.Error())
		return
	}

	voucher := models.Voucher{
		BranchID:         targetBranch.ID,
		VoucherNumber:    voucherNumber,
		VoucherType:      "DONATION_RECEIPT",
		BusinessDate:     bizDate,
		SourceType:       "DONATION",
		SourceID:         donation.ID,
		PayeeOrDonorName: donor.FullName,
		Amount:           req.Amount,
		AmountInWords:    shared.ConvertAmountToWords(req.Amount),
		PaymentMode:      models.PaymentModeBank,
		BankAccountID:    &bankAccount.ID,
		Details:          fmt.Sprintf("Online Payment Gateway donation (%s)", refNum),
		Status:           "ISSUED", // Automatically ISSUED
		CreatedByID:      systemUserID,
		ApprovedByID:     &systemUserID,
		ApprovedAt:       &now,
	}
	if err := tx.Create(&voucher).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to issue voucher receipt: "+err.Error())
		return
	}

	// 7. Audit log the auto-verified transaction
	auditLog := models.AuditLog{
		UserID:     &systemUserID,
		Action:     "GATEWAY_DONATION_AUTO_VERIFIED",
		EntityName: "Donation",
		EntityID:   donation.ID,
		AfterData:  shared.JSONOrNull(donation),
		Reason:     fmt.Sprintf("Payment Gateway confirmed payment %s", refNum),
		IPAddress:  c.ClientIP(),
	}
	_ = tx.Create(&auditLog)

	tx.Commit()

	response := gin.H{
		"donation_id":         donation.ID,
		"donation_number":     donation.DonationNumber,
		"voucher_number":      voucherNumber,
		"branch_id":           targetBranch.ID,
		"branch_name":         targetBranch.Name,
		"donor_name":          donor.FullName,
		"donor_phone":         donor.Phone,
		"donor_pan":           donor.PANNumber,
		"amount":              req.Amount,
		"amount_in_words":     voucher.AmountInWords,
		"category":            category,
		"purpose":             purpose,
		"scheme_name":         schemeName,
		"business_date":       bizDate.Format("2006-01-02"),
		"bank_name":           bankAccount.BankName,
		"account_name":        bankAccount.AccountName,
		"reference_number":    refNum,
		"verification_status": "VERIFIED",
		"is_verified":         true,
		"message":             fmt.Sprintf("Thank you! Your donation of Rs. %s has been successfully received and verified via payment gateway. Your official receipt (Voucher #%s) is issued.", req.Amount.StringFixed(2), voucherNumber),
	}

	shared.SendSuccess(c, http.StatusCreated, response)
}

