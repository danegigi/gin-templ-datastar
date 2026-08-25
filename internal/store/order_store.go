package store

import (
	"fmt"
	"strings"
	"time"

	"github.com/grandshipper/admin-v2/internal/models"
	"github.com/jmoiron/sqlx"
)

// OrderStore handles reads from the orders table.
type OrderStore struct {
	db *sqlx.DB
}

func NewOrderStore(db *sqlx.DB) *OrderStore {
	return &OrderStore{db: db}
}

// OrderFilter holds optional filter parameters for listing orders.
type OrderFilter struct {
	UserID    *uint
	StatusIDs []int
	ServiceIDs []int
	StartDate *time.Time
	EndDate   *time.Time
}

// LabelReportRow is a raw DB result row from the label-reports query.
type LabelReportRow struct {
	ServiceType  string   `db:"service_type"`
	Category     *string  `db:"category"`
	LabelType    *string  `db:"label_type"`
	ReturnStatus *string  `db:"return_status"`
	Pieces       int64    `db:"pieces"`
	Spend        float64  `db:"spend"`
	Insurance    float64  `db:"insurance"`
}

// QueryLabelReportRows executes the core label-report SQL.
// serviceIDs must be a non-empty subset of [2,3,4,5,6,8,13].
func (s *OrderStore) QueryLabelReportRows(startDate, endDate string, serviceIDs []int, userID *uint) ([]LabelReportRow, error) {
	if len(serviceIDs) == 0 {
		return nil, nil
	}

	var timeCond, userCond string
	var args []interface{}

	if startDate != "" && endDate != "" {
		timeCond = "AND orders.created_at >= ? AND orders.created_at <= ?"
		args = append(args, startDate, endDate)
	}
	if userID != nil {
		userCond = "AND orders.user_id = ?"
		args = append(args, *userID)
	}

	placeholders := make([]string, len(serviceIDs))
	for i, id := range serviceIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}
	inClause := strings.Join(placeholders, ",")

	query := fmt.Sprintf(`
		SELECT
			CASE
				WHEN shipping_service_id = 2 THEN 'PM'
				WHEN shipping_service_id = 3 THEN 'EM'
				WHEN shipping_service_id = 5 THEN 'EMI'
				WHEN shipping_service_id = 6 THEN 'PMI'
				WHEN shipping_service_id = 8 THEN 'MEDIA'
				WHEN shipping_service_id = 4 THEN 'FCPIS'
				WHEN shipping_service_id = 13 THEN 'GA'
			END AS service_type,
			CASE
				WHEN shipping_service_id = 13 AND total_weight < 16 THEN 'lightweight'
				WHEN shipping_service_id = 13 AND total_weight >= 16 AND total_weight <= 320 THEN 'heavyweight'
				WHEN shipping_service_id IN(2,3,5,6) AND total_weight <= 320
					AND package_type IN('PKG','SOFTPACK','IRRPKG','RBA','RBB') THEN 'b'
				WHEN shipping_service_id IN(2,3,5,6)
					AND package_type IN('FRE','LGLFRENV','PFRENV','SFRB','FRB','LFRB','MLFRB') THEN 'a'
				WHEN shipping_service_id IN(8,4) THEN NULL
				ELSE 'overweight'
			END AS category,
			NULL AS label_type,
			NULL AS return_status,
			COUNT(id) AS pieces,
			COALESCE(SUM(rate),0) AS spend,
			COALESCE(SUM(insurance_price),0) AS insurance
		FROM orders
		WHERE order_status_id = 3
			AND deleted_at IS NULL
			AND shipping_service_id IN (%s)
			%s %s
		GROUP BY service_type, category`, inClause, timeCond, userCond)

	var rows []LabelReportRow
	if err := s.db.Select(&rows, query, args...); err != nil {
		return nil, err
	}
	return rows, nil
}

// QueryReturnLabelReportRows fetches return label rows.
func (s *OrderStore) QueryReturnLabelReportRows(startDate, endDate string, userID *uint) ([]LabelReportRow, error) {
	var timeCond, userCond string
	var args []interface{}

	if startDate != "" && endDate != "" {
		timeCond = "AND orders.created_at >= ? AND orders.created_at <= ?"
		args = append(args, startDate, endDate)
	}
	if userID != nil {
		userCond = "AND orders.user_id = ?"
		args = append(args, *userID)
	}

	query := fmt.Sprintf(`
		SELECT
			CASE
				WHEN shipping_service_id = 2 THEN 'PM'
				WHEN shipping_service_id = 3 THEN 'EM'
				WHEN shipping_service_id = 13 THEN 'GA'
			END AS service_type,
			'return' AS label_type,
			order_labels.return_status,
			CASE
				WHEN shipping_service_id = 13 AND total_weight < 16 THEN 'lightweight'
				WHEN shipping_service_id = 13 AND total_weight >= 16 AND total_weight <= 320 THEN 'heavyweight'
				WHEN shipping_service_id IN (2,3) AND total_weight <= 320
					AND package_type IN ('PKG','SOFTPACK','IRRPKG','RBA','RBB') THEN 'b'
				WHEN shipping_service_id IN (2,3)
					AND package_type IN ('FRE','LGLFRENV','PFRENV','SFRB','FRB','LFRB','MLFRB') THEN 'a'
				ELSE 'overweight'
			END AS category,
			COUNT(orders.id) AS pieces,
			COALESCE(SUM(rate),0) AS spend,
			COALESCE(SUM(insurance_price),0) AS insurance
		FROM orders
		JOIN order_labels ON orders.id = order_labels.order_id
		WHERE order_status_id = 5
			AND orders.deleted_at IS NULL
			AND shipping_service_id IN (2,3,13)
			%s %s
		GROUP BY service_type, return_status, category`, timeCond, userCond)

	var rows []LabelReportRow
	if err := s.db.Select(&rows, query, args...); err != nil {
		return nil, err
	}
	return rows, nil
}

