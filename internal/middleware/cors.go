package middleware

import (
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS returns a Gin handler that implements the same whitelist logic as the
// existing Koa backend (comma-separated CORS_ORIGIN env var, regexp: prefix
// support, wildcard '*').
func CORS() gin.HandlerFunc {
	rawOrigins := os.Getenv("CORS_ORIGIN")
	whitelist := strings.Split(rawOrigins, ",")

	return func(c *gin.Context) {
		// Always allow ping / healthcheck
		if c.Request.URL.Path == "/ping" || c.Request.URL.Path == "/healthcheck" {
			c.Header("Access-Control-Allow-Origin", "*")
			c.Next()
			return
		}

		// Wildcard or empty whitelist
		if len(whitelist) == 0 || (len(whitelist) == 1 && (whitelist[0] == "" || whitelist[0] == "*")) {
			c.Header("Access-Control-Allow-Origin", "*")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Content-Type, x-auth-token, Authorization")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			if c.Request.Method == http.MethodOptions {
				c.AbortWithStatus(http.StatusNoContent)
				return
			}
			c.Next()
			return
		}

		origin := c.GetHeader("Origin")
		if origin == "" {
			origin = c.GetHeader("Referer")
		}

		parsedOrigin, err := url.Parse(origin)
		if err != nil || parsedOrigin.Host == "" {
			c.Next()
			return
		}

		matched := ""
		for _, w := range whitelist {
			w = strings.TrimSpace(w)
			if w == "" {
				continue
			}
			if strings.HasPrefix(w, "regexp:") {
				pattern := regexp.MustCompile(w[7:])
				if pattern.MatchString(parsedOrigin.Hostname()) {
					matched = parsedOrigin.Scheme + "://" + parsedOrigin.Host
					break
				}
			} else {
				if strings.ToLower(w) == strings.ToLower(parsedOrigin.Scheme+"://"+parsedOrigin.Host) ||
					strings.ToLower(w) == strings.ToLower(parsedOrigin.Hostname()) {
					matched = parsedOrigin.Scheme + "://" + parsedOrigin.Host
					break
				}
			}
		}

		if matched != "" {
			c.Header("Access-Control-Allow-Origin", matched)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Content-Type, x-auth-token, Authorization")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Vary", "Origin")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
