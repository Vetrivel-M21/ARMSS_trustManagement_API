package report

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"trust-management/backend/internal/database"
	"trust-management/backend/internal/dto"
	"trust-management/backend/internal/models"
	"trust-management/backend/internal/shared"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

type ReportHandler struct{}

func NewReportHandler() *ReportHandler {
	return &ReportHandler{}
}

// GetDailySummaryBook returns aggregated financial totals for a given date
func (h *ReportHandler) GetDailySummaryBook(c *gin.Context) {
	dateStr := c.DefaultQuery("date", shared.GetCurrentBusinessDate().Format("2006-01-02"))

	var cashInflow decimal.Decimal
	database.DB.Model(&models.CashTransaction{}).
		Where("business_date = ? AND transaction_type = ?", dateStr, "INFLOW").
		Select("COALESCE(SUM(amount), 0)").Scan(&cashInflow)

	var cashOutflow decimal.Decimal
	database.DB.Model(&models.CashTransaction{}).
		Where("business_date = ? AND transaction_type = ?", dateStr, "OUTFLOW").
		Select("COALESCE(SUM(amount), 0)").Scan(&cashOutflow)

	var bankCredit decimal.Decimal
	database.DB.Model(&models.BankTransaction{}).
		Where("business_date = ? AND transaction_type = ?", dateStr, "CREDIT").
		Select("COALESCE(SUM(amount), 0)").Scan(&bankCredit)

	var bankDebit decimal.Decimal
	database.DB.Model(&models.BankTransaction{}).
		Where("business_date = ? AND transaction_type = ?", dateStr, "DEBIT").
		Select("COALESCE(SUM(amount), 0)").Scan(&bankDebit)

	// Fetch scheme breakdown
	type SchemeBreakdown struct {
		SchemeName  string          `json:"scheme_name"`
		Category    string          `json:"category"`
		TotalAmount decimal.Decimal `json:"total_amount"`
		Count       int64           `json:"count"`
	}
	var schemeBreakdown []SchemeBreakdown
	database.DB.Model(&models.Donation{}).
		Select("COALESCE(schemes.name, NULLIF(donations.purpose, ''), 'General Donation') as scheme_name, COALESCE(schemes.category, 'GENERAL') as category, SUM(donations.amount) as total_amount, COUNT(donations.id) as count").
		Joins("LEFT JOIN schemes ON schemes.id = donations.scheme_id").
		Where("donations.business_date = ?", dateStr).
		Group("scheme_name, schemes.category").
		Scan(&schemeBreakdown)

	shared.SendSuccess(c, http.StatusOK, gin.H{
		"business_date":    dateStr,
		"cash_inflow":      cashInflow,
		"cash_outflow":     cashOutflow,
		"net_cash":         cashInflow.Sub(cashOutflow),
		"bank_credit":      bankCredit,
		"bank_debit":       bankDebit,
		"net_bank":         bankCredit.Sub(bankDebit),
		"total_collection": cashInflow.Add(bankCredit),
		"scheme_summary":   schemeBreakdown,
	})
}

// GetYoYComparison returns Month-over-Month donor collection comparison for current vs previous year
func (h *ReportHandler) GetYoYComparison(c *gin.Context) {
	currentYearStr := c.DefaultQuery("year", strconv.Itoa(time.Now().Year()))
	currentYear, _ := strconv.Atoi(currentYearStr)
	prevYear := currentYear - 1

	months := []string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}
	result := make([]dto.YoYMonthComparisonItem, 12)

	for i := 1; i <= 12; i++ {
		monthStr := fmt.Sprintf("%02d", i)
		var currAmt decimal.Decimal
		database.DB.Model(&models.Donation{}).
			Where("DATE_FORMAT(business_date, '%Y-%m') = ?", fmt.Sprintf("%d-%s", currentYear, monthStr)).
			Select("COALESCE(SUM(amount), 0)").Scan(&currAmt)

		var prevAmt decimal.Decimal
		database.DB.Model(&models.Donation{}).
			Where("DATE_FORMAT(business_date, '%Y-%m') = ?", fmt.Sprintf("%d-%s", prevYear, monthStr)).
			Select("COALESCE(SUM(amount), 0)").Scan(&prevAmt)

		varPct := 0.0
		if prevAmt.IsPositive() {
			varPct, _ = currAmt.Sub(prevAmt).Div(prevAmt).Mul(decimal.NewFromInt(100)).Float64()
		}

		result[i-1] = dto.YoYMonthComparisonItem{
			MonthName:       months[i-1],
			CurrentYearAmt:  currAmt,
			PreviousYearAmt: prevAmt,
			VarianceAmount:  currAmt.Sub(prevAmt),
			VariancePercent: varPct,
		}
	}

	shared.SendSuccess(c, http.StatusOK, gin.H{
		"current_year":  currentYear,
		"previous_year": prevYear,
		"months":        result,
	})
}

