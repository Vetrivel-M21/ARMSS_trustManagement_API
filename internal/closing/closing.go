package closing

import (
	"fmt"
	"net/http"
	"time"

	"trust-management/backend/internal/database"
	"trust-management/backend/internal/dto"
	"trust-management/backend/internal/models"
	"trust-management/backend/internal/shared"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

type ClosingHandler struct{}

func NewClosingHandler() *ClosingHandler {
	return &ClosingHandler{}
}

// GetDailyClosingStatus gets current closing state and calculations for business date
func (h *ClosingHandler) GetDailyClosingStatus(c *gin.Context) {
	dateStr := c.DefaultQuery("date", shared.GetCurrentBusinessDate().Format("2006-01-02"))
	bizDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		shared.SendAppError(c, http.StatusBadRequest, "Invalid date format")
		return
	}

	var dailyClosing models.DailyClosing
	err = database.DB.Where("business_date = ?", dateStr).First(&dailyClosing).Error
	if err != nil {
		// Initialize open status; opening cash carries forward from the
		// previous CLOSED day's physical count.
		dailyClosing = models.DailyClosing{
			BusinessDate:        bizDate,
			Status:              models.DayStatusOpen,
			OpeningCash:         shared.GetOpeningCash(database.DB, dateStr),
			ExpectedClosingCash: decimal.Zero,
		}
	}

	// Calculate current inflows & outflows
	figures := shared.RecomputeCashFigures(database.DB, dateStr, dailyClosing.OpeningCash)
	dailyClosing.CashInflow = figures.Inflow
	dailyClosing.CashOutflow = figures.Outflow
	dailyClosing.ExpectedClosingCash = figures.ExpectedClosing
	dailyClosing.CashDifference = dailyClosing.PhysicalCashCount.Sub(dailyClosing.ExpectedClosingCash)

	// Update status to READY_TO_CLOSE if physical cash entered and day is OPEN
	if dailyClosing.Status == models.DayStatusOpen && dailyClosing.PhysicalCashCount.IsPositive() {
		dailyClosing.Status = models.DayStatusReadyToClose
	}

	shared.SendSuccess(c, http.StatusOK, dailyClosing)
}

// ExecuteDailyClosing transition day status from READY_TO_CLOSE to CLOSED
func (h *ClosingHandler) ExecuteDailyClosing(c *gin.Context) {
	var req dto.SubmitDailyClosingRequest
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

	dateStr := req.BusinessDate
	var dailyClosing models.DailyClosing
	if err := database.DB.Where("business_date = ?", dateStr).First(&dailyClosing).Error; err != nil {
		shared.SendAppError(c, http.StatusNotFound, "Daily closing record not found for this date")
		return
	}

	if dailyClosing.Status == models.DayStatusClosed {
		shared.SendAppError(c, http.StatusBadRequest, "Business day is already CLOSED.")
		return
	}

	// Recompute the reconciliation figures from source transactions (never trust
	// stale/cached values) and refuse to close on any physical/expected mismatch,
	// per spec section 19 — closing must never silently paper over a difference.
	figures := shared.RecomputeCashFigures(database.DB, dateStr, dailyClosing.OpeningCash)
	dailyClosing.CashInflow = figures.Inflow
	dailyClosing.CashOutflow = figures.Outflow
	dailyClosing.ExpectedClosingCash = figures.ExpectedClosing
	dailyClosing.CashDifference = dailyClosing.PhysicalCashCount.Sub(dailyClosing.ExpectedClosingCash)

	if !dailyClosing.CashDifference.IsZero() {
		shared.SendError(c, http.StatusConflict, "CASH_MISMATCH",
			fmt.Sprintf("Physical cash (₹%s) does not match expected cash (₹%s). Difference: ₹%s. Recount denominations or correct the discrepancy before closing.",
				dailyClosing.PhysicalCashCount.StringFixed(2), dailyClosing.ExpectedClosingCash.StringFixed(2), dailyClosing.CashDifference.StringFixed(2)))
		return
	}

	now := time.Now()
	dailyClosing.Status = models.DayStatusClosed
	dailyClosing.ClosedByID = &userID
	dailyClosing.ClosedAt = &now

	tx := database.DB.Begin()
	if err := tx.Save(&dailyClosing).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to lock business day")
		return
	}

	// Create Audit Log entry for Daily Closing (same transaction — closing the
	// day without a recorded audit entry must never happen)
	audit := models.AuditLog{
		UserID:     &userID,
		Action:     "DAILY_CLOSING_LOCKED",
		EntityName: "DailyClosing",
		EntityID:   dailyClosing.ID,
		BeforeData: shared.JSONOrNull(nil),
		AfterData:  shared.JSONOrNull(dailyClosing),
		Reason:     fmt.Sprintf("Business day %s locked by staff/admin. Physical Cash: ₹%s, Expected: ₹%s, Diff: ₹%s", dateStr, dailyClosing.PhysicalCashCount.StringFixed(2), dailyClosing.ExpectedClosingCash.StringFixed(2), dailyClosing.CashDifference.StringFixed(2)),
		IPAddress:  c.ClientIP(),
	}
	if err := tx.Create(&audit).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to record closing audit log")
		return
	}
	tx.Commit()

	shared.SendSuccess(c, http.StatusOK, gin.H{
		"message":       fmt.Sprintf("Business day %s successfully CLOSED.", dateStr),
		"daily_closing": dailyClosing,
	})
}
