package handlers

import (
	"github.com/danegigi/go-tut/internal/store"
)

// Deps holds all handler dependencies.
type Deps struct {
	AdminStore *store.AdminStore
	UserStore  *store.UserStore
	OrderStore *store.OrderStore
	LabelStore *store.LabelStore
}

// Handler is the top-level handler struct carrying all deps.
type Handler struct {
	Deps
}

func New(d Deps) *Handler {
	return &Handler{Deps: d}
}
