package main

import (
	"github.com/joho/godotenv"
	"github.com/yourusername/wemake/api"
	"github.com/yourusername/wemake/internal/config"
	"github.com/yourusername/wemake/internal/cron"
	"github.com/yourusername/wemake/internal/jobs"
	"github.com/yourusername/wemake/internal/mailer"
	adminrepo "github.com/yourusername/wemake/internal/repository/admin"
	"github.com/yourusername/wemake/internal/logger"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		logger.Warn("env file not loaded", "err", err)
	}

	// Initialize configuration
	logger.Info("loading configuration...")
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Fatal("failed to load config", "err", err)
	}
	logger.Info("configuration loaded", "database_url", cfg.DatabaseURL)

	// Initialize database
	logger.Info("initializing database...")
	db, err := config.InitDatabase(cfg)
	if err != nil {
		logger.Fatal("failed to initialize database", "err", err)
	}
	logger.Info("database initialized successfully")
	defer db.Close()

	// Start background jobs (expiration + auto-matching notifications)
	jobs.Start(db)

	// Start commission invoice cron (generates + emails on 1st of each month)
	mailSvc := mailer.New(db)
	commInvoiceRepo := adminrepo.NewCommissionInvoiceRepository(db)
	commCron := cron.NewCommissionCron(db, commInvoiceRepo, mailSvc)
	go commCron.Start()

	// Initialize router and start server
	app := api.SetupRoutes(db, cfg)

	logger.Info("starting server", "port", cfg.Port)
	if err := app.Listen(":" + cfg.Port); err != nil {
		logger.Fatal("server failed to start", "err", err, "port", cfg.Port)
	}
}
