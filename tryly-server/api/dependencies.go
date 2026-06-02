package api

import (
	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/config"
	"github.com/yourusername/wemake/internal/handler"
	adminhandler "github.com/yourusername/wemake/internal/handler/admin"
	authhandler "github.com/yourusername/wemake/internal/handler/auth"
	boqhandler "github.com/yourusername/wemake/internal/handler/boq"
	mehandler "github.com/yourusername/wemake/internal/handler/me"
	cataloghandler "github.com/yourusername/wemake/internal/handler/catalog"
	conversationhandler "github.com/yourusername/wemake/internal/handler/conversation"
	factoryhandler "github.com/yourusername/wemake/internal/handler/factory"
	frontendhandler "github.com/yourusername/wemake/internal/handler/frontend"
	masterhandler "github.com/yourusername/wemake/internal/handler/master"
	messagehandler "github.com/yourusername/wemake/internal/handler/message"
	notificationhandler "github.com/yourusername/wemake/internal/handler/notification"
	orderhandler "github.com/yourusername/wemake/internal/handler/order"
	paymenthandler "github.com/yourusername/wemake/internal/handler/payment"
	platformconfighandler "github.com/yourusername/wemake/internal/handler/platform_config"
	tconfighandler "github.com/yourusername/wemake/internal/handler/tconfig"
	productionhandler "github.com/yourusername/wemake/internal/handler/production"
	profilehandler "github.com/yourusername/wemake/internal/handler/profile"
	quotationhandler "github.com/yourusername/wemake/internal/handler/quotation"
	rfqhandler "github.com/yourusername/wemake/internal/handler/rfq"
	showcasehandler "github.com/yourusername/wemake/internal/handler/showcase"
	userhandler "github.com/yourusername/wemake/internal/handler/user"
	wallethandler "github.com/yourusername/wemake/internal/handler/wallet"
	"github.com/yourusername/wemake/internal/logger"
	"github.com/yourusername/wemake/internal/media"
	"github.com/yourusername/wemake/internal/sse"
	adminrepo "github.com/yourusername/wemake/internal/repository/admin"
	authrepo "github.com/yourusername/wemake/internal/repository/auth"
	catalogrepo "github.com/yourusername/wemake/internal/repository/catalog"
	conversationrepo "github.com/yourusername/wemake/internal/repository/conversation"
	factoryrepo "github.com/yourusername/wemake/internal/repository/factory"
	frontendrepo "github.com/yourusername/wemake/internal/repository/frontend"
	masterrepo "github.com/yourusername/wemake/internal/repository/master"
	merepo "github.com/yourusername/wemake/internal/repository/me"
	messagerepo "github.com/yourusername/wemake/internal/repository/message"
	notificationrepo "github.com/yourusername/wemake/internal/repository/notification"
	orderrepo "github.com/yourusername/wemake/internal/repository/order"
	paymentrepo "github.com/yourusername/wemake/internal/repository/payment"
	platformrepo "github.com/yourusername/wemake/internal/repository/platform_config"
	tconfigrepo "github.com/yourusername/wemake/internal/repository/tconfig"
	productionrepo "github.com/yourusername/wemake/internal/repository/production"
	profilerepo "github.com/yourusername/wemake/internal/repository/profile"
	quotationrepo "github.com/yourusername/wemake/internal/repository/quotation"
	rfqrepo "github.com/yourusername/wemake/internal/repository/rfq"
	showcaserepo "github.com/yourusername/wemake/internal/repository/showcase"
	userrepo "github.com/yourusername/wemake/internal/repository/user"
	walletrepo "github.com/yourusername/wemake/internal/repository/wallet"
	adminservice "github.com/yourusername/wemake/internal/service/admin"
	authservice "github.com/yourusername/wemake/internal/service/auth"
	boqservice "github.com/yourusername/wemake/internal/service/boq"
	catalogservice "github.com/yourusername/wemake/internal/service/catalog"
	conversationservice "github.com/yourusername/wemake/internal/service/conversation"
	factoryservice "github.com/yourusername/wemake/internal/service/factory"
	frontendservice "github.com/yourusername/wemake/internal/service/frontend"
	meservice "github.com/yourusername/wemake/internal/service/me"
	masterservice "github.com/yourusername/wemake/internal/service/master"
	messageservice "github.com/yourusername/wemake/internal/service/message"
	notificationservice "github.com/yourusername/wemake/internal/service/notification"
	orderservice "github.com/yourusername/wemake/internal/service/order"
	paymentservice "github.com/yourusername/wemake/internal/service/payment"
	platformservice "github.com/yourusername/wemake/internal/service/platform_config"
	productionservice "github.com/yourusername/wemake/internal/service/production"
	profileservice "github.com/yourusername/wemake/internal/service/profile"
	quotationservice "github.com/yourusername/wemake/internal/service/quotation"
	rfqservice "github.com/yourusername/wemake/internal/service/rfq"
	showcaseservice "github.com/yourusername/wemake/internal/service/showcase"
	userservice "github.com/yourusername/wemake/internal/service/user"
	walletservice "github.com/yourusername/wemake/internal/service/wallet"
)

