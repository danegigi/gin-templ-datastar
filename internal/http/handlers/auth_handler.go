package handlers

import (
	"database/sql"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/grandshipper/admin-v2/internal/middleware"
	"github.com/grandshipper/admin-v2/internal/views/pages"
	"golang.org/x/crypto/bcrypt"
)

// GetLogin renders the login page.
func (h *Handler) GetLogin(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	pages.Login("").Render(c.Request.Context(), c.Writer)
}

// PostLogin handles HTML form login.
func (h *Handler) PostLogin(c *gin.Context) {
	email := c.PostForm("email")
	password := c.PostForm("password")

	admin, err := h.AdminStore.FindByUsername(email)
	if err != nil {
		if err == sql.ErrNoRows {
			renderLoginError(c, "Email or password don't match")
			return
		}
		renderLoginError(c, "Internal error")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(password)); err != nil {
		renderLoginError(c, "Email or password don't match")
		return
	}

	h.AdminStore.UpdateLastLogin(admin.ID) //nolint:errcheck
	middleware.SetSessionAdmin(c, admin.ID)
	c.Redirect(http.StatusFound, "/")
}

// Logout clears the session and redirects to login.
func (h *Handler) Logout(c *gin.Context) {
	middleware.ClearSession(c)
	c.Redirect(http.StatusFound, "/login")
}

// Ping is the health check handler.
func (h *Handler) Ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"serviceName": "admin-v2",
		"status":      "ok",
	})
}

func renderLoginError(c *gin.Context, msg string) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusUnauthorized)
	pages.Login(msg).Render(c.Request.Context(), c.Writer)
}

// ---------------------
// JSON API auth endpoints
// ---------------------

type loginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// APILogin is the token-returning JSON endpoint (used by the existing Next.js frontend).
func (h *Handler) APILogin(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	admin, err := h.AdminStore.FindByUsername(req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Email or password don't match"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Email or password don't match"})
		return
	}

	h.AdminStore.UpdateLastLogin(admin.ID) //nolint:errcheck

	token, err := generateJWT(admin.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token generation failed"})
		return
	}
	c.JSON(http.StatusOK, token)
}

func generateJWT(adminID uint) (string, error) {
	secret := []byte(os.Getenv("JWT_KEY"))
	expiryStr := os.Getenv("JWT_EXPIRATION")

	var expiry time.Duration
	switch expiryStr {
	case "":
		expiry = 8 * time.Hour
	default:
		d, err := time.ParseDuration(expiryStr)
		if err != nil {
			expiry = 8 * time.Hour
		} else {
			expiry = d
		}
	}

	claims := jwt.MapClaims{
		"id":  adminID,
		"exp": time.Now().Add(expiry).Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(secret)
}
