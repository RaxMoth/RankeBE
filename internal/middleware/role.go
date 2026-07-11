package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	db "ranke-be/internal/db/sqlc"
)

// RequireListRole returns middleware that checks if the authenticated user
// has one of the allowed roles for the list identified by the :id route param.
func RequireListRole(queries *db.Queries, allowedRoles ...string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(allowedRoles))
	for _, r := range allowedRoles {
		allowed[r] = true
	}

	return func(c *gin.Context) {
		userID, ok := GetUserID(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "authentication required"}})
			return
		}

		listIDStr := c.Param("id")
		listID, ok := ParseUUID(listIDStr)
		if !ok {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "invalid list id"}})
			return
		}

		member, err := queries.GetListMember(c.Request.Context(), db.GetListMemberParams{
			ListID: listID,
			UserID: userID,
		})
		if err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "FORBIDDEN", "message": "you are not a member of this list"}})
			return
		}

		if !allowed[member.Role] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "FORBIDDEN", "message": "insufficient permissions"}})
			return
		}

		// Store the role for downstream handlers
		c.Set("listRole", member.Role)
		c.Next()
	}
}

// RequireListMember is a shorthand that allows any member role.
func RequireListMember(queries *db.Queries) gin.HandlerFunc {
	return RequireListRole(queries, "owner", "admin", "member")
}

// UUIDFromString converts string to pgtype.UUID, returning zero value on error.
func UUIDFromString(s string) pgtype.UUID {
	u, _ := ParseUUID(s)
	return u
}
