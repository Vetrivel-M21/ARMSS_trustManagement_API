package ledger

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"trust-management/backend/internal/database"
	"trust-management/backend/internal/models"
	"trust-management/backend/internal/shared"

	"github.com/gin-gonic/gin"
)

type LedgerHandler struct{}

func NewLedgerHandler() *LedgerHandler {
	return &LedgerHandler{}
}

// ─── LEDGER MASTER ─────────────────────────────────────────────────────────────

type CreateLedgerRequest struct {
	LedgerNo    string `json:"ledger_no" binding:"required"`
	LedgerName  string `json:"ledger_name" binding:"required"`
	Description string `json:"description"`
}

type UpdateLedgerRequest struct {
	LedgerNo    string `json:"ledger_no" binding:"required"`
	LedgerName  string `json:"ledger_name" binding:"required"`
	Description string `json:"description"`
	IsActive    *bool  `json:"is_active"`
}

func (h *LedgerHandler) GetLedgers(c *gin.Context) {
	var ledgers []models.Ledger
	query := database.DB.Preload("CreatedBy").Order("ledger_name asc")

	if c.Query("active_only") == "true" {
		query = query.Where("is_active = ?", true)
	}

	if err := query.Find(&ledgers).Error; err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to fetch ledgers")
		return
	}

	// Compute titles count for each ledger
	type TitleCount struct {
		LedgerID uint
		Count    int64
	}
	var counts []TitleCount
	database.DB.Model(&models.VoucherTitle{}).Select("ledger_id, count(*) as count").Group("ledger_id").Scan(&counts)
	countMap := make(map[uint]int64)
	for _, cnt := range counts {
		countMap[cnt.LedgerID] = cnt.Count
	}

	for i := range ledgers {
		ledgers[i].TitlesCount = countMap[ledgers[i].ID]
	}

	shared.SendSuccess(c, http.StatusOK, ledgers)
}

func (h *LedgerHandler) CreateLedger(c *gin.Context) {
	var req CreateLedgerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.SendAppError(c, http.StatusBadRequest, "Ledger number and name are required")
		return
	}

	ledgerNo := strings.TrimSpace(req.LedgerNo)
	ledgerName := strings.TrimSpace(req.LedgerName)

	if ledgerNo == "" || ledgerName == "" {
		shared.SendAppError(c, http.StatusBadRequest, "Ledger number and name cannot be blank")
		return
	}

	// Check unique ledger_no
	var existing models.Ledger
	if err := database.DB.Where("ledger_no = ?", ledgerNo).First(&existing).Error; err == nil {
		shared.SendAppError(c, http.StatusConflict, fmt.Sprintf("Ledger number '%s' already exists", ledgerNo))
		return
	}

	userID := c.MustGet("user_id").(uint)

	ledger := models.Ledger{
		LedgerNo:    ledgerNo,
		LedgerName:  ledgerName,
		Description: strings.TrimSpace(req.Description),
		IsActive:    true,
		CreatedByID: userID,
	}

	if err := database.DB.Create(&ledger).Error; err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to create ledger: "+err.Error())
		return
	}

	shared.SendSuccess(c, http.StatusCreated, ledger)
}

func (h *LedgerHandler) UpdateLedger(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		shared.SendAppError(c, http.StatusBadRequest, "Invalid ledger ID")
		return
	}

	var req UpdateLedgerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.SendAppError(c, http.StatusBadRequest, "Invalid request payload")
		return
	}

	var ledger models.Ledger
	if err := database.DB.First(&ledger, id).Error; err != nil {
		shared.SendAppError(c, http.StatusNotFound, "Ledger not found")
		return
	}

	ledgerNo := strings.TrimSpace(req.LedgerNo)
	ledgerName := strings.TrimSpace(req.LedgerName)

	if ledgerNo == "" || ledgerName == "" {
		shared.SendAppError(c, http.StatusBadRequest, "Ledger number and name cannot be blank")
		return
	}

	// Verify ledger_no uniqueness if changed
	if ledgerNo != ledger.LedgerNo {
		var duplicate models.Ledger
		if err := database.DB.Where("ledger_no = ? AND id <> ?", ledgerNo, id).First(&duplicate).Error; err == nil {
			shared.SendAppError(c, http.StatusConflict, fmt.Sprintf("Ledger number '%s' is already in use", ledgerNo))
			return
		}
	}

	ledger.LedgerNo = ledgerNo
	ledger.LedgerName = ledgerName
	ledger.Description = strings.TrimSpace(req.Description)
	if req.IsActive != nil {
		ledger.IsActive = *req.IsActive
	}

	if err := database.DB.Save(&ledger).Error; err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to update ledger")
		return
	}

	shared.SendSuccess(c, http.StatusOK, ledger)
}

