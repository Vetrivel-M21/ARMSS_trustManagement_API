package expense

import (
	"fmt"
	"net/http"
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

type ExpenseHandler struct{}

func NewExpenseHandler() *ExpenseHandler {
	return &ExpenseHandler{}
}

// GetExpenses returns expense register with preloads and associated vouchers
func (h *ExpenseHandler) GetExpenses(c *gin.Context) {
	var expenses []models.Expense
	if err := database.DB.Preload("BankAccount").Preload("CreatedBy").Preload("ApprovedBy").Order("id desc").Find(&expenses).Error; err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to fetch expenses")
		return
	}

	// Fetch any existing vouchers for approved/active expenses
	var expenseIDs []uint
	for _, e := range expenses {
		if e.Status == "APPROVED" || e.Status == "ACTIVE" {
			expenseIDs = append(expenseIDs, e.ID)
		}
	}

	if len(expenseIDs) > 0 {
		var vouchers []models.Voucher
		database.DB.Where("source_type = ? AND source_id IN (?)", "EXPENSE", expenseIDs).Find(&vouchers)
		voucherMap := make(map[uint]*models.Voucher)
		for i := range vouchers {
			voucherMap[vouchers[i].SourceID] = &vouchers[i]
		}
		for i := range expenses {
			if v, ok := voucherMap[expenses[i].ID]; ok {
				expenses[i].Voucher = v
			}
		}
	}

	shared.SendSuccess(c, http.StatusOK, expenses)
}

// CreateExpense records a new expense in PENDING status awaiting Admin approval
func (h *ExpenseHandler) CreateExpense(c *gin.Context) {
	var req dto.CreateExpenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.SendAppError(c, http.StatusBadRequest, err.Error())
		return
	}

	userIDVal, exists := c.Get("userID")
	if !exists {
		shared.SendAppError(c, http.StatusUnauthorized, "User context missing")
		return
	}
	userID := userIDVal.(uint)

	bizDate := shared.GetCurrentBusinessDate()
	if req.BusinessDate != "" {
		if t, err := time.Parse("2006-01-02", req.BusinessDate); err == nil {
			bizDate = t
		}
	}

	if req.Amount.LessThanOrEqual(decimal.Zero) {
		shared.SendAppError(c, http.StatusBadRequest, "Expense amount must be greater than zero.")
		return
	}

	paymentMode := models.PaymentMode(strings.ToUpper(req.PaymentMode))
	if paymentMode != models.PaymentModeCash && paymentMode != models.PaymentModeBank {
		shared.SendAppError(c, http.StatusBadRequest, "Invalid payment mode. Must be CASH or BANK.")
		return
	}

	if paymentMode == models.PaymentModeBank {
		if req.BankAccountID == nil || *req.BankAccountID == 0 {
			shared.SendAppError(c, http.StatusBadRequest, "Bank Account selection is required for BANK payment mode expense.")
			return
		}
		var acc models.BankAccount
		if err := database.DB.First(&acc, *req.BankAccountID).Error; err != nil {
			shared.SendAppError(c, http.StatusBadRequest, "Selected Bank Account not found.")
			return
		}
	}

	tx := database.DB.Begin()

	// Generate unique Expense Number (e.g. EXP-2026-00001) via atomic sequence counter
	expSeq, err := shared.NextSequence(tx, "EXPENSE")
	if err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to generate expense number")
		return
	}
	expenseNumber := fmt.Sprintf("EXP-%d-%05d", bizDate.Year(), expSeq)

	expense := models.Expense{
		ExpenseNumber:   expenseNumber,
		BusinessDate:    bizDate,
		PaymentMode:     paymentMode,
		BankAccountID:   req.BankAccountID,
		Category:        req.Category,
		Amount:          req.Amount,
		PayeeName:       req.PayeeName,
		Description:     req.Description,
		ReferenceNumber: req.ReferenceNumber,
		AttachmentPath:  req.AttachmentPath,
		Status:          "PENDING",
		CreatedByID:     userID,
	}

	if err := tx.Create(&expense).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to record expense")
		return
	}

	audit := models.AuditLog{
		UserID:     &userID,
		Action:     "EXPENSE_SUBMITTED_FOR_APPROVAL",
		EntityName: "Expense",
		EntityID:   expense.ID,
		BeforeData: shared.JSONOrNull(nil),
		AfterData:  shared.JSONOrNull(expense),
		IPAddress:  c.ClientIP(),
	}
	if err := tx.Create(&audit).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to record audit log")
		return
	}

	tx.Commit()

	database.DB.Preload("BankAccount").Preload("CreatedBy").First(&expense, expense.ID)
	shared.SendSuccess(c, http.StatusCreated, gin.H{
		"expense": expense,
	})
}

