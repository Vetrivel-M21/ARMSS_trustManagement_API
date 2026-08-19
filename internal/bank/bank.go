package bank

import (
	"encoding/json"
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
	"gorm.io/gorm/clause"
)

type BankHandler struct{}

func NewBankHandler() *BankHandler {
	return &BankHandler{}
}

// GetBankAccounts returns list of bank accounts
func (h *BankHandler) GetBankAccounts(c *gin.Context) {
	var accounts []models.BankAccount
	if err := database.DB.Order("id desc").Find(&accounts).Error; err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to fetch bank accounts")
		return
	}
	shared.SendSuccess(c, http.StatusOK, accounts)
}

// GetActiveBankAccounts returns active bank accounts for dropdowns
func (h *BankHandler) GetActiveBankAccounts(c *gin.Context) {
	var accounts []models.BankAccount
	if err := database.DB.Where("is_active = ?", true).Order("bank_name asc").Find(&accounts).Error; err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to fetch active bank accounts")
		return
	}
	shared.SendSuccess(c, http.StatusOK, accounts)
}

// CreateBankAccount handles Admin adding a new bank account
func (h *BankHandler) CreateBankAccount(c *gin.Context) {
	var req dto.CreateBankAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.SendAppError(c, http.StatusBadRequest, err.Error())
		return
	}

	userIDVal, _ := c.Get("userID")
	userID, _ := userIDVal.(uint)

	account := models.BankAccount{
		BankName:            req.BankName,
		AccountName:         req.AccountName,
		AccountNumberMasked: req.AccountNumberMasked,
		IFSCCode:            req.IFSCCode,
		Branch:              req.Branch,
		Location:            req.Location,
		OpeningBalance:      req.OpeningBalance,
		QRCodePath:          req.QRCodePath,
		CurrentBalance:      req.OpeningBalance, // Initialized to opening balance
		IsActive:            true,
	}

	tx := database.DB.Begin()
	if err := tx.Create(&account).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to create bank account: "+err.Error())
		return
	}

	audit := models.AuditLog{
		UserID:     &userID,
		Action:     "BANK_ACCOUNT_CREATED",
		EntityName: "BankAccount",
		EntityID:   account.ID,
		BeforeData: shared.JSONOrNull(nil),
		AfterData:  shared.JSONOrNull(account),
		IPAddress:  c.ClientIP(),
	}
	if err := tx.Create(&audit).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to record audit log")
		return
	}
	tx.Commit()

	shared.SendSuccess(c, http.StatusCreated, account)
}

// UpdateBankAccount handles Admin modifying bank details
func (h *BankHandler) UpdateBankAccount(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		shared.SendAppError(c, http.StatusBadRequest, "Invalid bank account ID")
		return
	}

	var account models.BankAccount
	if err := database.DB.First(&account, id).Error; err != nil {
		shared.SendAppError(c, http.StatusNotFound, "Bank account not found")
		return
	}

	var req dto.UpdateBankAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.SendAppError(c, http.StatusBadRequest, err.Error())
		return
	}

	beforeData, _ := json.Marshal(account)

	if req.BankName != "" {
		account.BankName = req.BankName
	}
	if req.AccountName != "" {
		account.AccountName = req.AccountName
	}
	if req.AccountNumberMasked != "" {
		account.AccountNumberMasked = req.AccountNumberMasked
	}
	if req.IFSCCode != "" {
		account.IFSCCode = req.IFSCCode
	}
	if req.Branch != "" {
		account.Branch = req.Branch
	}
	if req.Location != "" {
		account.Location = req.Location
	}
	if req.QRCodePath != "" {
		account.QRCodePath = req.QRCodePath
	}
	if req.IsActive != nil {
		account.IsActive = *req.IsActive
	}

	userIDVal, _ := c.Get("userID")
	userID, _ := userIDVal.(uint)

	tx := database.DB.Begin()
	if err := tx.Save(&account).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to update bank account")
		return
	}

	afterData, _ := json.Marshal(account)
	audit := models.AuditLog{
		UserID:     &userID,
		Action:     "BANK_ACCOUNT_UPDATED",
		EntityName: "BankAccount",
		EntityID:   account.ID,
		BeforeData: string(beforeData),
		AfterData:  string(afterData),
		IPAddress:  c.ClientIP(),
	}
	if err := tx.Create(&audit).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to record audit log")
		return
	}
	tx.Commit()

	shared.SendSuccess(c, http.StatusOK, account)
}

