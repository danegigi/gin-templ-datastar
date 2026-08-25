package store

import (
	"github.com/grandshipper/admin-v2/internal/models"
	"github.com/jmoiron/sqlx"
)

// LabelStore handles order_labels reads/writes.
type LabelStore struct {
	db *sqlx.DB
}

func NewLabelStore(db *sqlx.DB) *LabelStore {
	return &LabelStore{db: db}
}

// GetByOrderID returns the label for an order.
func (s *LabelStore) GetByOrderID(orderID uint) (*models.OrderLabel, error) {
	var l models.OrderLabel
	err := s.db.Get(&l,
		`SELECT id, order_id, shipment_id, tracking_id, pdf_url,
			status, refund_status
		 FROM order_labels WHERE order_id = ? LIMIT 1`, orderID)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

// UpdatePBStatus updates pb_info JSON on the users table for a given user ID.
func (s *LabelStore) UpdatePBStatus(userID uint, pbInfo string) error {
	_, err := s.db.Exec(
		`UPDATE users SET pb_info = ?, updated_at = NOW() WHERE id = ?`,
		pbInfo, userID)
	return err
}
