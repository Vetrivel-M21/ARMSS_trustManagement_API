package middleware

import (
	"fmt"
	"net/http"

	"trust-management/backend/internal/shared"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type DeviceClaims struct {
	DeviceID string `json:"device_id"`
	Type     string `json:"type"`
	jwt.RegisteredClaims
}

// RequireDeviceToken restricts a route group to requests carrying a valid
// X-Device-Token — this is what makes the API non-functional from a plain
// browser, since only a pre-registered client (the MIS Desktop app) can
// obtain one via POST /api/v1/device/auth.
func RequireDeviceToken(deviceJWTSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("X-Device-Token")
		if tokenString == "" {
			shared.SendError(c, http.StatusForbidden, "DEVICE_TOKEN_MISSING", "this application must be accessed through the desktop client")
			c.Abort()
			return
		}

		claims := &DeviceClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(deviceJWTSecret), nil
		})

		if err != nil || !token.Valid || claims.Type != "device" {
			shared.SendError(c, http.StatusForbidden, "DEVICE_TOKEN_INVALID", "invalid or expired device token")
			c.Abort()
			return
		}

		c.Set("device_id", claims.DeviceID)
		c.Next()
	}
}
