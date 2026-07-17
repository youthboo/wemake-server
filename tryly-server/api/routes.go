package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	recovermw "github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/config"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/logger"
	"github.com/yourusername/wemake/internal/middleware"
)

func SetupRoutes(db *sqlx.DB, cfg *config.Config) *fiber.App {
	app := fiber.New(fiber.Config{
		// SSE connections are long-lived; disable write timeout so they aren't killed.
		WriteTimeout: 0,
		ReadTimeout:  0,
		IdleTimeout:  0,
		// Raise header buffer to 16 KB to accommodate large Authorization / Cookie headers
		// (JWT tokens + session cookies can easily exceed the 4 KB default).
		ReadBufferSize: 16 * 1024,
	})
	// Recover middleware must be registered first so a panic in any downstream
	// handler or middleware is turned into a 500 instead of killing the process.
	app.Use(recovermw.New(recovermw.Config{EnableStackTrace: true}))

	logger.Info("Setting up CORS", "corsOrigins", cfg.CORSOrigins)

	// Fiber CORS middleware
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSOrigins,
		AllowMethods:     "GET,HEAD,PUT,PATCH,POST,DELETE,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization,X-User-ID,X-Confirm-Payment-Trigger",
		AllowCredentials: true,
		MaxAge:           3600,
	}))

	// Serve static files for media uploads
	app.Static("/uploads", "./uploads")

	h := newRouteHandlers(db, cfg)

	// Health check route
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
	app.Post("/internal/mail/send", h.mailRelay.Send)

	// Factory routes
	api := app.Group("/api/v1")
	api.Use(middleware.AuthContext(cfg.JWTSecret))

	admin := app.Group("/api/admin")
	// Admin routes must use a signed JWT — X-User-ID bypass is blocked here.
	admin.Use(middleware.RequireJWT(cfg.JWTSecret))
	admin.Get("/platform-config", middleware.RequireRole(h.authService, domain.RoleAccountManager, domain.RoleAdmin, domain.RoleSuperAdmin), h.platformConfig.GetActive)
	admin.Post("/platform-config", middleware.RequireRole(h.authService, domain.RoleSuperAdmin), h.platformConfig.Create)
	admin.Get("/platform-config/history", middleware.RequireRole(h.authService, domain.RoleAccountManager, domain.RoleAdmin, domain.RoleSuperAdmin), h.platformConfig.ListHistory)
	admin.Get("/tconfig", middleware.RequireRole(h.authService, domain.RoleAccountManager, domain.RoleAdmin, domain.RoleSuperAdmin), h.tconfig.GetAll)
	admin.Patch("/tconfig", middleware.RequireRole(h.authService, domain.RoleSuperAdmin), h.tconfig.PatchBulk)
	admin.Get("/cronjobs", middleware.RequireRole(h.authService, domain.RoleAdmin, domain.RoleSuperAdmin), h.tcronjob.ListJobs)
	admin.Patch("/cronjobs/:job_key", middleware.RequireRole(h.authService, domain.RoleSuperAdmin), h.tcronjob.UpdateJob)
	admin.Get("/platform-configs", middleware.RequireRole(h.authService, domain.RoleAccountManager, domain.RoleAdmin, domain.RoleSuperAdmin), h.platformConfig.ListAll)
	// static sub-path must come before /:config_id to avoid param capture
	admin.Get("/platform-configs/default-comm", middleware.RequireRole(h.authService, domain.RoleAccountManager, domain.RoleAdmin, domain.RoleSuperAdmin), h.platformConfig.GetDefaultComm)
	admin.Post("/platform-configs", middleware.RequireRole(h.authService, domain.RoleSuperAdmin), h.platformConfig.CreateConfig)
	admin.Delete("/platform-configs/:config_id", middleware.RequireRole(h.authService, domain.RoleSuperAdmin), h.platformConfig.DeleteConfig)
	admin.Patch("/platform-configs/:config_id", middleware.RequireRole(h.authService, domain.RoleSuperAdmin), h.platformConfig.UpdateConfig)
	admin.Get("/dashboard/summary", middleware.RequireRole(h.authService, domain.RoleAccountManager, domain.RoleAdmin, domain.RoleSuperAdmin), h.adminDashboard.GetSummary)
	admin.Get("/dashboard/revenue-chart", middleware.RequireRole(h.authService, domain.RoleAccountManager, domain.RoleAdmin, domain.RoleSuperAdmin), h.adminDashboard.GetRevenueChart)
	admin.Get("/dashboard/top-factories", middleware.RequireRole(h.authService, domain.RoleAccountManager, domain.RoleAdmin, domain.RoleSuperAdmin), h.adminDashboard.GetTopFactories)
	admin.Get("/factories", middleware.RequireRole(h.authService, domain.RoleAccountManager, domain.RoleAdmin, domain.RoleSuperAdmin), h.adminFactory.List)
	admin.Get("/factories/:factory_id", middleware.RequireRole(h.authService, domain.RoleAccountManager, domain.RoleAdmin, domain.RoleSuperAdmin), h.adminFactory.GetByID)
	admin.Get("/factories/:factory_id/config", middleware.RequireRole(h.authService, domain.RoleAccountManager, domain.RoleAdmin, domain.RoleSuperAdmin), h.adminFactory.GetFactoryConfig)
	admin.Patch("/factories/:factory_id/config", middleware.RequireRole(h.authService, domain.RoleAdmin, domain.RoleSuperAdmin), h.adminFactory.AssignFactoryConfig)
	admin.Post("/factories/:factory_id/approve", middleware.RequireRole(h.authService, domain.RoleAdmin, domain.RoleSuperAdmin), h.adminFactory.Approve)
	admin.Post("/factories/:factory_id/reject", middleware.RequireRole(h.authService, domain.RoleAdmin, domain.RoleSuperAdmin), h.adminFactory.Reject)
	admin.Post("/factories/:factory_id/suspend", middleware.RequireRole(h.authService, domain.RoleSuperAdmin), h.adminFactory.Suspend)
	admin.Post("/factories/:factory_id/unsuspend", middleware.RequireRole(h.authService, domain.RoleSuperAdmin), h.adminFactory.Unsuspend)
	admin.Get("/factory-verification", middleware.RequireRole(h.authService, domain.RoleAccountManager, domain.RoleAdmin, domain.RoleSuperAdmin), h.adminFactory.List)
	admin.Patch("/factories/:factory_id/verification", middleware.RequireRole(h.authService, domain.RoleSuperAdmin), h.adminFactory.PatchVerification)
	admin.Patch("/factories/:factory_id/certificates/:map_id", middleware.RequireRole(h.authService, domain.RoleAdmin, domain.RoleSuperAdmin), h.adminFactory.PatchCertificateStatus)
	admin.Get("/rfqs", middleware.RequireRole(h.authService, domain.RoleAccountManager, domain.RoleAdmin, domain.RoleSuperAdmin), h.adminRFQ.List)
	admin.Get("/rfqs/:rfq_id", middleware.RequireRole(h.authService, domain.RoleAccountManager, domain.RoleAdmin, domain.RoleSuperAdmin), h.adminRFQ.GetByID)
	admin.Patch("/rfqs/:rfq_id/status", middleware.RequireRole(h.authService, domain.RoleAdmin, domain.RoleSuperAdmin), h.adminRFQ.PatchStatus)
	admin.Get("/orders", middleware.RequireRole(h.authService, domain.RoleAccountManager, domain.RoleAdmin, domain.RoleSuperAdmin), h.adminOrder.List)
	admin.Get("/orders/:order_id", middleware.RequireRole(h.authService, domain.RoleAccountManager, domain.RoleAdmin, domain.RoleSuperAdmin), h.adminOrder.GetByID)
	admin.Patch("/orders/:order_id/status", middleware.RequireRole(h.authService, domain.RoleAdmin, domain.RoleSuperAdmin), h.adminOrder.PatchStatus)
	// Escrow mode: superadmin verifies customer payment slips (instead of factory)
	admin.Patch("/orders/:order_id/verify-slip", middleware.RequireRole(h.authService, domain.RoleSuperAdmin), h.slip.VerifySlipAdmin)
	admin.Get("/withdrawals", middleware.RequireRole(h.authService, domain.RoleAccountManager, domain.RoleAdmin, domain.RoleSuperAdmin), h.adminOrder.ListWithdrawals)
	admin.Patch("/withdrawals/:request_id", middleware.RequireRole(h.authService, domain.RoleAdmin, domain.RoleSuperAdmin), h.adminOrder.PatchWithdrawal)
	// Customer complaint / refund tickets — superadmin reviews & resolves.
	admin.Get("/disputes", middleware.RequireRole(h.authService, domain.RoleAccountManager, domain.RoleAdmin, domain.RoleSuperAdmin), h.adminOrder.ListDisputes)
	admin.Patch("/disputes/:dispute_id", middleware.RequireRole(h.authService, domain.RoleSuperAdmin), h.adminOrder.PatchDispute)
	admin.Get("/commission-rules", middleware.RequireRole(h.authService, domain.RoleAccountManager, domain.RoleAdmin, domain.RoleSuperAdmin), h.adminConfig.ListRules)
	admin.Post("/commission-rules", middleware.RequireRole(h.authService, domain.RoleAdmin, domain.RoleSuperAdmin), h.adminConfig.CreateRule)
	admin.Delete("/commission-rules/:rule_id", middleware.RequireRole(h.authService, domain.RoleAdmin, domain.RoleSuperAdmin), h.adminConfig.DeleteRule)
	admin.Get("/commission-exemptions", middleware.RequireRole(h.authService, domain.RoleAccountManager, domain.RoleAdmin, domain.RoleSuperAdmin), h.adminConfig.ListExemptions)
	admin.Post("/commission-exemptions", middleware.RequireRole(h.authService, domain.RoleSuperAdmin), h.adminConfig.CreateExemption)
	admin.Delete("/commission-exemptions/:exemption_id", middleware.RequireRole(h.authService, domain.RoleSuperAdmin), h.adminConfig.DeleteExemption)
	admin.Get("/audit-log", middleware.RequireRole(h.authService, domain.RoleAdmin, domain.RoleSuperAdmin), h.adminConfig.ListAuditLog)
	admin.Post("/users", middleware.RequireRole(h.authService, domain.RoleSuperAdmin), h.adminUser.Create)
	admin.Get("/users", middleware.RequireRole(h.authService, domain.RoleSuperAdmin), h.adminUser.List)

	// Customer admin (AD + SA)
	admin.Get("/customers", middleware.RequireRole(h.authService, domain.RoleAdmin, domain.RoleSuperAdmin), h.adminCustomer.ListCustomers)
	admin.Get("/customers/:user_id", middleware.RequireRole(h.authService, domain.RoleAdmin, domain.RoleSuperAdmin), h.adminCustomer.GetCustomerDetail)
	admin.Get("/customers/:user_id/wallet", middleware.RequireRole(h.authService, domain.RoleAdmin, domain.RoleSuperAdmin), h.adminCustomer.GetCustomerWallet)
	admin.Get("/customers/:user_id/orders", middleware.RequireRole(h.authService, domain.RoleAdmin, domain.RoleSuperAdmin), h.adminCustomer.ListCustomerOrders)

	// Dashboard additions
	admin.Get("/dashboard/top-customers", middleware.RequireRole(h.authService, domain.RoleAccountManager, domain.RoleAdmin, domain.RoleSuperAdmin), h.adminCustomer.ListTopCustomers)

	// Factory settlements
	admin.Get("/factories/:factory_id/settlements", middleware.RequireRole(h.authService, domain.RoleAdmin, domain.RoleSuperAdmin), h.adminCustomer.ListFactorySettlements)

	// Commission invoices (B6, B7, B8)
	admin.Patch("/categories/:id/img", middleware.RequireRole(h.authService, domain.RoleSuperAdmin), h.adminCatalog.PatchCategoryImg)
	admin.Patch("/hubs/:id/img", middleware.RequireRole(h.authService, domain.RoleSuperAdmin), h.adminCatalog.PatchHubImg)

	admin.Post("/invoices/generate", middleware.RequireRole(h.authService, domain.RoleSuperAdmin), h.adminCommission.GenerateInvoices)
	admin.Get("/invoices", middleware.RequireRole(h.authService, domain.RoleAdmin, domain.RoleSuperAdmin), h.adminCommission.ListInvoices)
	admin.Get("/invoices/:invoice_id", middleware.RequireRole(h.authService, domain.RoleAdmin, domain.RoleSuperAdmin), h.adminCommission.GetInvoice)
	admin.Patch("/invoices/:invoice_id/verify", middleware.RequireRole(h.authService, domain.RoleSuperAdmin), h.adminCommission.VerifyInvoiceSlip)
	admin.Post("/invoices/:invoice_id/send", middleware.RequireRole(h.authService, domain.RoleSuperAdmin), h.adminCommission.SendInvoice)
	admin.Post("/invoices/:invoice_id/resend", middleware.RequireRole(h.authService, domain.RoleSuperAdmin), h.adminCommission.ResendInvoice)
	admin.Get("/commission/summary", middleware.RequireRole(h.authService, domain.RoleAdmin, domain.RoleSuperAdmin), h.adminCommission.GetSummary)
	admin.Get("/commission/orders", middleware.RequireRole(h.authService, domain.RoleAccountManager, domain.RoleAdmin, domain.RoleSuperAdmin), h.adminCommission.ListEscrowCommissions)

	auth := api.Group("/auth")
	auth.Post("/register", h.auth.Register)
	auth.Post("/login", h.auth.Login)
	auth.Post("/forgot-password", h.auth.ForgotPassword)
	auth.Post("/reset-password", h.auth.ResetPassword)
	auth.Get("/email-check", h.auth.CheckEmail)
	auth.Post("/upgrade-to-factory", middleware.RequireAuth, h.auth.UpgradeToFactory)
	auth.Post("/upgrade-to-customer", middleware.RequireAuth, h.auth.UpgradeToCustomer)
	auth.Post("/switch-role", middleware.RequireAuth, h.auth.SwitchRole)
	auth.Get("/available-roles", middleware.RequireAuth, h.auth.AvailableRoles)

	api.Get("/explore", h.explore.GetExplore)
	api.Get("/categories", h.catalog.GetCategories)
	api.Get("/lbi/hubs", h.catalog.GetHubs)
	api.Get("/lbi/categories", h.catalog.GetLBICategories)
	api.Get("/lbi/sub-categories", h.catalog.GetAllLBISubCategories)
	api.Get("/categories/:id/sub-categories", h.catalog.GetSubCategories)
	api.Get("/units", h.catalog.GetUnits)
	api.Get("/hubs/showcases", h.catalog.GetHubShowcases)
	api.Get("/hubs/:hub_id/showcases", h.catalog.GetHubShowcases)

	api.Get("/factories", h.factory.List)
	api.Post("/factories", h.factory.Create)
	api.Get("/factories/me", h.factory.GetMe)
	api.Get("/factories/me/profile-init", h.profileInit.GetProfileInit)
	api.Get("/factories/me/portal", h.factory.GetPortal)
	api.Get("/factories/me/dashboard", h.factory.GetDashboard)
	api.Get("/factories/me/analytics", h.factory.GetAnalytics)

	// Factory bank accounts (B1)
	api.Get("/factories/me/bank-accounts", middleware.RequireRole(h.authService, domain.RoleFactory), h.bankAccount.List)
	api.Post("/factories/me/bank-accounts", middleware.RequireRole(h.authService, domain.RoleFactory), h.bankAccount.Create)
	api.Patch("/factories/me/bank-accounts/:account_id", middleware.RequireRole(h.authService, domain.RoleFactory), h.bankAccount.Update)
	api.Delete("/factories/me/bank-accounts/:account_id", middleware.RequireRole(h.authService, domain.RoleFactory), h.bankAccount.Delete)

	// Factory invoices (B7 — factory side)
	api.Get("/factories/me/reviews", middleware.RequireRole(h.authService, domain.RoleFactory), h.factoryReview.ListMyReviews)
	api.Post("/factories/me/reviews/:id/reply", middleware.RequireRole(h.authService, domain.RoleFactory), h.factoryReview.ReplyToReview)

	api.Get("/factories/me/invoices", middleware.RequireRole(h.authService, domain.RoleFactory), h.factoryInvoice.ListMyInvoices)
	api.Get("/factories/me/invoices/current-period", middleware.RequireRole(h.authService, domain.RoleFactory), h.factoryInvoice.GetCurrentPeriod)
	api.Get("/factories/me/invoices/:invoice_id", middleware.RequireRole(h.authService, domain.RoleFactory), h.factoryInvoice.GetMyInvoice)
	api.Post("/factories/me/invoices/:invoice_id/slip", middleware.RequireRole(h.authService, domain.RoleFactory), h.factoryInvoice.AttachCommSlip)

	factories := api.Group("/factories")
	factories.Get("/:factory_id/bank-account", h.bankAccount.GetPublicDefault) // public default account
	factories.Get("/:factory_id/categories", h.factory.ListCategories)
	factories.Post("/:factory_id/categories", h.factory.AddCategory)
	factories.Put("/:factory_id/categories", h.factory.ReplaceCategories)
	factories.Delete("/:factory_id/categories/:category_id", h.factory.RemoveCategory)
	factories.Get("/:factory_id/sub-categories", h.factory.ListSubCategories)
	factories.Post("/:factory_id/sub-categories", h.factory.AddSubCategory)
	factories.Put("/:factory_id/sub-categories", h.factory.ReplaceSubCategories)
	factories.Delete("/:factory_id/sub-categories/:sub_category_id", h.factory.RemoveSubCategory)
	factories.Get("/:factory_id/reviews/summary", h.review.GetSummaryByFactory)
	factories.Get("/:factory_id/reviews", h.review.ListByFactory)
	factories.Post("/:factory_id/reviews", h.review.Create)
	factories.Get("/:factory_id/certificates", h.certificate.ListByFactory)
	factories.Post("/:factory_id/certificates", h.certificate.Create)
	factories.Delete("/:factory_id/certificates/:map_id", h.certificate.Delete)
	factories.Patch("/:factory_id/certificates/:cert_id", h.certificate.PatchByCertID)
	factories.Delete("/:factory_id/certificates/by-cert/:cert_id", h.certificate.DeleteByCertID)
	factories.Get("/:factory_id/showcases", h.showcase.ListByFactory)
	factories.Put("/:factory_id/profile", h.factory.SaveProfile)
	factories.Patch("/:factory_id", h.factory.PatchProfile)
	factories.Put("/:factory_id", h.factory.PatchProfile)
	factories.Get("/:factory_id", h.factory.GetByID)

	addresses := api.Group("/addresses")
	addresses.Get("/", h.address.ListAddresses)
	addresses.Post("/", h.address.CreateAddress)
	addresses.Patch("/:address_id", h.address.PatchAddress)
	addresses.Delete("/:address_id", h.address.DeleteAddress)

	// Public configs — whitelisted tconfig keys (config_payment + บัญชี Tryly)
	api.Get("/configs/public", h.tconfig.GetPublic)

	// Factory verify slip (B4) — ปิดเมื่อ escrow mode (config_payment != 1)
	api.Patch("/factory/orders/:order_id/verify-slip", middleware.RequireRole(h.authService, domain.RoleFactory), h.slip.VerifySlip)

	api.Get("/factory/rfq-board", h.factoryRFQBoard.GetBoard)
	api.Get("/factory/rfqs/:rfq_id/detail", h.factoryRFQBoard.GetDetail)

	rfqs := api.Group("/rfqs")
	rfqs.Post("/", h.rfq.CreateRFQ)
	rfqs.Get("/preview-factories", h.rfq.PreviewFactories)
	rfqs.Get("/matching", h.rfq.ListMatching)
	rfqs.Get("/", h.rfq.ListRFQs)
	rfqs.Get("/:rfq_id", h.rfq.GetRFQ)
	rfqs.Get("/:rfq_id/detail", h.rfq.GetDetail)
	rfqs.Patch("/:rfq_id", h.rfq.PatchRFQ)
	rfqs.Patch("/:rfq_id/cancel", h.rfq.CancelRFQ)
	rfqs.Patch("/:rfq_id/close", h.rfq.CloseRFQ)
	rfqs.Put("/:rfq_id/targets", h.rfq.UpdateRFQTargets)
	rfqs.Post("/:rfq_id/bulk-checkout", h.order.BulkCheckout)
	rfqs.Post("/:rfq_id/quotations", middleware.RequireRole(h.authService, domain.RoleFactory), h.quotation.CreateQuotation)
	rfqs.Get("/:rfq_id/quotations", h.quotation.ListQuotationsByRFQ)

	factoryRFQs := api.Group("/factory/rfqs")
	factoryRFQs.Use(middleware.RequireRole(h.authService, domain.RoleFactory))
	factoryRFQs.Get("/", h.rfq.ListMatching)
	factoryRFQs.Get("/:rfq_id", h.rfq.GetRFQ)
	factoryRFQs.Post("/:rfq_id/dismiss", h.rfq.DismissRFQ)
	factoryRFQs.Delete("/:rfq_id/dismiss", h.rfq.UndismissRFQ)
	factoryRFQs.Get("/:rfq_id/quotations", h.quotation.ListQuotationsByRFQ)

	quotations := api.Group("/quotations")
	quotations.Post("/preview", h.quotation.Preview)
	quotations.Post("/", middleware.RequireRole(h.authService, domain.RoleFactory), h.quotation.CreateDetailed)
	quotations.Get("/", middleware.RequireRole(h.authService, domain.RoleFactory), h.quotation.ListCollection)
	quotations.Get("/me", middleware.RequireRole(h.authService, domain.RoleFactory), h.quotation.ListMine)
	quotations.Get("/:quotation_id/history", middleware.RequireAuth, h.quotation.ListHistory)
	quotations.Get("/:quotation_id", h.quotation.GetQuotation)
	quotations.Post("/:quotation_id/revision", middleware.RequireRole(h.authService, domain.RoleFactory), h.quotation.CreateRevision)
	quotations.Post("/:quotation_id/accept", h.quotation.Accept)
	quotations.Post("/:quotation_id/reject", h.quotation.Reject)
	quotations.Patch("/:quotation_id", middleware.RequireRole(h.authService, domain.RoleFactory), h.quotation.PatchQuotation)
	quotations.Patch("/:quotation_id/status", h.quotation.PatchQuotationStatus)
	quotations.Patch("/:quotation_id/factory-note", middleware.RequireRole(h.authService, domain.RoleFactory), h.quotation.PatchFactoryNote)

	wallets := api.Group("/wallets")
	wallets.Get("/me", h.wallet.GetMyWallet)
	wallets.Get("/me/transactions", h.wallet.ListMyTransactions)
	wallets.Post("/topup", h.topup.CreateIntent)
	wallets.Get("/topup/:intent_id", h.topup.GetIntent)
	wallets.Post("/topup/:intent_id/confirm", h.topup.ConfirmIntent)
	wallets.Post("/withdraw", h.withdrawal.Create)
	wallets.Get("/withdraw", h.withdrawal.List)
	wallets.Patch("/withdraw/:request_id/status", h.withdrawal.PatchStatus)

	orders := api.Group("/orders")
	orders.Post("/", h.order.CreateOrder)
	orders.Get("/", h.order.ListOrders)
	orders.Get("/:order_id/activity", h.order.ListActivity)
	orders.Get("/:order_id", h.order.GetOrder)
	// Payment slip (B2, B4)
	orders.Post("/:order_id/slip", h.slip.AttachSlip)
	orders.Get("/:order_id/slip", h.slip.GetSlip)
	orders.Get("/:order_id/review", h.order.GetReviewState)
	orders.Post("/:order_id/review", h.order.CreateReview)
	orders.Post("/:order_id/confirm-receipt", h.order.ConfirmReceipt)
	orders.Post("/:order_id/ship", h.order.MarkShipped)
	orders.Post("/:order_id/payments", h.orderPayment.PayDeposit)
	orders.Post("/:order_id/payments/:tx_id/verify", h.order.VerifyPayment)
	orders.Patch("/:order_id/status", h.order.PatchOrderStatus)
	orders.Patch("/:order_id/cancel", h.order.CancelOrder)
	// Customer opens / views a complaint (refund ticket) on their own order,
	// and (when asked) attaches return-shipping evidence.
	orders.Post("/:order_id/disputes", h.dispute.Create)
	orders.Get("/:order_id/disputes", h.dispute.GetByOrderID)
	orders.Post("/:order_id/disputes/return", h.dispute.SubmitReturn)
	orders.Get("/:order_id/payment-schedules", h.paymentSchedule.List)
	orders.Post("/:order_id/payment-schedules", h.paymentSchedule.Create)
	orders.Post("/:order_id/production-updates", h.production.CreateUpdate)
	orders.Get("/:order_id/production-updates", h.production.ListUpdates)

	production := api.Group("/production")
	production.Get("/steps", h.production.ListSteps)

	productionUpdates := api.Group("/production-updates")
	productionUpdates.Patch("/:update_id/reject", h.production.RejectUpdate)

	messages := api.Group("/messages")
	messages.Post("/", h.message.CreateMessage)
	messages.Get("/", h.message.ListMessages)
	messages.Get("/threads", h.message.ListThreads)

	transactions := api.Group("/transactions")
	transactions.Post("/", h.transaction.CreateTransaction)
	transactions.Get("/", h.transaction.ListTransactions)
	transactions.Patch("/:tx_id/status", h.transaction.PatchTransactionStatus)

	master := api.Group("/master")
	master.Get("/provinces", h.master.GetProvinces)
	master.Get("/districts", h.master.GetDistricts)
	master.Get("/sub-districts", h.master.GetSubDistricts)
	master.Get("/factory-types", h.master.GetFactoryTypes)
	master.Get("/categories", h.master.GetCategories)
	master.Get("/product-categories", h.master.GetProductCategories)
	master.Get("/production-steps", h.master.GetProductionSteps)
	master.Get("/units", h.master.GetUnits)
	master.Get("/shipping-methods", h.master.GetShippingMethods)
	master.Get("/certificates", h.master.GetCertificates)

	media := api.Group("/media")
	media.Post("/upload", h.media.UploadFile)

	conversations := api.Group("/conversations")
	conversations.Get("/", h.conversation.List)
	conversations.Get("/:conv_id", h.conversation.Get)
	conversations.Post("/", h.conversation.Create)
	conversations.Post("/:conv_id/boq", h.boq.Create)
	conversations.Post("/:conv_id/share-rfq", h.conversation.ShareRFQ)
	conversations.Patch("/:conv_id/read", h.conversation.MarkAsRead)
	conversations.Post("/:conv_id/mark-read", h.conversation.MarkAsRead) // FE compat
	conversations.Get("/:conv_id/messages", h.message.ListMessagesByConvPath)
	conversations.Post("/:conv_id/messages", h.message.CreateMessageByConvPath)

	notifications := api.Group("/notifications")
	notifications.Post("/stream-ticket", h.notification.IssueStreamTicket) // mint short-lived SSE ticket
	notifications.Get("/stream", h.notification.Stream)                    // SSE — must be before /:noti_id
	notifications.Get("/", h.notification.List)
	notifications.Get("/unread-count", h.notification.GetUnreadCount)
	notifications.Post("/read-all", h.notification.MarkAllRead)      // spec
	notifications.Put("/read-all", h.notification.MarkAllRead)       // compat
	notifications.Post("/:noti_id/read", h.notification.MarkAsRead)  // spec
	notifications.Patch("/:noti_id/read", h.notification.MarkAsRead) // compat
	notifications.Delete("/:noti_id", h.notification.SoftDelete)

	profile := api.Group("/profile")
	profile.Get("/", h.profile.GetProfile)
	profile.Put("/", h.profile.UpdateProfile)
	profile.Post("/avatar", h.profile.UploadAvatar)
	profile.Put("/change-password", h.profile.ChangePassword)
	profile.Get("/summary", h.profile.GetSummary)
	profile.Get("/transactions", h.profile.ListTransactions)
	profile.Get("/reviews", h.profile.ListMyReviews)
	profile.Get("/reviews/received", h.profile.ListReceivedReviews)
	profile.Get("/notification-preferences", h.profile.GetNotifPrefs)
	profile.Put("/notification-preferences", h.profile.UpdateNotifPrefs)

	reviews := api.Group("/reviews")
	reviews.Put("/:review_id", h.review.UpdateByUser)
	reviews.Delete("/:review_id", h.review.DeleteByUser)

	showcases := api.Group("/showcases")
	showcases.Get("/", h.showcase.List)
	showcases.Post("/", h.showcase.Create)
	showcases.Get("/promo-slides", h.showcase.ListPromoSlides)
	showcases.Post("/:showcase_id/inquire", h.conversation.InquireShowcase)
	showcases.Get("/:showcase_id/analytics", h.showcase.GetAnalytics)
	showcases.Get("/:showcase_id/images", h.showcase.ListImages)
	showcases.Post("/:showcase_id/images", h.showcase.CreateImage)
	showcases.Post("/:showcase_id/view", h.showcase.RecordView)
	showcases.Patch("/:showcase_id/status", h.showcase.PatchStatus)
	showcases.Put("/:showcase_id", h.showcase.Put)
	showcases.Patch("/:showcase_id", h.showcase.Patch)
	showcases.Delete("/:showcase_id", h.showcase.Delete)
	showcases.Get("/:showcase_id", h.showcase.GetDetail)

	promoSlides := api.Group("/promo-slides")
	promoSlides.Get("/", h.showcase.ListHomePromoSlides)

	favorites := api.Group("/favorites")
	favorites.Get("/", h.favorite.List)
	favorites.Post("/", h.favorite.Add)
	favorites.Delete("/:showcase_id", h.favorite.Remove)

	settlements := api.Group("/settlements")
	settlements.Get("/", h.settlement.List)
	settlements.Post("/", h.settlement.Create)
	settlements.Get("/:settlement_id", h.settlement.GetByID)
	settlements.Patch("/:settlement_id/status", h.settlement.PatchStatus)

	// Dispute resolution is superadmin-only (via /admin/disputes/:id).

	paymentSchedules := api.Group("/payment-schedules")
	paymentSchedules.Patch("/:schedule_id", h.paymentSchedule.PatchStatus)

	quotationTemplates := api.Group("/quotation-templates")
	quotationTemplates.Get("/", h.quotationTemplate.List)
	quotationTemplates.Post("/", h.quotationTemplate.Create)
	quotationTemplates.Patch("/:template_id", h.quotationTemplate.Patch)
	quotationTemplates.Delete("/:template_id", h.quotationTemplate.Delete)

	boqs := api.Group("/boq")
	boqs.Get("/", h.boq.ListMine)
	boqs.Get("/:rfq_id", h.boq.Get)
	boqs.Put("/:rfq_id", h.boq.Update)
	boqs.Post("/:rfq_id/accept", h.boq.Accept)
	boqs.Post("/:rfq_id/decline", h.boq.Decline)

	// Unified endpoints for the /orders page — reduces multiple API calls to one.
	me := api.Group("/me")
	me.Get("/session", middleware.RequireAuth, h.session.GetSession)
	me.Get("/rfq-orders", h.meRFQOrders.ListRFQOrders)
	me.Get("/rfq-orders/:rfq_id", h.meRFQOrders.GetRFQOrderDetail)

	frontend := api.Group("/frontend")
	frontend.Get("/bootstrap", h.frontend.GetBootstrap)
	frontend.Get("/mock-data", h.frontend.GetMockData)
	frontend.Get("/products", h.frontend.GetProducts)
	frontend.Get("/promotions", h.frontend.GetPromotions)
	frontend.Get("/promo-codes", h.frontend.GetPromoCodes)
	frontend.Get("/explore", h.frontend.GetExplore)
	frontend.Get("/me", h.frontend.GetCurrentUser)
	frontend.Get("/factories", h.frontend.ListFactories)
	frontend.Get("/factories/:factory_id", h.frontend.GetFactoryDetail)
	frontend.Get("/rfqs/:rfq_id", h.frontend.GetRFQDetail)
	frontend.Get("/orders/:order_id", h.frontend.GetOrderDetail)
	frontend.Get("/messages/threads", h.frontend.ListThreads)

	return app
}
