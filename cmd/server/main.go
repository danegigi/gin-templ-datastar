package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/grandshipper/admin-v2/internal/http/handlers"
	"github.com/grandshipper/admin-v2/internal/middleware"
	"github.com/grandshipper/admin-v2/internal/store"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env if present
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from environment")
	}

	// Initialize DB
	db, err := store.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Stores
	adminStore := store.NewAdminStore(db)
	userStore := store.NewUserStore(db)
	orderStore := store.NewOrderStore(db)
	labelStore := store.NewLabelStore(db)

	// Services (injected into handlers)
	h := handlers.New(handlers.Deps{
		AdminStore: adminStore,
		UserStore:  userStore,
		OrderStore: orderStore,
		LabelStore: labelStore,
	})

	r := gin.Default()
	r.SetTrustedProxies(nil) //nolint:errcheck

	// Serve static assets
	r.Static("/static", "./static")

	// Middleware
	r.Use(middleware.CORS())

	// Register all routes
	h.RegisterRoutes(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("admin-v2 listening on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
