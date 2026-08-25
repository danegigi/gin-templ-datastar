package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/danegigi/go-tut/internal/middleware"
)

// RegisterRoutes wires all HTTP routes onto the Gin engine.
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	// Health
	r.GET("/ping", h.Ping)
	r.GET("/healthcheck", h.Ping)

	// Session middleware must be applied before auth-aware routes
	r.Use(middleware.Sessions())
	r.Use(middleware.LoadSessionAdmin())

	// Auth
	r.GET("/login", h.GetLogin)
	r.POST("/login", h.PostLogin)
	r.GET("/logout", h.Logout)

	// Dashboard (HTML, session-protected)
	protected := r.Group("/", middleware.SessionAuth())
	{
		protected.GET("/", h.GetHome)

		// Dashboard lazy-loaded fragments (each renders one section)
		protected.GET("/dashboard/stat/new-users", h.DashStatNewUsers)
		protected.GET("/dashboard/stat/new-labels", h.DashStatNewLabels)
		protected.GET("/dashboard/stat/printed", h.DashStatPrinted)
		protected.GET("/dashboard/stat/active-shippers", h.DashStatActiveShippers)
		protected.GET("/dashboard/table/new-users", h.DashTableNewUsers)
		protected.GET("/dashboard/table/new-labels", h.DashTableNewLabels)
		protected.GET("/dashboard/table/print-performance", h.DashTablePrintPerformance)
		protected.GET("/dashboard/table/active-users", h.DashTableActiveUsers)

		// Accounts
		protected.GET("/accounts", h.GetAccounts)
		protected.GET("/accounts/search", h.GetAccountsSearch)
		protected.GET("/accounts/:id", h.GetAccountDetail)
		protected.GET("/accounts/:id/info", h.GetAccountInfoFragment)
		protected.GET("/accounts/:id/activities", h.GetAccountActivitiesFragment)
		protected.GET("/accounts/:id/edit", h.GetEditAccount)
		protected.POST("/accounts/:id/change-password", h.PostAccountChangePassword)
		protected.POST("/accounts/:id/favorite", h.PostToggleFavorite)
		protected.POST("/accounts/:id/suspend", h.PostToggleSuspend)
		protected.POST("/accounts/:id/activity", h.PostAddActivity)
		protected.POST("/accounts/:id/restore", h.PostRestoreAccount)

		// Favorites
		protected.GET("/favorites", h.GetFavorites)

		// USPS Reps
		protected.GET("/usps-reps", h.GetUSPSReps)
		protected.GET("/usps-reps/list", h.GetUSPSRepsList)
		protected.GET("/usps-reps/create", h.GetCreateUSPSRep)
		protected.POST("/usps-reps/create", h.PostCreateUSPSRep)
		protected.GET("/usps-reps/:id", h.GetUSPSRepDetail)
		protected.POST("/usps-reps/:id/change-password", h.PostRepChangePassword)

		// Affiliates
		protected.GET("/affiliates", h.GetAffiliates)
		protected.GET("/affiliates/list", h.GetAffiliatesList)
		protected.GET("/affiliates/create", h.GetCreateAffiliate)
		protected.POST("/affiliates/create", h.PostCreateAffiliate)
		protected.GET("/affiliates/:id", h.GetAffiliateDetail)
		protected.POST("/affiliates/:id/change-password", h.PostAffiliateChangePassword)

		// Deactivated accounts
		protected.GET("/deactivated", h.GetDeactivatedAccounts)
		protected.GET("/deactivated/list", h.GetDeactivatedList)

		// Labels
		protected.GET("/labels", h.GetLabels)
		protected.GET("/labels/report", h.GetLabelsReport)
		protected.POST("/labels/report", h.PostLabelsReport)

		// Settings
		protected.GET("/settings", h.GetSettings)
		protected.POST("/settings/countries/:id/toggle", h.PostToggleCountry)
	}

	// JSON API (token auth – compatible with existing frontend)
	api := r.Group("/api", middleware.RequireAuth())
	{
		api.POST("/login", h.APILogin)
		api.POST("/register-rep", h.APIRegisterRep)
		api.POST("/account/change-password", h.APIAccountChangePassword)
		api.POST("/rep/change-password", h.APIRepChangePassword)
		api.POST("/affiliate/create", h.APICreateAffiliate)
		api.POST("/affiliate/change-password", h.APIAffiliateChangePassword)
		api.POST("/reports/print-performance", h.APIReportPrintPerformance)
		api.POST("/reports/active-users-performance", h.APIReportActiveUsers)
		api.POST("/reports/labels", h.APIReportLabels)
		api.POST("/reports/new-labels", h.APIReportNewLabels)
		api.POST("/reports/labels-by-marketplace", h.APIReportLabelsByMarketplace)
		api.POST("/get-label", h.APIGetLabel)
		api.GET("/get-total-billed/:customerId", h.APIGetTotalBilled)
		api.POST("/update-pb-status", h.APIUpdatePBStatus)
	}
}
