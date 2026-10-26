package dto

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token string      `json:"token"`
	User  UserSummary `json:"user"`
}

type UserSummary struct {
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
	StatusRemarks         string `json:"status_remarks,omitempty"`
	BranchID              *uint  `json:"branch_id"`
	BranchCode            string   `json:"branch_code,omitempty"`
	BranchName            string   `json:"branch_name,omitempty"`
	Permissions           []string `json:"permissions"`
	IsActive              bool     `json:"is_active"`
}

