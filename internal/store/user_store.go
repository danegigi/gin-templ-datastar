package store

import (
	"fmt"
	"strings"
	"time"

	"github.com/danegigi/go-tut/internal/models"
	"github.com/jmoiron/sqlx"
)

// UserStore handles users table reads/writes.
type UserStore struct {
	db *sqlx.DB
}

func NewUserStore(db *sqlx.DB) *UserStore {
	return &UserStore{db: db}
}

// UserFilter encapsulates optional list filters.
type UserFilter struct {
	Search    string
	RoleID    *uint
	Suspended *bool
	Favorite  *bool
	Deleted   bool // if true, return soft-deleted rows only
	StartDate *time.Time
	EndDate   *time.Time
	Page      int
	Limit     int
}

const userSelectFields = `
	u.id, u.user_role_id, u.name, u.email,
	u.company, u.phone, u.address, u.address2,
	u.city, u.state, u.postal_code, u.country,
	u.rep_id, u.title, u.district, u.notes,
	u.favorite, u.suspend, u.suspend_reason,
	u.emailNotif, u.plan_price, u.postage_balance,
	u.shipsurance_balance, u.email_verified_at,
	u.last_login, u.created_at, u.updated_at, u.deleted_at,
	ur.name AS role_name
`

func buildUserWhere(f UserFilter) (string, []interface{}) {
	var conds []string
	var args []interface{}

	if f.Deleted {
		conds = append(conds, "u.deleted_at IS NOT NULL")
	} else {
		conds = append(conds, "u.deleted_at IS NULL")
	}

	if f.RoleID != nil {
		conds = append(conds, "u.user_role_id = ?")
		args = append(args, *f.RoleID)
	}

	if f.Suspended != nil {
		if *f.Suspended {
			conds = append(conds, "u.suspend = 1")
		} else {
			conds = append(conds, "u.suspend = 0")
		}
	}

	if f.Favorite != nil {
		if *f.Favorite {
			conds = append(conds, "u.favorite = 1")
		} else {
			conds = append(conds, "u.favorite = 0")
		}
	}

	if f.Search != "" {
		s := "%" + f.Search + "%"
		conds = append(conds, "(u.name LIKE ? OR u.email LIKE ? OR u.company LIKE ? OR u.rep_id LIKE ?)")
		args = append(args, s, s, s, s)
	}

	if f.StartDate != nil {
		conds = append(conds, "u.created_at >= ?")
		args = append(args, f.StartDate.Format("2006-01-02 15:04:05"))
	}
	if f.EndDate != nil {
		conds = append(conds, "u.created_at <= ?")
		args = append(args, f.EndDate.Format("2006-01-02 15:04:05"))
	}

	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}
	return where, args
}

