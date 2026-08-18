package middleware

import (
	"trust-management/backend/internal/models"
	"trust-management/backend/internal/shared"

	"github.com/gin-gonic/gin"
)

func RequireRole(roles ...models.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRoleVal, exists := c.Get("role")
		if !exists {
			shared.SendUnauthorized(c, "User session not authenticated")
			c.Abort()
			return
		}

		userRole, ok := userRoleVal.(models.Role)
		if !ok {
			shared.SendUnauthorized(c, "Invalid user role in session")
			c.Abort()
			return
		}

		allowed := false
		for _, role := range roles {
			if userRole == role {
				allowed = true
				break
			}
		}

		if !allowed {
			shared.SendForbidden(c, "Administrative permission required for this action")
			c.Abort()
			return
		}

		c.Next()
	}
}
