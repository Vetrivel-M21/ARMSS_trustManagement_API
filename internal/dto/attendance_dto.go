package dto

type AttendanceSheetItem struct {
	UserID           uint   `json:"user_id"`
	StaffCode        string `json:"staff_code"`
	FullName         string `json:"full_name"`
	Designation      string `json:"designation"`
	Department       string `json:"department"`
	PhotoPath        string `json:"photo_path"`
	BranchID         uint   `json:"branch_id"`
	BranchName       string `json:"branch_name"`
	AttendanceID     *uint  `json:"attendance_id,omitempty"`
	Status           string `json:"status"` // FULL_DAY, FIRST_HALF, SECOND_HALF, ABSENT, LEAVE
	Remarks          string `json:"remarks"`
	MarkedByUserName string `json:"marked_by_user_name,omitempty"`
	UpdatedAt        string `json:"updated_at,omitempty"`
}

type AttendanceSheetResponse struct {
	BusinessDate    string                `json:"business_date"`
	TotalStaff      int                   `json:"total_staff"`
	PresentFullDay  int                   `json:"present_full_day"`
	Present1stHalf  int                   `json:"present_1st_half"`
	Present2ndHalf  int                   `json:"present_2nd_half"`
	AbsentCount     int                   `json:"absent_count"`
	LeaveCount      int                   `json:"leave_count"`
	Items           []AttendanceSheetItem `json:"items"`
}

type BatchAttendanceEntry struct {
	UserID  uint   `json:"user_id" binding:"required"`
	Status  string `json:"status" binding:"required"` // FULL_DAY, FIRST_HALF, SECOND_HALF, ABSENT, LEAVE
	Remarks string `json:"remarks"`
}

type BatchAttendanceRequest struct {
	BusinessDate string                 `json:"business_date" binding:"required"` // YYYY-MM-DD
	Entries      []BatchAttendanceEntry `json:"entries" binding:"required"`
}

type StaffAttendanceHistoryItem struct {
	ID           uint   `json:"id"`
	BusinessDate string `json:"business_date"`
	Status       string `json:"status"`
	Remarks      string `json:"remarks"`
	MarkedByName string `json:"marked_by_name"`
	CreatedAt    string `json:"created_at"`
}

type StaffAttendanceMySummary struct {
	TotalDays            int                          `json:"total_days"`
	FullDayCount         int                          `json:"full_day_count"`
	FirstHalfCount       int                          `json:"first_half_count"`
	SecondHalfCount      int                          `json:"second_half_count"`
	AbsentCount          int                          `json:"absent_count"`
	LeaveCount           int                          `json:"leave_count"`
	AttendancePercentage float64                      `json:"attendance_percentage"`
	History              []StaffAttendanceHistoryItem `json:"history"`
}

type AttendanceReportItem struct {
	ID           uint   `json:"id"`
	BusinessDate string `json:"business_date"`
	UserID       uint   `json:"user_id"`
	StaffCode    string `json:"staff_code"`
	FullName     string `json:"full_name"`
	Designation  string `json:"designation"`
	Department   string `json:"department"`
	BranchID     uint   `json:"branch_id"`
	BranchName   string `json:"branch_name"`
	Status       string `json:"status"`
	Remarks      string `json:"remarks"`
	MarkedByName string `json:"marked_by_name"`
}

type EmployeeAttendanceSummaryItem struct {
	UserID               uint    `json:"user_id"`
	StaffCode            string  `json:"staff_code"`
	FullName             string  `json:"full_name"`
	Phone                string  `json:"phone"`
	Email                string  `json:"email"`
	Designation          string  `json:"designation"`
	Department           string  `json:"department"`
	BranchID             uint    `json:"branch_id"`
	BranchName           string  `json:"branch_name"`
	EmploymentStatus     string  `json:"employment_status"`
	TotalDays            int     `json:"total_days"`
	FullDayCount         int     `json:"full_day_count"`
	FirstHalfCount       int     `json:"first_half_count"`
	SecondHalfCount      int     `json:"second_half_count"`
	LeaveCount           int     `json:"leave_count"`
	AbsentCount          int     `json:"absent_count"`
	AttendancePercentage float64 `json:"attendance_percentage"`
}

type AttendanceReportResponse struct {
	StartDate            string                          `json:"start_date"`
	EndDate              string                          `json:"end_date"`
	TotalRecords         int                             `json:"total_records"`
	FullDayCount         int                             `json:"full_day_count"`
	FirstHalfCount       int                             `json:"first_half_count"`
	SecondHalfCount      int                             `json:"second_half_count"`
	AbsentCount          int                             `json:"absent_count"`
	LeaveCount           int                             `json:"leave_count"`
	AttendancePercentage float64                         `json:"attendance_percentage"`
	Items                []AttendanceReportItem          `json:"items"`
	EmployeeSummaries    []EmployeeAttendanceSummaryItem `json:"employee_summaries"`
}