// BankTransactionDetail is one ledger row enriched with who/why details from
// its source Donation or Expense — the ledger alone only carries a generic
// category, not who gave/paid or why.
type BankTransactionDetail struct {
	models.BankTransaction
	DonorName      string `json:"donor_name,omitempty"`
	DonorPhone     string `json:"donor_phone,omitempty"`
	DonationNumber string `json:"donation_number,omitempty"`
	Purpose        string `json:"purpose,omitempty"`
	SchemeName     string `json:"scheme_name,omitempty"`
	FoodType       string `json:"food_type,omitempty"`
	MealType       string `json:"meal_type,omitempty"`
	PayeeName      string `json:"payee_name,omitempty"`
	ExpenseNumber  string `json:"expense_number,omitempty"`
	AttachmentPath string `json:"attachment_path,omitempty"`
}

// GetBankTransactions returns the transaction ledger for a bank account,
// enriched with donor/payee/purpose details from the underlying Donation or
// Expense record so the ledger can answer "who gave/paid this, and why."
func (h *BankHandler) GetBankTransactions(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		shared.SendAppError(c, http.StatusBadRequest, "Invalid bank account ID")
		return
	}

	var txs []models.BankTransaction
	if err := database.DB.Where("bank_account_id = ?", id).Order("created_at desc").Find(&txs).Error; err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to fetch bank transactions")
		return
	}

	var donationIDs, expenseIDs []uint
	for _, t := range txs {
		switch t.SourceType {
		case "DONATION":
			donationIDs = append(donationIDs, t.SourceID)
		case "EXPENSE":
			expenseIDs = append(expenseIDs, t.SourceID)
		}
	}

	donationByID := make(map[uint]models.Donation)
	if len(donationIDs) > 0 {
		var donations []models.Donation
		database.DB.Preload("Donor").Preload("Scheme").Where("id IN ?", donationIDs).Find(&donations)
		for _, d := range donations {
			donationByID[d.ID] = d
		}
	}

	expenseByID := make(map[uint]models.Expense)
	if len(expenseIDs) > 0 {
		var expenses []models.Expense
		database.DB.Where("id IN ?", expenseIDs).Find(&expenses)
		for _, e := range expenses {
			expenseByID[e.ID] = e
		}
	}

	result := make([]BankTransactionDetail, 0, len(txs))
	for _, t := range txs {
		row := BankTransactionDetail{BankTransaction: t}
		switch t.SourceType {
		case "DONATION":
			if d, ok := donationByID[t.SourceID]; ok {
				row.DonationNumber = d.DonationNumber
				row.Purpose = d.Purpose
				row.AttachmentPath = d.AttachmentPath
				if d.Donor != nil {
					row.DonorName = d.Donor.FullName
					row.DonorPhone = d.Donor.Phone
				}
				if d.Scheme != nil {
					row.SchemeName = d.Scheme.Name
					row.FoodType = d.Scheme.FoodType
					row.MealType = d.Scheme.MealType
				}
			}
		case "EXPENSE":
			if e, ok := expenseByID[t.SourceID]; ok {
				row.ExpenseNumber = e.ExpenseNumber
				row.PayeeName = e.PayeeName
				row.Purpose = e.Description
				row.AttachmentPath = e.AttachmentPath
			}
		}
		result = append(result, row)
	}

	shared.SendSuccess(c, http.StatusOK, result)
}

// BankBreakdownEntry is one underlying transaction inside a breakdown group —
// e.g. one specific donor's donation, or one specific expense payment — shown
// in the "View Details" drilldown since several donors/payees often share the
// same purpose/category on the same day.
type BankBreakdownEntry struct {
	Name            string          `json:"name"`
	Purpose         string          `json:"purpose"`
	ReferenceNumber string          `json:"reference_number"`
	Amount          decimal.Decimal `json:"amount"`
	BusinessDate    string          `json:"business_date"`
}

