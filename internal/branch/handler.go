package branch

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"trust-management/backend/internal/database"
	"trust-management/backend/internal/dto"
	"trust-management/backend/internal/models"
	"trust-management/backend/internal/shared"

	"github.com/gin-gonic/gin"
)

type BranchHandler struct{}

func NewBranchHandler() *BranchHandler {
	return &BranchHandler{}
}

// GetBranches returns list of branches
func (h *BranchHandler) GetBranches(c *gin.Context) {
	var list []models.Branch
	query := database.DB.Order("id asc")

	if c.Query("active_only") == "true" {
		query = query.Where("is_active = ?", true)
	}

	if err := query.Find(&list).Error; err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to fetch branches")
		return
	}

	shared.SendSuccess(c, http.StatusOK, list)
}

// GetBranchByID returns single branch details
func (h *BranchHandler) GetBranchByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		shared.SendAppError(c, http.StatusBadRequest, "Invalid branch ID")
		return
	}

	var b models.Branch
	if err := database.DB.First(&b, id).Error; err != nil {
		shared.SendAppError(c, http.StatusNotFound, "Branch not found")
		return
	}

	shared.SendSuccess(c, http.StatusOK, b)
}

// CreateBranch creates a new branch (Admin only)
func (h *BranchHandler) CreateBranch(c *gin.Context) {
	var req dto.CreateBranchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.SendAppError(c, http.StatusBadRequest, err.Error())
		return
	}

	code := strings.ToUpper(strings.TrimSpace(req.BranchCode))
	if len(code) < 2 {
		shared.SendAppError(c, http.StatusBadRequest, "Branch code must be at least 2 characters")
		return
	}

	var count int64
	database.DB.Model(&models.Branch{}).Where("branch_code = ?", code).Count(&count)
	if count > 0 {
		shared.SendAppError(c, http.StatusBadRequest, "Branch code already exists")
		return
	}

	branch := models.Branch{
		BranchCode:          code,
		Name:                req.Name,
		TamilName:           req.TamilName,
		LicenseNumber:       req.LicenseNumber,
		LicenseIssueDate:    req.LicenseIssueDate,
		LicenseExpiryDate:   req.LicenseExpiryDate,
		RegistrationDetails: req.RegistrationDetails,
		InchargeName:        req.InchargeName,
		Phone:               req.Phone,
		Email:               req.Email,
		AddressLine:         req.AddressLine,
		City:                req.City,
		State:               req.State,
		Pincode:             req.Pincode,
		UPIID:               strings.TrimSpace(req.UPIID),
		QRCodePath:          strings.TrimSpace(req.QRCodePath),
		LogoPath:            strings.TrimSpace(req.LogoPath),
		IsActive:            true,
	}
	if branch.State == "" {
		branch.State = "Tamil Nadu"
	}

	adminIDVal, _ := c.Get("userID")
	adminID, _ := adminIDVal.(uint)

	tx := database.DB.Begin()
	if err := tx.Create(&branch).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to create branch: "+err.Error())
		return
	}

	afterData, _ := json.Marshal(branch)
	audit := models.AuditLog{
		UserID:     &adminID,
		Action:     "BRANCH_CREATED",
		EntityName: "Branch",
		EntityID:   branch.ID,
		BeforeData: shared.JSONOrNull(nil),
		AfterData:  shared.JSONOrNull(json.RawMessage(afterData)),
		IPAddress:  c.ClientIP(),
	}
	_ = tx.Create(&audit).Error
	tx.Commit()

	shared.SendSuccess(c, http.StatusCreated, branch)
}