// GetYoYMonthDonors returns the donor-level donation list for one calendar
// month, split across the given year and the previous year, so a selected
// month's YoY comparison can be drilled down into who actually donated.
func (h *ReportHandler) GetYoYMonthDonors(c *gin.Context) {
	monthStr := c.Query("month")
	month, err := strconv.Atoi(monthStr)
	if err != nil || month < 1 || month > 12 {
		shared.SendAppError(c, http.StatusBadRequest, "month is required and must be between 1 and 12")
		return
	}

	currentYearStr := c.DefaultQuery("year", strconv.Itoa(time.Now().Year()))
	currentYear, _ := strconv.Atoi(currentYearStr)
	prevYear := currentYear - 1

	type row struct {
		DonorID      uint            `json:"donor_id"`
		DonorName    string          `json:"donor_name"`
		DonorCode    string          `json:"donor_code"`
		Amount       decimal.Decimal `json:"amount"`
		BusinessDate time.Time       `json:"business_date"`
		Purpose      string          `json:"purpose"`
	}

	fetchYear := func(year int) []dto.YoYMonthDonorItem {
		var rows []row
		database.DB.Model(&models.Donation{}).
			Select("donors.id as donor_id, donors.full_name as donor_name, donors.donor_code as donor_code, donations.amount as amount, donations.business_date as business_date, donations.purpose as purpose").
			Joins("JOIN donors ON donors.id = donations.donor_id").
			Where("MONTH(donations.business_date) = ? AND YEAR(donations.business_date) = ?", month, year).
			Order("donations.business_date asc").
			Scan(&rows)

		items := make([]dto.YoYMonthDonorItem, len(rows))
		for i, r := range rows {
			items[i] = dto.YoYMonthDonorItem{
				DonorID:      r.DonorID,
				DonorName:    r.DonorName,
				DonorCode:    r.DonorCode,
				Amount:       r.Amount,
				BusinessDate: r.BusinessDate.Format("2006-01-02"),
				Purpose:      r.Purpose,
				Year:         year,
			}
		}
		return items
	}

	shared.SendSuccess(c, http.StatusOK, gin.H{
		"month":                month,
		"current_year":         currentYear,
		"previous_year":        prevYear,
		"current_year_donors":  fetchYear(currentYear),
		"previous_year_donors": fetchYear(prevYear),
	})
}

// GetBirthdayReport returns birthday list for donors and children for specified month or upcoming
func (h *ReportHandler) GetBirthdayReport(c *gin.Context) {
	monthStr := c.DefaultQuery("month", strconv.Itoa(int(time.Now().Month())))
	month, _ := strconv.Atoi(monthStr)

	var items []dto.BirthdayItem

	// 1. Fetch Donors with DOB in target month
	var donors []models.Donor
	database.DB.Where("MONTH(date_of_birth) = ? AND is_active = ?", month, true).Find(&donors)
	for _, d := range donors {
		if d.DateOfBirth != nil {
			age := time.Now().Year() - d.DateOfBirth.Year()
			items = append(items, dto.BirthdayItem{
				Type:          "DONOR",
				DonorID:       d.ID,
				DonorName:     d.FullName,
				PersonName:    d.FullName,
				Phone:         d.Phone,
				Email:         d.Email,
				Relationship:  "SELF",
				DateOfBirth:   *d.DateOfBirth,
				BirthdayDay:   d.DateOfBirth.Day(),
				BirthdayMonth: int(d.DateOfBirth.Month()),
				Age:           age,
			})
		}
	}

	// 2. Fetch Family Members / Children with DOB in target month
	var familyMembers []models.DonorFamilyMember
	database.DB.Where("MONTH(date_of_birth) = ?", month).Find(&familyMembers)
	for _, fm := range familyMembers {
		var donor models.Donor
		database.DB.First(&donor, fm.DonorID)

		age := time.Now().Year() - fm.DateOfBirth.Year()
		familyMemberID := fm.ID
		items = append(items, dto.BirthdayItem{
			Type:           "FAMILY_MEMBER",
			DonorID:        donor.ID,
			DonorName:      donor.FullName,
			PersonName:     fm.FullName,
			Phone:          donor.Phone,
			Email:          donor.Email,
			Relationship:   fm.Relationship,
			DateOfBirth:    fm.DateOfBirth,
			BirthdayDay:    fm.DateOfBirth.Day(),
			BirthdayMonth:  int(fm.DateOfBirth.Month()),
			Age:            age,
			FamilyMemberID: &familyMemberID,
		})
	}

	// 3. Fetch Donors with a wedding anniversary in target month
	var anniversaryDonors []models.Donor
	database.DB.Where("MONTH(anniversary_date) = ? AND is_active = ?", month, true).Find(&anniversaryDonors)
	for _, d := range anniversaryDonors {
		if d.AnniversaryDate != nil {
			years := time.Now().Year() - d.AnniversaryDate.Year()
			items = append(items, dto.BirthdayItem{
				Type:          "ANNIVERSARY",
				DonorID:       d.ID,
				DonorName:     d.FullName,
				PersonName:    d.FullName,
				Phone:         d.Phone,
				Email:         d.Email,
				Relationship:  "SPOUSE",
				DateOfBirth:   *d.AnniversaryDate,
				BirthdayDay:   d.AnniversaryDate.Day(),
				BirthdayMonth: int(d.AnniversaryDate.Month()),
				Age:           years,
			})
		}
	}

	shared.SendSuccess(c, http.StatusOK, gin.H{
		"month":     month,
		"birthdays": items,
	})
}
