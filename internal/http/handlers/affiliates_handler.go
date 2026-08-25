package handlers

import (
	"fmt"
	"math/rand"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/danegigi/go-tut/internal/models"
	"github.com/danegigi/go-tut/internal/store"
	"github.com/danegigi/go-tut/internal/views/pages"
	"golang.org/x/crypto/bcrypt"
)

// Affiliates use user_role_id = 3 (non-rep, non-admin role).
// USPS Reps use user_role_id = 2.

const (
	roleIDRep       uint = 2
	roleIDAffiliate uint = 3
)

// ---------------------
// USPS REPS
// ---------------------

func (h *Handler) GetUSPSReps(c *gin.Context) {
	// Shell renders instantly; list lazy-loads via /usps-reps/list.
	c.Header("Content-Type", "text/html; charset=utf-8")
	pages.USPSReps(pages.USPSRepsData{Search: c.Query("q")}).Render(c.Request.Context(), c.Writer)
}

// GetUSPSRepsList returns the lazily-loaded USPS reps table fragment.
func (h *Handler) GetUSPSRepsList(c *gin.Context) {
	roleID := roleIDRep
	f := buildUserFilter(c, nil)
	f.RoleID = &roleID

	users, total, err := h.UserStore.ListUsers(f)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	pages.USPSRepsListFragment(pages.USPSRepsData{
		Users: users, TotalItems: total,
		Page: f.Page, Limit: f.Limit, Search: f.Search,
	}).Render(c.Request.Context(), c.Writer)
}

func (h *Handler) GetCreateUSPSRep(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	pages.CreateUSPSRep("", "").Render(c.Request.Context(), c.Writer)
}

func (h *Handler) PostCreateUSPSRep(c *gin.Context) {
	u, errMsg := parseUserForm(c)
	if errMsg != "" {
		c.Header("Content-Type", "text/html; charset=utf-8")
		pages.CreateUSPSRep("", errMsg).Render(c.Request.Context(), c.Writer)
		return
	}
	role := roleIDRep
	u.UserRoleID = &role
	id, err := createUserWithRepID(h, u, c.PostForm("password"))
	if err != nil {
		c.Header("Content-Type", "text/html; charset=utf-8")
		pages.CreateUSPSRep("", err.Error()).Render(c.Request.Context(), c.Writer)
		return
	}
	c.Redirect(http.StatusFound, fmt.Sprintf("/usps-reps/%d?flash=Rep+created", id))
}

func (h *Handler) GetUSPSRepDetail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	flash := c.Query("flash")
	c.Header("Content-Type", "text/html; charset=utf-8")
	pages.AccountDetailShell(uint(id), flash).Render(c.Request.Context(), c.Writer)
}

func (h *Handler) PostRepChangePassword(c *gin.Context) {
	u, ok := loadUser(c, h)
	if !ok {
		return
	}
	newPwd := c.PostForm("new_password")
	if newPwd == "" {
		c.Redirect(http.StatusFound, fmt.Sprintf("/usps-reps/%d", u.ID))
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(newPwd), bcrypt.DefaultCost)
	h.UserStore.UpdatePassword(u.ID, string(hash)) //nolint:errcheck
	c.Redirect(http.StatusFound, fmt.Sprintf("/usps-reps/%d?flash=Password+updated", u.ID))
}

// ---------------------
// AFFILIATES
// ---------------------

func (h *Handler) GetAffiliates(c *gin.Context) {
	// Shell renders instantly; the list lazy-loads via /affiliates/list.
	c.Header("Content-Type", "text/html; charset=utf-8")
	pages.Affiliates(pages.AffiliatesData{Search: c.Query("q")}).Render(c.Request.Context(), c.Writer)
}

// GetAffiliatesList returns the lazily-loaded affiliates table fragment.
func (h *Handler) GetAffiliatesList(c *gin.Context) {
	roleID := roleIDAffiliate
	f := buildUserFilter(c, nil)
	f.RoleID = &roleID

	users, total, err := h.UserStore.ListUsers(f)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	pages.AffiliatesListFragment(pages.AffiliatesData{
		Users: users, TotalItems: total,
		Page: f.Page, Limit: f.Limit, Search: f.Search,
	}).Render(c.Request.Context(), c.Writer)
}

func (h *Handler) GetCreateAffiliate(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	pages.CreateAffiliate("", "").Render(c.Request.Context(), c.Writer)
}

