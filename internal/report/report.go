package report

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
)

type ReportHandler struct{}

func NewReportHandler() *ReportHandler {
	return &ReportHandler{}
}

// GetDailySummaryBook returns aggregated financial totals for a given date
func (h *ReportHandler) GetDailySummaryBook(c *gin.Context) {
	dateStr := c.DefaultQuery("date", shared.GetCurrentBusinessDate().Format("2006-01-02"))
	targetBranchID := shared.ResolveBranchID(c)

	cashInQuery := database.DB.Model(&models.CashTransaction{}).
		Where("business_date = ? AND transaction_type = ?", dateStr, "INFLOW")
	if targetBranchID != nil {
		cashInQuery = cashInQuery.Where("branch_id = ?", *targetBranchID)
	}
	var cashInflow decimal.Decimal
	cashInQuery.Select("COALESCE(SUM(amount), 0)").Scan(&cashInflow)

	cashOutQuery := database.DB.Model(&models.CashTransaction{}).
		Where("business_date = ? AND transaction_type = ?", dateStr, "OUTFLOW")
	if targetBranchID != nil {
		cashOutQuery = cashOutQuery.Where("branch_id = ?", *targetBranchID)
	}
	var cashOutflow decimal.Decimal
	cashOutQuery.Select("COALESCE(SUM(amount), 0)").Scan(&cashOutflow)

	bankCreditQuery := database.DB.Model(&models.BankTransaction{}).
		Where("business_date = ? AND transaction_type = ?", dateStr, "CREDIT")
	if targetBranchID != nil {
		bankCreditQuery = bankCreditQuery.Joins("JOIN bank_accounts ON bank_accounts.id = bank_transactions.bank_account_id").
			Where("bank_accounts.branch_id = ?", *targetBranchID)
	}
	var bankCredit decimal.Decimal
	bankCreditQuery.Select("COALESCE(SUM(amount), 0)").Scan(&bankCredit)

	bankDebitQuery := database.DB.Model(&models.BankTransaction{}).
		Where("business_date = ? AND transaction_type = ?", dateStr, "DEBIT")
	if targetBranchID != nil {
		bankDebitQuery = bankDebitQuery.Joins("JOIN bank_accounts ON bank_accounts.id = bank_transactions.bank_account_id").
			Where("bank_accounts.branch_id = ?", *targetBranchID)
	}
	var bankDebit decimal.Decimal
	bankDebitQuery.Select("COALESCE(SUM(amount), 0)").Scan(&bankDebit)

	// Fetch scheme breakdown
	type SchemeBreakdown struct {
		SchemeName  string          `json:"scheme_name"`
		Category    string          `json:"category"`
		TotalAmount decimal.Decimal `json:"total_amount"`
		Count       int64           `json:"count"`
	}
	var schemeBreakdown []SchemeBreakdown
	donQuery := database.DB.Model(&models.Donation{}).
		Select("COALESCE(schemes.name, NULLIF(donations.purpose, ''), 'General Donation') as scheme_name, COALESCE(schemes.category, 'GENERAL') as category, SUM(donations.amount) as total_amount, COUNT(donations.id) as count").
		Joins("LEFT JOIN schemes ON schemes.id = donations.scheme_id").
		Where("donations.business_date = ?", dateStr)
	if targetBranchID != nil {
		donQuery = donQuery.Where("donations.branch_id = ?", *targetBranchID)
	}
	donQuery.Group("scheme_name, schemes.category").Scan(&schemeBreakdown)

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

// GetBranchComparisonReport provides side-by-side branch metrics for Super Admin
func (h *ReportHandler) GetBranchComparisonReport(c *gin.Context) {
	dateStr := c.DefaultQuery("date", shared.GetCurrentBusinessDate().Format("2006-01-02"))

	type BranchMetric struct {
		BranchID          uint            `json:"branch_id"`
		BranchCode        string          `json:"branch_code"`
		BranchName        string          `json:"branch_name"`
		TotalDonations    decimal.Decimal `json:"total_donations"`
		TotalExpenses     decimal.Decimal `json:"total_expenses"`
		CashInflow        decimal.Decimal `json:"cash_inflow"`
		CashOutflow       decimal.Decimal `json:"cash_outflow"`
		NetCash           decimal.Decimal `json:"net_cash"`
		VoucherCount      int64           `json:"voucher_count"`
		LicenseExpiryDate *time.Time      `json:"license_expiry_date"`
		LicenseStatus     string          `json:"license_status"`
	}

	var branches []models.Branch
	database.DB.Where("is_active = ?", true).Order("id asc").Find(&branches)

	var result []BranchMetric
	now := time.Now()

	for _, b := range branches {
		var totalDon decimal.Decimal
		database.DB.Model(&models.Donation{}).
			Where("branch_id = ? AND business_date = ?", b.ID, dateStr).
			Select("COALESCE(SUM(amount), 0)").Scan(&totalDon)

		var totalExp decimal.Decimal
		database.DB.Model(&models.Expense{}).
			Where("branch_id = ? AND business_date = ? AND status IN ('APPROVED', 'ACTIVE')", b.ID, dateStr).
			Select("COALESCE(SUM(amount), 0)").Scan(&totalExp)

		var cashIn decimal.Decimal
		database.DB.Model(&models.CashTransaction{}).
			Where("branch_id = ? AND business_date = ? AND transaction_type = ?", b.ID, dateStr, "INFLOW").
			Select("COALESCE(SUM(amount), 0)").Scan(&cashIn)

		var cashOut decimal.Decimal
		database.DB.Model(&models.CashTransaction{}).
			Where("branch_id = ? AND business_date = ? AND transaction_type = ?", b.ID, dateStr, "OUTFLOW").
			Select("COALESCE(SUM(amount), 0)").Scan(&cashOut)

		var vCount int64
		database.DB.Model(&models.Voucher{}).
			Where("branch_id = ? AND business_date = ?", b.ID, dateStr).
			Count(&vCount)

		licStatus := "ACTIVE"
		if b.LicenseExpiryDate != nil {
			days := int(b.LicenseExpiryDate.Sub(now).Hours() / 24)
			if days < 0 {
				licStatus = "EXPIRED"
			} else if days <= 30 {
				licStatus = "EXPIRING_SOON"
			}
		}

		result = append(result, BranchMetric{
			BranchID:          b.ID,
			BranchCode:        b.BranchCode,
			BranchName:        b.Name,
			TotalDonations:    totalDon,
			TotalExpenses:     totalExp,
			CashInflow:        cashIn,
			CashOutflow:       cashOut,
			NetCash:           cashIn.Sub(cashOut),
			VoucherCount:      vCount,
			LicenseExpiryDate: b.LicenseExpiryDate,
			LicenseStatus:     licStatus,
		})
	}

	shared.SendSuccess(c, http.StatusOK, gin.H{
		"business_date": dateStr,
		"branches":      result,
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

// GetStaffAnalyticsReport aggregates demographic, attendance, operational throughput, and permissions for a staff member
func (h *ReportHandler) GetStaffAnalyticsReport(c *gin.Context) {
	idStr := c.Param("id")
	staffID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		shared.SendAppError(c, http.StatusBadRequest, "Invalid staff ID")
		return
	}

	var user models.User
	if err := database.DB.Preload("Branch").Preload("Permissions").First(&user, uint(staffID)).Error; err != nil {
		shared.SendAppError(c, http.StatusNotFound, "Staff member not found")
		return
	}

	// Calculate formatted tenure
	tenureFormatted := "N/A"
	if user.DateOfJoining != nil {
		endDate := time.Now()
		if user.DateOfRelieving != nil {
			endDate = *user.DateOfRelieving
		}
		years := endDate.Year() - user.DateOfJoining.Year()
		months := int(endDate.Month()) - int(user.DateOfJoining.Month())
		if months < 0 {
			years--
			months += 12
		}
		if years > 0 {
			tenureFormatted = fmt.Sprintf("%d year%s, %d month%s", years, plural(years), months, plural(months))
		} else if months > 0 {
			tenureFormatted = fmt.Sprintf("%d month%s", months, plural(months))
		} else {
			tenureFormatted = "Less than a month"
		}
	}

	dobStr := ""
	if user.DateOfBirth != nil {
		dobStr = user.DateOfBirth.Format("2006-01-02")
	}
	dojStr := ""
	if user.DateOfJoining != nil {
		dojStr = user.DateOfJoining.Format("2006-01-02")
	}
	dorStr := ""
	if user.DateOfRelieving != nil {
		dorStr = user.DateOfRelieving.Format("2006-01-02")
	}

	branchID := uint(1)
	branchName := "Main Branch"
	if user.Branch != nil {
		branchID = user.Branch.ID
		branchName = user.Branch.Name
	} else if user.BranchID != nil {
		branchID = *user.BranchID
	}

	profileDetail := dto.StaffProfileDetail{
		ID:                    user.ID,
		StaffCode:             user.StaffCode,
		Username:              user.Username,
		FullName:              user.FullName,
		Email:                 user.Email,
		Phone:                 user.Phone,
		Gender:                user.Gender,
		Education:             user.Education,
		MaritalStatus:         user.MaritalStatus,
		DateOfBirth:           dobStr,
		DateOfJoining:         dojStr,
		DateOfRelieving:       dorStr,
		TenureFormatted:       tenureFormatted,
		Designation:           user.Designation,
		Department:            user.Department,
		AddressLine:           user.AddressLine,
		City:                  user.City,
		State:                 user.State,
		Pincode:               user.Pincode,
		PhotoPath:             user.PhotoPath,
		EmergencyContactName:  user.EmergencyContactName,
		EmergencyContactPhone: user.EmergencyContactPhone,
		Role:                  string(user.Role),
		EmploymentStatus:      string(user.EmploymentStatus),
		StatusRemarks:         user.StatusRemarks,
		BranchID:              branchID,
		BranchName:            branchName,
		IsActive:              user.IsActive,
	}

	// 2. Attendance Summary for Date Range or Year & Month
	bizDate := shared.GetCurrentBusinessDate()
	targetYear := bizDate.Year()
	targetMonth := int(bizDate.Month())

	if y, err := strconv.Atoi(c.Query("year")); err == nil && y > 2000 {
		targetYear = y
	}
	if m, err := strconv.Atoi(c.Query("month")); err == nil && m >= 1 && m <= 12 {
		targetMonth = m
	}

	startParam := strings.TrimSpace(c.Query("start_date"))
	endParam := strings.TrimSpace(c.Query("end_date"))

	startDate := ""
	endDate := ""
	if startParam != "" && endParam != "" {
		startDate = startParam
		endDate = endParam
		if parsedStart, err := time.Parse("2006-01-02", startDate); err == nil {
			targetYear = parsedStart.Year()
			targetMonth = int(parsedStart.Month())
		}
	} else {
		startDate = fmt.Sprintf("%04d-%02d-01", targetYear, targetMonth)
		nextMonth := time.Date(targetYear, time.Month(targetMonth)+1, 1, 0, 0, 0, 0, time.UTC)
		endDate = nextMonth.AddDate(0, 0, -1).Format("2006-01-02")
	}

	var attendanceRecords []models.StaffAttendance
	database.DB.Preload("MarkedByUser").
		Where("user_id = ? AND business_date >= ? AND business_date <= ?", user.ID, startDate, endDate).
		Order("business_date desc").
		Find(&attendanceRecords)

	fullDay := 0
	half1 := 0
	half2 := 0
	absent := 0
	leave := 0
	historyItems := make([]dto.StaffAttendanceHistoryItem, 0, len(attendanceRecords))

	for _, r := range attendanceRecords {
		markedByName := "Admin"
		if r.MarkedByUser != nil {
			markedByName = r.MarkedByUser.FullName
		}
		switch r.Status {
		case models.AttendanceStatusFullDay:
			fullDay++
		case models.AttendanceStatusFirstHalf:
			half1++
		case models.AttendanceStatusSecondHalf:
			half2++
		case models.AttendanceStatusAbsent:
			absent++
		case models.AttendanceStatusLeave:
			leave++
		}
		historyItems = append(historyItems, dto.StaffAttendanceHistoryItem{
			ID:           r.ID,
			BusinessDate: r.BusinessDate.Format("2006-01-02"),
			Status:       string(r.Status),
			Remarks:      r.Remarks,
			MarkedByName: markedByName,
			CreatedAt:    r.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	totalMarked := len(attendanceRecords)
	var attRate float64 = 0
	if totalMarked > 0 {
		presentWeight := float64(fullDay) + (float64(half1+half2) * 0.5)
		attRate = (presentWeight / float64(totalMarked)) * 100
	}

	monthNames := []string{
		"", "January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December",
	}

	attSummary := dto.StaffAttendanceMonthSummary{
		Year:                 targetYear,
		Month:                targetMonth,
		MonthName:            monthNames[targetMonth],
		StartDate:            startDate,
		EndDate:              endDate,
		TotalBusinessDays:    totalMarked,
		FullDayCount:         fullDay,
		FirstHalfCount:       half1,
		SecondHalfCount:      half2,
		LeaveCount:           leave,
		AbsentCount:          absent,
		AttendancePercentage: attRate,
		History:              historyItems,
	}

	// 3. Operational Activity
	var vCreatedCount int64
	var vCreatedTotal decimal.Decimal
	database.DB.Model(&models.Voucher{}).Where("created_by_id = ?", user.ID).Count(&vCreatedCount)
	database.DB.Model(&models.Voucher{}).Where("created_by_id = ?", user.ID).Select("COALESCE(SUM(amount), 0)").Scan(&vCreatedTotal)

	var vApprovedCount int64
	var vApprovedTotal decimal.Decimal
	database.DB.Model(&models.Voucher{}).Where("approved_by_id = ?", user.ID).Count(&vApprovedCount)
	database.DB.Model(&models.Voucher{}).Where("approved_by_id = ?", user.ID).Select("COALESCE(SUM(amount), 0)").Scan(&vApprovedTotal)

	var eCreatedCount int64
	var eCreatedTotal decimal.Decimal
	database.DB.Model(&models.Expense{}).Where("created_by_id = ?", user.ID).Count(&eCreatedCount)
	database.DB.Model(&models.Expense{}).Where("created_by_id = ?", user.ID).Select("COALESCE(SUM(amount), 0)").Scan(&eCreatedTotal)

	var eApprovedCount int64
	var eApprovedTotal decimal.Decimal
	database.DB.Model(&models.Expense{}).Where("approved_by_id = ?", user.ID).Count(&eApprovedCount)
	database.DB.Model(&models.Expense{}).Where("approved_by_id = ?", user.ID).Select("COALESCE(SUM(amount), 0)").Scan(&eApprovedTotal)

	var dCount int64
	var dTotal decimal.Decimal
	database.DB.Model(&models.Donation{}).Where("created_by_id = ?", user.ID).Count(&dCount)
	database.DB.Model(&models.Donation{}).Where("created_by_id = ?", user.ID).Select("COALESCE(SUM(amount), 0)").Scan(&dTotal)

	operational := dto.StaffOperationalActivity{
		VouchersCreatedCount:        vCreatedCount,
		VouchersCreatedTotalAmount:  vCreatedTotal.StringFixed(2),
		VouchersApprovedCount:       vApprovedCount,
		VouchersApprovedTotalAmount: vApprovedTotal.StringFixed(2),
		ExpensesCreatedCount:        eCreatedCount,
		ExpensesCreatedTotalAmount:  eCreatedTotal.StringFixed(2),
		ExpensesApprovedCount:       eApprovedCount,
		ExpensesApprovedTotalAmount: eApprovedTotal.StringFixed(2),
		DonationsCollectedCount:     dCount,
		DonationsCollectedAmount:    dTotal.StringFixed(2),
	}

	// 4. Permissions
	perms := make([]string, 0)
	if user.Role == models.RoleAdmin {
		var allPerms []models.Permission
		database.DB.Select("code").Find(&allPerms)
		for _, p := range allPerms {
			perms = append(perms, p.Code)
		}
	} else {
		for _, p := range user.Permissions {
			perms = append(perms, p.Code)
		}
	}

	shared.SendSuccess(c, http.StatusOK, dto.StaffAnalyticsReportResponse{
		Profile:             profileDetail,
		AttendanceSummary:   attSummary,
		OperationalActivity: operational,
		Permissions:         perms,
	})
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

