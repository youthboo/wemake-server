package api

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/config"
	"github.com/yourusername/wemake/internal/handler"
	"github.com/yourusername/wemake/internal/media"
	"github.com/yourusername/wemake/internal/middleware"
	"github.com/yourusername/wemake/internal/repository"
	"github.com/yourusername/wemake/internal/service"
)

func SetupRoutes(db *sqlx.DB, cfg *config.Config) *fiber.App {
	app := fiber.New()
	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.CORSOrigins,
		AllowHeaders: "Origin, Content-Type, Accept, Authorization, X-User-ID, X-Confirm-Payment-Trigger",
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
	}))

	// Serve static files for media uploads
	app.Static("/uploads", "./uploads")

	// Initialize repositories
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
	reviewRepo := repository.NewReviewRepository(db)
	conversationRepo := repository.NewConversationRepository(db)
	notificationRepo := repository.NewNotificationRepository(db)
	showcaseRepo := repository.NewShowcaseRepository(db)
	factoryRepo := repository.NewFactoryRepository(db)
	favoriteRepo := repository.NewFavoriteRepository(db)
	certificateRepo := repository.NewCertificateRepository(db)
	settlementRepo := repository.NewSettlementRepository(db)
	topupRepo := repository.NewTopupRepository(db)
	withdrawalRepo := repository.NewWithdrawalRepository(db)
	disputeRepo := repository.NewDisputeRepository(db)
	quotationTemplateRepo := repository.NewQuotationTemplateRepository(db)
	paymentScheduleRepo := repository.NewPaymentScheduleRepository(db)

	// Initialize services
	authService := service.NewAuthService(authRepo, cfg.JWTSecret)
	catalogService := service.NewCatalogService(catalogRepo)
	addressService := service.NewAddressService(addressRepo)
	walletService := service.NewWalletService(walletRepo, transactionRepo)
	rfqService := service.NewRFQService(rfqRepo)
	quotationService := service.NewQuotationService(quotationRepo, rfqRepo)
	orderService := service.NewOrderService(db, orderRepo, paymentScheduleRepo, walletRepo, transactionRepo, quotationRepo, rfqRepo)
	orderPaymentService := service.NewOrderPaymentService(db)
	productionService := service.NewProductionService(productionRepo)
	messageService := service.NewMessageService(messageRepo, conversationRepo)
	masterService := service.NewMasterService(masterRepo)
	transactionService := service.NewTransactionService(transactionRepo)
	frontendService := service.NewFrontendService(frontendRepo)
	reviewService := service.NewReviewService(reviewRepo)
	conversationService := service.NewConversationService(conversationRepo)
	notificationService := service.NewNotificationService(notificationRepo)
	showcaseService := service.NewShowcaseService(showcaseRepo)
	factoryService := service.NewFactoryService(factoryRepo)
	favoriteService := service.NewFavoriteService(favoriteRepo)
	certificateService := service.NewCertificateService(certificateRepo)
	settlementService := service.NewSettlementService(settlementRepo)
	topupService := service.NewTopupService(topupRepo, walletRepo)
	withdrawalService := service.NewWithdrawalService(withdrawalRepo, walletRepo)
	disputeService := service.NewDisputeService(disputeRepo)
	quotationTemplateService := service.NewQuotationTemplateService(quotationTemplateRepo)
	paymentScheduleService := service.NewPaymentScheduleService(paymentScheduleRepo)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService)
	catalogHandler := handler.NewCatalogHandler(catalogService)
	addressHandler := handler.NewAddressHandler(addressService)
	walletHandler := handler.NewWalletHandler(walletService)
	rfqHandler := handler.NewRFQHandler(rfqService, authService)
	quotationHandler := handler.NewQuotationHandler(quotationService, authService)
	orderHandler := handler.NewOrderHandler(orderService, authService)
	orderPaymentHandler := handler.NewOrderPaymentHandler(orderPaymentService)
	productionHandler := handler.NewProductionHandler(productionService)
	messageHandler := handler.NewMessageHandler(messageService)
	masterHandler := handler.NewMasterHandler(masterService)
	transactionHandler := handler.NewTransactionHandler(transactionService)
	frontendHandler := handler.NewFrontendHandler(frontendService)
	cld, err := media.NewCloudinaryClient(cfg)
	if err != nil {
		log.Printf("cloudinary disabled: invalid configuration: %v", err)
		cld = nil
	}
	mediaHandler := handler.NewMediaHandler(cfg.PublicBaseURL, cld)
	reviewHandler := handler.NewReviewHandler(reviewService)
	conversationHandler := handler.NewConversationHandler(conversationService)
	notificationHandler := handler.NewNotificationHandler(notificationService)
	showcaseHandler := handler.NewShowcaseHandler(showcaseService)
	factoryHandler := handler.NewFactoryHandler(factoryService, authService)
	favoriteHandler := handler.NewFavoriteHandler(favoriteService)
	certificateHandler := handler.NewCertificateHandler(certificateService)
	settlementHandler := handler.NewSettlementHandler(settlementService)
	topupHandler := handler.NewTopupHandler(topupService)
	withdrawalHandler := handler.NewWithdrawalHandler(withdrawalService)
	disputeHandler := handler.NewDisputeHandler(disputeService)
	quotationTemplateHandler := handler.NewQuotationTemplateHandler(quotationTemplateService)
	paymentScheduleHandler := handler.NewPaymentScheduleHandler(paymentScheduleService)

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
	api.Get("/categories/:id/sub-categories", catalogHandler.GetSubCategories)
	api.Get("/units", catalogHandler.GetUnits)

	api.Get("/factories", factoryHandler.List)
	api.Get("/factories/me", factoryHandler.GetMe)
	api.Get("/factories/me/dashboard", factoryHandler.GetDashboard)
	api.Get("/factories/me/analytics", factoryHandler.GetAnalytics)

	factories := api.Group("/factories")
	factories.Get("/:factory_id/categories", factoryHandler.ListCategories)
	factories.Post("/:factory_id/categories", factoryHandler.AddCategory)
	factories.Put("/:factory_id/categories", factoryHandler.ReplaceCategories)
	factories.Delete("/:factory_id/categories/:category_id", factoryHandler.RemoveCategory)
	factories.Get("/:factory_id/sub-categories", factoryHandler.ListSubCategories)
	factories.Post("/:factory_id/sub-categories", factoryHandler.AddSubCategory)
	factories.Put("/:factory_id/sub-categories", factoryHandler.ReplaceSubCategories)
	factories.Delete("/:factory_id/sub-categories/:sub_category_id", factoryHandler.RemoveSubCategory)
	factories.Get("/:factory_id/reviews", reviewHandler.ListByFactory)
	factories.Post("/:factory_id/reviews", reviewHandler.Create)
	factories.Get("/:factory_id/certificates", certificateHandler.ListByFactory)
	factories.Post("/:factory_id/certificates", certificateHandler.Create)
	factories.Delete("/:factory_id/certificates/:map_id", certificateHandler.Delete)
	factories.Patch("/:factory_id/certificates/:cert_id", certificateHandler.PatchByCertID)
	factories.Delete("/:factory_id/certificates/by-cert/:cert_id", certificateHandler.DeleteByCertID)
	factories.Get("/:factory_id/showcases", showcaseHandler.ListByFactory)
	factories.Patch("/:factory_id", factoryHandler.PatchProfile)
	factories.Put("/:factory_id", factoryHandler.PatchProfile)
	factories.Get("/:factory_id", factoryHandler.GetByID)

	addresses := api.Group("/addresses")
	addresses.Get("/", addressHandler.ListAddresses)
	addresses.Post("/", addressHandler.CreateAddress)
	addresses.Patch("/:address_id", addressHandler.PatchAddress)
	addresses.Delete("/:address_id", addressHandler.DeleteAddress)

	rfqs := api.Group("/rfqs")
	rfqs.Post("/", rfqHandler.CreateRFQ)
	rfqs.Get("/matching", rfqHandler.ListMatching)
	rfqs.Get("/", rfqHandler.ListRFQs)
	rfqs.Get("/:rfq_id", rfqHandler.GetRFQ)
	rfqs.Patch("/:rfq_id/cancel", rfqHandler.CancelRFQ)
	rfqs.Post("/:rfq_id/quotations", quotationHandler.CreateQuotation)
	rfqs.Get("/:rfq_id/quotations", quotationHandler.ListQuotationsByRFQ)

	quotations := api.Group("/quotations")
	quotations.Get("/", quotationHandler.ListCollection)
	quotations.Get("/me", quotationHandler.ListMine)
	quotations.Get("/:quotation_id/history", quotationHandler.ListHistory)
	quotations.Get("/:quotation_id", quotationHandler.GetQuotation)
	quotations.Patch("/:quotation_id", quotationHandler.PatchQuotation)
	quotations.Patch("/:quotation_id/status", quotationHandler.PatchQuotationStatus)

	wallets := api.Group("/wallets")
	wallets.Get("/me", walletHandler.GetMyWallet)
	wallets.Get("/me/transactions", walletHandler.ListMyTransactions)
	wallets.Post("/topup", topupHandler.CreateIntent)
	wallets.Get("/topup/:intent_id", topupHandler.GetIntent)
	wallets.Post("/topup/:intent_id/confirm", topupHandler.ConfirmIntent)
	wallets.Post("/withdraw", withdrawalHandler.Create)
	wallets.Get("/withdraw", withdrawalHandler.List)
	wallets.Patch("/withdraw/:request_id/status", withdrawalHandler.PatchStatus)

	orders := api.Group("/orders")
	orders.Post("/", orderHandler.CreateOrder)
	orders.Get("/", orderHandler.ListOrders)
	orders.Get("/:order_id/activity", orderHandler.ListActivity)
	orders.Get("/:order_id", orderHandler.GetOrder)
	orders.Post("/:order_id/ship", orderHandler.MarkShipped)
	orders.Post("/:order_id/payments", orderPaymentHandler.PayDeposit)
	orders.Post("/:order_id/payments/:tx_id/verify", orderHandler.VerifyPayment)
	orders.Patch("/:order_id/status", orderHandler.PatchOrderStatus)
	orders.Patch("/:order_id/cancel", orderHandler.CancelOrder)
	orders.Post("/:order_id/disputes", disputeHandler.Create)
	orders.Get("/:order_id/disputes", disputeHandler.GetByOrderID)
	orders.Get("/:order_id/payment-schedules", paymentScheduleHandler.List)
	orders.Post("/:order_id/payment-schedules", paymentScheduleHandler.Create)
	orders.Post("/:order_id/production-updates", productionHandler.CreateUpdate)
	orders.Get("/:order_id/production-updates", productionHandler.ListUpdates)

	production := api.Group("/production")
	production.Get("/steps", productionHandler.ListSteps)

	productionUpdates := api.Group("/production-updates")
	productionUpdates.Patch("/:update_id/reject", productionHandler.RejectUpdate)

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
	master.Get("/categories", masterHandler.GetCategories)
	master.Get("/product-categories", masterHandler.GetProductCategories)
	master.Get("/production-steps", masterHandler.GetProductionSteps)
	master.Get("/units", masterHandler.GetUnits)
	master.Get("/shipping-methods", masterHandler.GetShippingMethods)
	master.Get("/certificates", masterHandler.GetCertificates)

	media := api.Group("/media")
	media.Post("/upload", mediaHandler.UploadFile)

	conversations := api.Group("/conversations")
	conversations.Get("/", conversationHandler.List)
	conversations.Get("/:conv_id", conversationHandler.Get)
	conversations.Post("/", conversationHandler.Create)
	conversations.Patch("/:conv_id/read", conversationHandler.MarkAsRead)

	notifications := api.Group("/notifications")
	notifications.Get("/", notificationHandler.List)
	notifications.Patch("/:noti_id/read", notificationHandler.MarkAsRead)

	showcases := api.Group("/showcases")
	showcases.Get("/", showcaseHandler.List)
	showcases.Post("/", showcaseHandler.Create)
	showcases.Get("/promo-slides", showcaseHandler.ListPromoSlides)
	showcases.Get("/:showcase_id/analytics", showcaseHandler.GetAnalytics)
	showcases.Post("/:showcase_id/view", showcaseHandler.RecordView)
	showcases.Patch("/:showcase_id/status", showcaseHandler.PatchStatus)
	showcases.Put("/:showcase_id", showcaseHandler.Put)
	showcases.Patch("/:showcase_id", showcaseHandler.Patch)
	showcases.Delete("/:showcase_id", showcaseHandler.Delete)
	showcases.Get("/:showcase_id", showcaseHandler.GetDetail)

	promoSlides := api.Group("/promo-slides")
	promoSlides.Get("/", showcaseHandler.ListPromoSlides)

	favorites := api.Group("/favorites")
	favorites.Get("/", favoriteHandler.List)
	favorites.Post("/", favoriteHandler.Add)
	favorites.Delete("/:showcase_id", favoriteHandler.Remove)

	settlements := api.Group("/settlements")
	settlements.Get("/", settlementHandler.List)
	settlements.Post("/", settlementHandler.Create)
	settlements.Get("/:settlement_id", settlementHandler.GetByID)
	settlements.Patch("/:settlement_id/status", settlementHandler.PatchStatus)

	disputes := api.Group("/disputes")
	disputes.Patch("/:dispute_id", disputeHandler.PatchStatus)

	paymentSchedules := api.Group("/payment-schedules")
	paymentSchedules.Patch("/:schedule_id", paymentScheduleHandler.PatchStatus)

	quotationTemplates := api.Group("/quotation-templates")
	quotationTemplates.Get("/", quotationTemplateHandler.List)
	quotationTemplates.Post("/", quotationTemplateHandler.Create)
	quotationTemplates.Patch("/:template_id", quotationTemplateHandler.Patch)
	quotationTemplates.Delete("/:template_id", quotationTemplateHandler.Delete)

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