// ApproveExpense authorizes a PENDING expense: creates ledger transactions, deducts bank funds, and issues a voucher
func (h *ExpenseHandler) ApproveExpense(c *gin.Context) {
	id := c.Param("id")

	adminIDVal, exists := c.Get("userID")
	if !exists {
		shared.SendAppError(c, http.StatusUnauthorized, "User context missing")
		return
	}
	adminID := adminIDVal.(uint)

	var expense models.Expense
	if err := database.DB.Preload("BankAccount").First(&expense, id).Error; err != nil {
		shared.SendAppError(c, http.StatusNotFound, "Expense not found")
		return
	}

	if expense.Status != "PENDING" {
		shared.SendAppError(c, http.StatusBadRequest, fmt.Sprintf("Expense cannot be approved because it is already in %s status", expense.Status))
		return
	}

	tx := database.DB.Begin()

	// If BANK mode, re-validate balance under row lock and deduct
	if expense.PaymentMode == models.PaymentModeBank {
		if expense.BankAccountID == nil || *expense.BankAccountID == 0 {
			tx.Rollback()
			shared.SendAppError(c, http.StatusBadRequest, "Expense has no associated bank account")
			return
		}

		var lockedAcc models.BankAccount
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&lockedAcc, *expense.BankAccountID).Error; err != nil {
			tx.Rollback()
			shared.SendAppError(c, http.StatusInternalServerError, "Failed to lock bank account for balance deduction")
			return
		}

		if lockedAcc.CurrentBalance.LessThan(expense.Amount) {
			tx.Rollback()
			shared.SendAppError(c, http.StatusBadRequest, fmt.Sprintf("Insufficient bank balance (Current: ₹%s, Expense: ₹%s)", lockedAcc.CurrentBalance.StringFixed(2), expense.Amount.StringFixed(2)))
			return
		}

		if err := tx.Model(&lockedAcc).Update("current_balance", gorm.Expr("current_balance - ?", expense.Amount)).Error; err != nil {
			tx.Rollback()
			shared.SendAppError(c, http.StatusInternalServerError, "Failed to update bank balance")
			return
		}

		bankTx := models.BankTransaction{
			BankAccountID:   *expense.BankAccountID,
			BusinessDate:    expense.BusinessDate,
			TransactionType: "DEBIT",
			Amount:          expense.Amount,
			Category:        expense.Category,
			ReferenceNumber: expense.ExpenseNumber,
			SourceType:      "EXPENSE",
			SourceID:        expense.ID,
			Description:     fmt.Sprintf("Bank Expense %s to %s", expense.ExpenseNumber, expense.PayeeName),
			CreatedByID:     adminID,
		}
		if err := tx.Create(&bankTx).Error; err != nil {
			tx.Rollback()
			shared.SendAppError(c, http.StatusInternalServerError, "Failed to record bank debit transaction")
			return
		}
	} else if expense.PaymentMode == models.PaymentModeCash {
		cashTx := models.CashTransaction{
			BusinessDate:    expense.BusinessDate,
			TransactionType: "OUTFLOW",
			Amount:          expense.Amount,
			SourceType:      "EXPENSE",
			SourceID:        expense.ID,
			Description:     fmt.Sprintf("Expense %s (%s to %s)", expense.ExpenseNumber, expense.Category, expense.PayeeName),
			CreatedByID:     adminID,
		}
		if err := tx.Create(&cashTx).Error; err != nil {
			tx.Rollback()
			shared.SendAppError(c, http.StatusInternalServerError, "Failed to record cash outflow")
			return
		}
	}

	// Generate Expense Voucher Code (e.g. ACHT/1/26-27) via atomic sequence counter
	voucherNumber, err := shared.GenerateVoucherNumber(tx, expense.BusinessDate)
	if err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to generate voucher number")
		return
	}

	voucher := models.Voucher{
		VoucherNumber:    voucherNumber,
		VoucherType:      "EXPENSE_VOUCHER",
		BusinessDate:     expense.BusinessDate,
		SourceType:       "EXPENSE",
		SourceID:         expense.ID,
		PayeeOrDonorName: expense.PayeeName,
		Amount:           expense.Amount,
		AmountInWords:    shared.ConvertAmountToWords(expense.Amount),
		PaymentMode:      expense.PaymentMode,
		Status:           "ISSUED",
		CreatedByID:      adminID,
	}
	if err := tx.Create(&voucher).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to generate expense voucher")
		return
	}

	now := time.Now()
	beforeExpense := expense

	if err := tx.Model(&models.Expense{}).Where("id = ?", expense.ID).Updates(map[string]interface{}{
		"status":          "APPROVED",
		"approved_by_id":  adminID,
		"approved_at":     now,
	}).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to update expense status")
		return
	}

	expense.Status = "APPROVED"
	expense.ApprovedByID = &adminID
	expense.ApprovedAt = &now

	audit := models.AuditLog{
		UserID:     &adminID,
		Action:     "EXPENSE_APPROVED",
		EntityName: "Expense",
		EntityID:   expense.ID,
		BeforeData: shared.JSONOrNull(beforeExpense),
		AfterData:  shared.JSONOrNull(expense),
		IPAddress:  c.ClientIP(),
	}
	if err := tx.Create(&audit).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to record audit log")
		return
	}

	tx.Commit()

	database.DB.Preload("BankAccount").Preload("CreatedBy").Preload("ApprovedBy").First(&expense, expense.ID)
	expense.Voucher = &voucher

	shared.SendSuccess(c, http.StatusOK, gin.H{
		"expense": expense,
		"voucher": voucher,
	})
}

