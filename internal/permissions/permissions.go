package permissions

import (
	"net/http"
	"strconv"

	"trust-management/backend/internal/database"
	"trust-management/backend/internal/dto"
	"trust-management/backend/internal/models"
	"trust-management/backend/internal/shared"

	"github.com/gin-gonic/gin"
)

type PermissionHandler struct{}

func NewPermissionHandler() *PermissionHandler {
	return &PermissionHandler{}
}

// GetPermissions returns the master list of all available system permissions.
func (h *PermissionHandler) GetPermissions(c *gin.Context) {
	var perms []models.Permission
	if err := database.DB.Order("module asc, id asc").Find(&perms).Error; err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to load permissions: "+err.Error())
		return
	}
	shared.SendSuccess(c, http.StatusOK, perms)
}

// GetUserPermissions returns all permission codes assigned to a specific user.
func (h *PermissionHandler) GetUserPermissions(c *gin.Context) {
	idStr := c.Param("id")
	userID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil || userID == 0 {
		shared.SendAppError(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	var user models.User
	if err := database.DB.First(&user, uint(userID)).Error; err != nil {
		shared.SendAppError(c, http.StatusNotFound, "User not found")
		return
	}

	var permCodes []string
	if user.Role == models.RoleAdmin {
		// Admin has all permissions
		database.DB.Model(&models.Permission{}).Pluck("code", &permCodes)
	} else {
		database.DB.Table("permissions").
			Joins("JOIN user_permissions ON user_permissions.permission_id = permissions.id").
			Where("user_permissions.user_id = ?", uint(userID)).
			Pluck("permissions.code", &permCodes)
	}

	if permCodes == nil {
		permCodes = make([]string, 0)
	}

	shared.SendSuccess(c, http.StatusOK, dto.UserPermissionsResponse{
		UserID:          uint(userID),
		PermissionCodes: permCodes,
	})
}

// UpdateUserPermissions updates the permissions assigned to a specific user.
func (h *PermissionHandler) UpdateUserPermissions(c *gin.Context) {
	idStr := c.Param("id")
	userID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil || userID == 0 {
		shared.SendAppError(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	var user models.User
	if err := database.DB.First(&user, uint(userID)).Error; err != nil {
		shared.SendAppError(c, http.StatusNotFound, "User not found")
		return
	}

	var req dto.UpdateUserPermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.SendAppError(c, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Delete existing user permissions
	if err := tx.Where("user_id = ?", uint(userID)).Delete(&models.UserPermission{}).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to clear existing permissions: "+err.Error())
		return
	}

	// Insert newly selected permissions
	if len(req.PermissionCodes) > 0 {
		var matchedPerms []models.Permission
		if err := tx.Where("code IN ?", req.PermissionCodes).Find(&matchedPerms).Error; err != nil {
			tx.Rollback()
			shared.SendAppError(c, http.StatusInternalServerError, "Failed to resolve permissions: "+err.Error())
			return
		}

		userPerms := make([]models.UserPermission, 0, len(matchedPerms))
		for _, p := range matchedPerms {
			userPerms = append(userPerms, models.UserPermission{
				UserID:       uint(userID),
				PermissionID: p.ID,
			})
		}

		if len(userPerms) > 0 {
			if err := tx.Create(&userPerms).Error; err != nil {
				tx.Rollback()
				shared.SendAppError(c, http.StatusInternalServerError, "Failed to assign permissions: "+err.Error())
				return
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to commit permissions: "+err.Error())
		return
	}

	shared.SendSuccess(c, http.StatusOK, gin.H{
		"message":          "Permissions updated successfully",
		"user_id":          uint(userID),
		"permission_count": len(req.PermissionCodes),
	})
}
