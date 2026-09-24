package device

import (
	"net/http"
	"time"

	"trust-management/backend/internal/config"
	"trust-management/backend/internal/dto"
	"trust-management/backend/internal/shared"

	"github.com/gin-gonic/gin"
)

type DeviceHandler struct {
	deviceService *DeviceService
	cfg           *config.Config
}

func NewDeviceHandler(cfg *config.Config) *DeviceHandler {
	return &DeviceHandler{
		deviceService: NewDeviceService(),
		cfg:           cfg,
	}
}

// Authenticate issues a short-lived device token to a pre-registered client
// (the MIS Desktop app). Deliberately NOT behind RequireDeviceToken — this is
// how a client obtains its first token.
func (h *DeviceHandler) Authenticate(c *gin.Context) {
	var req dto.DeviceAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.SendBadRequest(c, "INVALID_REQUEST", "client_id and client_secret are required")
		return
	}

	deviceClient, err := h.deviceService.ValidateClient(req.ClientID, req.ClientSecret)
	if err != nil {
		shared.SendError(c, http.StatusUnauthorized, "DEVICE_AUTH_FAILED", "invalid device credentials")
		return
	}

	token, expiresAt, err := h.deviceService.IssueDeviceToken(deviceClient.ID, h.cfg.DeviceJWTSecret)
	if err != nil {
		shared.SendInternalError(c, "unable to issue device token")
		return
	}

	shared.SendSuccess(c, http.StatusOK, dto.DeviceAuthResponse{
		DeviceToken: token,
		ExpiresAt:   expiresAt.Format(time.RFC3339),
	})
}
