package donor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"trust-management/backend/internal/database"
	"trust-management/backend/internal/dto"
	"trust-management/backend/internal/models"
	"trust-management/backend/internal/shared"

	"github.com/gin-gonic/gin"
)

type DonorHandler struct{}

func NewDonorHandler() *DonorHandler {
	return &DonorHandler{}
}

// GetDonors handles fetching donors with optional search query
func (h *DonorHandler) GetDonors(c *gin.Context) {
	search := c.Query("search")
	var donors []models.Donor

	query := database.DB.Preload("FamilyMembers").Order("id desc")
	if search != "" {
		likePattern := "%" + search + "%"
		query = query.Where("full_name LIKE ? OR phone LIKE ? OR email LIKE ? OR donor_code LIKE ? OR city LIKE ?",
			likePattern, likePattern, likePattern, likePattern, likePattern)
	}

	if err := query.Find(&donors).Error; err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to fetch donors")
		return
	}

	shared.SendSuccess(c, http.StatusOK, donors)
}

// GetDonorByID handles fetching single donor with family members
func (h *DonorHandler) GetDonorByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		shared.SendAppError(c, http.StatusBadRequest, "Invalid donor ID")
		return
	}

	var donor models.Donor
	if err := database.DB.Preload("FamilyMembers").First(&donor, id).Error; err != nil {
		shared.SendAppError(c, http.StatusNotFound, "Donor not found")
		return
	}

	shared.SendSuccess(c, http.StatusOK, donor)
}

// CreateDonor creates a new donor record with unique donor code and family members
func (h *DonorHandler) CreateDonor(c *gin.Context) {
	var req dto.CreateDonorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.SendAppError(c, http.StatusBadRequest, err.Error())
		return
	}

	var dob *time.Time
	if req.DateOfBirth != "" {
		if t, err := time.Parse("2006-01-02", req.DateOfBirth); err == nil {
			dob = &t
		}
	}

	var anniv *time.Time
	if req.AnniversaryDate != "" {
		if t, err := time.Parse("2006-01-02", req.AnniversaryDate); err == nil {
			anniv = &t
		}
	}

	userIDVal, _ := c.Get("userID")
	userID, _ := userIDVal.(uint)

	tx := database.DB.Begin()

	// Generate unique donor code (e.g. DNR-00001) via atomic sequence counter
	donorSeq, err := shared.NextSequence(tx, "DONOR")
	if err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to generate donor code")
		return
	}
	donorCode := fmt.Sprintf("DNR-%05d", donorSeq)

	donor := models.Donor{
		DonorCode:       donorCode,
		FullName:        req.FullName,
		FatherName:      req.FatherName,
		Phone:           req.Phone,
		Email:           req.Email,
		AddressLine:     req.AddressLine,
		City:            req.City,
		State:           req.State,
		Pincode:         req.Pincode,
		DateOfBirth:     dob,
		AnniversaryDate: anniv,
		MaritalStatus:   req.MaritalStatus,
		AadhaarNumber:   req.AadhaarNumber,
		AadhaarDocPath:  req.AadhaarDocPath,
		PANNumber:       req.PANNumber,
		PANDocPath:      req.PANDocPath,
		PhotoPath:       req.PhotoPath,
		Notes:           req.Notes,
		IsActive:        true,
	}

	if err := tx.Create(&donor).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to create donor")
		return
	}

	// Process family members / child registrations
	for _, fm := range req.FamilyMembers {
		var fmDob time.Time
		if t, err := time.Parse("2006-01-02", fm.DateOfBirth); err == nil {
			fmDob = t
		}
		familyMember := models.DonorFamilyMember{
			DonorID:      donor.ID,
			FullName:     fm.FullName,
			Relationship: fm.Relationship,
			DateOfBirth:  fmDob,
			Notes:        fm.Notes,
		}
		if err := tx.Create(&familyMember).Error; err != nil {
			tx.Rollback()
			shared.SendAppError(c, http.StatusInternalServerError, "Failed to save family member details")
			return
		}
	}

	audit := models.AuditLog{
		UserID:     &userID,
		Action:     "DONOR_CREATED",
		EntityName: "Donor",
		EntityID:   donor.ID,
		BeforeData: shared.JSONOrNull(nil),
		AfterData:  shared.JSONOrNull(donor),
		IPAddress:  c.ClientIP(),
	}
	if err := tx.Create(&audit).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to record audit log")
		return
	}

	tx.Commit()

	// Load complete created donor with preloads
	database.DB.Preload("FamilyMembers").First(&donor, donor.ID)
	shared.SendSuccess(c, http.StatusCreated, donor)
}

