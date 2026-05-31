package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pabloju2003/url-shortener/internal/ratelimit"
)

func RateLimitMiddleware(limiter *ratelimit.IPRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !limiter.GetLimiter(c.ClientIP()).Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			c.Abort()
			return
		}
		c.Next()
	}
}
