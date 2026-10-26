package users

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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

// GetUsers returns all users or filtered staff (admin-only).
// Supports query params: status (ALL, ACTIVE, RELIEVED, TEMPORARILY_BLOCKED), branch_id, search.
func (h *UserHandler) GetUsers(c *gin.Context) {
	query := database.DB.Preload("Branch").Order("id asc")

	statusParam := strings.ToUpper(strings.TrimSpace(c.Query("status")))
	if statusParam != "" && statusParam != "ALL" {
		query = query.Where("employment_status = ?", statusParam)
	}

	branchIDParam := strings.TrimSpace(c.Query("branch_id"))
	if branchIDParam != "" && branchIDParam != "ALL" {
		if bID, err := strconv.ParseUint(branchIDParam, 10, 32); err == nil && bID > 0 {
			query = query.Where("branch_id = ?", uint(bID))
		}
	}

	search := strings.TrimSpace(c.Query("search"))
	if search != "" {
		searchTerm := "%" + search + "%"
		query = query.Where("full_name LIKE ? OR username LIKE ? OR staff_code LIKE ? OR phone LIKE ? OR email LIKE ? OR designation LIKE ?",
			searchTerm, searchTerm, searchTerm, searchTerm, searchTerm, searchTerm)
	}

	list := make([]models.User, 0)
	if err := query.Find(&list).Error; err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to fetch staff members: "+err.Error())
		return
	}
	shared.SendSuccess(c, http.StatusOK, list)
}

// GetUserByID returns details of a single staff user
func (h *UserHandler) GetUserByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		shared.SendAppError(c, http.StatusBadRequest, "Invalid staff ID")
		return
	}

	var user models.User
	if err := database.DB.Preload("Branch").First(&user, id).Error; err != nil {
		shared.SendAppError(c, http.StatusNotFound, "Staff member not found")
		return
	}
	shared.SendSuccess(c, http.StatusOK, user)
}

// CreateUser creates a new staff account with demographic and employment details (admin-only).
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.SendAppError(c, http.StatusBadRequest, err.Error())
		return
	}

	role := models.Role(strings.ToUpper(strings.TrimSpace(req.Role)))
	if role != models.RoleStaff && role != models.RoleAdmin {
		shared.SendAppError(c, http.StatusBadRequest, "Invalid role. Must be STAFF or ADMIN.")
		return
	}

	username := strings.TrimSpace(req.Username)
	if username == "" {
		shared.SendAppError(c, http.StatusBadRequest, "Username is required")
		return
	}

	email := strings.TrimSpace(req.Email)
	if email == "" {
		shared.SendAppError(c, http.StatusBadRequest, "Email is required")
		return
	}

	// Check for unique username and email
	var count int64
	database.DB.Model(&models.User{}).Where("username = ?", username).Count(&count)
	if count > 0 {
		shared.SendAppError(c, http.StatusConflict, "Username is already taken")
		return
	}
	database.DB.Model(&models.User{}).Where("email = ?", email).Count(&count)
	if count > 0 {
		shared.SendAppError(c, http.StatusConflict, "Email is already registered")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	branchID := req.BranchID
	if branchID == nil || *branchID == 0 {
		defaultB := uint(1)
		branchID = &defaultB
	}

	tx := database.DB.Begin()

	// Auto-generate staff code if not provided
	staffCode := strings.TrimSpace(req.StaffCode)
	if staffCode == "" {
		seq, err := shared.NextSequenceAutoSeed(tx, "STAFF")
		if err != nil {
			tx.Rollback()
			shared.SendAppError(c, http.StatusInternalServerError, "Failed to generate staff code")
			return
		}
		staffCode = fmt.Sprintf("STF-%04d", seq)
	}

	var dob *time.Time
	if strings.TrimSpace(req.DateOfBirth) != "" {
		if t, err := time.Parse("2006-01-02", strings.TrimSpace(req.DateOfBirth)); err == nil {
			dob = &t
		}
	}

	var doj *time.Time
	if strings.TrimSpace(req.DateOfJoining) != "" {
		if t, err := time.Parse("2006-01-02", strings.TrimSpace(req.DateOfJoining)); err == nil {
			doj = &t
		}
	} else {
		now := time.Now()
		doj = &now
	}

	gender := strings.ToUpper(strings.TrimSpace(req.Gender))
	if gender == "" {
		gender = "OTHER"
	}

	maritalStatus := strings.ToUpper(strings.TrimSpace(req.MaritalStatus))
	if maritalStatus == "" {
		maritalStatus = "SINGLE"
	}

	user := models.User{
		StaffCode:             staffCode,
		Username:              username,
		FullName:              strings.TrimSpace(req.FullName),
		Email:                 email,
		Phone:                 strings.TrimSpace(req.Phone),
		Gender:                gender,
		Education:             strings.TrimSpace(req.Education),
		MaritalStatus:         maritalStatus,
		DateOfBirth:           dob,
		DateOfJoining:         doj,
		Designation:           strings.TrimSpace(req.Designation),
		Department:            strings.TrimSpace(req.Department),
		AddressLine:           strings.TrimSpace(req.AddressLine),
		City:                  strings.TrimSpace(req.City),
		State:                 strings.TrimSpace(req.State),
		Pincode:               strings.TrimSpace(req.Pincode),
		PhotoPath:             strings.TrimSpace(req.PhotoPath),
		EmergencyContactName:  strings.TrimSpace(req.EmergencyContactName),
		EmergencyContactPhone: strings.TrimSpace(req.EmergencyContactPhone),
		PasswordHash:          string(hash),
		Role:                  role,
		EmploymentStatus:      models.EmploymentStatusActive,
		BranchID:              branchID,
		IsActive:              true,
	}

	adminIDVal, _ := c.Get("userID")
	adminID, _ := adminIDVal.(uint)

	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to create staff member: "+err.Error())
		return
	}

	afterData, _ := json.Marshal(user)
	audit := models.AuditLog{
		UserID:     &adminID,
		Action:     "STAFF_CREATED",
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

	_ = database.DB.Preload("Branch").First(&user, user.ID)
	shared.SendSuccess(c, http.StatusCreated, user)
}

