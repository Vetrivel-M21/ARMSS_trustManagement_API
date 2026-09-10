package dto

type DeviceAuthRequest struct {
	ClientID     string `json:"client_id" binding:"required"`
	ClientSecret string `json:"client_secret" binding:"required"`
}

type DeviceAuthResponse struct {
	DeviceToken string `json:"device_token"`
	ExpiresAt   string `json:"expires_at"`
}