// QueryPrintPerformance returns weekly order counts grouped by week.
func (s *OrderStore) QueryPrintPerformance(startDate, endDate string) ([]models.WeeklyPerformance, error) {
	var timeCond string
	var args []interface{}
	if startDate != "" && endDate != "" {
		timeCond = "AND orders.created_at >= ? AND orders.created_at <= ?"
		args = append(args, startDate, endDate)
	}

	q := fmt.Sprintf(`
		SELECT
			DATE_FORMAT(STR_TO_DATE(CONCAT(YEARWEEK(orders.created_at,0),' Sunday'),'%%X%%V %%W'),'%%m/%%d/%%y') AS first_day_of_week,
			DATE_FORMAT(ADDDATE(STR_TO_DATE(CONCAT(YEARWEEK(orders.created_at,0),' Sunday'),'%%X%%V %%W'),6),'%%m/%%d/%%y') AS last_day_of_week,
			YEARWEEK(orders.created_at,0) AS year_week,
			COUNT(DISTINCT user_id) AS distinct_users,
			COUNT(*) AS total_records
		FROM orders
		WHERE orders.deleted_at IS NULL
			AND orders.order_status_id IN (3,5)
			AND orders.shipping_service_id IN (4,13,8,2,3,5,6)
			%s
		GROUP BY year_week
		ORDER BY year_week`, timeCond)

	var out []models.WeeklyPerformance
	if err := s.db.Select(&out, q, args...); err != nil {
		return nil, err
	}
	return out, nil
}

// QueryActiveUsersPerformance returns weekly new-user counts.
func (s *OrderStore) QueryActiveUsersPerformance(startDate, endDate string) ([]models.WeeklyPerformance, error) {
	var timeCond string
	var args []interface{}
	if startDate != "" && endDate != "" {
		timeCond = "AND users.created_at >= ? AND users.created_at <= ?"
		args = append(args, startDate, endDate)
	}

	q := fmt.Sprintf(`
		SELECT
			DATE_FORMAT(STR_TO_DATE(CONCAT(YEARWEEK(created_at,0),' Sunday'),'%%X%%V %%W'),'%%m/%%d/%%y') AS first_day_of_week,
			DATE_FORMAT(ADDDATE(STR_TO_DATE(CONCAT(YEARWEEK(created_at,0),' Sunday'),'%%X%%V %%W'),6),'%%m/%%d/%%y') AS last_day_of_week,
			YEARWEEK(created_at,0) AS year_week,
			COUNT(*) AS total_records
		FROM users
		WHERE deleted_at IS NULL %s
		GROUP BY year_week
		ORDER BY year_week`, timeCond)

	var out []models.WeeklyPerformance
	if err := s.db.Select(&out, q, args...); err != nil {
		return nil, err
	}
	return out, nil
}

// QueryNewLabels returns counts of new order_labels grouped by week.
func (s *OrderStore) QueryNewLabels(startDate, endDate string) ([]models.WeeklyPerformance, error) {
	var timeCond string
	var args []interface{}
	if startDate != "" && endDate != "" {
		timeCond = "AND ol.created_at >= ? AND ol.created_at <= ?"
		args = append(args, startDate, endDate)
	}

	q := fmt.Sprintf(`
		SELECT
			DATE_FORMAT(STR_TO_DATE(CONCAT(YEARWEEK(ol.created_at,0),' Sunday'),'%%X%%V %%W'),'%%m/%%d/%%y') AS first_day_of_week,
			DATE_FORMAT(ADDDATE(STR_TO_DATE(CONCAT(YEARWEEK(ol.created_at,0),' Sunday'),'%%X%%V %%W'),6),'%%m/%%d/%%y') AS last_day_of_week,
			YEARWEEK(ol.created_at,0) AS year_week,
			COUNT(*) AS total_records
		FROM order_labels ol
		WHERE ol.deleted_at IS NULL %s
		GROUP BY year_week
		ORDER BY year_week`, timeCond)

	var out []models.WeeklyPerformance
	if err := s.db.Select(&out, q, args...); err != nil {
		return nil, err
	}
	return out, nil
}

// MarketplaceRow holds one marketplace label-count row.
type MarketplaceRow struct {
	MarketplaceName string  `db:"marketplace_name"`
	Pieces          int64   `db:"pieces"`
	TotalSpend      float64 `db:"total_spend"`
}

// QueryLabelsByMarketplace returns label counts grouped by marketplace.
func (s *OrderStore) QueryLabelsByMarketplace(startDate, endDate string) ([]MarketplaceRow, error) {
	var timeCond string
	var args []interface{}
	if startDate != "" && endDate != "" {
		timeCond = "AND o.created_at >= ? AND o.created_at <= ?"
		args = append(args, startDate, endDate)
	}

	q := fmt.Sprintf(`
		SELECT
			m.name AS marketplace_name,
			COUNT(o.id) AS pieces,
			COALESCE(SUM(o.rate),0) AS total_spend
		FROM orders o
		JOIN marketplaces m ON m.id = o.marketplace_id
		WHERE o.deleted_at IS NULL
			AND o.order_status_id IN (3,5)
			%s
		GROUP BY m.id, m.name
		ORDER BY pieces DESC`, timeCond)

	var out []MarketplaceRow
	if err := s.db.Select(&out, q, args...); err != nil {
		return nil, err
	}
	return out, nil
}
