package dto

type RequestOtpResponse struct {
	RequestID string `json:"request_id"`
}

type VerifyOtpRequest struct {
	RequestID string `json:"request_id" binding:"required"`
	Otp       string `json:"otp" binding:"required"`
}

type VerifyOtpResponse struct {
	Valid bool `json:"valid"`
}
