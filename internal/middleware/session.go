package middleware

import (
	"os"

	ginsessions "github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

const sessionName = "admin_session"
const sessionAdminKey = "admin_id"

// Sessions returns session middleware backed by an encrypted cookie store.
func Sessions() gin.HandlerFunc {
	secret := os.Getenv("SESSION_SECRET")
	if secret == "" {
		secret = "change-me-in-production-32bytes!"
	}
	store := cookie.NewStore([]byte(secret))
	return ginsessions.Sessions(sessionName, store)
}

// LoadSessionAdmin reads the admin_id from the session and sets it on the
// context so RequireAuth / SessionAuth can find it.
func LoadSessionAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := ginsessions.Default(c)
		v := session.Get(sessionAdminKey)
		if v != nil {
			c.Set(AdminIDKey, v)
		}
		c.Next()
	}
}

// SetSessionAdmin writes the admin ID into the session cookie.
func SetSessionAdmin(c *gin.Context, adminID uint) {
	session := ginsessions.Default(c)
	session.Set(sessionAdminKey, adminID)
	session.Save() //nolint:errcheck
}

// ClearSession removes admin session data.
func ClearSession(c *gin.Context) {
	session := ginsessions.Default(c)
	session.Clear()
	session.Save() //nolint:errcheck
}