// UpdateDonor updates donor details and family members
func (h *DonorHandler) UpdateDonor(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		shared.SendAppError(c, http.StatusBadRequest, "Invalid donor ID")
		return
	}

	var donor models.Donor
	if err := database.DB.Preload("FamilyMembers").First(&donor, id).Error; err != nil {
		shared.SendAppError(c, http.StatusNotFound, "Donor not found")
		return
	}

	var req dto.UpdateDonorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.SendAppError(c, http.StatusBadRequest, err.Error())
		return
	}

	beforeData, _ := json.Marshal(donor)

	if req.FullName != "" {
		donor.FullName = req.FullName
	}
	if req.Phone != "" {
		donor.Phone = req.Phone
	}
	donor.FatherName = req.FatherName
	donor.Email = req.Email
	donor.AddressLine = req.AddressLine
	donor.City = req.City
	donor.State = req.State
	donor.Pincode = req.Pincode
	donor.MaritalStatus = req.MaritalStatus
	donor.AadhaarNumber = req.AadhaarNumber
	donor.AadhaarDocPath = req.AadhaarDocPath
	donor.PANNumber = req.PANNumber
	donor.PANDocPath = req.PANDocPath
	donor.PhotoPath = req.PhotoPath
	donor.Notes = req.Notes
	if req.IsActive != nil {
		donor.IsActive = *req.IsActive
	}

	if req.DateOfBirth != "" {
		if t, err := time.Parse("2006-01-02", req.DateOfBirth); err == nil {
			donor.DateOfBirth = &t
		}
	}
	if req.AnniversaryDate != "" {
		if t, err := time.Parse("2006-01-02", req.AnniversaryDate); err == nil {
			donor.AnniversaryDate = &t
		}
	}

	tx := database.DB.Begin()
	if err := tx.Save(&donor).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to update donor")
		return
	}

	// Update family members if provided
	if req.FamilyMembers != nil {
		tx.Where("donor_id = ?", donor.ID).Delete(&models.DonorFamilyMember{})
		for _, fm := range req.FamilyMembers {
			var fmDob time.Time
			if t, err := time.Parse("2006-01-02", fm.DateOfBirth); err == nil {
				fmDob = t
			}
			familyMember := models.DonorFamilyMember{
				DonorID:      donor.ID,
				FullName:     fm.FullName,
				Relationship: fm.Relationship,
				DateOfBirth:  fmDob,
				Notes:        fm.Notes,
			}
			tx.Create(&familyMember)
		}
	}

	userIDVal, _ := c.Get("userID")
	userID, _ := userIDVal.(uint)

	afterData, _ := json.Marshal(donor)
	audit := models.AuditLog{
		UserID:     &userID,
		Action:     "DONOR_UPDATED",
		EntityName: "Donor",
		EntityID:   donor.ID,
		BeforeData: string(beforeData),
		AfterData:  string(afterData),
		IPAddress:  c.ClientIP(),
	}
	if err := tx.Create(&audit).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to record audit log")
		return
	}

	tx.Commit()
	database.DB.Preload("FamilyMembers").First(&donor, donor.ID)
	shared.SendSuccess(c, http.StatusOK, donor)
}