// BankBreakdownRow is one grouped line item in a credit or debit breakdown —
// e.g. a donation purpose/scheme name for credits, or an expense category for
// debits — carrying the individual entries that make it up.
type BankBreakdownRow struct {
	Label       string               `json:"label"`
	FoodType    string               `json:"food_type,omitempty"`
	MealType    string               `json:"meal_type,omitempty"`
	Category    string               `json:"category,omitempty"`
	Count       int64                `json:"count"`
	TotalAmount decimal.Decimal      `json:"total_amount"`
	Entries     []BankBreakdownEntry `json:"entries"`
}

// BankAccountBreakdown is one bank account's opening balance and credit/debit
// totals for a single business date — used only for the per-account
// reconciliation/closing table. The credit/debit breakdown rows themselves
// are returned flattened across all banks (see credit_breakdown/
// debit_breakdown in GetBankDaySummary's response), not nested here.
type BankAccountBreakdown struct {
	BankAccountID  uint            `json:"bank_account_id"`
	BankName       string          `json:"bank_name"`
	AccountName    string          `json:"account_name"`
	OpeningBalance decimal.Decimal `json:"opening_balance"`
	TotalCredits   decimal.Decimal `json:"total_credits"`
	TotalDebits    decimal.Decimal `json:"total_debits"`
}

type bankCreditRawRow struct {
	BankAccountID   uint
	Amount          decimal.Decimal
	BusinessDate    time.Time
	SourceType      string
	TxReference     string
	SchemeName      string
	FoodType        string
	MealType        string
	Category        string
	Purpose         string
	ReferenceNumber string
	DonorName       string
}

type bankDebitRawRow struct {
	BankAccountID uint
	Amount        decimal.Decimal
	BusinessDate  time.Time
	SourceType    string
	TxReference   string
	Category      string
	PayeeName     string
	Description   string
	ExpenseRef    string
}

type breakdownGroupKey struct {
	label    string
	foodType string
	mealType string
}

