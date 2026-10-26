package middleware

import (
	"trust-management/backend/internal/database"
	"trust-management/backend/internal/models"
	"trust-management/backend/internal/shared"

	"github.com/gin-gonic/gin"
)

// RequirePermission checks whether the authenticated user has the specified action permission.
// Users with RoleAdmin have full access to all system actions by default.
func RequirePermission(permissionCode string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRoleVal, exists := c.Get("role")
		if !exists {
			shared.SendUnauthorized(c, "User session not authenticated")
			c.Abort()
			return
		}

		userRole, ok := userRoleVal.(models.Role)
		if ok && userRole == models.RoleAdmin {
			c.Next()
			return
		}

		userIDVal, exists := c.Get("userID")
		if !exists {
			userIDVal, exists = c.Get("user_id")
		}
		if !exists {
			shared.SendUnauthorized(c, "User ID missing from session")
			c.Abort()
			return
		}

		userID, ok := userIDVal.(uint)
		if !ok {
			shared.SendUnauthorized(c, "Invalid user ID in session")
			c.Abort()
			return
		}

		var count int64
		err := database.DB.Table("user_permissions").
			Joins("JOIN permissions ON permissions.id = user_permissions.permission_id").
			Where("user_permissions.user_id = ? AND permissions.code = ?", userID, permissionCode).
			Count(&count).Error

		if err != nil || count == 0 {
			shared.SendForbidden(c, "Access denied: Missing required permission '"+permissionCode+"'")
			c.Abort()
			return
		}

		c.Next()
	}
}
