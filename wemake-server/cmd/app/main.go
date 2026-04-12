package main

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/yourusername/wemake/api"
	"github.com/yourusername/wemake/internal/config"
	"github.com/yourusername/wemake/internal/jobs"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Initialize configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	db, err := config.InitDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Start background jobs (expiration + auto-matching notifications)
	jobs.Start(db)

	// Initialize router and start server
	app := api.SetupRoutes(db, cfg)

	log.Printf("Starting server on port %s", cfg.Port)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
