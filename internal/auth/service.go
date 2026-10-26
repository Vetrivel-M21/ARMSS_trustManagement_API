package auth

import (
	"errors"

	"trust-management/backend/internal/database"
	"trust-management/backend/internal/dto"
	"trust-management/backend/internal/middleware"
	"trust-management/backend/internal/models"

	"golang.org/x/crypto/bcrypt"
)

type AuthService struct{}

func NewAuthService() *AuthService {
	return &AuthService{}
}

func buildUserSummary(user *models.User) dto.UserSummary {
	dobStr := ""
	if user.DateOfBirth != nil {
		dobStr = user.DateOfBirth.Format("2006-01-02")
	}
	dojStr := ""
	if user.DateOfJoining != nil {
		dojStr = user.DateOfJoining.Format("2006-01-02")
	}

	summary := dto.UserSummary{
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
		BranchID:              user.BranchID,
		IsActive:              user.IsActive,
	}

	if user.Branch != nil {
		summary.BranchCode = user.Branch.BranchCode
		summary.BranchName = user.Branch.Name
	}

	var permCodes []string
	if user.Role == models.RoleAdmin {
		database.DB.Model(&models.Permission{}).Pluck("code", &permCodes)
	} else {
		database.DB.Table("permissions").
			Joins("JOIN user_permissions ON user_permissions.permission_id = permissions.id").
			Where("user_permissions.user_id = ?", user.ID).
			Pluck("permissions.code", &permCodes)
	}
	if permCodes == nil {
		permCodes = make([]string, 0)
	}
	summary.Permissions = permCodes

	return summary
}

func (s *AuthService) Login(req *dto.LoginRequest, jwtSecret string) (*dto.LoginResponse, error) {
	var user models.User
	if err := database.DB.Preload("Branch").Where("username = ?", req.Username).First(&user).Error; err != nil {
		return nil, errors.New("invalid username or password")
	}

	if !user.IsActive {
		return nil, errors.New("this account is inactive; please contact administrator")
	}

	if user.EmploymentStatus == models.EmploymentStatusTemporarilyBlocked {
		return nil, errors.New("your account has been temporarily blocked; please contact administrator")
	}

	if user.EmploymentStatus == models.EmploymentStatusRelieved {
		return nil, errors.New("this staff account has been relieved")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid username or password")
	}

	token, err := middleware.GenerateToken(&user, jwtSecret)
	if err != nil {
		return nil, errors.New("failed to generate access token")
	}

	return &dto.LoginResponse{
		Token: token,
		User:  buildUserSummary(&user),
	}, nil
}

func (s *AuthService) GetUserProfile(userID uint) (*dto.UserSummary, error) {
	var user models.User
	if err := database.DB.Preload("Branch").First(&user, userID).Error; err != nil {
		return nil, errors.New("user not found")
	}
	summary := buildUserSummary(&user)
	return &summary, nil
}
