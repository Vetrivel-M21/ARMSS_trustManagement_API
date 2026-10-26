package dto

type CreateUserRequest struct {
	StaffCode             string `json:"staff_code"`
	Username              string `json:"username" binding:"required"`
	FullName              string `json:"full_name" binding:"required"`
	Email                 string `json:"email" binding:"required,email"`
	Phone                 string `json:"phone"`
	Gender                string `json:"gender"`
	Education             string `json:"education"`
	MaritalStatus         string `json:"marital_status"`
	DateOfBirth           string `json:"date_of_birth"`   // YYYY-MM-DD
	DateOfJoining         string `json:"date_of_joining"` // YYYY-MM-DD
	Designation           string `json:"designation"`
	Department            string `json:"department"`
	AddressLine           string `json:"address_line"`
	City                  string `json:"city"`
	State                 string `json:"state"`
	Pincode               string `json:"pincode"`
	PhotoPath             string `json:"photo_path"`
	EmergencyContactName  string `json:"emergency_contact_name"`
	EmergencyContactPhone string `json:"emergency_contact_phone"`
	Password              string `json:"password" binding:"required,min=8"`
	Role                  string `json:"role" binding:"required"` // STAFF / ADMIN
	BranchID              *uint  `json:"branch_id"`
}

type UpdateUserRequest struct {
	FullName              string  `json:"full_name"`
	Email                 string  `json:"email"`
	Phone                 string  `json:"phone"`
	Gender                string  `json:"gender"`
	Education             string  `json:"education"`
	MaritalStatus         string  `json:"marital_status"`
	DateOfBirth           *string `json:"date_of_birth"`
	DateOfJoining         *string `json:"date_of_joining"`
	DateOfRelieving       *string `json:"date_of_relieving"`
	Designation           string  `json:"designation"`
	Department            string  `json:"department"`
	AddressLine           string  `json:"address_line"`
	City                  string  `json:"city"`
	State                 string  `json:"state"`
	Pincode               string  `json:"pincode"`
	PhotoPath             string  `json:"photo_path"`
	EmergencyContactName  string  `json:"emergency_contact_name"`
	EmergencyContactPhone string  `json:"emergency_contact_phone"`
	Role                  string  `json:"role"`
	EmploymentStatus      string  `json:"employment_status"` // ACTIVE / RELIEVED / TEMPORARILY_BLOCKED
	StatusRemarks         string  `json:"status_remarks"`
	BranchID              *uint   `json:"branch_id"`
	IsActive              *bool   `json:"is_active"`
	Password              *string `json:"password"` // set to reset the password; omit to leave unchanged
}

type UpdateUserPermissionsRequest struct {
	PermissionCodes []string `json:"permission_codes"`
}

type UserPermissionsResponse struct {
	UserID          uint     `json:"user_id"`
	PermissionCodes []string `json:"permission_codes"`
}

