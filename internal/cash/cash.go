package cash

import (
	"net/http"
	"time"

	"trust-management/backend/internal/database"
	"trust-management/backend/internal/dto"
	"trust-management/backend/internal/models"
	"trust-management/backend/internal/shared"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

type CashHandler struct{}

func NewCashHandler() *CashHandler {
	return &CashHandler{}
}

// GetDailyCashSummary returns cash inflows, outflows, opening balance, and physical count for a date
func (h *CashHandler) GetDailyCashSummary(c *gin.Context) {
	dateStr := c.DefaultQuery("date", shared.GetCurrentBusinessDate().Format("2006-01-02"))
	_, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		shared.SendAppError(c, http.StatusBadRequest, "Invalid date format. Use YYYY-MM-DD.")
		return
	}

	var inflows decimal.Decimal
	database.DB.Model(&models.CashTransaction{}).
		Where("business_date = ? AND transaction_type = ?", dateStr, "INFLOW").
		Select("COALESCE(SUM(amount), 0)").Scan(&inflows)

	var outflows decimal.Decimal
	database.DB.Model(&models.CashTransaction{}).
		Where("business_date = ? AND transaction_type = ?", dateStr, "OUTFLOW").
		Select("COALESCE(SUM(amount), 0)").Scan(&outflows)

	var dailyClosing models.DailyClosing
	err = database.DB.Where("business_date = ?", dateStr).First(&dailyClosing).Error
	openingCash := decimal.Zero
	status := "OPEN"

	if err == nil {
		openingCash = dailyClosing.OpeningCash
		status = string(dailyClosing.Status)
	}

	expectedClosing := openingCash.Add(inflows).Sub(outflows)

	// Fetch existing closing denomination entries for this date if present
	var denoms []models.CashDenomination
	if dailyClosing.ID > 0 {
		database.DB.Where("entity_type = ? AND entity_id = ?", "DAILY_CLOSING", dailyClosing.ID).Find(&denoms)
	}

	physicalCash := decimal.Zero
	for _, d := range denoms {
		physicalCash = physicalCash.Add(d.TotalAmount)
	}

	// Aggregated Cash Donations by Purpose & Meal Type (Section 17 of SPEC)
	type AggregatedCashGroup struct {
		Label       string            `json:"label"`
		FoodType    string            `json:"food_type"`
		MealType    string            `json:"meal_type"`
		Count       int64             `json:"count"`
		TotalAmount decimal.Decimal   `json:"total_amount"`
		Donations   []models.Donation `json:"donations"`
	}

	var cashDonations []models.Donation
	database.DB.Preload("Donor").Preload("Scheme").
		Where("business_date = ? AND payment_mode = ?", dateStr, "CASH").
		Find(&cashDonations)

	groupMap := make(map[string]*AggregatedCashGroup)
	for _, don := range cashDonations {
		// The donor's own purpose text takes priority over the scheme name —
		// schemes now get auto-generated names (e.g. "Veg Breakfast") that are
		// less meaningful here than what the donor actually typed.
		label := don.Purpose
		foodType := "NA"
		mealType := "NA"
		if don.Scheme != nil {
			foodType = don.Scheme.FoodType
			mealType = don.Scheme.MealType
			if label == "" {
				label = don.Scheme.Name
			}
		}
		if label == "" {
			label = "General Cash Donation"
		}

		key := label + "|" + foodType + "|" + mealType
		if grp, ok := groupMap[key]; ok {
			grp.Count++
			grp.TotalAmount = grp.TotalAmount.Add(don.Amount)
			grp.Donations = append(grp.Donations, don)
		} else {
			groupMap[key] = &AggregatedCashGroup{
				Label:       label,
				FoodType:    foodType,
				MealType:    mealType,
				Count:       1,
				TotalAmount: don.Amount,
				Donations:   []models.Donation{don},
			}
		}
	}

	var aggregatedGroups []AggregatedCashGroup
	for _, grp := range groupMap {
		aggregatedGroups = append(aggregatedGroups, *grp)
	}

	// Aggregated Cash Outflow (expenses paid in cash) by Category — the debit-side
	// counterpart of the donation breakdown above, so cash outflow isn't just a
	// single total with no detail behind it.
	type AggregatedExpenseGroup struct {
		Category    string           `json:"category"`
		Count       int64            `json:"count"`
		TotalAmount decimal.Decimal  `json:"total_amount"`
		Expenses    []models.Expense `json:"expenses"`
	}

	var cashExpenses []models.Expense
	database.DB.Where("business_date = ? AND payment_mode = ?", dateStr, "CASH").Find(&cashExpenses)

	expenseGroupMap := make(map[string]*AggregatedExpenseGroup)
	for _, exp := range cashExpenses {
		if grp, ok := expenseGroupMap[exp.Category]; ok {
			grp.Count++
			grp.TotalAmount = grp.TotalAmount.Add(exp.Amount)
			grp.Expenses = append(grp.Expenses, exp)
		} else {
			expenseGroupMap[exp.Category] = &AggregatedExpenseGroup{
				Category:    exp.Category,
				Count:       1,
				TotalAmount: exp.Amount,
				Expenses:    []models.Expense{exp},
			}
		}
	}

	var expenseAggregations []AggregatedExpenseGroup
	for _, grp := range expenseGroupMap {
		expenseAggregations = append(expenseAggregations, *grp)
	}

	shared.SendSuccess(c, http.StatusOK, gin.H{
		"business_date":         dateStr,
		"status":                status,
		"opening_cash":          openingCash,
		"cash_inflow":           inflows,
		"cash_outflow":          outflows,
		"expected_closing_cash": expectedClosing,
		"physical_cash_count":   physicalCash,
		"cash_difference":       physicalCash.Sub(expectedClosing),
		"denominations":         denoms,
		"scheme_aggregations":   aggregatedGroups,
		"expense_aggregations":  expenseAggregations,
	})
}