func (h *Handler) PostCreateAffiliate(c *gin.Context) {
	u, errMsg := parseUserForm(c)
	if errMsg != "" {
		c.Header("Content-Type", "text/html; charset=utf-8")
		pages.CreateAffiliate("", errMsg).Render(c.Request.Context(), c.Writer)
		return
	}
	role := roleIDAffiliate
	u.UserRoleID = &role
	id, err := createUserWithRepID(h, u, c.PostForm("password"))
	if err != nil {
		c.Header("Content-Type", "text/html; charset=utf-8")
		pages.CreateAffiliate("", err.Error()).Render(c.Request.Context(), c.Writer)
		return
	}
	c.Redirect(http.StatusFound, fmt.Sprintf("/affiliates/%d?flash=Affiliate+created", id))
}

func (h *Handler) GetAffiliateDetail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	flash := c.Query("flash")
	c.Header("Content-Type", "text/html; charset=utf-8")
	pages.AccountDetailShell(uint(id), flash).Render(c.Request.Context(), c.Writer)
}

func (h *Handler) PostAffiliateChangePassword(c *gin.Context) {
	u, ok := loadUser(c, h)
	if !ok {
		return
	}
	newPwd := c.PostForm("new_password")
	if newPwd == "" {
		c.Redirect(http.StatusFound, fmt.Sprintf("/affiliates/%d", u.ID))
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(newPwd), bcrypt.DefaultCost)
	h.UserStore.UpdatePassword(u.ID, string(hash)) //nolint:errcheck
	c.Redirect(http.StatusFound, fmt.Sprintf("/affiliates/%d?flash=Password+updated", u.ID))
}

// ---------------------
// JSON API wrappers
// ---------------------

type createAffiliateRequest struct {
	Name     string  `json:"name" binding:"required"`
	Email    string  `json:"email" binding:"required"`
	Password string  `json:"password" binding:"required"`
	Company  *string `json:"company"`
	Phone    *string `json:"phone"`
	Title    *string `json:"title"`
	District *string `json:"district"`
}

func (h *Handler) APICreateAffiliate(c *gin.Context) {
	var req createAffiliateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	role := roleIDAffiliate
	u := &models.User{
		Name: req.Name, Email: req.Email,
		Company: req.Company, Phone: req.Phone,
		Title: req.Title, District: req.District,
		UserRoleID: &role,
	}
	id, err := createUserWithRepID(h, u, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	newU, _ := h.UserStore.GetUserByID(id)
	c.JSON(http.StatusOK, newU)
}

func (h *Handler) APIAffiliateChangePassword(c *gin.Context) {
	h.APIAccountChangePassword(c)
}

type registerRepRequest struct {
	Name     string  `json:"name" binding:"required"`
	Email    string  `json:"email" binding:"required"`
	Password string  `json:"password" binding:"required"`
	Company  *string `json:"company"`
	Phone    *string `json:"phone"`
	Title    *string `json:"title"`
	District *string `json:"district"`
}

func (h *Handler) APIRegisterRep(c *gin.Context) {
	var req registerRepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	role := roleIDRep
	u := &models.User{
		Name: req.Name, Email: req.Email,
		Company: req.Company, Phone: req.Phone,
		Title: req.Title, District: req.District,
		UserRoleID: &role,
	}
	id, err := createUserWithRepID(h, u, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	newU, _ := h.UserStore.GetUserByID(id)
	c.JSON(http.StatusOK, newU)
}

func (h *Handler) APIRepChangePassword(c *gin.Context) {
	h.APIAccountChangePassword(c)
}

// ---------------------
// helpers
// ---------------------

func parseUserForm(c *gin.Context) (*models.User, string) {
	name := c.PostForm("name")
	email := c.PostForm("email")
	if name == "" || email == "" {
		return nil, "Name and email are required"
	}
	company := strPtr(c.PostForm("company"))
	phone := strPtr(c.PostForm("phone"))
	title := strPtr(c.PostForm("title"))
	district := strPtr(c.PostForm("district"))
	return &models.User{
		Name: name, Email: email,
		Company: company, Phone: phone,
		Title: title, District: district,
	}, ""
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func createUserWithRepID(h *Handler, u *models.User, password string) (uint, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	id, err := h.UserStore.CreateUser(u, string(hash))
	if err != nil {
		return 0, err
	}
	repID := fmt.Sprintf("GS%d%d", id, rand.Intn(10))
	h.UserStore.UpdateRepID(id, repID) //nolint:errcheck
	return id, nil
}

func buildAffiliateFilter(c *gin.Context, roleID uint) store.UserFilter {
	f := buildUserFilter(c, nil)
	f.RoleID = &roleID
	return f
}
