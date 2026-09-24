package installer

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"trust-management/backend/internal/config"
	"trust-management/backend/internal/database"
	"trust-management/backend/internal/mail"
	"trust-management/backend/internal/models"
)

const otpTTL = 15 * time.Minute

type Service struct {
	cfg  *config.Config
	mail *mail.Service
}

func NewService(cfg *config.Config) *Service {
	return &Service{cfg: cfg, mail: mail.NewService(cfg)}
}

func (s *Service) RequestOtp() (string, error) {
	requestID, err := randomHex(16)
	if err != nil {
		return "", err
	}
	otp, err := randomOtp()
	if err != nil {
		return "", err
	}

	record := models.InstallerOtpRequest{
		ID:        requestID,
		OtpCode:   otp,
		ExpiresAt: time.Now().Add(otpTTL),
	}
	if err := database.DB.Create(&record).Error; err != nil {
		return "", err
	}

	subject := "ARMSS Gateway installer OTP"
	body := fmt.Sprintf("A new ARMSS Gateway installation is requesting an OTP.\n\nOTP: %s\n\nThis code expires in 15 minutes.", otp)
	if err := s.mail.Send(s.cfg.InstallerAdminEmail, subject, body); err != nil {
		return "", err
	}

	return requestID, nil
}

// VerifyOtp does the match/expiry/single-use check and the "mark verified"
// write as one atomic UPDATE (rather than a SELECT followed by a separate
// UPDATE) so two concurrent verify calls for the same request can't both
// observe "not yet verified" and both succeed.
func (s *Service) VerifyOtp(requestID, otp string) (bool, error) {
	result := database.DB.Model(&models.InstallerOtpRequest{}).
		Where("id = ? AND otp_code = ? AND verified = ? AND expires_at > ?", requestID, otp, false, time.Now()).
		Update("verified", true)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected == 1, nil
}

func randomHex(numBytes int) (string, error) {
	b := make([]byte, numBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func randomOtp() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
