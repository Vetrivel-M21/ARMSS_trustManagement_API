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
	if err := database.DB.Where("username = ? AND is_active = ?", req.Username, true).First(&user).Error; err != nil {
		return nil, errors.New("invalid username or password")
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
		User: dto.UserSummary{
			ID:       user.ID,
			Username: user.Username,
			FullName: user.FullName,
			Email:    user.Email,
			Role:     string(user.Role),
		},
	}, nil
}

func (s *AuthService) GetUserProfile(userID uint) (*dto.UserSummary, error) {
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return nil, errors.New("user not found")
	}
	return &dto.UserSummary{
		ID:       user.ID,
		Username: user.Username,
		FullName: user.FullName,
		Email:    user.Email,
		Role:     string(user.Role),
	}, nil
}
