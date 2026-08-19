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

// GetExpenses returns expense register with preloads
func (h *ExpenseHandler) GetExpenses(c *gin.Context) {
	var expenses []models.Expense
	if err := database.DB.Preload("BankAccount").Order("id desc").Find(&expenses).Error; err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to fetch expenses")
		return
	}
	shared.SendSuccess(c, http.StatusOK, expenses)
}

// CreateExpense records an outflow expense, updates ledger, and issues an expense voucher
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

	// Closed-day protection is enforced centrally by middleware.ClosedDayProtectionMiddleware
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		shared.SendAppError(c, http.StatusBadRequest, "Expense amount must be greater than zero.")
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
			shared.SendAppError(c, http.StatusBadRequest, "Bank Account selection is required for BANK payment mode expense.")
			return
		}
		var acc models.BankAccount
		if err := database.DB.First(&acc, *req.BankAccountID).Error; err != nil {
			shared.SendAppError(c, http.StatusBadRequest, "Selected Bank Account not found.")
			return
		}
		if acc.CurrentBalance.LessThan(req.Amount) {
			shared.SendAppError(c, http.StatusBadRequest, fmt.Sprintf("Insufficient bank balance (Current: ₹%s, Expense: ₹%s)", acc.CurrentBalance.StringFixed(2), req.Amount.StringFixed(2)))
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
		Status:          "ACTIVE",
		CreatedByID:     userID,
	}

	if err := tx.Create(&expense).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to record expense")
		return
	}

	// Ledger Entries
	if paymentMode == models.PaymentModeCash {
		cashTx := models.CashTransaction{
			BusinessDate:    bizDate,
			TransactionType: "OUTFLOW",
			Amount:          req.Amount,
			SourceType:      "EXPENSE",
			SourceID:        expense.ID,
			Description:     fmt.Sprintf("Expense %s (%s to %s)", expense.ExpenseNumber, req.Category, req.PayeeName),
			CreatedByID:     userID,
		}
		if err := tx.Create(&cashTx).Error; err != nil {
			tx.Rollback()
			shared.SendAppError(c, http.StatusInternalServerError, "Failed to record cash outflow")
			return
		}
	} else if paymentMode == models.PaymentModeBank {
		bankTx := models.BankTransaction{
			BankAccountID:   *req.BankAccountID,
			BusinessDate:    bizDate,
			TransactionType: "DEBIT",
			Amount:          req.Amount,
			Category:        req.Category,
			ReferenceNumber: expense.ExpenseNumber,
			SourceType:      "EXPENSE",
			SourceID:        expense.ID,
			Description:     fmt.Sprintf("Bank Expense %s to %s", expense.ExpenseNumber, req.PayeeName),
			CreatedByID:     userID,
		}
		if err := tx.Create(&bankTx).Error; err != nil {
			tx.Rollback()
			shared.SendAppError(c, http.StatusInternalServerError, "Failed to record bank debit transaction")
			return
		}

		// Re-check the balance under a row lock inside the transaction — the
		// check above (before the transaction started) is only advisory; this
		// is the authoritative check, closing the race where two concurrent
		// debits could otherwise both pass against the same stale balance.
		var lockedAcc models.BankAccount
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&lockedAcc, bankAccount.ID).Error; err != nil {
			tx.Rollback()
			shared.SendAppError(c, http.StatusInternalServerError, "Failed to lock bank account for balance update")
			return
		}
		if lockedAcc.CurrentBalance.LessThan(req.Amount) {
			tx.Rollback()
			shared.SendAppError(c, http.StatusBadRequest, fmt.Sprintf("Insufficient bank balance (Current: ₹%s, Expense: ₹%s)", lockedAcc.CurrentBalance.StringFixed(2), req.Amount.StringFixed(2)))
			return
		}
		if err := tx.Model(&lockedAcc).Update("current_balance", gorm.Expr("current_balance - ?", req.Amount)).Error; err != nil {
			tx.Rollback()
			shared.SendAppError(c, http.StatusInternalServerError, "Failed to update bank balance")
			return
		}
	}

	// Generate Expense Voucher Code (e.g. ACHT/1/26-27) via atomic per-financial-year sequence counter
	voucherNumber, err := shared.GenerateVoucherNumber(tx, bizDate)
	if err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to generate voucher number")
		return
	}

	voucher := models.Voucher{
		VoucherNumber:    voucherNumber,
		VoucherType:      "EXPENSE_VOUCHER",
		BusinessDate:     bizDate,
		SourceType:       "EXPENSE",
		SourceID:         expense.ID,
		PayeeOrDonorName: req.PayeeName,
		Amount:           req.Amount,
		AmountInWords:    shared.ConvertAmountToWords(req.Amount),
		PaymentMode:      paymentMode,
		Status:           "ISSUED",
		CreatedByID:      userID,
	}
	if err := tx.Create(&voucher).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to generate expense voucher")
		return
	}

	audit := models.AuditLog{
		UserID:     &userID,
		Action:     "EXPENSE_CREATED",
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

	database.DB.Preload("BankAccount").First(&expense, expense.ID)
	shared.SendSuccess(c, http.StatusCreated, gin.H{
		"expense": expense,
		"voucher": voucher,
	})
}