type StaffMasterReportItem struct {
	ID               uint   `json:"id"`
	StaffCode        string `json:"staff_code"`
	Username         string `json:"username"`
	FullName         string `json:"full_name"`
	Email            string `json:"email"`
	Phone            string `json:"phone"`
	Gender           string `json:"gender"`
	Education        string `json:"education"`
	MaritalStatus    string `json:"marital_status"`
	DateOfBirth      string `json:"date_of_birth,omitempty"`
	DateOfJoining    string `json:"date_of_joining,omitempty"`
	DateOfRelieving  string `json:"date_of_relieving,omitempty"`
	Designation      string `json:"designation"`
	Department       string `json:"department"`
	Address          string `json:"address"`
	City             string `json:"city"`
	State            string `json:"state"`
	Pincode          string `json:"pincode"`
	EmergencyContact string `json:"emergency_contact"`
	Role             string `json:"role"`
	EmploymentStatus string `json:"employment_status"`
	StatusRemarks    string `json:"status_remarks"`
	BranchID         uint   `json:"branch_id"`
	BranchName       string `json:"branch_name"`
	IsActive         bool   `json:"is_active"`
}

type StaffProfileDetail struct {
	ID                    uint   `json:"id"`
	StaffCode             string `json:"staff_code"`
	Username              string `json:"username"`
	FullName              string `json:"full_name"`
	Email                 string `json:"email"`
	Phone                 string `json:"phone"`
	Gender                string `json:"gender"`
	Education             string `json:"education"`
	MaritalStatus         string `json:"marital_status"`
	DateOfBirth           string `json:"date_of_birth,omitempty"`
	DateOfJoining         string `json:"date_of_joining,omitempty"`
	DateOfRelieving       string `json:"date_of_relieving,omitempty"`
	TenureFormatted       string `json:"tenure_formatted"`
	Designation           string `json:"designation"`
	Department            string `json:"department"`
	AddressLine           string `json:"address_line"`
	City                  string `json:"city"`
	State                 string `json:"state"`
	Pincode               string `json:"pincode"`
	PhotoPath             string `json:"photo_path"`
	EmergencyContactName  string `json:"emergency_contact_name"`
	EmergencyContactPhone string `json:"emergency_contact_phone"`
	Role                  string `json:"role"`
	EmploymentStatus      string `json:"employment_status"`
	StatusRemarks         string `json:"status_remarks"`
	BranchID              uint   `json:"branch_id"`
	BranchName            string `json:"branch_name"`
	IsActive              bool   `json:"is_active"`
}

type StaffOperationalActivity struct {
	VouchersCreatedCount        int64  `json:"vouchers_created_count"`
	VouchersCreatedTotalAmount  string `json:"vouchers_created_total_amount"`
	VouchersApprovedCount       int64  `json:"vouchers_approved_count"`
	VouchersApprovedTotalAmount string `json:"vouchers_approved_total_amount"`
	ExpensesCreatedCount        int64  `json:"expenses_created_count"`
	ExpensesCreatedTotalAmount  string `json:"expenses_created_total_amount"`
	ExpensesApprovedCount       int64  `json:"expenses_approved_count"`
	ExpensesApprovedTotalAmount string `json:"expenses_approved_total_amount"`
	DonationsCollectedCount     int64  `json:"donations_collected_count"`
	DonationsCollectedAmount    string `json:"donations_collected_amount"`
}

type StaffAttendanceMonthSummary struct {
	Year                 int                          `json:"year"`
	Month                int                          `json:"month"`
	MonthName            string                       `json:"month_name"`
	StartDate            string                       `json:"start_date,omitempty"`
	EndDate              string                       `json:"end_date,omitempty"`
	TotalBusinessDays    int                          `json:"total_business_days"`
	FullDayCount         int                          `json:"full_day_count"`
	FirstHalfCount       int                          `json:"first_half_count"`
	SecondHalfCount      int                          `json:"second_half_count"`
	LeaveCount           int                          `json:"leave_count"`
	AbsentCount          int                          `json:"absent_count"`
	AttendancePercentage float64                      `json:"attendance_percentage"`
	History              []StaffAttendanceHistoryItem `json:"history"`
}

type StaffAnalyticsReportResponse struct {
	Profile             StaffProfileDetail          `json:"profile"`
	AttendanceSummary   StaffAttendanceMonthSummary `json:"attendance_summary"`
	OperationalActivity StaffOperationalActivity    `json:"operational_activity"`
	Permissions         []string                    `json:"permissions"`
}

