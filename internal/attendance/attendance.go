package attendance

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
)

type AttendanceHandler struct{}

func NewAttendanceHandler() *AttendanceHandler {
	return &AttendanceHandler{}
}

// parseDateOrDefault parses a YYYY-MM-DD string or returns the current business date
func parseDateOrDefault(dStr string) time.Time {
	clean := strings.TrimSpace(dStr)
	if clean != "" {
		if t, err := time.Parse("2006-01-02", clean); err == nil {
			return t
		}
	}
	return shared.GetCurrentBusinessDate()
}

// GetAttendanceSheet returns the attendance marking sheet for a given business date
func (h *AttendanceHandler) GetAttendanceSheet(c *gin.Context) {
	dateParam := c.Query("date")
	bizDate := parseDateOrDefault(dateParam)
	dateStr := bizDate.Format("2006-01-02")

	userQuery := database.DB.Preload("Branch").
		Where("is_active = ? AND employment_status = ?", true, models.EmploymentStatusActive).
		Order("full_name asc")

	branchIDParam := strings.TrimSpace(c.Query("branch_id"))
	if branchIDParam != "" && branchIDParam != "ALL" {
		if bID, err := strconv.ParseUint(branchIDParam, 10, 32); err == nil && bID > 0 {
			userQuery = userQuery.Where("branch_id = ?", uint(bID))
		}
	}

	var activeUsers []models.User
	if err := userQuery.Find(&activeUsers).Error; err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to fetch staff list: "+err.Error())
		return
	}

	// Fetch existing attendance records for this date
	var existingRecords []models.StaffAttendance
	database.DB.Preload("MarkedByUser").
		Where("business_date = ?", dateStr).
		Find(&existingRecords)

	recordMap := make(map[uint]models.StaffAttendance, len(existingRecords))
	for _, r := range existingRecords {
		recordMap[r.UserID] = r
	}

	items := make([]dto.AttendanceSheetItem, 0, len(activeUsers))
	presentFull := 0
	present1st := 0
	present2nd := 0
	absent := 0
	leave := 0

	for _, u := range activeUsers {
		branchName := "Main Branch"
		branchID := uint(1)
		if u.Branch != nil {
			branchName = u.Branch.Name
			branchID = u.Branch.ID
		} else if u.BranchID != nil {
			branchID = *u.BranchID
		}

		status := "FULL_DAY" // default for easy marking
		remarks := ""
		var attID *uint
		markedByName := ""
		updatedAtStr := ""

		if rec, exists := recordMap[u.ID]; exists {
			attID = &rec.ID
			status = string(rec.Status)
			remarks = rec.Remarks
			if rec.MarkedByUser != nil {
				markedByName = rec.MarkedByUser.FullName
			}
			updatedAtStr = rec.UpdatedAt.Format("2006-01-02 15:04:05")
		}

		switch status {
		case "FULL_DAY":
			presentFull++
		case "FIRST_HALF":
			present1st++
		case "SECOND_HALF":
			present2nd++
		case "ABSENT":
			absent++
		case "LEAVE":
			leave++
		}

		items = append(items, dto.AttendanceSheetItem{
			UserID:           u.ID,
			StaffCode:        u.StaffCode,
			FullName:         u.FullName,
			Designation:      u.Designation,
			Department:       u.Department,
			PhotoPath:        u.PhotoPath,
			BranchID:         branchID,
			BranchName:       branchName,
			AttendanceID:     attID,
			Status:           status,
			Remarks:          remarks,
			MarkedByUserName: markedByName,
			UpdatedAt:        updatedAtStr,
		})
	}

	shared.SendSuccess(c, http.StatusOK, dto.AttendanceSheetResponse{
		BusinessDate:    dateStr,
		TotalStaff:      len(activeUsers),
		PresentFullDay:  presentFull,
		Present1stHalf:  present1st,
		Present2ndHalf:  present2nd,
		AbsentCount:     absent,
		LeaveCount:      leave,
		Items:           items,
	})
}