// SubmitCashDenominations stores physical note counts for daily cash closing reconciliation
func (h *CashHandler) SubmitCashDenominations(c *gin.Context) {
	var req dto.SubmitDailyCashDenominationsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.SendAppError(c, http.StatusBadRequest, err.Error())
		return
	}

	dateStr := req.BusinessDate
	bizDate, err := shared.ParseBusinessDate(dateStr)
	if err != nil {
		shared.SendAppError(c, http.StatusBadRequest, "Invalid business date format. Use YYYY-MM-DD.")
		return
	}

	var dailyClosing models.DailyClosing
	err = database.DB.Where("business_date = ?", dateStr).First(&dailyClosing).Error

	tx := database.DB.Begin()
	if err != nil {
		// Create initial daily closing record for the REQUESTED business date.
		// Opening cash carries forward from the previous CLOSED day's physical count.
		dailyClosing = models.DailyClosing{
			BusinessDate:        bizDate,
			Status:              models.DayStatusOpen,
			OpeningCash:         shared.GetOpeningCash(tx, dateStr),
			ExpectedClosingCash: decimal.Zero,
		}
		if err := tx.Create(&dailyClosing).Error; err != nil {
			tx.Rollback()
			shared.SendAppError(c, http.StatusInternalServerError, "Failed to initialize daily closing record")
			return
		}
	}
	// Closed-day protection is enforced centrally by middleware.ClosedDayProtectionMiddleware

	// Clear previous closing denominations for this date
	tx.Where("entity_type = ? AND entity_id = ?", "DAILY_CLOSING", dailyClosing.ID).Delete(&models.CashDenomination{})

	totalPhysical := shared.SumDenominations(req.Denominations)
	for _, item := range req.Denominations {
		if item.Quantity >= 0 {
			amt := decimal.NewFromInt(int64(item.Value * item.Quantity))
			denom := models.CashDenomination{
				EntityType:        "DAILY_CLOSING",
				EntityID:          dailyClosing.ID,
				DenominationValue: item.Value,
				Quantity:          item.Quantity,
				TotalAmount:       amt,
			}
			if err := tx.Create(&denom).Error; err != nil {
				tx.Rollback()
				shared.SendAppError(c, http.StatusInternalServerError, "Failed to save denomination entry")
				return
			}
		}
	}

	// Recompute expected closing cash from current transactions — never trust a
	// previously persisted value, which goes stale as soon as a new cash
	// transaction is recorded for this date.
	figures := shared.RecomputeCashFigures(tx, dateStr, dailyClosing.OpeningCash)
	dailyClosing.CashInflow = figures.Inflow
	dailyClosing.CashOutflow = figures.Outflow
	dailyClosing.ExpectedClosingCash = figures.ExpectedClosing
	dailyClosing.PhysicalCashCount = totalPhysical
	dailyClosing.CashDifference = totalPhysical.Sub(dailyClosing.ExpectedClosingCash)

	if err := tx.Save(&dailyClosing).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to update daily physical cash count")
		return
	}

	tx.Commit()

	shared.SendSuccess(c, http.StatusOK, gin.H{
		"message":             "Cash denominations recorded successfully",
		"physical_cash_count": totalPhysical,
		"cash_difference":     dailyClosing.CashDifference,
	})
}
