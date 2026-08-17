package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/watt-siwat/agnos-backend/internal/auth"
)

const (
	ContextKeyStaffID    = "staff_id"
	ContextKeyHospitalID = "hospital_id"
)

// JWTAuth validates the Bearer access token and injects staff_id/hospital_id
// into the request context — handlers read hospital scope from here, never
// from client-supplied input.
func JWTAuth(secret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(header, prefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid authorization header"})
			return
		}

		tokenStr := strings.TrimPrefix(header, prefix)
		claims, err := auth.ParseAccessToken(secret, tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		c.Set(ContextKeyStaffID, claims.StaffID)
		c.Set(ContextKeyHospitalID, claims.HospitalID)
		c.Next()
	}
}