// BatchSubmitAttendance saves or updates attendance for multiple staff members on a business date
func (h *AttendanceHandler) BatchSubmitAttendance(c *gin.Context) {
	var req dto.BatchAttendanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.SendAppError(c, http.StatusBadRequest, err.Error())
		return
	}

	bizDate, err := time.Parse("2006-01-02", strings.TrimSpace(req.BusinessDate))
	if err != nil {
		shared.SendAppError(c, http.StatusBadRequest, "Invalid business date format. Must be YYYY-MM-DD.")
		return
	}

	adminIDVal, _ := c.Get("userID")
	adminID, _ := adminIDVal.(uint)

	tx := database.DB.Begin()

	savedCount := 0
	for _, entry := range req.Entries {
		if entry.UserID == 0 {
			continue
		}

		status := strings.ToUpper(strings.TrimSpace(entry.Status))
		switch status {
		case "FULL_DAY", "FIRST_HALF", "SECOND_HALF", "ABSENT", "LEAVE":
			// valid
		default:
			tx.Rollback()
			shared.SendAppError(c, http.StatusBadRequest, fmt.Sprintf("Invalid status '%s' for user %d", status, entry.UserID))
			return
		}

		var user models.User
		if err := tx.First(&user, entry.UserID).Error; err != nil {
			continue // Skip non-existing user
		}

		branchID := uint(1)
		if user.BranchID != nil && *user.BranchID > 0 {
			branchID = *user.BranchID
		}

		var existing models.StaffAttendance
		findErr := tx.Where("user_id = ? AND business_date = ?", entry.UserID, bizDate.Format("2006-01-02")).First(&existing).Error

		if findErr == nil {
			// Update
			existing.Status = models.AttendanceStatus(status)
			existing.Remarks = strings.TrimSpace(entry.Remarks)
			existing.MarkedByUserID = adminID
			existing.BranchID = branchID
			if err := tx.Save(&existing).Error; err != nil {
				tx.Rollback()
				shared.SendAppError(c, http.StatusInternalServerError, "Failed to update attendance record")
				return
			}
		} else {
			// Create
			newRec := models.StaffAttendance{
				UserID:         entry.UserID,
				BranchID:       branchID,
				BusinessDate:   bizDate,
				Status:         models.AttendanceStatus(status),
				Remarks:        strings.TrimSpace(entry.Remarks),
				MarkedByUserID: adminID,
			}
			if err := tx.Create(&newRec).Error; err != nil {
				tx.Rollback()
				shared.SendAppError(c, http.StatusInternalServerError, "Failed to create attendance record")
				return
			}
		}
		savedCount++
	}

	auditPayload, _ := json.Marshal(map[string]interface{}{
		"business_date": req.BusinessDate,
		"total_entries": savedCount,
	})
	audit := models.AuditLog{
		UserID:     &adminID,
		Action:     "STAFF_ATTENDANCE_BATCH_RECORDED",
		EntityName: "StaffAttendance",
		EntityID:   adminID,
		AfterData:  shared.JSONOrNull(json.RawMessage(auditPayload)),
		IPAddress:  c.ClientIP(),
	}
	_ = tx.Create(&audit)

	tx.Commit()

	shared.SendSuccess(c, http.StatusOK, gin.H{
		"message":       fmt.Sprintf("Attendance recorded successfully for %d staff members on %s", savedCount, req.BusinessDate),
		"business_date": req.BusinessDate,
		"saved_count":   savedCount,
	})
}

// GetMyAttendance returns attendance history for the currently logged-in staff member
func (h *AttendanceHandler) GetMyAttendance(c *gin.Context) {
	adminIDVal, _ := c.Get("userID")
	userID, _ := adminIDVal.(uint)

	query := database.DB.Preload("MarkedByUser").Where("user_id = ?", userID)

	yearStr := strings.TrimSpace(c.Query("year"))
	monthStr := strings.TrimSpace(c.Query("month"))

	currentDate := shared.GetCurrentBusinessDate()
	targetYear := currentDate.Year()
	targetMonth := int(currentDate.Month())

	if y, err := strconv.Atoi(yearStr); err == nil && y > 2000 {
		targetYear = y
	}
	if m, err := strconv.Atoi(monthStr); err == nil && m >= 1 && m <= 12 {
		targetMonth = m
	}

	startDate := fmt.Sprintf("%04d-%02d-01", targetYear, targetMonth)
	// calculate end of month
	nextMonth := time.Date(targetYear, time.Month(targetMonth)+1, 1, 0, 0, 0, 0, time.UTC)
	endDate := nextMonth.AddDate(0, 0, -1).Format("2006-01-02")

	query = query.Where("business_date >= ? AND business_date <= ?", startDate, endDate).Order("business_date desc")

	var records []models.StaffAttendance
	if err := query.Find(&records).Error; err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to fetch attendance history: "+err.Error())
		return
	}

	fullDay := 0
	half1 := 0
	half2 := 0
	absent := 0
	leave := 0

	historyItems := make([]dto.StaffAttendanceHistoryItem, 0, len(records))
	for _, r := range records {
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

	totalMarked := len(records)
	var attRate float64 = 0
	if totalMarked > 0 {
		presentWeight := float64(fullDay) + (float64(half1+half2) * 0.5)
		attRate = (presentWeight / float64(totalMarked)) * 100
	}

	shared.SendSuccess(c, http.StatusOK, dto.StaffAttendanceMySummary{
		TotalDays:            totalMarked,
		FullDayCount:         fullDay,
		FirstHalfCount:       half1,
		SecondHalfCount:      half2,
		AbsentCount:          absent,
		LeaveCount:           leave,
		AttendancePercentage: attRate,
		History:              historyItems,
	})
}