// GetBankDaySummary returns a per-bank-account breakdown for one business
// date — each bank's opening balance, and its credits grouped by the
// donation's purpose/scheme (so "why was this credited" is visible instead of
// the generic "DONATION" category stored on the ledger row) and debits
// grouped by category — plus portfolio-wide totals for the footer. Each group
// carries its own entry list (donor/payee, raw purpose, reference number) for
// a "View Details" drilldown, since several donors often share one purpose on
// the same day. This is a read-only aggregate — the per-account
// reconciliation/closing mechanics (GetBankClosingStatus/CloseBankDay) are
// unaffected.
func (h *BankHandler) GetBankDaySummary(c *gin.Context) {
	dateStr := c.DefaultQuery("date", shared.GetCurrentBusinessDate().Format("2006-01-02"))

	var accounts []models.BankAccount
	if err := database.DB.Find(&accounts).Error; err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to fetch bank accounts")
		return
	}

	var creditRaw []bankCreditRawRow
	database.DB.Table("bank_transactions").
		Select(`bank_transactions.bank_account_id as bank_account_id,
			bank_transactions.amount as amount,
			bank_transactions.business_date as business_date,
			bank_transactions.source_type as source_type,
			bank_transactions.reference_number as tx_reference,
			COALESCE(schemes.name, '') as scheme_name,
			COALESCE(schemes.food_type, '') as food_type,
			COALESCE(schemes.meal_type, '') as meal_type,
			COALESCE(schemes.category, '') as category,
			COALESCE(donations.purpose, '') as purpose,
			COALESCE(donations.reference_number, '') as reference_number,
			COALESCE(donors.full_name, '') as donor_name`).
		Joins("LEFT JOIN donations ON bank_transactions.source_type = 'DONATION' AND bank_transactions.source_id = donations.id").
		Joins("LEFT JOIN schemes ON schemes.id = donations.scheme_id").
		Joins("LEFT JOIN donors ON donors.id = donations.donor_id").
		Where("bank_transactions.transaction_type = ? AND bank_transactions.business_date = ?", "CREDIT", dateStr).
		Scan(&creditRaw)

	var debitRaw []bankDebitRawRow
	database.DB.Table("bank_transactions").
		Select(`bank_transactions.bank_account_id as bank_account_id,
			bank_transactions.amount as amount,
			bank_transactions.business_date as business_date,
			bank_transactions.source_type as source_type,
			bank_transactions.reference_number as tx_reference,
			bank_transactions.category as category,
			COALESCE(expenses.payee_name, '') as payee_name,
			COALESCE(expenses.description, '') as description,
			COALESCE(expenses.reference_number, '') as expense_ref`).
		Joins("LEFT JOIN expenses ON bank_transactions.source_type = 'EXPENSE' AND bank_transactions.source_id = expenses.id").
		Where("bank_transactions.transaction_type = ? AND bank_transactions.business_date = ?", "DEBIT", dateStr).
		Scan(&debitRaw)

	acctCreditTotal := make(map[uint]decimal.Decimal)
	acctDebitTotal := make(map[uint]decimal.Decimal)

	creditGroups := make(map[breakdownGroupKey]*BankBreakdownRow)
	var creditOrder []breakdownGroupKey
	for _, r := range creditRaw {
		// The donor's own purpose text takes priority over the scheme name —
		// schemes now get auto-generated names (e.g. "Veg Breakfast") that are
		// less meaningful here than what the donor actually typed. Mirrors the
		// cash-side aggregation in internal/cash/cash.go.
		label := r.Purpose
		if label == "" {
			if r.SchemeName != "" {
				label = r.SchemeName
			} else if r.SourceType == "INTER_BANK_TRANSFER" {
				label = "Inter-Bank Transfer"
			} else {
				label = "Other Bank Credit"
			}
		}
		key := breakdownGroupKey{label: label, foodType: r.FoodType, mealType: r.MealType}
		grp, ok := creditGroups[key]
		if !ok {
			grp = &BankBreakdownRow{Label: label, FoodType: r.FoodType, MealType: r.MealType, Category: r.Category, Entries: []BankBreakdownEntry{}}
			creditGroups[key] = grp
			creditOrder = append(creditOrder, key)
		}
		grp.Count++
		grp.TotalAmount = grp.TotalAmount.Add(r.Amount)

		refNum := r.ReferenceNumber
		if refNum == "" {
			refNum = r.TxReference
		}
		name := r.DonorName
		if name == "" {
			name = "—"
		}
		purpose := r.Purpose
		if purpose == "" {
			purpose = label
		}
		grp.Entries = append(grp.Entries, BankBreakdownEntry{Name: name, Purpose: purpose, ReferenceNumber: refNum, Amount: r.Amount, BusinessDate: r.BusinessDate.Format("2006-01-02")})

		acctCreditTotal[r.BankAccountID] = acctCreditTotal[r.BankAccountID].Add(r.Amount)
	}

	debitGroups := make(map[breakdownGroupKey]*BankBreakdownRow)
	var debitOrder []breakdownGroupKey
	for _, r := range debitRaw {
		label := r.Category
		key := breakdownGroupKey{label: label}
		grp, ok := debitGroups[key]
		if !ok {
			grp = &BankBreakdownRow{Label: label, Entries: []BankBreakdownEntry{}}
			debitGroups[key] = grp
			debitOrder = append(debitOrder, key)
		}
		grp.Count++
		grp.TotalAmount = grp.TotalAmount.Add(r.Amount)

		refNum := r.ExpenseRef
		if refNum == "" {
			refNum = r.TxReference
		}
		name := r.PayeeName
		if name == "" {
			if r.SourceType == "INTER_BANK_TRANSFER" {
				name = "Inter-Bank Transfer"
			} else {
				name = "—"
			}
		}
		purpose := r.Description
		if purpose == "" {
			purpose = label
		}
		grp.Entries = append(grp.Entries, BankBreakdownEntry{Name: name, Purpose: purpose, ReferenceNumber: refNum, Amount: r.Amount, BusinessDate: r.BusinessDate.Format("2006-01-02")})

		acctDebitTotal[r.BankAccountID] = acctDebitTotal[r.BankAccountID].Add(r.Amount)
	}

	// Flattened across all banks — the Credits/Debits cards no longer group by
	// bank account, only the per-account reconciliation table below does.
	creditBreakdown := make([]BankBreakdownRow, 0, len(creditOrder))
	for _, key := range creditOrder {
		creditBreakdown = append(creditBreakdown, *creditGroups[key])
	}
	debitBreakdown := make([]BankBreakdownRow, 0, len(debitOrder))
	for _, key := range debitOrder {
		debitBreakdown = append(debitBreakdown, *debitGroups[key])
	}

	openingTotal := decimal.Zero
	totalCredits := decimal.Zero
	totalDebits := decimal.Zero
	banks := make([]BankAccountBreakdown, 0, len(accounts))

	for _, a := range accounts {
		opening := shared.GetOpeningBankBalance(database.DB, a.ID, dateStr, a.OpeningBalance)
		acctCredits := acctCreditTotal[a.ID]
		acctDebits := acctDebitTotal[a.ID]

		openingTotal = openingTotal.Add(opening)
		totalCredits = totalCredits.Add(acctCredits)
		totalDebits = totalDebits.Add(acctDebits)

		banks = append(banks, BankAccountBreakdown{
			BankAccountID:  a.ID,
			BankName:       a.BankName,
			AccountName:    a.AccountName,
			OpeningBalance: opening,
			TotalCredits:   acctCredits,
			TotalDebits:    acctDebits,
		})
	}

	shared.SendSuccess(c, http.StatusOK, gin.H{
		"business_date":    dateStr,
		"opening_total":    openingTotal,
		"total_credits":    totalCredits,
		"total_debits":     totalDebits,
		"net_position":     openingTotal.Add(totalCredits).Sub(totalDebits),
		"banks":            banks,
		"credit_breakdown": creditBreakdown,
		"debit_breakdown":  debitBreakdown,
	})
}

