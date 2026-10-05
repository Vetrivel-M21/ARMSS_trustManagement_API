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
	ID         uint   `json:"id"`
	Username   string `json:"username"`
	FullName   string `json:"full_name"`
	Email      string `json:"email"`
	Role       string `json:"role"`
	BranchID   *uint  `json:"branch_id"`
	BranchCode string `json:"branch_code,omitempty"`
	BranchName string `json:"branch_name,omitempty"`
}