// GetAttendanceReport returns aggregated attendance records across staff and date ranges
func (h *AttendanceHandler) GetAttendanceReport(c *gin.Context) {
	currentBizDate := shared.GetCurrentBusinessDate().Format("2006-01-02")

	startParam := strings.TrimSpace(c.Query("start_date"))
	if startParam == "" {
		startParam = currentBizDate
	}
	endParam := strings.TrimSpace(c.Query("end_date"))
	if endParam == "" {
		endParam = startParam
	}

	query := database.DB.Preload("User").Preload("Branch").Preload("MarkedByUser").
		Where("business_date >= ? AND business_date <= ?", startParam, endParam).
		Order("business_date desc, user_id asc")

	branchParam := strings.TrimSpace(c.Query("branch_id"))
	if branchParam != "" && branchParam != "ALL" {
		if bID, err := strconv.ParseUint(branchParam, 10, 32); err == nil && bID > 0 {
			query = query.Where("branch_id = ?", uint(bID))
		}
	}

	statusParam := strings.ToUpper(strings.TrimSpace(c.Query("status")))
	if statusParam != "" && statusParam != "ALL" {
		query = query.Where("status = ?", statusParam)
	}

	userIDParam := strings.TrimSpace(c.Query("user_id"))
	if userIDParam != "" && userIDParam != "ALL" {
		if uID, err := strconv.ParseUint(userIDParam, 10, 32); err == nil && uID > 0 {
			query = query.Where("user_id = ?", uint(uID))
		}
	}

	var records []models.StaffAttendance
	if err := query.Find(&records).Error; err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to fetch attendance report: "+err.Error())
		return
	}

	fullCount := 0
	half1Count := 0
	half2Count := 0
	absentCount := 0
	leaveCount := 0

	items := make([]dto.AttendanceReportItem, 0, len(records))
	for _, r := range records {
		staffCode := ""
		fullName := "Unknown Staff"
		designation := ""
		department := ""
		if r.User != nil {
			staffCode = r.User.StaffCode
			fullName = r.User.FullName
			designation = r.User.Designation
			department = r.User.Department
		}

		branchName := "Main Branch"
		if r.Branch != nil {
			branchName = r.Branch.Name
		}

		markedByName := "System Admin"
		if r.MarkedByUser != nil {
			markedByName = r.MarkedByUser.FullName
		}

		switch r.Status {
		case models.AttendanceStatusFullDay:
			fullCount++
		case models.AttendanceStatusFirstHalf:
			half1Count++
		case models.AttendanceStatusSecondHalf:
			half2Count++
		case models.AttendanceStatusAbsent:
			absentCount++
		case models.AttendanceStatusLeave:
			leaveCount++
		}

		items = append(items, dto.AttendanceReportItem{
			ID:           r.ID,
			BusinessDate: r.BusinessDate.Format("2006-01-02"),
			UserID:       r.UserID,
			StaffCode:    staffCode,
			FullName:     fullName,
			Designation:  designation,
			Department:   department,
			BranchID:     r.BranchID,
			BranchName:   branchName,
			Status:       string(r.Status),
			Remarks:      r.Remarks,
			MarkedByName: markedByName,
		})
	}

	total := len(records)
	var rate float64 = 0
	if total > 0 {
		rate = ((float64(fullCount) + float64(half1Count+half2Count)*0.5) / float64(total)) * 100
	}

	// Aggregate per employee summary for period
	userSummaryMap := make(map[uint]*dto.EmployeeAttendanceSummaryItem)
	for _, r := range records {
		s, exists := userSummaryMap[r.UserID]
		if !exists {
			staffCode := ""
			fullName := "Unknown Staff"
			phone := ""
			email := ""
			designation := ""
			department := ""
			empStatus := "ACTIVE"
			if r.User != nil {
				staffCode = r.User.StaffCode
				fullName = r.User.FullName
				phone = r.User.Phone
				email = r.User.Email
				designation = r.User.Designation
				department = r.User.Department
				empStatus = string(r.User.EmploymentStatus)
			}
			branchName := "Main Branch"
			if r.Branch != nil {
				branchName = r.Branch.Name
			}
			s = &dto.EmployeeAttendanceSummaryItem{
				UserID:           r.UserID,
				StaffCode:        staffCode,
				FullName:         fullName,
				Phone:            phone,
				Email:            email,
				Designation:      designation,
				Department:       department,
				BranchID:         r.BranchID,
				BranchName:       branchName,
				EmploymentStatus: empStatus,
			}
			userSummaryMap[r.UserID] = s
		}

		s.TotalDays++
		switch r.Status {
		case models.AttendanceStatusFullDay:
			s.FullDayCount++
		case models.AttendanceStatusFirstHalf:
			s.FirstHalfCount++
		case models.AttendanceStatusSecondHalf:
			s.SecondHalfCount++
		case models.AttendanceStatusAbsent:
			s.AbsentCount++
		case models.AttendanceStatusLeave:
			s.LeaveCount++
		}
	}

	employeeSummaries := make([]dto.EmployeeAttendanceSummaryItem, 0, len(userSummaryMap))
	for _, s := range userSummaryMap {
		if s.TotalDays > 0 {
			weight := float64(s.FullDayCount) + (float64(s.FirstHalfCount+s.SecondHalfCount) * 0.5)
			s.AttendancePercentage = (weight / float64(s.TotalDays)) * 100
		}
		employeeSummaries = append(employeeSummaries, *s)
	}

	shared.SendSuccess(c, http.StatusOK, dto.AttendanceReportResponse{
		StartDate:            startParam,
		EndDate:              endParam,
		TotalRecords:         total,
		FullDayCount:         fullCount,
		FirstHalfCount:       half1Count,
		SecondHalfCount:      half2Count,
		AbsentCount:          absentCount,
		LeaveCount:           leaveCount,
		AttendancePercentage: rate,
		Items:                items,
		EmployeeSummaries:    employeeSummaries,
	})
}