// RejectExpense marks a PENDING expense as REJECTED with a required reason
func (h *ExpenseHandler) RejectExpense(c *gin.Context) {
	id := c.Param("id")

	adminIDVal, exists := c.Get("userID")
	if !exists {
		shared.SendAppError(c, http.StatusUnauthorized, "User context missing")
		return
	}
	adminID := adminIDVal.(uint)

	var req dto.RejectExpenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.SendAppError(c, http.StatusBadRequest, "Rejection reason is required")
		return
	}

	var expense models.Expense
	if err := database.DB.First(&expense, id).Error; err != nil {
		shared.SendAppError(c, http.StatusNotFound, "Expense not found")
		return
	}

	if expense.Status != "PENDING" {
		shared.SendAppError(c, http.StatusBadRequest, fmt.Sprintf("Expense cannot be rejected because it is already in %s status", expense.Status))
		return
	}

	now := time.Now()
	beforeExpense := expense

	tx := database.DB.Begin()

	if err := tx.Model(&models.Expense{}).Where("id = ?", expense.ID).Updates(map[string]interface{}{
		"status":           "REJECTED",
		"rejection_reason": req.Reason,
		"approved_by_id":   adminID,
		"approved_at":      now,
	}).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to reject expense")
		return
	}

	expense.Status = "REJECTED"
	expense.RejectionReason = req.Reason
	expense.ApprovedByID = &adminID
	expense.ApprovedAt = &now

	audit := models.AuditLog{
		UserID:     &adminID,
		Action:     "EXPENSE_REJECTED",
		EntityName: "Expense",
		EntityID:   expense.ID,
		BeforeData: shared.JSONOrNull(beforeExpense),
		AfterData:  shared.JSONOrNull(expense),
		IPAddress:  c.ClientIP(),
	}
	if err := tx.Create(&audit).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to record audit log")
		return
	}

	tx.Commit()

	database.DB.Preload("BankAccount").Preload("CreatedBy").Preload("ApprovedBy").First(&expense, expense.ID)
	shared.SendSuccess(c, http.StatusOK, gin.H{
		"expense": expense,
	})
}