// ListUsers returns a paginated list of users with an optional filter.
func (s *UserStore) ListUsers(f UserFilter) ([]models.User, int64, error) {
	if f.Limit == 0 {
		f.Limit = 50
	}
	if f.Page < 1 {
		f.Page = 1
	}
	offset := (f.Page - 1) * f.Limit

	where, args := buildUserWhere(f)

	countSQL := fmt.Sprintf(`
		SELECT COUNT(*) FROM users u
		LEFT JOIN user_roles ur ON ur.id = u.user_role_id
		%s`, where)

	var total int64
	if err := s.db.Get(&total, countSQL, args...); err != nil {
		return nil, 0, err
	}

	listSQL := fmt.Sprintf(`
		SELECT %s
		FROM users u
		LEFT JOIN user_roles ur ON ur.id = u.user_role_id
		%s
		ORDER BY u.created_at DESC
		LIMIT ? OFFSET ?`, userSelectFields, where)

	args = append(args, f.Limit, offset)
	var users []models.User
	if err := s.db.Select(&users, listSQL, args...); err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

// GetUserByID returns a single user with role name.
func (s *UserStore) GetUserByID(id uint) (*models.User, error) {
	var u models.User
	err := s.db.Get(&u, fmt.Sprintf(`
		SELECT %s
		FROM users u
		LEFT JOIN user_roles ur ON ur.id = u.user_role_id
		WHERE u.id = ?`, userSelectFields), id)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// GetUserByEmail finds a user by email.
func (s *UserStore) GetUserByEmail(email string) (*models.User, error) {
	var u models.User
	err := s.db.Get(&u, fmt.Sprintf(`
		SELECT %s
		FROM users u
		LEFT JOIN user_roles ur ON ur.id = u.user_role_id
		WHERE u.email = ? AND u.deleted_at IS NULL
		LIMIT 1`, userSelectFields), email)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// CreateUser inserts a new user row and returns the new ID.
func (s *UserStore) CreateUser(u *models.User, hashedPwd string) (uint, error) {
	res, err := s.db.Exec(`
		INSERT INTO users
		  (user_role_id, name, email, password, company, phone,
		   address, city, state, postal_code, country,
		   email_verified_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW(), NOW())`,
		u.UserRoleID, u.Name, u.Email, hashedPwd,
		u.Company, u.Phone, u.Address, u.City, u.State,
		u.PostalCode, u.Country,
	)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return uint(id), nil
}

// UpdateRepID sets the rep_id field after user creation.
func (s *UserStore) UpdateRepID(id uint, repID string) error {
	_, err := s.db.Exec(`UPDATE users SET rep_id = ? WHERE id = ?`, repID, id)
	return err
}

// UpdatePassword sets a hashed password for the given user.
func (s *UserStore) UpdatePassword(id uint, hashedPwd string) error {
	_, err := s.db.Exec(`UPDATE users SET password = ?, updated_at = NOW() WHERE id = ?`, hashedPwd, id)
	return err
}

// SoftDelete marks a user as deleted.
func (s *UserStore) SoftDelete(id uint) error {
	_, err := s.db.Exec(`UPDATE users SET deleted_at = NOW() WHERE id = ?`, id)
	return err
}

// Restore un-deletes a soft-deleted user.
func (s *UserStore) Restore(id uint) error {
	_, err := s.db.Exec(`UPDATE users SET deleted_at = NULL, updated_at = NOW() WHERE id = ?`, id)
	return err
}

// ToggleFavorite flips the favorite flag.
func (s *UserStore) ToggleFavorite(id uint, fav bool) error {
	_, err := s.db.Exec(`UPDATE users SET favorite = ?, updated_at = NOW() WHERE id = ?`, fav, id)
	return err
}

// ToggleSuspend sets suspend + reason.
func (s *UserStore) ToggleSuspend(id uint, suspend bool, reason string) error {
	_, err := s.db.Exec(
		`UPDATE users SET suspend = ?, suspend_reason = ?, updated_at = NOW() WHERE id = ?`,
		suspend, reason, id)
	return err
}

// GetActivities returns admin activity notes for a user.
func (s *UserStore) GetActivities(userID uint) ([]models.AdminActivity, error) {
	var out []models.AdminActivity
	err := s.db.Select(&out,
		`SELECT id, note, user_id, admin_name, type, created_at
		 FROM admin_activities WHERE user_id = ? ORDER BY created_at DESC`, userID)
	return out, err
}

// AddActivity inserts an admin activity note.
func (s *UserStore) AddActivity(a *models.AdminActivity) error {
	_, err := s.db.Exec(
		`INSERT INTO admin_activities (note, user_id, admin_name, type, created_at)
		 VALUES (?, ?, ?, ?, NOW())`,
		a.Note, a.UserID, a.AdminName, a.Type)
	return err
}

// ListCountries returns all countries.
func (s *UserStore) ListCountries() ([]models.Country, error) {
	var out []models.Country
	err := s.db.Select(&out, `SELECT id, name, code, is_us_territory, active FROM countries ORDER BY name`)
	return out, err
}

// UpdateCountryStatus toggles a country's active flag.
func (s *UserStore) UpdateCountryStatus(id uint, active bool) error {
	_, err := s.db.Exec(`UPDATE countries SET active = ? WHERE id = ?`, active, id)
	return err
}
