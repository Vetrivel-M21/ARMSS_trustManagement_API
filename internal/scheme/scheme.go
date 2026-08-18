package scheme

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"trust-management/backend/internal/database"
	"trust-management/backend/internal/dto"
	"trust-management/backend/internal/models"
	"trust-management/backend/internal/shared"

	"github.com/gin-gonic/gin"
)

type SchemeHandler struct{}

func NewSchemeHandler() *SchemeHandler {
	return &SchemeHandler{}
}

func (h *SchemeHandler) GetSchemes(c *gin.Context) {
	var schemes []models.Scheme
	if err := database.DB.Order("id desc").Find(&schemes).Error; err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to fetch schemes")
		return
	}
	shared.SendSuccess(c, http.StatusOK, schemes)
}

func (h *SchemeHandler) GetActiveSchemes(c *gin.Context) {
	var schemes []models.Scheme
	if err := database.DB.Where("is_active = ?", true).Order("name asc").Find(&schemes).Error; err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to fetch active schemes")
		return
	}
	shared.SendSuccess(c, http.StatusOK, schemes)
}

func (h *SchemeHandler) CreateScheme(c *gin.Context) {
	var req dto.CreateSchemeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.SendAppError(c, http.StatusBadRequest, err.Error())
		return
	}

	foodType := req.FoodType
	if foodType == "" {
		foodType = "NA"
	}
	mealType := req.MealType
	if mealType == "" {
		mealType = "NA"
	}

	scheme := models.Scheme{
		Name:          req.Name,
		Category:      req.Category,
		FoodType:      foodType,
		MealType:      mealType,
		DefaultAmount: req.DefaultAmount,
		Description:   req.Description,
		IsActive:      true,
	}

	userIDVal, _ := c.Get("userID")
	userID, _ := userIDVal.(uint)

	tx := database.DB.Begin()
	if err := tx.Create(&scheme).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to create scheme")
		return
	}

	audit := models.AuditLog{
		UserID:     &userID,
		Action:     "SCHEME_CREATED",
		EntityName: "Scheme",
		EntityID:   scheme.ID,
		BeforeData: shared.JSONOrNull(nil),
		AfterData:  shared.JSONOrNull(scheme),
		IPAddress:  c.ClientIP(),
	}
	if err := tx.Create(&audit).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to record audit log")
		return
	}
	tx.Commit()

	shared.SendSuccess(c, http.StatusCreated, scheme)
}

// schemeCellName derives a human-readable scheme name from its food/meal
// combination (e.g. "Veg Breakfast") since the matrix dialog no longer asks
// for a name — each cell gets a name distinct enough for the donation-form
// disambiguation dropdown and the bank breakdown's purpose label.
func schemeCellName(foodType, mealType string) string {
	foodLabel := "Veg"
	if foodType == "NON_VEG" {
		foodLabel = "Non-Veg"
	}
	mealLabel := mealType
	switch mealType {
	case "BREAKFAST":
		mealLabel = "Breakfast"
	case "LUNCH":
		mealLabel = "Lunch"
	case "DINNER":
		mealLabel = "Dinner"
	}
	return foodLabel + " " + mealLabel
}

// CreateSchemesBulk creates one Scheme row per priced cell in the Veg/Non-Veg
// x Breakfast/Lunch/Dinner matrix dialog, all sharing the same description,
// in a single transaction — either all cells are created or none are, so a
// malformed cell never leaves a half-created scheme family. Name and
// Category are derived server-side (not collected from the dialog) since
// this endpoint only ever creates FOOD/Annadhanam meal schemes.
func (h *SchemeHandler) CreateSchemesBulk(c *gin.Context) {
	var req dto.CreateSchemeBulkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.SendAppError(c, http.StatusBadRequest, err.Error())
		return
	}

	priced := make([]dto.CreateSchemeBulkCell, 0, len(req.Cells))
	for _, cell := range req.Cells {
		if cell.DefaultAmount.IsPositive() {
			priced = append(priced, cell)
		}
	}
	if len(priced) == 0 {
		shared.SendAppError(c, http.StatusBadRequest, "At least one price cell must be filled in.")
		return
	}

	userIDVal, _ := c.Get("userID")
	userID, _ := userIDVal.(uint)

	tx := database.DB.Begin()
	created := make([]models.Scheme, 0, len(priced))
	for _, cell := range priced {
		scheme := models.Scheme{
			Name:          schemeCellName(cell.FoodType, cell.MealType),
			Category:      "FOOD",
			FoodType:      cell.FoodType,
			MealType:      cell.MealType,
			DefaultAmount: cell.DefaultAmount,
			Description:   req.Description,
			IsActive:      true,
		}
		if err := tx.Create(&scheme).Error; err != nil {
			tx.Rollback()
			shared.SendAppError(c, http.StatusInternalServerError, "Failed to create scheme: "+err.Error())
			return
		}
		created = append(created, scheme)
	}

	audit := models.AuditLog{
		UserID:     &userID,
		Action:     "SCHEME_BULK_CREATED",
		EntityName: "Scheme",
		EntityID:   created[0].ID,
		BeforeData: shared.JSONOrNull(nil),
		AfterData:  shared.JSONOrNull(created),
		Reason:     fmt.Sprintf("Created %d scheme(s) via matrix dialog", len(created)),
		IPAddress:  c.ClientIP(),
	}
	if err := tx.Create(&audit).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to record audit log")
		return
	}
	tx.Commit()

	shared.SendSuccess(c, http.StatusCreated, created)
}

func (h *SchemeHandler) UpdateScheme(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		shared.SendAppError(c, http.StatusBadRequest, "Invalid scheme ID")
		return
	}

	var scheme models.Scheme
	if err := database.DB.First(&scheme, id).Error; err != nil {
		shared.SendAppError(c, http.StatusNotFound, "Scheme not found")
		return
	}

	beforeData, _ := json.Marshal(scheme)

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.SendAppError(c, http.StatusBadRequest, err.Error())
		return
	}

	userIDVal, _ := c.Get("userID")
	userID, _ := userIDVal.(uint)

	tx := database.DB.Begin()
	if err := tx.Model(&scheme).Updates(req).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to update scheme")
		return
	}

	afterData, _ := json.Marshal(scheme)
	audit := models.AuditLog{
		UserID:     &userID,
		Action:     "SCHEME_UPDATED",
		EntityName: "Scheme",
		EntityID:   scheme.ID,
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

	shared.SendSuccess(c, http.StatusOK, scheme)
}
