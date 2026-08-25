package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/grandshipper/admin-v2/internal/models"
	"github.com/grandshipper/admin-v2/internal/store"
	"github.com/grandshipper/admin-v2/internal/views/pages"
	"golang.org/x/crypto/bcrypt"
)

// GetAccounts renders the paginated accounts list.
func (h *Handler) GetAccounts(c *gin.Context) {
	f := buildUserFilter(c, nil)
	// Accounts = users with role_id NULL or 1
	roleID := uint(1)
	f.RoleID = &roleID

	users, total, err := h.UserStore.ListUsers(f)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	pages.Accounts(pages.AccountsData{
		Users:      users,
		TotalItems: total,
		Page:       f.Page,
		Limit:      f.Limit,
		Search:     f.Search,
	}).Render(c.Request.Context(), c.Writer)
}

// GetAccountsSearch returns ONLY the accounts table fragment (Datastar swaps it
// into #accounts-table on debounced search input).
func (h *Handler) GetAccountsSearch(c *gin.Context) {
	f := buildUserFilter(c, nil)
	roleID := uint(1)
	f.RoleID = &roleID

	users, total, err := h.UserStore.ListUsers(f)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	pages.AccountsTable(pages.AccountsData{
		Users:      users,
		TotalItems: total,
		Page:       f.Page,
		Limit:      f.Limit,
		Search:     f.Search,
	}).Render(c.Request.Context(), c.Writer)
}

// GetAccountDetail renders the account SHELL only — no DB query. The info card
// and activity feed lazy-load via /accounts/:id/info and /accounts/:id/activities.
func (h *Handler) GetAccountDetail(c *gin.Context) {
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

// GetAccountInfoFragment fetches the user record and renders the info + actions card.
func (h *Handler) GetAccountInfoFragment(c *gin.Context) {
	u, ok := loadUser(c, h)
	if !ok {
		return
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	pages.AccountInfoFragment(u).Render(c.Request.Context(), c.Writer)
}

// GetAccountActivitiesFragment fetches the activity feed and renders it.
func (h *Handler) GetAccountActivitiesFragment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	activities, _ := h.UserStore.GetActivities(uint(id))

	c.Header("Content-Type", "text/html; charset=utf-8")
	pages.AccountActivitiesFragment(activities, uint(id)).Render(c.Request.Context(), c.Writer)
}

// GetEditAccount – placeholder (redirects to detail for now; extend with a full edit form).
func (h *Handler) GetEditAccount(c *gin.Context) {
	id := c.Param("id")
	c.Redirect(http.StatusFound, fmt.Sprintf("/accounts/%s", id))
}

// PostToggleFavorite flips the favorite flag on an account.
func (h *Handler) PostToggleFavorite(c *gin.Context) {
	u, ok := loadUser(c, h)
	if !ok {
		return
	}
	favStr := c.PostForm("favorite")
	fav := favStr == "true"
	if err := h.UserStore.ToggleFavorite(u.ID, fav); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Redirect(http.StatusFound, fmt.Sprintf("/accounts/%d?flash=Updated", u.ID))
}

// PostToggleSuspend suspends or un-suspends an account.
func (h *Handler) PostToggleSuspend(c *gin.Context) {
	u, ok := loadUser(c, h)
	if !ok {
		return
	}
	suspend := c.PostForm("suspend") == "true"
	reason := c.PostForm("reason")
	if err := h.UserStore.ToggleSuspend(u.ID, suspend, reason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Redirect(http.StatusFound, fmt.Sprintf("/accounts/%d?flash=Updated", u.ID))
}

// PostAccountChangePassword changes the password on an account.
func (h *Handler) PostAccountChangePassword(c *gin.Context) {
	u, ok := loadUser(c, h)
	if !ok {
		return
	}
	newPwd := c.PostForm("new_password")
	if newPwd == "" {
		c.Redirect(http.StatusFound, fmt.Sprintf("/accounts/%d?flash=error", u.ID))
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPwd), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "hashing failed"})
		return
	}
	if err := h.UserStore.UpdatePassword(u.ID, string(hash)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Redirect(http.StatusFound, fmt.Sprintf("/accounts/%d?flash=Password+updated", u.ID))
}

// PostAddActivity adds an admin note to the user's activity feed.
func (h *Handler) PostAddActivity(c *gin.Context) {
	u, ok := loadUser(c, h)
	if !ok {
		return
	}
	note := c.PostForm("note")
	if note == "" {
		c.Redirect(http.StatusFound, fmt.Sprintf("/accounts/%d", u.ID))
		return
	}
	adminName := "Admin" // Could pull from session
	if err := h.UserStore.AddActivity(&models.AdminActivity{
		Note:      note,
		UserID:    u.ID,
		AdminName: adminName,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Redirect(http.StatusFound, fmt.Sprintf("/accounts/%d?flash=Note+added", u.ID))
}

// PostRestoreAccount un-deletes an account.
func (h *Handler) PostRestoreAccount(c *gin.Context) {
	u, ok := loadUser(c, h)
	if !ok {
		return
	}
	if err := h.UserStore.Restore(u.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Redirect(http.StatusFound, fmt.Sprintf("/accounts/%d?flash=Restored", u.ID))
}

// GetFavorites renders the favorites list.
func (h *Handler) GetFavorites(c *gin.Context) {
	t := true
	users, _, err := h.UserStore.ListUsers(store.UserFilter{
		Favorite: &t,
		Limit:    200,
		Page:     1,
	})
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	pages.Favorites(users).Render(c.Request.Context(), c.Writer)
}

// GetDeactivatedAccounts renders the shell; the table lazy-loads.
func (h *Handler) GetDeactivatedAccounts(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	pages.DeactivatedAccounts(pages.DeactivatedData{Search: c.Query("q")}).Render(c.Request.Context(), c.Writer)
}

// GetDeactivatedList returns the lazily-loaded deactivated-accounts table fragment.
func (h *Handler) GetDeactivatedList(c *gin.Context) {
	f := buildUserFilter(c, nil)
	f.Deleted = true

	users, total, err := h.UserStore.ListUsers(f)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	pages.DeactivatedListFragment(pages.DeactivatedData{
		Users:      users,
		TotalItems: total,
		Page:       f.Page,
		Limit:      f.Limit,
		Search:     f.Search,
	}).Render(c.Request.Context(), c.Writer)
}

// ---------------------
// JSON API wrappers
// ---------------------

type changePasswordRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// APIAccountChangePassword is the existing JSON API for account password change.
func (h *Handler) APIAccountChangePassword(c *gin.Context) {
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	u, err := h.UserStore.GetUserByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "hashing failed"})
		return
	}
	if err := h.UserStore.UpdatePassword(u.ID, string(hash)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "password updated"})
}

// ---------------------
// helpers
// ---------------------

func loadUser(c *gin.Context, h *Handler) (*models.User, bool) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return nil, false
	}
	u, err := h.UserStore.GetUserByID(uint(id))
	if err != nil {
		c.Status(http.StatusNotFound)
		return nil, false
	}
	return u, true
}

func buildUserFilter(c *gin.Context, roleID *uint) store.UserFilter {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	search := c.Query("q")
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	return store.UserFilter{
		RoleID: roleID,
		Search: search,
		Page:   page,
		Limit:  limit,
	}
}
