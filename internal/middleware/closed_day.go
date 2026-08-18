package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"time"

	"trust-management/backend/internal/database"
	"trust-management/backend/internal/models"
	"trust-management/backend/internal/shared"

	"github.com/gin-gonic/gin"
)

// ClosedDayProtectionMiddleware must be attached only to the specific financial
// mutation routes that are actually scoped to a business date (donations, cash
// denominations, expenses, closing execution, bank transfers, bank closing) —
// never to the whole router. Master data (donors, schemes, bank accounts, users)
// has no business-date concept and must never be blocked by a day's closed status.
func ClosedDayProtectionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		businessDate := extractBusinessDate(c)

		var closing models.DailyClosing
		err := database.DB.Where("business_date = ?", businessDate.Format("2006-01-02")).First(&closing).Error

		if err == nil && closing.Status == models.DayStatusClosed {
			userRoleVal, _ := c.Get("role")
			userRole, _ := userRoleVal.(models.Role)

			if userRole != models.RoleAdmin {
				shared.SendConflict(
					c,
					"BUSINESS_DAY_CLOSED",
					"Business day "+businessDate.Format("2006-01-02")+" is CLOSED. Financial entries cannot be modified without administrative unlock.",
				)
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// extractBusinessDate resolves the effective business date for a mutating request.
// Handlers accept business_date either as a query param or as a field in the JSON
// body; the body must be peeked and restored so downstream handlers can still read it.
func extractBusinessDate(c *gin.Context) time.Time {
	dateStr := c.Query("business_date")
	if dateStr == "" {
		dateStr = c.Query("date")
	}

	if dateStr == "" && c.Request.Body != nil {
		bodyBytes, err := c.GetRawData()
		if err == nil {
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			var payload struct {
				BusinessDate string `json:"business_date"`
				Date         string `json:"date"`
			}
			if json.Unmarshal(bodyBytes, &payload) == nil {
				if payload.BusinessDate != "" {
					dateStr = payload.BusinessDate
				} else if payload.Date != "" {
					dateStr = payload.Date
				}
			}
		}
	}

	if dateStr != "" {
		if t, err := shared.ParseBusinessDate(dateStr); err == nil {
			return t
		}
	}

	return shared.GetCurrentBusinessDate()
}