// recomputeBankFigures sums bank_transactions for one account/date and derives
// the expected closing balance from the supplied opening balance.
func recomputeBankFigures(dateStr string, bankAccountID uint, opening decimal.Decimal) (credits, debits, expected decimal.Decimal) {
	database.DB.Model(&models.BankTransaction{}).
		Where("bank_account_id = ? AND business_date = ? AND transaction_type = ?", bankAccountID, dateStr, "CREDIT").
		Select("COALESCE(SUM(amount), 0)").Scan(&credits)
	database.DB.Model(&models.BankTransaction{}).
		Where("bank_account_id = ? AND business_date = ? AND transaction_type = ?", bankAccountID, dateStr, "DEBIT").
		Select("COALESCE(SUM(amount), 0)").Scan(&debits)
	expected = opening.Add(credits).Sub(debits)
	return
}

// GetBankClosingStatus returns the reconciliation figures for one bank account/date,
// including the existing closing record if that date has already been closed.
func (h *BankHandler) GetBankClosingStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		shared.SendAppError(c, http.StatusBadRequest, "Invalid bank account ID")
		return
	}

	var account models.BankAccount
	if err := database.DB.First(&account, id).Error; err != nil {
		shared.SendAppError(c, http.StatusNotFound, "Bank account not found")
		return
	}

	dateStr := c.DefaultQuery("date", shared.GetCurrentBusinessDate().Format("2006-01-02"))

	var existing models.BankClosing
	closedErr := database.DB.Where("bank_account_id = ? AND business_date = ?", id, dateStr).First(&existing).Error
	if closedErr == nil {
		shared.SendSuccess(c, http.StatusOK, gin.H{
			"business_date":    dateStr,
			"bank_account_id":  account.ID,
			"status":           "CLOSED",
			"opening_balance":  existing.OpeningBalance,
			"total_credits":    existing.TotalCredits,
			"total_debits":     existing.TotalDebits,
			"expected_closing": existing.ExpectedClosing,
			"actual_closing":   existing.ActualClosing,
			"difference":       existing.Difference,
		})
		return
	}

	opening := shared.GetOpeningBankBalance(database.DB, uint(id), dateStr, account.OpeningBalance)
	credits, debits, expected := recomputeBankFigures(dateStr, uint(id), opening)

	shared.SendSuccess(c, http.StatusOK, gin.H{
		"business_date":    dateStr,
		"bank_account_id":  account.ID,
		"status":           "OPEN",
		"opening_balance":  opening,
		"total_credits":    credits,
		"total_debits":     debits,
		"expected_closing": expected,
	})
}

