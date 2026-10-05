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

func (s *AuthService) Login(req *dto.LoginRequest, jwtSecret string) (*dto.LoginResponse, error) {
	var user models.User
	if err := database.DB.Preload("Branch").Where("username = ? AND is_active = ?", req.Username, true).First(&user).Error; err != nil {
		return nil, errors.New("invalid username or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid username or password")
	}

	token, err := middleware.GenerateToken(&user, jwtSecret)
	if err != nil {
		return nil, errors.New("failed to generate access token")
	}

	summary := dto.UserSummary{
		ID:       user.ID,
		Username: user.Username,
		FullName: user.FullName,
		Email:    user.Email,
		Role:     string(user.Role),
		BranchID: user.BranchID,
	}
	if user.Branch != nil {
		summary.BranchCode = user.Branch.BranchCode
		summary.BranchName = user.Branch.Name
	}

	return &dto.LoginResponse{
		Token: token,
		User:  summary,
	}, nil
}

func (s *AuthService) GetUserProfile(userID uint) (*dto.UserSummary, error) {
	var user models.User
	if err := database.DB.Preload("Branch").First(&user, userID).Error; err != nil {
		return nil, errors.New("user not found")
	}
	summary := dto.UserSummary{
		ID:       user.ID,
		Username: user.Username,
		FullName: user.FullName,
		Email:    user.Email,
		Role:     string(user.Role),
		BranchID: user.BranchID,
	}
	if user.Branch != nil {
		summary.BranchCode = user.Branch.BranchCode
		summary.BranchName = user.Branch.Name
	}
	return &summary, nil
}
