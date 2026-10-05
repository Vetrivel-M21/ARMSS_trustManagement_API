package shared

import (
	"strconv"
	"trust-management/backend/internal/models"

	"github.com/gin-gonic/gin"
)

// ResolveBranchID checks user role and headers/queries to determine target branch.
// For STAFF: always returns the user's assigned BranchID (cannot be overridden).
// For ADMIN:
//   - If X-Branch-ID header or ?branch_id is provided and > 0, returns that branchID.
//   - If "0", "ALL", or omitted, returns nil (meaning ALL branches / consolidated).
func ResolveBranchID(c *gin.Context) *uint {
	roleVal, _ := c.Get("role")
	role, _ := roleVal.(models.Role)

	// If user is STAFF, strictly enforce their assigned BranchID
	if role == models.RoleStaff {
		branchIDVal, exists := c.Get("branch_id")
		if exists && branchIDVal != nil {
			if bid, ok := branchIDVal.(*uint); ok && bid != nil {
				return bid
			}
			if bid, ok := branchIDVal.(uint); ok && bid > 0 {
				return &bid
			}
		}
		defaultBranch := uint(1)
		return &defaultBranch
	}

	// For ADMIN: check X-Branch-ID header or branch_id query param
	headerVal := c.GetHeader("X-Branch-ID")
	if headerVal == "" {
		headerVal = c.Query("branch_id")
	}

	if headerVal != "" && headerVal != "0" && headerVal != "ALL" {
		if id, err := strconv.ParseUint(headerVal, 10, 32); err == nil && id > 0 {
			uid := uint(id)
			return &uid
		}
	}

	return nil
}