// CloseBankDay reconciles and locks one bank account for a business date. Per
// the same financial-integrity rule as cash closing (spec section 19/42), a
// non-zero difference between expected and actual closing balance blocks the
// close — the discrepancy must be investigated, never silently absorbed.
func (h *BankHandler) CloseBankDay(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		shared.SendAppError(c, http.StatusBadRequest, "Invalid bank account ID")
		return
	}

	var account models.BankAccount
	if err := database.DB.First(&account, id).Error; err != nil {
		shared.SendAppError(c, http.StatusNotFound, "Bank account not found")
		return
	}

	var req dto.SubmitBankClosingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.SendAppError(c, http.StatusBadRequest, err.Error())
		return
	}

	var existing models.BankClosing
	if err := database.DB.Where("bank_account_id = ? AND business_date = ?", id, req.BusinessDate).First(&existing).Error; err == nil {
		shared.SendAppError(c, http.StatusBadRequest, "This bank account is already CLOSED for this business date.")
		return
	}

	opening := shared.GetOpeningBankBalance(database.DB, uint(id), req.BusinessDate, account.OpeningBalance)
	credits, debits, expected := recomputeBankFigures(req.BusinessDate, uint(id), opening)
	difference := expected.Sub(req.ActualClosing)

	if !difference.IsZero() {
		shared.SendError(c, http.StatusConflict, "BANK_MISMATCH",
			fmt.Sprintf("Expected closing balance (₹%s) does not match actual closing balance (₹%s) for %s. Difference: ₹%s.",
				expected.StringFixed(2), req.ActualClosing.StringFixed(2), account.BankName, difference.StringFixed(2)))
		return
	}

	bankClosing := models.BankClosing{
		BusinessDate:    mustParseDate(req.BusinessDate),
		BankAccountID:   uint(id),
		OpeningBalance:  opening,
		TotalCredits:    credits,
		TotalDebits:     debits,
		ExpectedClosing: expected,
		ActualClosing:   req.ActualClosing,
		Difference:      difference,
		Status:          models.DayStatusClosed,
	}

	userIDVal, _ := c.Get("userID")
	userID, _ := userIDVal.(uint)

	tx := database.DB.Begin()
	if err := tx.Create(&bankClosing).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to record bank closing")
		return
	}

	audit := models.AuditLog{
		UserID:     &userID,
		Action:     "BANK_CLOSING_LOCKED",
		EntityName: "BankClosing",
		EntityID:   bankClosing.ID,
		BeforeData: shared.JSONOrNull(nil),
		AfterData:  shared.JSONOrNull(bankClosing),
		Reason:     fmt.Sprintf("%s closed for %s. Expected: ₹%s, Actual: ₹%s", account.BankName, req.BusinessDate, expected.StringFixed(2), req.ActualClosing.StringFixed(2)),
		IPAddress:  c.ClientIP(),
	}
	if err := tx.Create(&audit).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to record audit log")
		return
	}
	tx.Commit()

	shared.SendSuccess(c, http.StatusOK, gin.H{
		"message":      fmt.Sprintf("%s successfully CLOSED for %s.", account.BankName, req.BusinessDate),
		"bank_closing": bankClosing,
	})
}

