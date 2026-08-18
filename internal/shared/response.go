package shared

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

type Meta struct {
	Page       int   `json:"page,omitempty"`
	Limit      int   `json:"limit,omitempty"`
	Total      int64 `json:"total,omitempty"`
	TotalPages int   `json:"total_pages,omitempty"`
}

type ErrorResponse struct {
	Success bool        `json:"success"`
	Error   ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

func SendSuccess(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, APIResponse{
		Success: true,
		Data:    data,
	})
}

func SendPaginated(c *gin.Context, statusCode int, data interface{}, page, limit int, total int64) {
	totalPages := 1
	if limit > 0 {
		totalPages = int((total + int64(limit) - 1) / int64(limit))
	}
	c.JSON(statusCode, APIResponse{
		Success: true,
		Data:    data,
		Meta: &Meta{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

func SendError(c *gin.Context, statusCode int, code string, message string, details ...interface{}) {
	var detail interface{}
	if len(details) > 0 {
		detail = details[0]
	}
	c.JSON(statusCode, ErrorResponse{
		Success: false,
		Error: ErrorDetail{
			Code:    code,
			Message: message,
			Details: detail,
		},
	})
}

func SendAppError(c *gin.Context, statusCode int, message string) {
	code := "ERROR"
	switch statusCode {
	case http.StatusBadRequest:
		code = "BAD_REQUEST"
	case http.StatusUnauthorized:
		code = "UNAUTHORIZED"
	case http.StatusForbidden:
		code = "FORBIDDEN"
	case http.StatusNotFound:
		code = "NOT_FOUND"
	case http.StatusInternalServerError:
		code = "INTERNAL_ERROR"
	}
	SendError(c, statusCode, code, message)
}

func SendBadRequest(c *gin.Context, code string, message string, details ...interface{}) {
	SendError(c, http.StatusBadRequest, code, message, details...)
}

func SendUnauthorized(c *gin.Context, message string) {
	SendError(c, http.StatusUnauthorized, "UNAUTHORIZED", message)
}

func SendForbidden(c *gin.Context, message string) {
	SendError(c, http.StatusForbidden, "FORBIDDEN", message)
}

func SendNotFound(c *gin.Context, message string) {
	SendError(c, http.StatusNotFound, "NOT_FOUND", message)
}

func SendConflict(c *gin.Context, code string, message string) {
	SendError(c, http.StatusConflict, code, message)
}

func SendInternalError(c *gin.Context, message string) {
	SendError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", message)
}