func (h *LedgerHandler) DeleteLedger(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		shared.SendAppError(c, http.StatusBadRequest, "Invalid ledger ID")
		return
	}

	var ledger models.Ledger
	if err := database.DB.First(&ledger, id).Error; err != nil {
		shared.SendAppError(c, http.StatusNotFound, "Ledger not found")
		return
	}

	// Safety check 1: check if titles belong to this ledger
	var titlesCount int64
	database.DB.Model(&models.VoucherTitle{}).Where("ledger_id = ?", id).Count(&titlesCount)
	if titlesCount > 0 {
		shared.SendAppError(c, http.StatusConflict, fmt.Sprintf("Cannot delete ledger: %d voucher title(s) are assigned to it. Please reassign or delete the titles first.", titlesCount))
		return
	}

	// Safety check 2: check if any vouchers reference this ledger
	var vouchersCount int64
	database.DB.Model(&models.Voucher{}).Where("ledger_id = ?", id).Count(&vouchersCount)
	if vouchersCount > 0 {
		shared.SendAppError(c, http.StatusConflict, fmt.Sprintf("Cannot delete ledger: %d voucher(s) are associated with this ledger.", vouchersCount))
		return
	}

	if err := database.DB.Delete(&ledger).Error; err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to delete ledger")
		return
	}

	shared.SendSuccess(c, http.StatusOK, gin.H{"message": "Ledger deleted successfully", "id": id})
}

// ─── VOUCHER TITLE MASTER ──────────────────────────────────────────────────────

type CreateTitleRequest struct {
	TitleNo     string `json:"title_no" binding:"required"`
	Title       string `json:"title" binding:"required"`
	VoucherType string `json:"voucher_type" binding:"required"` // INCOME, EXPENSE, ASSET, LIABILITY, SELF_TRANSFER
	LedgerID    uint   `json:"ledger_id" binding:"required"`
	Description string `json:"description"`
}

type UpdateTitleRequest struct {
	TitleNo     string `json:"title_no" binding:"required"`
	Title       string `json:"title" binding:"required"`
	VoucherType string `json:"voucher_type" binding:"required"`
	LedgerID    uint   `json:"ledger_id" binding:"required"`
	Description string `json:"description"`
	IsActive    *bool  `json:"is_active"`
}

func (h *LedgerHandler) GetTitles(c *gin.Context) {
	var titles []models.VoucherTitle
	query := database.DB.Preload("Ledger").Preload("CreatedBy").Order("title asc")

	if voucherType := c.Query("voucher_type"); voucherType != "" && voucherType != "ALL" {
		query = query.Where("voucher_type = ?", strings.ToUpper(voucherType))
	}

	if ledgerIDStr := c.Query("ledger_id"); ledgerIDStr != "" {
		if lid, err := strconv.ParseUint(ledgerIDStr, 10, 32); err == nil {
			query = query.Where("ledger_id = ?", lid)
		}
	}

	if c.Query("active_only") == "true" {
		query = query.Where("is_active = ?", true)
	}

	if err := query.Find(&titles).Error; err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to fetch titles")
		return
	}

	shared.SendSuccess(c, http.StatusOK, titles)
}