// CloseAllBankDays closes every listed bank account for one business date in
// a single all-or-nothing action. Every account's expected-vs-statement
// figures are validated FIRST; if any one mismatches or is already closed,
// the whole request is rejected and nothing is written — a real error in one
// account must never be masked by, or allowed to coexist with, successful
// closes on the others.
func (h *BankHandler) CloseAllBankDays(c *gin.Context) {
	var req dto.CloseAllBanksRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.SendAppError(c, http.StatusBadRequest, err.Error())
		return
	}
	if len(req.Closings) == 0 {
		shared.SendAppError(c, http.StatusBadRequest, "No bank accounts to close.")
		return
	}

	type prepared struct {
		account       models.BankAccount
		opening       decimal.Decimal
		credits       decimal.Decimal
		debits        decimal.Decimal
		expected      decimal.Decimal
		actualClosing decimal.Decimal
		difference    decimal.Decimal
	}

	var toClose []prepared
	var mismatches []string

	for _, entry := range req.Closings {
		var account models.BankAccount
		if err := database.DB.First(&account, entry.BankAccountID).Error; err != nil {
			shared.SendAppError(c, http.StatusBadRequest, fmt.Sprintf("Bank account %d not found.", entry.BankAccountID))
			return
		}

		var existing models.BankClosing
		if err := database.DB.Where("bank_account_id = ? AND business_date = ?", entry.BankAccountID, req.BusinessDate).First(&existing).Error; err == nil {
			mismatches = append(mismatches, fmt.Sprintf("%s is already CLOSED for this date", account.BankName))
			continue
		}

		opening := shared.GetOpeningBankBalance(database.DB, entry.BankAccountID, req.BusinessDate, account.OpeningBalance)
		credits, debits, expected := recomputeBankFigures(req.BusinessDate, entry.BankAccountID, opening)
		difference := expected.Sub(entry.ActualClosing)

		if !difference.IsZero() {
			mismatches = append(mismatches, fmt.Sprintf("%s: expected ₹%s vs statement ₹%s (diff ₹%s)",
				account.BankName, expected.StringFixed(2), entry.ActualClosing.StringFixed(2), difference.StringFixed(2)))
			continue
		}

		toClose = append(toClose, prepared{
			account: account, opening: opening, credits: credits, debits: debits,
			expected: expected, actualClosing: entry.ActualClosing, difference: difference,
		})
	}

	if len(mismatches) > 0 {
		shared.SendError(c, http.StatusConflict, "BANK_MISMATCH",
			"Closing rejected — no accounts were closed. Fix the following and retry: "+strings.Join(mismatches, "; "))
		return
	}

	userIDVal, _ := c.Get("userID")
	userID, _ := userIDVal.(uint)
	bizDate := mustParseDate(req.BusinessDate)

	tx := database.DB.Begin()
	created := make([]models.BankClosing, 0, len(toClose))
	for _, p := range toClose {
		bankClosing := models.BankClosing{
			BusinessDate:    bizDate,
			BankAccountID:   p.account.ID,
			OpeningBalance:  p.opening,
			TotalCredits:    p.credits,
			TotalDebits:     p.debits,
			ExpectedClosing: p.expected,
			ActualClosing:   p.actualClosing,
			Difference:      p.difference,
			Status:          models.DayStatusClosed,
		}
		if err := tx.Create(&bankClosing).Error; err != nil {
			tx.Rollback()
			shared.SendAppError(c, http.StatusInternalServerError, "Failed to record bank closing for "+p.account.BankName)
			return
		}
		created = append(created, bankClosing)
	}

	audit := models.AuditLog{
		UserID:     &userID,
		Action:     "BANK_CLOSING_LOCKED_ALL",
		EntityName: "BankClosing",
		EntityID:   created[0].ID,
		BeforeData: shared.JSONOrNull(nil),
		AfterData:  shared.JSONOrNull(created),
		Reason:     fmt.Sprintf("Closed %d bank account(s) for %s in one combined action", len(created), req.BusinessDate),
		IPAddress:  c.ClientIP(),
	}
	if err := tx.Create(&audit).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to record audit log")
		return
	}
	tx.Commit()

	shared.SendSuccess(c, http.StatusOK, gin.H{
		"message":       fmt.Sprintf("%d bank account(s) successfully CLOSED for %s.", len(created), req.BusinessDate),
		"bank_closings": created,
	})
}

func mustParseDate(s string) time.Time {
	t, err := shared.ParseBusinessDate(s)
	if err != nil {
		return shared.GetCurrentBusinessDate()
	}
	return t
}

