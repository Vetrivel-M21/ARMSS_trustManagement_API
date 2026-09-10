package device

import (
	"errors"
	"time"

	"trust-management/backend/internal/database"
	"trust-management/backend/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// DeviceTokenTTL is intentionally short — the desktop app is expected to
// refresh well before expiry (see mis_desktop's TrustPortalTokenController).
const DeviceTokenTTL = 15 * time.Minute

type DeviceService struct{}

func NewDeviceService() *DeviceService {
	return &DeviceService{}
}

func (s *DeviceService) ValidateClient(clientID, clientSecret string) (*models.DeviceClient, error) {
	var deviceClient models.DeviceClient
	if err := database.DB.Where("id = ? AND is_active = ?", clientID, true).First(&deviceClient).Error; err != nil {
		return nil, errors.New("invalid device credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(deviceClient.ClientSecretHash), []byte(clientSecret)); err != nil {
		return nil, errors.New("invalid device credentials")
	}

	database.DB.Model(&deviceClient).Update("last_used_at", time.Now())

	return &deviceClient, nil
}

func (s *DeviceService) IssueDeviceToken(deviceID string, deviceJWTSecret string) (string, time.Time, error) {
	expiresAt := time.Now().Add(DeviceTokenTTL)

	claims := jwt.MapClaims{
		"device_id": deviceID,
		"type":      "device",
		"exp":       expiresAt.Unix(),
		"iat":       time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(deviceJWTSecret))
	if err != nil {
		return "", time.Time{}, err
	}

	return signed, expiresAt, nil
}
