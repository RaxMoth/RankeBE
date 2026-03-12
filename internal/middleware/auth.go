package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const UserIDKey = "userID"

func AuthRequired(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "missing authorization header"}})
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "invalid authorization header format"}})
			return
		}

		token, err := jwt.Parse(parts[1], func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "invalid or expired token"}})
			return
		}

		sub, err := token.Claims.GetSubject()
		if err != nil || sub == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "invalid token claims"}})
			return
		}

		c.Set(UserIDKey, sub)
		c.Next()
	}
}

// GetUserID extracts the authenticated user's UUID from the Gin context.
func GetUserID(c *gin.Context) (pgtype.UUID, bool) {
	raw, exists := c.Get(UserIDKey)
	if !exists {
		return pgtype.UUID{}, false
	}
	s, ok := raw.(string)
	if !ok {
		return pgtype.UUID{}, false
	}
	return ParseUUID(s)
}

// ParseUUID converts a string UUID to pgtype.UUID.
func ParseUUID(s string) (pgtype.UUID, bool) {
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		return pgtype.UUID{}, false
	}
	return u, true
}