// UpdateBranch updates an existing branch (Admin only)
func (h *BranchHandler) UpdateBranch(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		shared.SendAppError(c, http.StatusBadRequest, "Invalid branch ID")
		return
	}

	var branch models.Branch
	if err := database.DB.First(&branch, id).Error; err != nil {
		shared.SendAppError(c, http.StatusNotFound, "Branch not found")
		return
	}

	var req dto.UpdateBranchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.SendAppError(c, http.StatusBadRequest, err.Error())
		return
	}

	beforeData, _ := json.Marshal(branch)

	if req.Name != "" {
		branch.Name = req.Name
	}
	branch.TamilName = req.TamilName
	branch.LicenseNumber = req.LicenseNumber
	branch.LicenseIssueDate = req.LicenseIssueDate
	branch.LicenseExpiryDate = req.LicenseExpiryDate
	branch.RegistrationDetails = req.RegistrationDetails
	branch.InchargeName = req.InchargeName
	branch.Phone = req.Phone
	branch.Email = req.Email
	branch.AddressLine = req.AddressLine
	if req.City != "" {
		branch.City = req.City
	}
	if req.State != "" {
		branch.State = req.State
	}
	branch.Pincode = req.Pincode
	branch.UPIID = strings.TrimSpace(req.UPIID)
	branch.QRCodePath = strings.TrimSpace(req.QRCodePath)
	branch.LogoPath = strings.TrimSpace(req.LogoPath)
	if req.IsActive != nil {
		branch.IsActive = *req.IsActive
	}

	adminIDVal, _ := c.Get("userID")
	adminID, _ := adminIDVal.(uint)

	tx := database.DB.Begin()
	if err := tx.Save(&branch).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to update branch")
		return
	}

	afterData, _ := json.Marshal(branch)
	audit := models.AuditLog{
		UserID:     &adminID,
		Action:     "BRANCH_UPDATED",
		EntityName: "Branch",
		EntityID:   branch.ID,
		BeforeData: shared.JSONOrNull(json.RawMessage(beforeData)),
		AfterData:  shared.JSONOrNull(json.RawMessage(afterData)),
		IPAddress:  c.ClientIP(),
	}
	_ = tx.Create(&audit).Error
	tx.Commit()

	shared.SendSuccess(c, http.StatusOK, branch)
}

// DeleteBranch deletes or deactivates a branch (Admin only)
func (h *BranchHandler) DeleteBranch(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		shared.SendAppError(c, http.StatusBadRequest, "Invalid branch ID")
		return
	}

	if id == 1 {
		shared.SendAppError(c, http.StatusBadRequest, "The Main Branch (Head Office) cannot be deleted")
		return
	}

	var branch models.Branch
	if err := database.DB.First(&branch, id).Error; err != nil {
		shared.SendAppError(c, http.StatusNotFound, "Branch not found")
		return
	}

	// Check if any vouchers, expenses, or donations reference this branch
	var vCount, eCount, dCount int64
	database.DB.Model(&models.Voucher{}).Where("branch_id = ?", id).Count(&vCount)
	database.DB.Model(&models.Expense{}).Where("branch_id = ?", id).Count(&eCount)
	database.DB.Model(&models.Donation{}).Where("branch_id = ?", id).Count(&dCount)

	if vCount > 0 || eCount > 0 || dCount > 0 {
		// Soft deactivate to protect financial history
		branch.IsActive = false
		if err := database.DB.Save(&branch).Error; err != nil {
			shared.SendAppError(c, http.StatusInternalServerError, "Failed to deactivate branch: "+err.Error())
			return
		}
		shared.SendSuccess(c, http.StatusOK, gin.H{
			"message":     "Branch has financial records (vouchers/expenses/donations) and was marked as inactive.",
			"deactivated": true,
		})
		return
	}

	if err := database.DB.Delete(&branch).Error; err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to delete branch: "+err.Error())
		return
	}

	shared.SendSuccess(c, http.StatusOK, gin.H{
		"message": "Branch deleted successfully.",
		"deleted": true,
	})
}

// LicenseAlert holds license alert info
type LicenseAlert struct {
	BranchID          uint       `json:"branch_id"`
	BranchName        string     `json:"branch_name"`
	BranchCode        string     `json:"branch_code"`
	LicenseNumber     string     `json:"license_number"`
	LicenseExpiryDate *time.Time `json:"license_expiry_date"`
	DaysRemaining     int        `json:"days_remaining"`
	Status            string     `json:"status"` // EXPIRED, EXPIRING_SOON, ACTIVE
}

// GetLicenseAlerts returns branches needing license attention
func (h *BranchHandler) GetLicenseAlerts(c *gin.Context) {
	var branches []models.Branch
	if err := database.DB.Where("is_active = ? AND license_expiry_date IS NOT NULL", true).Find(&branches).Error; err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to check license alerts")
		return
	}

	now := time.Now()
	var alerts []LicenseAlert
	for _, b := range branches {
		if b.LicenseExpiryDate == nil {
			continue
		}
		days := int(b.LicenseExpiryDate.Sub(now).Hours() / 24)
		status := "ACTIVE"
		if days < 0 {
			status = "EXPIRED"
		} else if days <= 30 {
			status = "EXPIRING_SOON"
		}

		if status != "ACTIVE" {
			alerts = append(alerts, LicenseAlert{
				BranchID:          b.ID,
				BranchName:        b.Name,
				BranchCode:        b.BranchCode,
				LicenseNumber:     b.LicenseNumber,
				LicenseExpiryDate: b.LicenseExpiryDate,
				DaysRemaining:     days,
				Status:            status,
			})
		}
	}

	shared.SendSuccess(c, http.StatusOK, alerts)
}
