package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/config"
	"github.com/yourusername/wemake/internal/handler"
	"github.com/yourusername/wemake/internal/middleware"
	"github.com/yourusername/wemake/internal/repository"
	"github.com/yourusername/wemake/internal/service"
)

func SetupRoutes(db *sqlx.DB, cfg *config.Config) *fiber.App {
	app := fiber.New()
	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.CORSOrigins,
		AllowHeaders: "Origin, Content-Type, Accept, Authorization, X-User-ID",
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
	}))

	// Serve static files for media uploads
	app.Static("/uploads", "./uploads")

	// Initialize repositories
	factoryRepo := repository.NewFactoryRepository(db)
	authRepo := repository.NewAuthRepository(db)
	catalogRepo := repository.NewCatalogRepository(db)
	addressRepo := repository.NewAddressRepository(db)
	walletRepo := repository.NewWalletRepository(db)
	rfqRepo := repository.NewRFQRepository(db)
	quotationRepo := repository.NewQuotationRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	productionRepo := repository.NewProductionRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	masterRepo := repository.NewMasterRepository(db)
	transactionRepo := repository.NewTransactionRepository(db)
	frontendRepo := repository.NewFrontendRepository(db)

	// Initialize services
	factoryService := service.NewFactoryService(factoryRepo)
	authService := service.NewAuthService(authRepo, cfg.JWTSecret)
	catalogService := service.NewCatalogService(catalogRepo)
	addressService := service.NewAddressService(addressRepo)
	walletService := service.NewWalletService(walletRepo)
	rfqService := service.NewRFQService(rfqRepo)
	quotationService := service.NewQuotationService(quotationRepo)
	orderService := service.NewOrderService(orderRepo)
	productionService := service.NewProductionService(productionRepo)
	messageService := service.NewMessageService(messageRepo)
	masterService := service.NewMasterService(masterRepo)
	transactionService := service.NewTransactionService(transactionRepo)
	frontendService := service.NewFrontendService(frontendRepo)

	// Initialize handlers
	factoryHandler := handler.NewFactoryHandler(factoryService)
	authHandler := handler.NewAuthHandler(authService)
	catalogHandler := handler.NewCatalogHandler(catalogService)
	addressHandler := handler.NewAddressHandler(addressService)
	walletHandler := handler.NewWalletHandler(walletService)
	rfqHandler := handler.NewRFQHandler(rfqService)
	quotationHandler := handler.NewQuotationHandler(quotationService)
	orderHandler := handler.NewOrderHandler(orderService)
	productionHandler := handler.NewProductionHandler(productionService)
	messageHandler := handler.NewMessageHandler(messageService)
	masterHandler := handler.NewMasterHandler(masterService)
	transactionHandler := handler.NewTransactionHandler(transactionService)
	frontendHandler := handler.NewFrontendHandler(frontendService)
	mediaHandler := handler.NewMediaHandler()

	// Health check route
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// Factory routes
	api := app.Group("/api/v1")
	api.Use(middleware.AuthContext(cfg.JWTSecret))

	auth := api.Group("/auth")
	auth.Post("/register", authHandler.Register)
	auth.Post("/login", authHandler.Login)
	auth.Post("/forgot-password", authHandler.ForgotPassword)
	auth.Post("/reset-password", authHandler.ResetPassword)

	api.Get("/categories", catalogHandler.GetCategories)
	api.Get("/units", catalogHandler.GetUnits)

	factories := api.Group("/factories")
	factories.Post("/", factoryHandler.CreateFactory)
	factories.Get("/", factoryHandler.GetAllFactories)
	factories.Get("/:id", factoryHandler.GetFactory)
	factories.Patch("/:id", factoryHandler.UpdateFactory)
	factories.Delete("/:id", factoryHandler.DeleteFactory)

	addresses := api.Group("/addresses")
	addresses.Get("/", addressHandler.ListAddresses)
	addresses.Post("/", addressHandler.CreateAddress)
	addresses.Patch("/:address_id", addressHandler.PatchAddress)

	rfqs := api.Group("/rfqs")
	rfqs.Post("/", rfqHandler.CreateRFQ)
	rfqs.Get("/", rfqHandler.ListRFQs)
	rfqs.Get("/:rfq_id", rfqHandler.GetRFQ)
	rfqs.Post("/:rfq_id/images", rfqHandler.AddRFQImage)
	rfqs.Patch("/:rfq_id/cancel", rfqHandler.CancelRFQ)
	rfqs.Post("/:rfq_id/quotations", quotationHandler.CreateQuotation)
	rfqs.Get("/:rfq_id/quotations", quotationHandler.ListQuotationsByRFQ)

	quotations := api.Group("/quotations")
	quotations.Get("/:quotation_id", quotationHandler.GetQuotation)
	quotations.Patch("/:quotation_id/status", quotationHandler.PatchQuotationStatus)

	wallets := api.Group("/wallets")
	wallets.Get("/me", walletHandler.GetMyWallet)

	orders := api.Group("/orders")
	orders.Post("/", orderHandler.CreateOrder)
	orders.Get("/", orderHandler.ListOrders)
	orders.Get("/:order_id", orderHandler.GetOrder)
	orders.Patch("/:order_id/status", orderHandler.PatchOrderStatus)
	orders.Post("/:order_id/production-updates", productionHandler.CreateUpdate)
	orders.Get("/:order_id/production-updates", productionHandler.ListUpdates)

	productionUpdates := api.Group("/production-updates")
	productionUpdates.Patch("/:update_id", productionHandler.PatchUpdate)

	messages := api.Group("/messages")
	messages.Post("/", messageHandler.CreateMessage)
	messages.Get("/", messageHandler.ListMessages)
	messages.Get("/threads", messageHandler.ListThreads)

	transactions := api.Group("/transactions")
	transactions.Post("/", transactionHandler.CreateTransaction)
	transactions.Get("/", transactionHandler.ListTransactions)
	transactions.Patch("/:tx_id/status", transactionHandler.PatchTransactionStatus)

	master := api.Group("/master")
	master.Get("/provinces", masterHandler.GetProvinces)
	master.Get("/districts", masterHandler.GetDistricts)
	master.Get("/sub-districts", masterHandler.GetSubDistricts)
	master.Get("/factory-types", masterHandler.GetFactoryTypes)
	master.Get("/product-categories", masterHandler.GetProductCategories)
	master.Get("/production-steps", masterHandler.GetProductionSteps)
	master.Get("/units", masterHandler.GetUnits)
	master.Get("/shipping-methods", masterHandler.GetShippingMethods)

	media := api.Group("/media")
	media.Post("/upload", mediaHandler.UploadFile)

	frontend := api.Group("/frontend")
	frontend.Get("/bootstrap", frontendHandler.GetBootstrap)
	frontend.Get("/mock-data", frontendHandler.GetMockData)
	frontend.Get("/products", frontendHandler.GetProducts)
	frontend.Get("/promotions", frontendHandler.GetPromotions)
	frontend.Get("/promo-codes", frontendHandler.GetPromoCodes)
	frontend.Get("/explore", frontendHandler.GetExplore)
	frontend.Get("/me", frontendHandler.GetCurrentUser)
	frontend.Get("/factories", frontendHandler.ListFactories)
	frontend.Get("/factories/:factory_id", frontendHandler.GetFactoryDetail)
	frontend.Get("/rfqs/:rfq_id", frontendHandler.GetRFQDetail)
	frontend.Get("/orders/:order_id", frontendHandler.GetOrderDetail)
	frontend.Get("/messages/threads", frontendHandler.ListThreads)

	return app
}
