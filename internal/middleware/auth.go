package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const AdminIDKey = "admin_id"

// RequireAuth validates the x-auth-token JWT header (or Authorization: Bearer).
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("x-auth-token")
		if token == "" {
			raw := c.GetHeader("Authorization")
			token = strings.TrimPrefix(raw, "Bearer ")
		}
		// Allow session-cookie based auth for HTML pages
		if token == "" {
			id, exists := c.Get(AdminIDKey)
			if exists && id != nil {
				c.Next()
				return
			}
		}

		if token == "" {
			// For HTML pages: redirect to login
			if isHTMLRequest(c) {
				c.Redirect(http.StatusFound, "/login")
				c.Abort()
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing auth token"})
			return
		}

		claims, err := parseJWT(token)
		if err != nil {
			if isHTMLRequest(c) {
				c.Redirect(http.StatusFound, "/login")
				c.Abort()
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		c.Set(AdminIDKey, claims)
		c.Next()
	}
}

// SessionAuth checks the gin session for an authenticated admin (HTML-first auth).
func SessionAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		adminID, exists := c.Get(AdminIDKey)
		if !exists || adminID == nil {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		c.Next()
	}
}

type Claims struct {
	ID uint `json:"id"`
	jwt.RegisteredClaims
}

func parseJWT(tokenStr string) (*Claims, error) {
	secret := []byte(os.Getenv("JWT_KEY"))
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	return claims, err
}

func isHTMLRequest(c *gin.Context) bool {
	accept := c.GetHeader("Accept")
	return strings.Contains(accept, "text/html")
}
