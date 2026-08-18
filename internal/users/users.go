package users

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"trust-management/backend/internal/database"
	"trust-management/backend/internal/dto"
	"trust-management/backend/internal/models"
	"trust-management/backend/internal/shared"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type UserHandler struct{}

func NewUserHandler() *UserHandler {
	return &UserHandler{}
}

// GetUsers returns all users (admin-only — password hashes are never serialized,
// see models.User's `json:"-"` tag on PasswordHash).
func (h *UserHandler) GetUsers(c *gin.Context) {
	var list []models.User
	if err := database.DB.Order("id asc").Find(&list).Error; err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to fetch users")
		return
	}
	shared.SendSuccess(c, http.StatusOK, list)
}

// CreateUser creates a new STAFF or ADMIN account (admin-only).
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.SendAppError(c, http.StatusBadRequest, err.Error())
		return
	}

	role := models.Role(strings.ToUpper(req.Role))
	if role != models.RoleStaff && role != models.RoleAdmin {
		shared.SendAppError(c, http.StatusBadRequest, "Invalid role. Must be STAFF or ADMIN.")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	user := models.User{
		Username:     req.Username,
		FullName:     req.FullName,
		Email:        req.Email,
		PasswordHash: string(hash),
		Role:         role,
		IsActive:     true,
	}

	adminIDVal, _ := c.Get("userID")
	adminID, _ := adminIDVal.(uint)

	tx := database.DB.Begin()
	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to create user: "+err.Error())
		return
	}

	afterData, _ := json.Marshal(user)
	audit := models.AuditLog{
		UserID:     &adminID,
		Action:     "USER_CREATED",
		EntityName: "User",
		EntityID:   user.ID,
		BeforeData: shared.JSONOrNull(nil),
		AfterData:  shared.JSONOrNull(json.RawMessage(afterData)),
		IPAddress:  c.ClientIP(),
	}
	if err := tx.Create(&audit).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to record audit log")
		return
	}
	tx.Commit()

	shared.SendSuccess(c, http.StatusCreated, user)
}

// UpdateUser edits an existing user's profile, role, active status, or resets
// their password (admin-only). This is also the mechanism for deactivating a
// user (soft delete) — accounts are never hard-deleted.
func (h *UserHandler) UpdateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		shared.SendAppError(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		shared.SendAppError(c, http.StatusNotFound, "User not found")
		return
	}

	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.SendAppError(c, http.StatusBadRequest, err.Error())
		return
	}

	beforeData, _ := json.Marshal(user)

	if req.FullName != "" {
		user.FullName = req.FullName
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.Role != "" {
		role := models.Role(strings.ToUpper(req.Role))
		if role != models.RoleStaff && role != models.RoleAdmin {
			shared.SendAppError(c, http.StatusBadRequest, "Invalid role. Must be STAFF or ADMIN.")
			return
		}
		user.Role = role
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}
	if req.Password != nil && *req.Password != "" {
		if len(*req.Password) < 8 {
			shared.SendAppError(c, http.StatusBadRequest, "Password must be at least 8 characters")
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			shared.SendAppError(c, http.StatusInternalServerError, "Failed to hash password")
			return
		}
		user.PasswordHash = string(hash)
	}

	adminIDVal, _ := c.Get("userID")
	adminID, _ := adminIDVal.(uint)

	tx := database.DB.Begin()
	if err := tx.Save(&user).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to update user")
		return
	}

	afterData, _ := json.Marshal(user)
	audit := models.AuditLog{
		UserID:     &adminID,
		Action:     "USER_UPDATED",
		EntityName: "User",
		EntityID:   user.ID,
		BeforeData: shared.JSONOrNull(json.RawMessage(beforeData)),
		AfterData:  shared.JSONOrNull(json.RawMessage(afterData)),
		IPAddress:  c.ClientIP(),
	}
	if err := tx.Create(&audit).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to record audit log")
		return
	}
	tx.Commit()

	shared.SendSuccess(c, http.StatusOK, user)
}
