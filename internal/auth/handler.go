package auth

import (
	"net/http"

	"trust-management/backend/internal/config"
	"trust-management/backend/internal/dto"
	"trust-management/backend/internal/shared"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService *AuthService
	cfg         *config.Config
}

func NewAuthHandler(cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		authService: NewAuthService(),
		cfg:         cfg,
	}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.SendBadRequest(c, "INVALID_INPUT", "Username and password are required")
		return
	}

	resp, err := h.authService.Login(&req, h.cfg.JWTSecret)
	if err != nil {
		shared.SendUnauthorized(c, err.Error())
		return
	}

	shared.SendSuccess(c, http.StatusOK, resp)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	shared.SendSuccess(c, http.StatusOK, gin.H{"message": "Logged out successfully"})
}

func (h *AuthHandler) Me(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		shared.SendUnauthorized(c, "Not authenticated")
		return
	}

	userID, ok := userIDVal.(uint)
	if !ok {
		shared.SendUnauthorized(c, "Invalid session user ID")
		return
	}

	profile, err := h.authService.GetUserProfile(userID)
	if err != nil {
		shared.SendNotFound(c, err.Error())
		return
	}

	shared.SendSuccess(c, http.StatusOK, profile)
}
