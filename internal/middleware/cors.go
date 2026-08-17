package middleware

import "github.com/gin-gonic/gin"

// CORS is permissive by design — it exists only so the local test UI
// (served from a different origin/port, or opened as a file://) can call the
// API during development. Not intended for a production deployment.
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
