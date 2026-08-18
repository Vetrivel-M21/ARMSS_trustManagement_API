package unlock

import (
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

type UnlockHandler struct{}

func NewUnlockHandler() *UnlockHandler {
	return &UnlockHandler{}
}

// GetUnlockRequests returns list of unlock requests with user preloads
func (h *UnlockHandler) GetUnlockRequests(c *gin.Context) {
	var requests []models.UnlockRequest
	if err := database.DB.Preload("RequestedBy").Preload("ReviewedBy").Preload("BankAccount").Order("id desc").Find(&requests).Error; err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to fetch unlock requests")
		return
	}
	shared.SendSuccess(c, http.StatusOK, requests)
}

// SubmitUnlockRequest allows staff/admin to submit an unlock request for a
// closed cash day (CASH_DAY, the default) or a closed bank account/day
// (BANK_DAY, which also requires bank_account_id).
func (h *UnlockHandler) SubmitUnlockRequest(c *gin.Context) {
	var req dto.SubmitUnlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.SendAppError(c, http.StatusBadRequest, err.Error())
		return
	}

	entityType := req.EntityType
	if entityType == "" {
		entityType = "CASH_DAY"
	}
	if entityType != "CASH_DAY" && entityType != "BANK_DAY" {
		shared.SendAppError(c, http.StatusBadRequest, "entity_type must be CASH_DAY or BANK_DAY")
		return
	}
	if entityType == "BANK_DAY" && (req.BankAccountID == nil || *req.BankAccountID == 0) {
		shared.SendAppError(c, http.StatusBadRequest, "bank_account_id is required for a BANK_DAY unlock request")
		return
	}

	userIDVal, exists := c.Get("userID")
	if !exists {
		shared.SendAppError(c, http.StatusUnauthorized, "User context missing")
		return
	}
	userID := userIDVal.(uint)

	bizDate, err := time.Parse("2006-01-02", req.BusinessDate)
	if err != nil {
		shared.SendAppError(c, http.StatusBadRequest, "Invalid date format")
		return
	}

	unlockReq := models.UnlockRequest{
		EntityType:    entityType,
		BankAccountID: req.BankAccountID,
		BusinessDate:  bizDate,
		RequestedByID: userID,
		RequestReason: req.Reason,
		Status:        "PENDING",
	}

	if err := database.DB.Create(&unlockReq).Error; err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to submit unlock request")
		return
	}

	database.DB.Preload("RequestedBy").Preload("BankAccount").First(&unlockReq, unlockReq.ID)
	shared.SendSuccess(c, http.StatusCreated, unlockReq)
}

// ReviewUnlockRequest allows Admin to approve or reject an unlock request
func (h *UnlockHandler) ReviewUnlockRequest(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		shared.SendAppError(c, http.StatusBadRequest, "Invalid unlock request ID")
		return
	}

	userIDVal, exists := c.Get("userID")
	if !exists {
		shared.SendAppError(c, http.StatusUnauthorized, "User context missing")
		return
	}
	adminID := userIDVal.(uint)

	var req dto.ReviewUnlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.SendAppError(c, http.StatusBadRequest, err.Error())
		return
	}

	var unlockReq models.UnlockRequest
	if err := database.DB.First(&unlockReq, id).Error; err != nil {
		shared.SendAppError(c, http.StatusNotFound, "Unlock request not found")
		return
	}

	if unlockReq.Status != "PENDING" {
		shared.SendAppError(c, http.StatusBadRequest, "Unlock request has already been reviewed")
		return
	}

	now := time.Now()
	unlockReq.Status = req.Status // APPROVED or REJECTED
	unlockReq.ReviewedByID = &adminID
	unlockReq.ReviewReason = req.ReviewNotes
	unlockReq.ReviewedAt = &now

	tx := database.DB.Begin()
	if err := tx.Save(&unlockReq).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to update unlock request status")
		return
	}

	// If Approved, unlock the underlying closed record: DailyClosing for a
	// CASH_DAY request, or the matching BankClosing row for a BANK_DAY request.
	dateStr := unlockReq.BusinessDate.Format("2006-01-02")
	if req.Status == "APPROVED" {
		if unlockReq.EntityType == "BANK_DAY" && unlockReq.BankAccountID != nil {
			var bankClosing models.BankClosing
			if err := tx.Where("bank_account_id = ? AND business_date = ?", *unlockReq.BankAccountID, dateStr).First(&bankClosing).Error; err == nil {
				bankClosing.Status = models.DayStatusUnlocked
				tx.Save(&bankClosing)
			}
		} else {
			var dailyClosing models.DailyClosing
			if err := tx.Where("business_date = ?", dateStr).First(&dailyClosing).Error; err == nil {
				dailyClosing.Status = models.DayStatusUnlocked
				tx.Save(&dailyClosing)
			}
		}
	}

	// Record Immutable Audit Log
	audit := models.AuditLog{
		UserID:     &adminID,
		Action:     fmt.Sprintf("UNLOCK_REQUEST_%s", req.Status),
		EntityName: "UnlockRequest",
		EntityID:   unlockReq.ID,
		BeforeData: shared.JSONOrNull(nil),
		AfterData:  shared.JSONOrNull(unlockReq),
		Reason:     fmt.Sprintf("Unlock request for %s %s. Notes: %s", dateStr, req.Status, req.ReviewNotes),
		IPAddress:  c.ClientIP(),
	}
	if err := tx.Create(&audit).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to record unlock audit log")
		return
	}

	tx.Commit()

	database.DB.Preload("RequestedBy").Preload("ReviewedBy").First(&unlockReq, unlockReq.ID)
	shared.SendSuccess(c, http.StatusOK, unlockReq)
}

// GetAuditLogs fetches immutable audit trail
func (h *UnlockHandler) GetAuditLogs(c *gin.Context) {
	var logs []models.AuditLog
	if err := database.DB.Preload("User").Order("id desc").Limit(100).Find(&logs).Error; err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to fetch audit logs")
		return
	}
	shared.SendSuccess(c, http.StatusOK, logs)
}
