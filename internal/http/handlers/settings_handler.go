package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/grandshipper/admin-v2/internal/views/pages"
)

// GetSettings renders the settings page (countries list).
func (h *Handler) GetSettings(c *gin.Context) {
	countries, err := h.UserStore.ListCountries()
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	pages.Settings(countries).Render(c.Request.Context(), c.Writer)
}

// PostToggleCountry flips a country's active status.
func (h *Handler) PostToggleCountry(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	// Look up current state so we can toggle it.
	countries, err := h.UserStore.ListCountries()
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	for _, co := range countries {
		if co.ID == uint(id) {
			h.UserStore.UpdateCountryStatus(uint(id), !co.Active) //nolint:errcheck
			break
		}
	}
	c.Redirect(http.StatusFound, "/settings?flash=Updated")
}
