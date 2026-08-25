package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/danegigi/go-tut/internal/models"
	"github.com/danegigi/go-tut/internal/views/pages"
)

// dashboardRange returns the default 30-day window used by every dashboard fragment.
func dashboardRange() (string, string) {
	end := time.Now()
	start := end.AddDate(0, -1, 0)
	return start.Format("2006-01-02"), end.Format("2006-01-02")
}

// GetHome renders the dashboard SHELL only — zero DB work, paints instantly.
// Each section fetches its own data via the /dashboard/* fragment endpoints below.
func (h *Handler) GetHome(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	pages.Home().Render(c.Request.Context(), c.Writer)
}

// ── Dashboard stat-card fragments ──────────────────────────────────────────

func (h *Handler) DashStatNewUsers(c *gin.Context) {
	start, end := dashboardRange()
	rows, err := h.OrderStore.QueryActiveUsersPerformance(start, end)
	h.renderStat(c, "stat-new-users", "New Users (latest week)", latestTotal(rows), "indigo", err)
}

func (h *Handler) DashStatNewLabels(c *gin.Context) {
	start, end := dashboardRange()
	rows, err := h.OrderStore.QueryNewLabels(start, end)
	h.renderStat(c, "stat-new-labels", "New Labels (latest week)", latestTotal(rows), "emerald", err)
}

func (h *Handler) DashStatPrinted(c *gin.Context) {
	start, end := dashboardRange()
	rows, err := h.OrderStore.QueryPrintPerformance(start, end)
	h.renderStat(c, "stat-printed", "Printed Labels (latest week)", latestTotal(rows), "violet", err)
}

func (h *Handler) DashStatActiveShippers(c *gin.Context) {
	start, end := dashboardRange()
	rows, err := h.OrderStore.QueryPrintPerformance(start, end)
	h.renderStat(c, "stat-active", "Active Shippers (latest week)", latestDistinct(rows), "amber", err)
}

// ── Dashboard weekly-table fragments ───────────────────────────────────────

func (h *Handler) DashTableNewUsers(c *gin.Context) {
	start, end := dashboardRange()
	rows, err := h.OrderStore.QueryActiveUsersPerformance(start, end)
	h.renderTable(c, "tbl-new-users", "New Users by Week", rows, false, err)
}

func (h *Handler) DashTableNewLabels(c *gin.Context) {
	start, end := dashboardRange()
	rows, err := h.OrderStore.QueryNewLabels(start, end)
	h.renderTable(c, "tbl-new-labels", "New Labels by Week", rows, false, err)
}

func (h *Handler) DashTablePrintPerformance(c *gin.Context) {
	start, end := dashboardRange()
	rows, err := h.OrderStore.QueryPrintPerformance(start, end)
	h.renderTable(c, "tbl-print-perf", "Print Performance by Week", rows, true, err)
}

func (h *Handler) DashTableActiveUsers(c *gin.Context) {
	start, end := dashboardRange()
	rows, err := h.OrderStore.QueryActiveUsersPerformance(start, end)
	h.renderTable(c, "tbl-active", "Active Users (Signup) by Week", rows, false, err)
}

// renderStat writes a stat-card fragment (or a value of "—" on error, so a
// single failed query never blanks the page).
func (h *Handler) renderStat(c *gin.Context, id, label, value, color string, err error) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err != nil {
		value = "—"
	}
	pages.StatCardFragment(id, label, value, color).Render(c.Request.Context(), c.Writer)
}

// renderTable writes a weekly-table fragment.
func (h *Handler) renderTable(c *gin.Context, id, title string, rows []models.WeeklyPerformance, showDistinct bool, err error) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	pages.WeeklyTableFragment(id, title, rows, showDistinct).Render(c.Request.Context(), c.Writer)
}

// latestTotal returns the TotalRecords of the most recent week, or "—".
func latestTotal(rows []models.WeeklyPerformance) string {
	if len(rows) == 0 {
		return "—"
	}
	return strconv.FormatInt(rows[len(rows)-1].TotalRecords, 10)
}

// latestDistinct returns the DistinctUsers of the most recent week, or "—".
func latestDistinct(rows []models.WeeklyPerformance) string {
	if len(rows) == 0 {
		return "—"
	}
	d := rows[len(rows)-1].DistinctUsers
	if d == nil {
		return "—"
	}
	return strconv.FormatInt(*d, 10)
}

// ---------------------
// Reports – API endpoints
// ---------------------

type reportRequest struct {
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
	UserID    *uint  `json:"userId"`
}

type reportServiceRequest struct {
	reportRequest
	ShippingServices []int `json:"shippingServices"`
}

// APIReportPrintPerformance returns weekly print-performance data as JSON.
func (h *Handler) APIReportPrintPerformance(c *gin.Context) {
	var req reportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	rows, err := h.OrderStore.QueryPrintPerformance(req.StartDate, req.EndDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Return as {yearWeek: row, ...} to match existing frontend contract
	out := make(map[int64]interface{}, len(rows))
	for _, r := range rows {
		out[r.YearWeek] = r
	}
	c.JSON(http.StatusOK, out)
}

// APIReportActiveUsers returns weekly active-user signup counts.
func (h *Handler) APIReportActiveUsers(c *gin.Context) {
	var req reportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	rows, err := h.OrderStore.QueryActiveUsersPerformance(req.StartDate, req.EndDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make(map[int64]interface{}, len(rows))
	for _, r := range rows {
		out[r.YearWeek] = r
	}
	c.JSON(http.StatusOK, out)
}

// APIReportNewLabels returns weekly new-label counts.
func (h *Handler) APIReportNewLabels(c *gin.Context) {
	var req reportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	rows, err := h.OrderStore.QueryNewLabels(req.StartDate, req.EndDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make(map[int64]interface{}, len(rows))
	for _, r := range rows {
		out[r.YearWeek] = r
	}
	c.JSON(http.StatusOK, out)
}

// APIReportLabelsByMarketplace returns label counts grouped by marketplace.
func (h *Handler) APIReportLabelsByMarketplace(c *gin.Context) {
	var req reportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	rows, err := h.OrderStore.QueryLabelsByMarketplace(req.StartDate, req.EndDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rows)
}
