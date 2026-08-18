package expensecategory

import (
	"encoding/json"
	"net/http"
	"strconv"

	"trust-management/backend/internal/database"
	"trust-management/backend/internal/dto"
	"trust-management/backend/internal/models"
	"trust-management/backend/internal/shared"

	"github.com/gin-gonic/gin"
)

type ExpenseCategoryHandler struct{}

func NewExpenseCategoryHandler() *ExpenseCategoryHandler {
	return &ExpenseCategoryHandler{}
}

func (h *ExpenseCategoryHandler) GetExpenseCategories(c *gin.Context) {
	var categories []models.ExpenseCategory
	if err := database.DB.Order("name asc").Find(&categories).Error; err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to fetch expense categories")
		return
	}
	shared.SendSuccess(c, http.StatusOK, categories)
}

func (h *ExpenseCategoryHandler) GetActiveExpenseCategories(c *gin.Context) {
	var categories []models.ExpenseCategory
	if err := database.DB.Where("is_active = ?", true).Order("name asc").Find(&categories).Error; err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to fetch active expense categories")
		return
	}
	shared.SendSuccess(c, http.StatusOK, categories)
}

func (h *ExpenseCategoryHandler) CreateExpenseCategory(c *gin.Context) {
	var req dto.CreateExpenseCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.SendAppError(c, http.StatusBadRequest, err.Error())
		return
	}

	category := models.ExpenseCategory{
		Name:     req.Name,
		IsActive: true,
	}

	userIDVal, _ := c.Get("userID")
	userID, _ := userIDVal.(uint)

	tx := database.DB.Begin()
	if err := tx.Create(&category).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to create expense category — the name may already exist")
		return
	}

	audit := models.AuditLog{
		UserID:     &userID,
		Action:     "EXPENSE_CATEGORY_CREATED",
		EntityName: "ExpenseCategory",
		EntityID:   category.ID,
		BeforeData: shared.JSONOrNull(nil),
		AfterData:  shared.JSONOrNull(category),
		IPAddress:  c.ClientIP(),
	}
	if err := tx.Create(&audit).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to record audit log")
		return
	}
	tx.Commit()

	shared.SendSuccess(c, http.StatusCreated, category)
}

// UpdateExpenseCategory handles both renaming and activate/deactivate — a
// category is never hard-deleted so any expense already recorded against it
// keeps a valid, readable category name.
func (h *ExpenseCategoryHandler) UpdateExpenseCategory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		shared.SendAppError(c, http.StatusBadRequest, "Invalid expense category ID")
		return
	}

	var category models.ExpenseCategory
	if err := database.DB.First(&category, id).Error; err != nil {
		shared.SendAppError(c, http.StatusNotFound, "Expense category not found")
		return
	}

	beforeData, _ := json.Marshal(category)

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.SendAppError(c, http.StatusBadRequest, err.Error())
		return
	}

	userIDVal, _ := c.Get("userID")
	userID, _ := userIDVal.(uint)

	tx := database.DB.Begin()
	if err := tx.Model(&category).Updates(req).Error; err != nil {
		tx.Rollback()
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to update expense category — the name may already exist")
		return
	}

	afterData, _ := json.Marshal(category)
	audit := models.AuditLog{
		UserID:     &userID,
		Action:     "EXPENSE_CATEGORY_UPDATED",
		EntityName: "ExpenseCategory",
		EntityID:   category.ID,
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

	shared.SendSuccess(c, http.StatusOK, category)
}
