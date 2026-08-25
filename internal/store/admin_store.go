package store

import (
	"github.com/grandshipper/admin-v2/internal/models"
	"github.com/jmoiron/sqlx"
)

// AdminStore handles reads/writes to the admins table.
type AdminStore struct {
	db *sqlx.DB
}

func NewAdminStore(db *sqlx.DB) *AdminStore {
	return &AdminStore{db: db}
}

// FindByUsername looks up an admin by their username (email).
func (s *AdminStore) FindByUsername(username string) (*models.Admin, error) {
	var a models.Admin
	err := s.db.Get(&a,
		`SELECT id, username, COALESCE(name,'') AS name, password, last_login
		 FROM admins WHERE username = ? LIMIT 1`, username)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// UpdateLastLogin stamps the last_login field.
func (s *AdminStore) UpdateLastLogin(id uint) error {
	_, err := s.db.Exec(`UPDATE admins SET last_login = NOW() WHERE id = ?`, id)
	return err
}
