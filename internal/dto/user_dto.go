package dto

type CreateUserRequest struct {
	Username string `json:"username" binding:"required"`
	FullName string `json:"full_name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Role     string `json:"role" binding:"required"` // STAFF / ADMIN
}

type UpdateUserRequest struct {
	FullName string  `json:"full_name"`
	Email    string  `json:"email"`
	Role     string  `json:"role"`
	IsActive *bool   `json:"is_active"`
	Password *string `json:"password"` // set to reset the password; omit to leave unchanged
}
