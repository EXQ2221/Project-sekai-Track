package middleware

import (
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"Project_sekai_search/internal/pkg/response"
	"Project_sekai_search/internal/service"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func AuthMiddleware(authSvc *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		accessToken := bearerToken(c.GetHeader("Authorization"))
		if accessToken == "" {
			response.Error(c, http.StatusUnauthorized, "missing access token")
			c.Abort()
			return
		}

		identity, err := authSvc.ValidateAccessToken(c.Request.Context(), accessToken)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, err.Error())
			c.Abort()
			return
		}

		c.Set("user_id", identity.UserID)
		c.Set("username", identity.Username)
		c.Set("session_id", identity.SessionID)
		c.Set("access_token", accessToken)
		c.Next()
	}
}

func bearerToken(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}

	parts := strings.SplitN(header, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return strings.TrimSpace(parts[1])
	}

	return ""
}

var visitors = make(map[string]*rate.Limiter)
var mu sync.Mutex

func getLimiter(ip string) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	limiter, exists := visitors[ip]
	if !exists {
		limiter = rate.NewLimiter(5, 10)
		visitors[ip] = limiter
		time.AfterFunc(60*time.Minute, func() {
			mu.Lock()
			delete(visitors, ip)
			mu.Unlock()
		})
	}

	return limiter
}

func RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		limiter := getLimiter(c.ClientIP())
		if !limiter.Allow() {
			response.JSON(c, http.StatusTooManyRequests, "too many requests", gin.H{"retry_after": "1s"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func CORSMiddleware() gin.HandlerFunc {
	allowed := parseAllowedOrigins(os.Getenv("CORS_ALLOWED_ORIGINS"))
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			if len(allowed) == 0 || slices.Contains(allowed, origin) {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
			}
		}
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func parseAllowedOrigins(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