type routeHandlers struct {
	authService *authservice.AuthService

	auth              *authhandler.AuthHandler
	explore           *cataloghandler.ExploreHandler
	catalog           *cataloghandler.CatalogHandler
	address           *userhandler.AddressHandler
	wallet            *wallethandler.WalletHandler
	rfq               *rfqhandler.RFQHandler
	quotation         *quotationhandler.QuotationHandler
	order             *orderhandler.OrderHandler
	orderPayment      *paymenthandler.OrderPaymentHandler
	production        *productionhandler.ProductionHandler
	message           *messagehandler.MessageHandler
	master            *masterhandler.MasterHandler
	transaction       *wallethandler.TransactionHandler
	frontend          *frontendhandler.FrontendHandler
	session           *mehandler.SessionHandler
	media             *handler.MediaHandler
	review            *userhandler.ReviewHandler
	conversation      *conversationhandler.ConversationHandler
	notification      *notificationhandler.NotificationHandler
	showcase          *showcasehandler.ShowcaseHandler
	boq               *boqhandler.BOQHandler
	profile           *profilehandler.ProfileHandler
	factory           *factoryhandler.FactoryHandler
	profileInit       *factoryhandler.ProfileInitHandler
	favorite          *userhandler.FavoriteHandler
	certificate       *userhandler.CertificateHandler
	settlement        *wallethandler.SettlementHandler
	topup             *wallethandler.TopupHandler
	withdrawal        *wallethandler.WithdrawalHandler
	dispute           *wallethandler.DisputeHandler
	quotationTemplate *quotationhandler.QuotationTemplateHandler
	paymentSchedule   *paymenthandler.PaymentScheduleHandler
	platformConfig    *platformconfighandler.PlatformConfigHandler
	tconfig           *tconfighandler.TConfigHandler
	adminFactory      *adminhandler.AdminFactoryHandler
	adminDashboard    *adminhandler.AdminDashboardHandler
	adminRFQ          *adminhandler.AdminRFQHandler
	adminOrder        *adminhandler.AdminOrderHandler
	adminConfig       *adminhandler.AdminConfigHandler
	adminUser         *adminhandler.AdminUserHandler
	adminCustomer     *adminhandler.AdminCustomerHandler
	meRFQOrders       *mehandler.MeRFQOrdersHandler
	factoryRFQBoard   *rfqhandler.FactoryRFQBoardHandler
}