// GetStaffMasterReport returns complete demographic and employment data for all staff
func (h *AttendanceHandler) GetStaffMasterReport(c *gin.Context) {
	query := database.DB.Preload("Branch").Order("id asc")

	statusParam := strings.ToUpper(strings.TrimSpace(c.Query("status")))
	if statusParam != "" && statusParam != "ALL" {
		query = query.Where("employment_status = ?", statusParam)
	}

	branchParam := strings.TrimSpace(c.Query("branch_id"))
	if branchParam != "" && branchParam != "ALL" {
		if bID, err := strconv.ParseUint(branchParam, 10, 32); err == nil && bID > 0 {
			query = query.Where("branch_id = ?", uint(bID))
		}
	}

	var users []models.User
	if err := query.Find(&users).Error; err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to generate staff report: "+err.Error())
		return
	}

	reportItems := make([]dto.StaffMasterReportItem, 0, len(users))
	for _, u := range users {
		dob := ""
		if u.DateOfBirth != nil {
			dob = u.DateOfBirth.Format("2006-01-02")
		}
		doj := ""
		if u.DateOfJoining != nil {
			doj = u.DateOfJoining.Format("2006-01-02")
		}
		dor := ""
		if u.DateOfRelieving != nil {
			dor = u.DateOfRelieving.Format("2006-01-02")
		}

		branchName := "Main Branch"
		branchID := uint(1)
		if u.Branch != nil {
			branchName = u.Branch.Name
			branchID = u.Branch.ID
		} else if u.BranchID != nil {
			branchID = *u.BranchID
		}

		emer := ""
		if u.EmergencyContactName != "" {
			emer = u.EmergencyContactName
			if u.EmergencyContactPhone != "" {
				emer += " (" + u.EmergencyContactPhone + ")"
			}
		}

		reportItems = append(reportItems, dto.StaffMasterReportItem{
			ID:               u.ID,
			StaffCode:        u.StaffCode,
			Username:         u.Username,
			FullName:         u.FullName,
			Email:            u.Email,
			Phone:            u.Phone,
			Gender:           u.Gender,
			Education:        u.Education,
			MaritalStatus:    u.MaritalStatus,
			DateOfBirth:      dob,
			DateOfJoining:    doj,
			DateOfRelieving:  dor,
			Designation:      u.Designation,
			Department:       u.Department,
			Address:          u.AddressLine,
			City:             u.City,
			State:            u.State,
			Pincode:          u.Pincode,
			EmergencyContact: emer,
			Role:             string(u.Role),
			EmploymentStatus: string(u.EmploymentStatus),
			StatusRemarks:    u.StatusRemarks,
			BranchID:         branchID,
			BranchName:       branchName,
			IsActive:         u.IsActive,
		})
	}

	shared.SendSuccess(c, http.StatusOK, reportItems)
}
