package middleware

import (
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// CORSMiddleware allows the configured frontend origin plus any additional
// origins from extraOrigins (comma-separated, e.g. from CORS_ALLOWED_ORIGINS)
// — no origin is ever hardcoded in source, so a deployment fully controls the
// allow-list via env.
func CORSMiddleware(frontendURL, extraOrigins string) gin.HandlerFunc {
	origins := []string{frontendURL}
	for _, o := range strings.Split(extraOrigins, ",") {
		if o = strings.TrimSpace(o); o != "" {
			origins = append(origins, o)
		}
	}

	config := cors.DefaultConfig()
	config.AllowOrigins = origins
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"}
	config.AllowCredentials = true
	config.MaxAge = 12 * time.Hour
	return cors.New(config)
}

func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()

		if raw != "" {
			path = path + "?" + raw
		}

		log.Info().
			Str("method", c.Request.Method).
			Str("path", path).
			Int("status", statusCode).
			Str("ip", clientIP).
			Dur("latency", latency).
			Msg("HTTP Request")
	}
}