func newRouteHandlers(db *sqlx.DB, cfg *config.Config) *routeHandlers {
	authRepo := authrepo.NewAuthRepository(db)
	catalogRepo := catalogrepo.NewCatalogRepository(db)
	addressRepo := userrepo.NewAddressRepository(db)
	walletRepo := walletrepo.NewWalletRepository(db)
	rfqRepo := rfqrepo.NewRFQRepository(db)
	quotationRepo := quotationrepo.NewQuotationRepository(db)
	orderRepo := orderrepo.NewOrderRepository(db)
	productionRepo := productionrepo.NewProductionRepository(db)
	messageRepo := messagerepo.NewMessageRepository(db)
	masterRepo := masterrepo.NewMasterRepository(db)
	transactionRepo := walletrepo.NewTransactionRepository(db)
	frontendRepo := frontendrepo.NewFrontendRepository(db)
	sessionRepo := frontendrepo.NewSessionRepository(db)
	reviewRepo := userrepo.NewReviewRepository(db)
	conversationRepo := conversationrepo.NewConversationRepository(db)
	notificationRepo := notificationrepo.NewNotificationRepository(db)
	rfqItemRepo := rfqrepo.NewRFQItemRepository(db)
	profileRepo := profilerepo.NewProfileRepository(db)
	showcaseRepo := showcaserepo.NewShowcaseRepository(db)
	factoryRepo := factoryrepo.NewFactoryRepository(db)
	favoriteRepo := userrepo.NewFavoriteRepository(db)
	certificateRepo := userrepo.NewCertificateRepository(db)
	settlementRepo := walletrepo.NewSettlementRepository(db)
	topupRepo := walletrepo.NewTopupRepository(db)
	withdrawalRepo := walletrepo.NewWithdrawalRepository(db)
	disputeRepo := walletrepo.NewDisputeRepository(db)
	quotationTemplateRepo := quotationrepo.NewQuotationTemplateRepository(db)
	paymentScheduleRepo := paymentrepo.NewPaymentScheduleRepository(db)
	platformConfigRepo := platformrepo.NewPlatformConfigRepository(db)
	tconfigRepo := tconfigrepo.NewTConfigRepository(db)
	quotationItemRepo := quotationrepo.NewQuotationItemRepository(db)
	commissionRepo := walletrepo.NewCommissionRepository(db)
	adminAuditRepo := adminrepo.NewAdminAuditRepository(db)
	adminDashboardRepo := adminrepo.NewAdminDashboardRepository(db)
	adminFactoryRepo := adminrepo.NewAdminFactoryRepository(db, factoryRepo)
	adminOrderRepo := adminrepo.NewAdminOrderRepository(db)
	adminRFQRepo := adminrepo.NewAdminRFQRepository(db, rfqRepo)
	adminWithdrawalRepo := adminrepo.NewAdminWithdrawalRepository(db)
	adminDisputeRepo := adminrepo.NewAdminDisputeRepository(db)
	customerAdminRepo := adminrepo.NewCustomerAdminRepository(db)
	settlementAdminRepo := adminrepo.NewSettlementAdminRepository(db)

	sseHub := sse.NewHub()

	authService := authservice.NewAuthService(authRepo, cfg.JWTSecret)
	showcaseService := showcaseservice.NewShowcaseService(showcaseRepo, factoryRepo)
	catalogService := catalogservice.NewCatalogService(catalogRepo, showcaseService, frontendRepo)
	addressService := userservice.NewAddressService(addressRepo, factoryRepo)
	walletService := walletservice.NewWalletService(walletRepo, transactionRepo)
	notificationService := notificationservice.NewNotificationService(notificationRepo)
	rfqService := rfqservice.NewRFQService(rfqRepo, factoryRepo, notificationService)
	messageService := messageservice.NewMessageService(messageRepo, conversationRepo, quotationRepo, notificationService)
	orderService := orderservice.NewOrderService(db, orderRepo, paymentScheduleRepo, walletRepo, transactionRepo, quotationRepo, rfqRepo, reviewRepo, notificationService, messageService)
	commissionService := walletservice.NewCommissionService(platformConfigRepo, commissionRepo)
	platformConfigService := platformservice.NewPlatformConfigService(db, platformConfigRepo, adminAuditRepo)
	quotationService := quotationservice.NewQuotationService(db, quotationRepo, rfqRepo, quotationItemRepo, commissionService, orderService, factoryRepo, notificationService, messageService)
	orderPaymentService := paymentservice.NewOrderPaymentService(db)
	productionService := productionservice.NewProductionService(productionRepo)
	masterService := masterservice.NewMasterService(masterRepo)
	transactionService := walletservice.NewTransactionService(transactionRepo)
	frontendService := frontendservice.NewFrontendService(frontendRepo, sessionRepo, factoryRepo)
	reviewService := userservice.NewReviewService(reviewRepo)
	conversationService := conversationservice.NewConversationService(conversationRepo, rfqRepo, messageService, sseHub)
	boqService := boqservice.NewBOQService(db, conversationRepo, rfqRepo, rfqItemRepo, quotationRepo, quotationItemRepo, orderService, messageService, notificationService, commissionService)
	profileService := profileservice.NewProfileService(profileRepo, authRepo)
	factoryService := factoryservice.NewFactoryService(factoryRepo)
	profileInitService := factoryservice.NewProfileInitService(factoryService, masterService, catalogService, addressService)
	meRFQOrdersRepo := merepo.NewRFQOrdersRepository(db)
	meRFQOrdersService := meservice.NewRFQOrdersService(meRFQOrdersRepo)
	favoriteService := userservice.NewFavoriteService(favoriteRepo)
	certificateService := userservice.NewCertificateService(certificateRepo)
	settlementService := walletservice.NewSettlementService(settlementRepo)
	topupService := walletservice.NewTopupService(topupRepo, walletRepo)
	withdrawalService := walletservice.NewWithdrawalService(withdrawalRepo, walletRepo)
	disputeService := walletservice.NewDisputeService(disputeRepo)
	quotationTemplateService := quotationservice.NewQuotationTemplateService(quotationTemplateRepo)
	paymentScheduleService := paymentservice.NewPaymentScheduleService(paymentScheduleRepo)
	adminFactoryService := adminservice.NewAdminFactoryService(adminFactoryRepo, adminAuditRepo, commissionRepo, platformConfigRepo)
	adminDashboardService := adminservice.NewAdminDashboardService(adminDashboardRepo)

	cld, err := media.NewCloudinaryClient(cfg)
	if err != nil {
		logger.Warn("cloudinary disabled", "reason", "invalid configuration", "err", err)
		cld = nil
	}

	return &routeHandlers{
		authService:       authService,
		auth:              authhandler.NewAuthHandler(authService),
		explore:           cataloghandler.NewExploreHandler(catalogService),
		catalog:           cataloghandler.NewCatalogHandler(catalogService),
		address:           userhandler.NewAddressHandler(addressService),
		wallet:            wallethandler.NewWalletHandler(walletService),
		rfq:               rfqhandler.NewRFQHandler(rfqService, quotationService, authService),
		quotation:         quotationhandler.NewQuotationHandler(quotationService, authService),
		order:             orderhandler.NewOrderHandler(orderService, authService, productionService),
		orderPayment:      paymenthandler.NewOrderPaymentHandler(orderPaymentService),
		production:        productionhandler.NewProductionHandler(productionService),
		message:           messagehandler.NewMessageHandler(messageService, sseHub),
		master:            masterhandler.NewMasterHandler(masterService),
		transaction:       wallethandler.NewTransactionHandler(transactionService),
		frontend:          frontendhandler.NewFrontendHandler(frontendService),
		session:           mehandler.NewSessionHandler(frontendService),
		media:             handler.NewMediaHandler(cfg.PublicBaseURL, cld),
		review:            userhandler.NewReviewHandler(reviewService),
		conversation:      conversationhandler.NewConversationHandler(conversationService),
		notification:      notificationhandler.NewNotificationHandler(notificationService, sseHub),
		showcase:          showcasehandler.NewShowcaseHandler(showcaseService),
		boq:               boqhandler.NewBOQHandler(boqService),
		profile:           profilehandler.NewProfileHandler(profileService, cfg.PublicBaseURL, cld),
		factory:           factoryhandler.NewFactoryHandler(factoryService, authService),
		profileInit:       factoryhandler.NewProfileInitHandler(profileInitService, authService),
		favorite:          userhandler.NewFavoriteHandler(favoriteService),
		certificate:       userhandler.NewCertificateHandler(certificateService),
		settlement:        wallethandler.NewSettlementHandler(settlementService),
		topup:             wallethandler.NewTopupHandler(topupService),
		withdrawal:        wallethandler.NewWithdrawalHandler(withdrawalService),
		dispute:           wallethandler.NewDisputeHandler(disputeService),
		quotationTemplate: quotationhandler.NewQuotationTemplateHandler(quotationTemplateService),
		paymentSchedule:   paymenthandler.NewPaymentScheduleHandler(paymentScheduleService),
		platformConfig:    platformconfighandler.NewPlatformConfigHandler(platformConfigService, authService),
		tconfig:           tconfighandler.NewTConfigHandler(tconfigRepo),
		adminFactory:      adminhandler.NewAdminFactoryHandler(adminFactoryRepo, adminFactoryService),
		adminDashboard:    adminhandler.NewAdminDashboardHandler(adminDashboardService),
		adminRFQ:          adminhandler.NewAdminRFQHandler(adminRFQRepo, adminAuditRepo),
		adminOrder:        adminhandler.NewAdminOrderHandler(adminOrderRepo, orderService, withdrawalRepo, adminWithdrawalRepo, disputeRepo, adminDisputeRepo, adminAuditRepo),
		adminConfig:       adminhandler.NewAdminConfigHandler(commissionRepo, adminAuditRepo),
		adminUser:         adminhandler.NewAdminUserHandler(authService, authRepo),
		adminCustomer:     adminhandler.NewAdminCustomerHandler(customerAdminRepo, settlementAdminRepo),
		meRFQOrders:       mehandler.NewMeRFQOrdersHandler(meRFQOrdersService),
		factoryRFQBoard:   rfqhandler.NewFactoryRFQBoardHandler(rfqService, quotationService, authService, platformConfigRepo),
	}
}
