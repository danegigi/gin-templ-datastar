package models

import "time"

// Admin represents the admins table.
type Admin struct {
	ID        uint      `db:"id"`
	Username  string    `db:"username"`
	Name      string    `db:"name"`
	Password  string    `db:"password"`
	LastLogin time.Time `db:"last_login"`
}

// User represents the users table (core fields used in admin views).
type User struct {
	ID                  uint       `db:"id" json:"id"`
	UserRoleID          *uint      `db:"user_role_id" json:"user_role_id"`
	Name                string     `db:"name" json:"name"`
	Email               string     `db:"email" json:"email"`
	Company             *string    `db:"company" json:"company"`
	Phone               *string    `db:"phone" json:"phone"`
	Address             *string    `db:"address" json:"address"`
	Address2            *string    `db:"address2" json:"address2"`
	City                *string    `db:"city" json:"city"`
	State               *string    `db:"state" json:"state"`
	PostalCode          *string    `db:"postal_code" json:"postal_code"`
	Country             *string    `db:"country" json:"country"`
	RepID               *string    `db:"rep_id" json:"rep_id"`
	Title               *string    `db:"title" json:"title"`
	District            *string    `db:"district" json:"district"`
	Notes               *string    `db:"notes" json:"notes"`
	Favorite            bool       `db:"favorite" json:"favorite"`
	Suspend             bool       `db:"suspend" json:"suspend"`
	SuspendReason       *string    `db:"suspend_reason" json:"suspend_reason"`
	EmailNotif          bool       `db:"emailNotif" json:"emailNotif"`
	PlanPrice           *float64   `db:"plan_price" json:"plan_price"`
	PostageBalance      *float64   `db:"postage_balance" json:"postage_balance"`
	ShipsuranceBalance  *float64   `db:"shipsurance_balance" json:"shipsurance_balance"`
	EmailVerifiedAt     *time.Time `db:"email_verified_at" json:"email_verified_at"`
	LastLogin           *time.Time `db:"last_login" json:"last_login"`
	CreatedAt           *time.Time `db:"created_at" json:"created_at"`
	UpdatedAt           *time.Time `db:"updated_at" json:"updated_at"`
	DeletedAt           *time.Time `db:"deleted_at" json:"deleted_at"`
	// Joined fields
	RoleName            *string    `db:"role_name" json:"role_name"`
}

// UserRole represents the user_roles table.
type UserRole struct {
	ID    uint   `db:"id"`
	Name  string `db:"name"`
	Admin bool   `db:"admin"`
	Rep   bool   `db:"rep"`
}

// Order is a summarised view of the orders table.
type Order struct {
	ID               uint       `db:"id" json:"id"`
	UserID           uint       `db:"user_id" json:"user_id"`
	MarketplaceID    uint       `db:"marketplace_id" json:"marketplace_id"`
	OrderStatusID    uint       `db:"order_status_id" json:"order_status_id"`
	ShippingServiceID *uint     `db:"shipping_service_id" json:"shipping_service_id"`
	Rate             *float64   `db:"rate" json:"rate"`
	TotalWeight      *float64   `db:"total_weight" json:"total_weight"`
	PackageType      string     `db:"package_type" json:"package_type"`
	CreatedAt        *time.Time `db:"created_at" json:"created_at"`
	DeletedAt        *time.Time `db:"deleted_at" json:"deleted_at"`
	// Joined
	UserName         *string    `db:"user_name" json:"user_name"`
	UserEmail        *string    `db:"user_email" json:"user_email"`
	ServiceCode      *string    `db:"service_code" json:"service_code"`
	StatusName       *string    `db:"status_name" json:"status_name"`
}

// OrderLabel holds label and tracking state.
type OrderLabel struct {
	ID           uint    `db:"id" json:"id"`
	OrderID      uint    `db:"order_id" json:"order_id"`
	ShipmentID   *string `db:"shipment_id" json:"shipment_id"`
	TrackingID   *string `db:"tracking_id" json:"tracking_id"`
	PdfURL       *string `db:"pdf_url" json:"pdf_url"`
	Status       string  `db:"status" json:"status"`
	RefundStatus *string `db:"refund_status" json:"refund_status"`
}

// LabelReport is a report row returned by the label-reports query.
type LabelReport struct {
	Name         string  `json:"name"`
	Code         string  `json:"code"`
	TotalLabels  int64   `json:"total_labels"`
	TotalSpent   float64 `json:"total_spent"`
	InsuranceSpent float64 `json:"insurance_spent"`
	TotalRevenue *float64 `json:"total_revenue,omitempty"`
}

// WeeklyPerformance is one row from the print-performance / active-users queries.
type WeeklyPerformance struct {
	FirstDay      string `db:"first_day_of_week" json:"first_day_of_week"`
	LastDay       string `db:"last_day_of_week" json:"last_day_of_week"`
	YearWeek      int64  `db:"year_week" json:"year_week"`
	TotalRecords  int64  `db:"total_records" json:"total_records"`
	DistinctUsers *int64 `db:"distinct_users" json:"distinct_users,omitempty"`
}

// Country represents the countries table.
type Country struct {
	ID           uint   `db:"id" json:"id"`
	Name         string `db:"name" json:"name"`
	Code         string `db:"code" json:"code"`
	IsUSTerritory bool  `db:"is_us_territory" json:"is_us_territory"`
	Active        bool  `db:"active" json:"active"`
}

// Marketplace represents the marketplaces table.
type Marketplace struct {
	ID   uint   `db:"id" json:"id"`
	Name string `db:"name" json:"name"`
	Code string `db:"code" json:"code"`
}

// Affiliate is a user with role "rep" (user_role_id = 2).
// We reuse the User struct but provide helper methods.
type Affiliate = User

// AdminActivity is a note logged against a user.
type AdminActivity struct {
	ID        uint      `db:"id" json:"id"`
	Note      string    `db:"note" json:"note"`
	UserID    uint      `db:"user_id" json:"user_id"`
	AdminName string    `db:"admin_name" json:"admin_name"`
	Type      *string   `db:"type" json:"type"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// Pagination helpers
type PageResult[T any] struct {
	Items      []T   `json:"items"`
	TotalItems int64 `json:"total_items"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
}