func (h *LedgerHandler) CreateTitle(c *gin.Context) {
	var req CreateTitleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.SendAppError(c, http.StatusBadRequest, "Title number, title name, voucher type and ledger ID are required")
		return
	}

	titleNo := strings.TrimSpace(req.TitleNo)
	titleName := strings.TrimSpace(req.Title)
	vType := strings.ToUpper(strings.TrimSpace(req.VoucherType))

	if titleNo == "" || titleName == "" {
		shared.SendAppError(c, http.StatusBadRequest, "Title number and title name cannot be blank")
		return
	}

	validTypes := map[string]bool{
		"INCOME": true, "EXPENSE": true, "ASSET": true, "LIABILITY": true, "SELF_TRANSFER": true,
	}
	if !validTypes[vType] {
		shared.SendAppError(c, http.StatusBadRequest, "Invalid voucher type. Must be INCOME, EXPENSE, ASSET, LIABILITY, or SELF_TRANSFER")
		return
	}

	// Verify parent ledger exists
	var ledger models.Ledger
	if err := database.DB.First(&ledger, req.LedgerID).Error; err != nil {
		shared.SendAppError(c, http.StatusBadRequest, "Selected parent ledger does not exist")
		return
	}

	// Verify unique title_no
	var existing models.VoucherTitle
	if err := database.DB.Where("title_no = ?", titleNo).First(&existing).Error; err == nil {
		shared.SendAppError(c, http.StatusConflict, fmt.Sprintf("Title number '%s' already exists", titleNo))
		return
	}

	userID := c.MustGet("user_id").(uint)

	voucherTitle := models.VoucherTitle{
		TitleNo:     titleNo,
		Title:       titleName,
		VoucherType: vType,
		LedgerID:    req.LedgerID,
		Description: strings.TrimSpace(req.Description),
		IsActive:    true,
		CreatedByID: userID,
	}

	if err := database.DB.Create(&voucherTitle).Error; err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to create title: "+err.Error())
		return
	}

	voucherTitle.Ledger = &ledger
	shared.SendSuccess(c, http.StatusCreated, voucherTitle)
}

func (h *LedgerHandler) UpdateTitle(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		shared.SendAppError(c, http.StatusBadRequest, "Invalid title ID")
		return
	}

	var req UpdateTitleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.SendAppError(c, http.StatusBadRequest, "Invalid request payload")
		return
	}

	var voucherTitle models.VoucherTitle
	if err := database.DB.First(&voucherTitle, id).Error; err != nil {
		shared.SendAppError(c, http.StatusNotFound, "Title not found")
		return
	}

	titleNo := strings.TrimSpace(req.TitleNo)
	titleName := strings.TrimSpace(req.Title)
	vType := strings.ToUpper(strings.TrimSpace(req.VoucherType))

	if titleNo == "" || titleName == "" {
		shared.SendAppError(c, http.StatusBadRequest, "Title number and title name cannot be blank")
		return
	}

	validTypes := map[string]bool{
		"INCOME": true, "EXPENSE": true, "ASSET": true, "LIABILITY": true, "SELF_TRANSFER": true,
	}
	if !validTypes[vType] {
		shared.SendAppError(c, http.StatusBadRequest, "Invalid voucher type")
		return
	}

	// Verify ledger
	var ledger models.Ledger
	if err := database.DB.First(&ledger, req.LedgerID).Error; err != nil {
		shared.SendAppError(c, http.StatusBadRequest, "Selected parent ledger does not exist")
		return
	}

	// Verify title_no uniqueness if changed
	if titleNo != voucherTitle.TitleNo {
		var duplicate models.VoucherTitle
		if err := database.DB.Where("title_no = ? AND id <> ?", titleNo, id).First(&duplicate).Error; err == nil {
			shared.SendAppError(c, http.StatusConflict, fmt.Sprintf("Title number '%s' is already in use", titleNo))
			return
		}
	}

	voucherTitle.TitleNo = titleNo
	voucherTitle.Title = titleName
	voucherTitle.VoucherType = vType
	voucherTitle.LedgerID = req.LedgerID
	voucherTitle.Description = strings.TrimSpace(req.Description)
	if req.IsActive != nil {
		voucherTitle.IsActive = *req.IsActive
	}

	if err := database.DB.Save(&voucherTitle).Error; err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to update title")
		return
	}

	voucherTitle.Ledger = &ledger
	shared.SendSuccess(c, http.StatusOK, voucherTitle)
}

func (h *LedgerHandler) DeleteTitle(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		shared.SendAppError(c, http.StatusBadRequest, "Invalid title ID")
		return
	}

	var voucherTitle models.VoucherTitle
	if err := database.DB.First(&voucherTitle, id).Error; err != nil {
		shared.SendAppError(c, http.StatusNotFound, "Title not found")
		return
	}

	// Safety check: check if any vouchers reference this title
	var vouchersCount int64
	database.DB.Model(&models.Voucher{}).Where("title_id = ?", id).Count(&vouchersCount)
	if vouchersCount > 0 {
		shared.SendAppError(c, http.StatusConflict, fmt.Sprintf("Cannot delete title: %d voucher(s) are recorded under this title. You may deactivate it instead.", vouchersCount))
		return
	}

	if err := database.DB.Delete(&voucherTitle).Error; err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to delete title")
		return
	}

	shared.SendSuccess(c, http.StatusOK, gin.H{"message": "Title deleted successfully", "id": id})
}