// TransferFunds performs an inter-bank account transfer (debit source, credit destination)
func (h *BankHandler) TransferFunds(c *gin.Context) {
	var req dto.InterBankTransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.SendAppError(c, http.StatusBadRequest, err.Error())
		return
	}

	if req.FromAccountID == req.ToAccountID {
		shared.SendAppError(c, http.StatusBadRequest, "Source and destination bank accounts must be different.")
		return
	}

	if req.Amount.LessThanOrEqual(decimal.Zero) {
		shared.SendAppError(c, http.StatusBadRequest, "Transfer amount must be greater than zero.")
		return
	}

	userIDVal, _ := c.Get("userID")
	userID := userIDVal.(uint)
	bizDate := shared.GetCurrentBusinessDate()

	var fromAcc models.BankAccount
	if err := database.DB.First(&fromAcc, req.FromAccountID).Error; err != nil {
		shared.SendAppError(c, http.StatusBadRequest, "Source bank account not found.")
		return
	}

	if fromAcc.CurrentBalance.LessThan(req.Amount) {
		shared.SendAppError(c, http.StatusBadRequest, "Insufficient funds in source bank account.")
		return
	}

	var toAcc models.BankAccount
	if err := database.DB.First(&toAcc, req.ToAccountID).Error; err != nil {
		shared.SendAppError(c, http.StatusBadRequest, "Destination bank account not found.")
		return
	}

	userRoleVal, _ := c.Get("role")
	userRole, _ := userRoleVal.(models.Role)
	if userRole != models.RoleAdmin {
		dateStr := bizDate.Format("2006-01-02")
		if err := shared.EnsureBankAccountOpen(database.DB, fromAcc.ID, dateStr); err != nil {
			shared.SendError(c, http.StatusConflict, "BANK_ACCOUNT_CLOSED",
				fmt.Sprintf("%s is CLOSED for %s. Ask an admin to unlock it before transferring funds.", fromAcc.BankName, dateStr))
			return
		}
		if err := shared.EnsureBankAccountOpen(database.DB, toAcc.ID, dateStr); err != nil {
			shared.SendError(c, http.StatusConflict, "BANK_ACCOUNT_CLOSED",
				fmt.Sprintf("%s is CLOSED for %s. Ask an admin to unlock it before transferring funds.", toAcc.BankName, dateStr))
			return
		}
	}

	tx := database.DB.Begin()

	// 1. Debit Source Bank Account
	debitTx := models.BankTransaction{
		BankAccountID:   fromAcc.ID,
		BusinessDate:    bizDate,
		TransactionType: "DEBIT",
		Amount:          req.Amount,
		Category:        "TRANSFER",
		ReferenceNumber: req.ReferenceNumber,
		SourceType:      "INTER_BANK_TRANSFER",
		SourceID:        toAcc.ID,
		Description:     "Transfer to " + toAcc.BankName + " (" + toAcc.AccountNumberMasked + ")",
		CreatedByID:     userID,
	}
	if err := tx.Create(&debitTx).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to record source bank debit")
		return
	}

	// Re-check the source balance under a row lock inside the transaction —
	// the check above (before the transaction started) is only advisory; this
	// is the authoritative check, closing the race where two concurrent
	// debits/transfers could otherwise both pass against the same stale balance.
	var lockedFromAcc models.BankAccount
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&lockedFromAcc, fromAcc.ID).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to lock source bank account for balance update")
		return
	}
	if lockedFromAcc.CurrentBalance.LessThan(req.Amount) {
		tx.Rollback()
		shared.SendAppError(c, http.StatusBadRequest, "Insufficient funds in source bank account.")
		return
	}
	if err := tx.Model(&lockedFromAcc).Update("current_balance", gorm.Expr("current_balance - ?", req.Amount)).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to update source bank balance")
		return
	}

	// 2. Credit Destination Bank Account
	creditTx := models.BankTransaction{
		BankAccountID:   toAcc.ID,
		BusinessDate:    bizDate,
		TransactionType: "CREDIT",
		Amount:          req.Amount,
		Category:        "TRANSFER",
		ReferenceNumber: req.ReferenceNumber,
		SourceType:      "INTER_BANK_TRANSFER",
		SourceID:        fromAcc.ID,
		Description:     "Transfer from " + fromAcc.BankName + " (" + fromAcc.AccountNumberMasked + ")",
		CreatedByID:     userID,
	}
	if err := tx.Create(&creditTx).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to record destination bank credit")
		return
	}
	if err := tx.Model(&toAcc).Update("current_balance", gorm.Expr("current_balance + ?", req.Amount)).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to update destination bank balance")
		return
	}

	transferDetails := gin.H{
		"from_account_id": fromAcc.ID,
		"to_account_id":   toAcc.ID,
		"amount":          req.Amount,
		"reference":       req.ReferenceNumber,
	}
	audit := models.AuditLog{
		UserID:     &userID,
		Action:     "BANK_TRANSFER_EXECUTED",
		EntityName: "BankAccount",
		EntityID:   fromAcc.ID,
		BeforeData: shared.JSONOrNull(nil),
		AfterData:  shared.JSONOrNull(transferDetails),
		Reason:     req.Description,
		IPAddress:  c.ClientIP(),
	}
	if err := tx.Create(&audit).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to record audit log")
		return
	}

	tx.Commit()

	shared.SendSuccess(c, http.StatusOK, gin.H{
		"message": "Inter-bank transfer executed successfully",
		"amount":  req.Amount,
	})
}
