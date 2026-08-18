package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"trust-management/backend/internal/models"

	"github.com/gin-gonic/gin"
)

func runRBAC(t *testing.T, sessionRole any, requiredRoles ...models.Role) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/", nil)
	if sessionRole != nil {
		c.Set("role", sessionRole)
	}

	handlerCalled := false
	RequireRole(requiredRoles...)(c)
	if !c.IsAborted() {
		handlerCalled = true
	}
	if handlerCalled {
		w.WriteHeader(http.StatusOK)
	}
	return w
}

func TestRequireRole_StaffBlockedFromAdminOnly(t *testing.T) {
	w := runRBAC(t, models.RoleStaff, models.RoleAdmin)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for STAFF hitting an ADMIN-only route, got %d", w.Code)
	}
}

func TestRequireRole_AdminAllowedOnAdminOnly(t *testing.T) {
	w := runRBAC(t, models.RoleAdmin, models.RoleAdmin)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK for ADMIN hitting an ADMIN-only route, got %d", w.Code)
	}
}

func TestRequireRole_UnauthenticatedRejected(t *testing.T) {
	w := runRBAC(t, nil, models.RoleAdmin)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized when no role is set in session, got %d", w.Code)
	}
}