// UpdateUser edits an existing staff's profile, designation, demographic data,
// employment status (ACTIVE, RELIEVED, TEMPORARILY_BLOCKED), or password.
func (h *UserHandler) UpdateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		shared.SendAppError(c, http.StatusBadRequest, "Invalid staff ID")
		return
	}

	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		shared.SendAppError(c, http.StatusNotFound, "Staff member not found")
		return
	}

	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.SendAppError(c, http.StatusBadRequest, err.Error())
		return
	}

	beforeData, _ := json.Marshal(user)

	if req.FullName != "" {
		user.FullName = strings.TrimSpace(req.FullName)
	}
	if req.Email != "" {
		newEmail := strings.TrimSpace(req.Email)
		if newEmail != user.Email {
			var count int64
			database.DB.Model(&models.User{}).Where("email = ? AND id != ?", newEmail, user.ID).Count(&count)
			if count > 0 {
				shared.SendAppError(c, http.StatusConflict, "Email is already taken by another account")
				return
			}
			user.Email = newEmail
		}
	}
	if req.Phone != "" {
		user.Phone = strings.TrimSpace(req.Phone)
	}
	if req.Gender != "" {
		user.Gender = strings.ToUpper(strings.TrimSpace(req.Gender))
	}
	if req.Education != "" {
		user.Education = strings.TrimSpace(req.Education)
	}
	if req.MaritalStatus != "" {
		user.MaritalStatus = strings.ToUpper(strings.TrimSpace(req.MaritalStatus))
	}
	if req.Designation != "" {
		user.Designation = strings.TrimSpace(req.Designation)
	}
	if req.Department != "" {
		user.Department = strings.TrimSpace(req.Department)
	}
	if req.AddressLine != "" {
		user.AddressLine = strings.TrimSpace(req.AddressLine)
	}
	if req.City != "" {
		user.City = strings.TrimSpace(req.City)
	}
	if req.State != "" {
		user.State = strings.TrimSpace(req.State)
	}
	if req.Pincode != "" {
		user.Pincode = strings.TrimSpace(req.Pincode)
	}
	if req.PhotoPath != "" {
		user.PhotoPath = strings.TrimSpace(req.PhotoPath)
	}
	if req.EmergencyContactName != "" {
		user.EmergencyContactName = strings.TrimSpace(req.EmergencyContactName)
	}
	if req.EmergencyContactPhone != "" {
		user.EmergencyContactPhone = strings.TrimSpace(req.EmergencyContactPhone)
	}

	if req.DateOfBirth != nil {
		if *req.DateOfBirth == "" {
			user.DateOfBirth = nil
		} else if t, err := time.Parse("2006-01-02", *req.DateOfBirth); err == nil {
			user.DateOfBirth = &t
		}
	}
	if req.DateOfJoining != nil {
		if *req.DateOfJoining == "" {
			user.DateOfJoining = nil
		} else if t, err := time.Parse("2006-01-02", *req.DateOfJoining); err == nil {
			user.DateOfJoining = &t
		}
	}
	if req.DateOfRelieving != nil {
		if *req.DateOfRelieving == "" {
			user.DateOfRelieving = nil
		} else if t, err := time.Parse("2006-01-02", *req.DateOfRelieving); err == nil {
			user.DateOfRelieving = &t
		}
	}

	if req.Role != "" {
		role := models.Role(strings.ToUpper(strings.TrimSpace(req.Role)))
		if role != models.RoleStaff && role != models.RoleAdmin {
			shared.SendAppError(c, http.StatusBadRequest, "Invalid role. Must be STAFF or ADMIN.")
			return
		}
		user.Role = role
	}

	if req.EmploymentStatus != "" {
		st := models.EmploymentStatus(strings.ToUpper(strings.TrimSpace(req.EmploymentStatus)))
		if st == models.EmploymentStatusActive || st == models.EmploymentStatusRelieved || st == models.EmploymentStatusTemporarilyBlocked {
			user.EmploymentStatus = st
			if st == models.EmploymentStatusRelieved && user.DateOfRelieving == nil {
				now := time.Now()
				user.DateOfRelieving = &now
			}
			if st == models.EmploymentStatusActive {
				user.IsActive = true
			}
		} else {
			shared.SendAppError(c, http.StatusBadRequest, "Invalid employment status. Must be ACTIVE, RELIEVED, or TEMPORARILY_BLOCKED.")
			return
		}
	}

	if req.StatusRemarks != "" {
		user.StatusRemarks = strings.TrimSpace(req.StatusRemarks)
	}

	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}
	if req.BranchID != nil && *req.BranchID > 0 {
		user.BranchID = req.BranchID
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
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to update staff member: "+err.Error())
		return
	}

	afterData, _ := json.Marshal(user)
	audit := models.AuditLog{
		UserID:     &adminID,
		Action:     "STAFF_UPDATED",
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

	_ = database.DB.Preload("Branch").First(&user, user.ID)
	shared.SendSuccess(c, http.StatusOK, user)
}
