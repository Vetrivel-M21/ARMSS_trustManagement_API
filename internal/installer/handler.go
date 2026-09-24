package installer

import (
	"net/http"

	"trust-management/backend/internal/config"
	"trust-management/backend/internal/dto"
	"trust-management/backend/internal/shared"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
	cfg     *config.Config
}

func NewHandler(cfg *config.Config) *Handler {
	return &Handler{service: NewService(cfg), cfg: cfg}
}

// RequireInstallerSecret is a simple shared-secret check — same "deterrent,
// not a cryptographic boundary" tradeoff as the desktop app's embedded Trust
// Portal client secret, just enough to stop the admin's inbox being spammed
// by random internet traffic.
func (h *Handler) RequireInstallerSecret(c *gin.Context) {
	if c.GetHeader("X-Installer-Secret") != h.cfg.InstallerAPISecret {
		shared.SendUnauthorized(c, "invalid installer secret")
		c.Abort()
		return
	}
	c.Next()
}

func (h *Handler) RequestOtp(c *gin.Context) {
	requestID, err := h.service.RequestOtp()
	if err != nil {
		shared.SendInternalError(c, "unable to send installer otp")
		return
	}
	shared.SendSuccess(c, http.StatusOK, dto.RequestOtpResponse{RequestID: requestID})
}

func (h *Handler) VerifyOtp(c *gin.Context) {
	var req dto.VerifyOtpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.SendBadRequest(c, "INVALID_REQUEST", "request_id and otp are required")
		return
	}

	valid, err := h.service.VerifyOtp(req.RequestID, req.Otp)
	if err != nil {
		shared.SendInternalError(c, "unable to verify installer otp")
		return
	}
	shared.SendSuccess(c, http.StatusOK, dto.VerifyOtpResponse{Valid: valid})
}
