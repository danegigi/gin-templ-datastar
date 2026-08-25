package handlers

import (
	"encoding/base64"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/danegigi/go-tut/internal/models"
	"github.com/danegigi/go-tut/internal/store"
	"github.com/danegigi/go-tut/internal/views/pages"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/invoice"
	"golang.org/x/sync/errgroup"
)

// GetLabels renders the labels list page (overview placeholder; deep per-label
// view goes through the existing GraphQL / API layer).
func (h *Handler) GetLabels(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	// For now, render the report page as the default labels view.
	pages.LabelsReport(pages.LabelsReportData{}).Render(c.Request.Context(), c.Writer)
}

// GetLabelsReport renders the label report filter form.
func (h *Handler) GetLabelsReport(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	pages.LabelsReport(pages.LabelsReportData{
		StartDate: c.Query("start_date"),
		EndDate:   c.Query("end_date"),
	}).Render(c.Request.Context(), c.Writer)
}

// PostLabelsReport runs the label report query and renders results.
func (h *Handler) PostLabelsReport(c *gin.Context) {
	startDate := c.PostForm("start_date")
	endDate := c.PostForm("end_date")
	serviceStrs := c.PostFormArray("services")

	var serviceIDs []int
	for _, s := range serviceStrs {
		id, err := strconv.Atoi(s)
		if err == nil {
			serviceIDs = append(serviceIDs, id)
		}
	}

	results := map[string]models.LabelReport{}

	if len(serviceIDs) > 0 {
		rows, err := h.OrderStore.QueryLabelReportRows(startDate, endDate, serviceIDs, nil)
		if err == nil {
			for _, r := range rows {
				name := buildReportName(r.LabelType, r.ServiceType, r.Category, r.ReturnStatus)
				results[name] = models.LabelReport{
					Name:           name,
					Code:           r.ServiceType,
					TotalLabels:    r.Pieces,
					TotalSpent:     r.Spend,
					InsuranceSpent: r.Insurance,
				}
			}
		}
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	pages.LabelsReport(pages.LabelsReportData{
		Reports:   results,
		StartDate: startDate,
		EndDate:   endDate,
		Services:  serviceIDs,
	}).Render(c.Request.Context(), c.Writer)
}

// ---------------------
// JSON API handlers
// ---------------------

type labelReportRequest struct {
	StartDate        string `json:"startDate"`
	EndDate          string `json:"endDate"`
	UserID           *uint  `json:"userId"`
	ShippingServices []int  `json:"shippingServices"`
}

// APIReportLabels mirrors POST /reports/labels.
// The two independent report queries (main services + return labels) run
// concurrently in goroutines via errgroup, so total latency is the slower of
// the two rather than their sum.
func (h *Handler) APIReportLabels(c *gin.Context) {
	var req labelReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	mainIDs := filterInts(req.ShippingServices, []int{2, 3, 4, 5, 6, 8, 13})
	wantReturns := false
	for _, s := range req.ShippingServices {
		if s == -1 {
			wantReturns = true
			break
		}
	}

	var (
		mainRows   []store.LabelReportRow
		returnRows []store.LabelReportRow
	)

	// Fan out the two independent queries.
	g, _ := errgroup.WithContext(c.Request.Context())
	if len(mainIDs) > 0 {
		g.Go(func() error {
			rows, err := h.OrderStore.QueryLabelReportRows(req.StartDate, req.EndDate, mainIDs, req.UserID)
			if err != nil {
				return err
			}
			mainRows = rows
			return nil
		})
	}
	if wantReturns {
		g.Go(func() error {
			rows, err := h.OrderStore.QueryReturnLabelReportRows(req.StartDate, req.EndDate, req.UserID)
			if err != nil {
				return err
			}
			returnRows = rows
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Merge results after both goroutines have finished (no concurrent map writes).
	results := map[string]models.LabelReport{}
	for _, r := range append(mainRows, returnRows...) {
		name := buildReportName(r.LabelType, r.ServiceType, r.Category, r.ReturnStatus)
		results[name] = models.LabelReport{
			Name:           name,
			Code:           r.ServiceType,
			TotalLabels:    r.Pieces,
			TotalSpent:     r.Spend,
			InsuranceSpent: r.Insurance,
		}
	}

	c.JSON(http.StatusOK, results)
}

// APIGetLabel fetches and merges PDF label data from S3 / the label service.
// Returns base64-encoded merged PDF, matching the existing API contract.
func (h *Handler) APIGetLabel(c *gin.Context) {
	var req struct {
		URLs []string `json:"urls" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.URLs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "urls is empty"})
		return
	}

	// Fetch every label PDF concurrently. Results are written into a
	// pre-sized slice by index so the merged output preserves URL order
	// (no shared append, so no mutex needed).
	fetched := make([][]byte, len(req.URLs))
	var wg sync.WaitGroup
	for i, url := range req.URLs {
		wg.Add(1)
		go func(idx int, u string) {
			defer wg.Done()
			data, err := fetchURL(u)
			if err == nil {
				fetched[idx] = data // each goroutine writes a distinct index
			}
		}(i, url)
	}
	wg.Wait()

	// Collect the successful fetches in original order.
	var pdfParts [][]byte
	for _, p := range fetched {
		if p != nil {
			pdfParts = append(pdfParts, p)
		}
	}

	if len(pdfParts) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no labels fetched"})
		return
	}

	// Merge: for now concatenate (proper PDF merging requires a library).
	merged := concatPDFs(pdfParts)
	c.JSON(http.StatusOK, base64.StdEncoding.EncodeToString(merged))
}

// APIGetTotalBilled proxies to Stripe to sum invoices for a customer.
func (h *Handler) APIGetTotalBilled(c *gin.Context) {
	customerID := c.Param("customerId")
	stripeSecret := os.Getenv("STRIPE_SECRET")
	if stripeSecret == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Billing service unavailable"})
		return
	}

	stripe.Key = stripeSecret
	var totalCents int64
	currency := "usd"

	params := &stripe.InvoiceListParams{}
	params.Filters.AddFilter("customer", "", customerID)
	params.Filters.AddFilter("limit", "", "100")
	iter := invoice.List(params)
	for iter.Next() {
		inv := iter.Invoice()
		if inv.Status != stripe.InvoiceStatusVoid && inv.Status != stripe.InvoiceStatusDraft {
			totalCents += inv.Total
			currency = string(inv.Currency)
		}
	}
	if err := iter.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"customerId": customerID,
		"value":      float64(totalCents) / 100,
		"currency":   strings.ToUpper(currency),
	})
}

// APIUpdatePBStatus updates the pb_info JSON on a user.
func (h *Handler) APIUpdatePBStatus(c *gin.Context) {
	var req struct {
		ID     uint   `json:"id" binding:"required"`
		PBInfo string `json:"pbInfo"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.LabelStore.UpdatePBStatus(req.ID, req.PBInfo); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "pb status updated"})
}

// ---------------------
// helpers
// ---------------------

func buildReportName(labelType *string, serviceType string, category, returnStatus *string) string {
	parts := []string{}
	if labelType != nil && *labelType != "" {
		parts = append(parts, *labelType)
	}
	if serviceType != "" {
		parts = append(parts, serviceType)
	}
	if category != nil && *category != "" {
		parts = append(parts, *category)
	}
	if returnStatus != nil && *returnStatus != "" {
		parts = append(parts, *returnStatus)
	}
	if len(parts) == 0 {
		return "UNKNOWN"
	}
	return strings.ToUpper(strings.Join(parts, "_"))
}

func filterInts(input, allowed []int) []int {
	set := map[int]bool{}
	for _, v := range allowed {
		set[v] = true
	}
	var out []int
	for _, v := range input {
		if set[v] {
			out = append(out, v)
		}
	}
	return out
}

func fetchURL(rawURL string) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(rawURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// concatPDFs is a naive concatenation – replace with a proper PDF-merge lib
// (e.g. github.com/unidoc/unipdf) for production use.
func concatPDFs(parts [][]byte) []byte {
	if len(parts) == 1 {
		return parts[0]
	}
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}
