package frontend

import (
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/yourusername/wemake/internal/dbutil"
	"github.com/yourusername/wemake/internal/domain"
	domainstatus "github.com/yourusername/wemake/internal/domain/status"
	"github.com/yourusername/wemake/internal/helper"
	"github.com/yourusername/wemake/internal/logger"
	factoryrepo "github.com/yourusername/wemake/internal/repository/factory"
	frontendrepo "github.com/yourusername/wemake/internal/repository/frontend"
)

type FrontendService struct {
	repo        *frontendrepo.FrontendRepository
	sessionRepo *frontendrepo.SessionRepository
	factoryRepo *factoryrepo.FactoryRepository
}

func NewFrontendService(repo *frontendrepo.FrontendRepository, sessionRepo *frontendrepo.SessionRepository, factoryRepo *factoryrepo.FactoryRepository) *FrontendService {
	return &FrontendService{repo: repo, sessionRepo: sessionRepo, factoryRepo: factoryRepo}
}

func (s *FrontendService) GetBootstrap(userID int64) (*domain.FrontendBootstrapResponse, error) {
	logger.Debug("building frontend bootstrap", "user_id", userID)

	if userID <= 0 {
		return &domain.FrontendBootstrapResponse{
			RFQs:    []domain.FrontendRFQCard{},
			Orders:  []domain.FrontendOrderCard{},
			Threads: []domain.FrontendMessageThread{},
		}, nil
	}

	var currentUser *domain.FrontendCurrentUser
	{
		cu, err := s.GetCurrentUser(userID)
		if err != nil {
			logger.Warn("frontend current user lookup failed", "user_id", userID, "err", err, "err_type", fmt.Sprintf("%T", err))
			if !dbutil.IsNotFoundError(err) {
				logger.Error("frontend current user lookup returned non-not-found error", "user_id", userID, "err", err)
				return nil, err
			}
		} else {
			logger.Debug("frontend current user loaded", "user_id", cu.ID, "name", cu.Name)
			currentUser = cu
		}
	}

	var rfqRows []frontendrepo.FrontendRFQRow
	var orderRows []frontendrepo.FrontendOrderRow
	var threadRows []frontendrepo.FrontendMessageThreadRow

	if rows, e := s.repo.ListRFQsByUserID(userID); e == nil {
		rfqRows = rows
	} else {
		logger.Warn("frontend rfqs query failed, continuing", "user_id", userID, "err", e)
	}
	if rows, e := s.repo.ListOrdersByUserID(userID); e == nil {
		orderRows = rows
	} else {
		logger.Warn("frontend orders query failed, continuing", "user_id", userID, "err", e)
	}
	if rows, e := s.repo.ListMessageThreads(userID); e == nil {
		threadRows = rows
	} else {
		logger.Warn("frontend message threads query failed, continuing", "user_id", userID, "err", e)
	}

	// Fetch user's favorites (showcase IDs) for heart icons on explore/factory-ideas cards.
	var favoriteIDs []int64
	if rows, e := s.repo.ListFavoriteShowcaseIDs(userID); e == nil {
		favoriteIDs = rows
	} else {
		logger.Warn("frontend favorites query failed, continuing", "user_id", userID, "err", e)
	}
	if favoriteIDs == nil {
		favoriteIDs = []int64{}
	}

	response := &domain.FrontendBootstrapResponse{
		CurrentUser: currentUser,
		RFQs:        make([]domain.FrontendRFQCard, 0, len(rfqRows)),
		Orders:      make([]domain.FrontendOrderCard, 0, len(orderRows)),
		Threads:     make([]domain.FrontendMessageThread, 0, len(threadRows)),
		Favorites:   favoriteIDs,
	}

	for _, item := range rfqRows {
		response.RFQs = append(response.RFQs, mapRFQCard(item))
	}
	for _, item := range orderRows {
		response.Orders = append(response.Orders, mapOrderCard(item))
	}

	threads, err := s.buildThreads(threadRows)
	if err != nil {
		return nil, err
	}
	response.Threads = threads

	return response, nil
}

func (s *FrontendService) GetSession(userID int64) (*domain.SessionResponse, error) {
	logger.Debug("building me session", "user_id", userID)

	userRow, err := s.sessionRepo.GetUser(userID)
	if err != nil {
		return nil, err
	}

	currentUser, wallet := mapSessionUser(userRow)

	var rfqRows []frontendrepo.SessionRFQRow
	if rows, e := s.sessionRepo.ListRFQsByUserID(userID); e == nil {
		rfqRows = rows
	} else {
		logger.Warn("session rfqs query failed, continuing", "user_id", userID, "err", e)
	}

	rfqIDs := make([]int64, 0, len(rfqRows))
	for _, row := range rfqRows {
		rfqIDs = append(rfqIDs, row.RFQID)
	}

	var offerRows []frontendrepo.SessionOfferRow
	if rows, e := s.sessionRepo.ListOffersForRFQs(userID, rfqIDs); e == nil {
		offerRows = rows
	} else {
		logger.Warn("session offers query failed, continuing", "user_id", userID, "err", e)
	}

	offersByRFQ := make(map[int64][]domain.SessionOffer)
	for _, row := range offerRows {
		offersByRFQ[row.RFQID] = append(offersByRFQ[row.RFQID], mapSessionOffer(row))
	}

	rfqs := make([]domain.SessionRFQ, 0, len(rfqRows))
	for _, row := range rfqRows {
		offers := offersByRFQ[row.RFQID]
		if offers == nil {
			offers = []domain.SessionOffer{}
		}
		rfqs = append(rfqs, mapSessionRFQ(row, offers))
	}

	var orderRows []frontendrepo.SessionOrderRow
	if rows, e := s.sessionRepo.ListOrdersByUserID(userID); e == nil {
		orderRows = rows
	} else {
		logger.Warn("session orders query failed, continuing", "user_id", userID, "err", e)
	}
	orders := make([]domain.SessionOrder, 0, len(orderRows))
	for _, row := range orderRows {
		orders = append(orders, mapSessionOrder(row))
	}

	var threadRows []frontendrepo.SessionThreadRow
	if rows, e := s.sessionRepo.ListThreadsByUserID(userID); e == nil {
		threadRows = rows
	} else {
		logger.Warn("session threads query failed, continuing", "user_id", userID, "err", e)
	}
	threads := make([]domain.SessionThread, 0, len(threadRows))
	for _, row := range threadRows {
		threads = append(threads, mapSessionThread(row))
	}

	var favorites []int64
	if rows, e := s.repo.ListFavoriteShowcaseIDs(userID); e == nil {
		favorites = rows
	} else {
		logger.Warn("session favorites query failed, continuing", "user_id", userID, "err", e)
	}
	if favorites == nil {
		favorites = []int64{}
	}

	return &domain.SessionResponse{
		CurrentUser: currentUser,
		Wallet:      wallet,
		RFQs:        rfqs,
		Orders:      orders,
		Threads:     threads,
		Favorites:   favorites,
	}, nil
}

func (s *FrontendService) GetCurrentUser(userID int64) (*domain.FrontendCurrentUser, error) {
	row, err := s.repo.GetCurrentUser(userID)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(strings.Join([]string{row.FirstName.String, row.LastName.String}, " "))
	if row.FactoryName.Valid {
		name = row.FactoryName.String
	}
	if name == "" {
		name = row.Email
	}

	company := ""
	if row.FactoryName.Valid {
		company = row.FactoryName.String
	}

	return &domain.FrontendCurrentUser{
		ID:             row.ID,
		Role:           row.Role,
		Name:           name,
		Company:        company,
		Email:          row.Email,
		Phone:          row.Phone.String,
		Avatar:         "",
		WalletBalance:  row.WalletBalance.Float64,
		PendingBalance: row.PendingBalance.Float64,
		MemberSince:    row.MemberSince,
	}, nil
}

func (s *FrontendService) ListFactories(scope ...string) ([]domain.FrontendFactoryCard, error) {
	rows, err := s.repo.ListFactories(scope...)
	if err != nil {
		return nil, err
	}
	items := make([]domain.FrontendFactoryCard, 0, len(rows))
	for _, item := range rows {
		items = append(items, mapFactoryCard(item))
	}
	return items, nil
}

func (s *FrontendService) GetFactoryDetail(factoryID int64) (*domain.FrontendFactoryDetail, error) {
	row, err := s.repo.GetFactoryDetail(factoryID)
	if err != nil {
		return nil, err
	}

	// Build full Thai address line so the FE can render it directly.
	// Bangkok uses แขวง/เขต prefixes; other provinces use ตำบล/อำเภอ.
	addressDetail := nullStr(row.AddressDetail)
	subDistrictName := nullStr(row.SubDistrictName)
	districtName := nullStr(row.DistrictName)
	provinceName := nullStr(row.ProvinceName)
	zipCode := nullStr(row.ZipCode)

	subDistrictLabel := "ตำบล"
	districtLabel := "อำเภอ"
	if strings.Contains(provinceName, "กรุงเทพ") {
		subDistrictLabel = "แขวง"
		districtLabel = "เขต"
	}
	addressParts := []string{}
	if addressDetail != "" {
		addressParts = append(addressParts, addressDetail)
	}
	if subDistrictName != "" {
		addressParts = append(addressParts, subDistrictLabel+subDistrictName)
	}
	if districtName != "" {
		addressParts = append(addressParts, districtLabel+districtName)
	}
	if provinceName != "" {
		addressParts = append(addressParts, "จ."+provinceName)
	}
	if zipCode != "" {
		addressParts = append(addressParts, zipCode)
	}

	// Reuse the same helpers used by /factories/:id and /factories/me so the
	// FE useFactoryProfile hook receives the identical sub-category contract
	// (each sub-category carries its parent `category_name`). Soft-fail with
	// empty arrays when joins return no rows — never block the whole detail.
	categories, _ := s.factoryRepo.ListFactoryCategories(factoryID)
	if categories == nil {
		categories = []domain.FactoryProfileCategory{}
	}
	subCategories, _ := s.factoryRepo.ListFactorySubCategories(factoryID)
	if subCategories == nil {
		subCategories = []domain.FactoryProfileSubCategory{}
	}

	// Fetch reviews from DB (soft-fail with empty slice)
	reviewRows, _ := s.repo.ListFactoryReviews(factoryID)
	reviews := make([]domain.FrontendFactoryReview, 0, len(reviewRows))
	for _, rr := range reviewRows {
		reviewer := "ลูกค้า"
		if rr.ReviewerName.Valid && rr.ReviewerName.String != "" {
			reviewer = rr.ReviewerName.String
		}
		imgURLs := rr.ImageURLs
		if imgURLs == nil {
			imgURLs = domain.StringArray{}
		}
		reviews = append(reviews, domain.FrontendFactoryReview{
			ID:        strconv.FormatInt(rr.ReviewID, 10),
			Reviewer:  reviewer,
			Rating:    rr.Rating,
			Comment:   rr.Comment,
			Date:      rr.CreatedAt,
			ImageURLs: imgURLs,
		})
	}

	return &domain.FrontendFactoryDetail{
		Factory: mapFactoryCard(frontendrepo.FrontendFactoryRow{
			ID:              row.ID,
			Name:            row.Name,
			Location:        row.Location,
			Specialization:  row.Specialization,
			Verified:        row.Verified,
			CompletedOrders: row.CompletedOrders,
			AverageLeadDays: row.AverageLeadDays,
			Description:     row.Description,
			Rating:          row.Rating,
			ReviewCount:     row.ReviewCount,
			LeadTimeDesc:    row.LeadTimeDesc,
			ImageURL:        row.ImageURL,
			PriceRange:      row.PriceRange,
		}),
		Profile: domain.FrontendFactoryProfile{
			Address:              strings.Join(addressParts, " "),
			AddressDetail:        addressDetail,
			SubDistrictName:      subDistrictName,
			DistrictName:         districtName,
			ProvinceName:         provinceName,
			ZipCode:              zipCode,
			AcceptedProductTypes: []string{},
			Certificates:         []string{},
		},
		Reviews:       reviews,
		Products:      []domain.FrontendShowcaseItem{},
		Promos:        []domain.FrontendShowcaseItem{},
		Ideas:         []domain.FrontendShowcaseItem{},
		Categories:    categories,
		SubCategories: subCategories,
	}, nil
}

func (s *FrontendService) GetRFQDetail(userID, rfqID int64) (*domain.FrontendRFQCard, error) {
	row, err := s.repo.GetRFQByUserID(userID, rfqID)
	if err != nil {
		return nil, err
	}

	offerRows, err := s.repo.ListQuotationsByRFQID(rfqID)
	if err != nil {
		return nil, err
	}
	imageRows, err := s.repo.ListRFQImages(rfqID)
	if err != nil {
		return nil, err
	}

	item := mapRFQCard(*row)
	item.Offers = make([]domain.FrontendQuotationCard, 0, len(offerRows))
	item.Images = make([]string, 0, len(imageRows))

	for index, offer := range offerRows {
		item.Offers = append(item.Offers, domain.FrontendQuotationCard{
			ID:              offer.ID,
			FactoryID:       offer.FactoryID,
			FactoryName:     offer.FactoryName,
			Price:           offer.TotalPrice,
			LeadTime:        offer.LeadTime,
			Verified:        offer.Verified,
			Recommended:     index == 0,
			CompletedOrders: offer.CompletedOrders,
			Status:          domainstatus.FrontendQuotation(offer.Status),
		})
	}
	for _, image := range imageRows {
		item.Images = append(item.Images, image.ImageURL)
	}

	return &item, nil
}

func (s *FrontendService) GetOrderDetail(userID, orderID int64) (*domain.FrontendOrderDetail, error) {
	row, err := s.repo.GetOrderByUserID(userID, orderID)
	if err != nil {
		return nil, err
	}

	timelineRows, err := s.repo.ListOrderTimeline(orderID)
	if err != nil {
		return nil, err
	}

	detail := &domain.FrontendOrderDetail{
		Order:    mapOrderCard(*row),
		Timeline: make([]domain.FrontendOrderTimelineItem, 0, len(timelineRows)),
	}

	lastIndex := len(timelineRows) - 1
	for index, item := range timelineRows {
		status := "upcoming"
		if index < lastIndex {
			status = "completed"
		}
		if index == lastIndex {
			status = "current"
		}
		detail.Timeline = append(detail.Timeline, domain.FrontendOrderTimelineItem{
			ID:          item.ID,
			Title:       fallbackString(item.Title.String, "Production Update"),
			Date:        item.Date,
			Status:      status,
			Description: item.Description.String,
			Photo:       item.Photo.String,
		})
	}

	return detail, nil
}

func (s *FrontendService) ListThreads(userID int64) ([]domain.FrontendMessageThread, error) {
	rows, err := s.repo.ListMessageThreads(userID)
	if err != nil {
		return nil, err
	}
	return s.buildThreads(rows)
}

func (s *FrontendService) GetMockData(userID int64) (*domain.FrontendMockDataResponse, error) {
	currentUserRow, err := s.repo.GetCurrentUser(userID)
	if err != nil {
		return nil, err
	}
	categoryRows, err := s.repo.ListCategories()
	if err != nil {
		return nil, err
	}
	factoryRows, err := s.repo.ListFactories()
	if err != nil {
		return nil, err
	}
	rfqRows, err := s.repo.ListRFQsByUserID(userID)
	if err != nil {
		return nil, err
	}
	orderRows, err := s.repo.ListOrdersByUserID(userID)
	if err != nil {
		return nil, err
	}
	threadRows, err := s.repo.ListMessageThreads(userID)
	if err != nil {
		return nil, err
	}

	currentUser := mapMockCurrentUser(currentUserRow)
	categories := make([]domain.MockCategory, 0, len(categoryRows))
	categoryNameToIcon := map[string]string{}
	for _, item := range categoryRows {
		mockCategory := mapMockCategory(item)
		categories = append(categories, mockCategory)
		categoryNameToIcon[mockCategory.Name] = mockCategory.Icon
	}

	factories := make([]domain.MockFactory, 0, len(factoryRows))
	factoryProfiles := make([]domain.MockFactoryProfile, 0, len(factoryRows))
	factoryReviews := make([]domain.MockFactoryReview, 0, len(factoryRows))
	ideaArticles := make([]domain.MockIdeaArticle, 0, len(factoryRows))
	showcases := make([]domain.MockShowcase, 0, len(factoryRows)*2)
	factoryMap := map[int64]domain.MockFactory{}

	for index, row := range factoryRows {
		factory := mapMockFactory(row)
		factoryID := helper.ParseFactoryID(factory.ID)
		factories = append(factories, factory)
		factoryMap[factoryID] = factory

		detailRow, detailErr := s.repo.GetFactoryDetail(factoryID)
		if detailErr == nil {
			factoryProfiles = append(factoryProfiles, buildMockFactoryProfile(*detailRow))
		} else {
			factoryProfiles = append(factoryProfiles, domain.MockFactoryProfile{
				FactoryID:            factory.ID,
				Address:              factory.Location,
				AcceptedProductTypes: []string{},
				Certificates:         []string{},
			})
		}

		factoryReviews = append(factoryReviews, buildMockFactoryReview(factory, index))
		ideaArticles = append(ideaArticles, buildMockIdeaArticle(factory, index))
		showcases = append(showcases, buildMockShowcases(factory, index)...)
	}

	sort.Slice(showcases, func(i, j int) bool { return showcases[i].PostedAt > showcases[j].PostedAt })
	sort.Slice(ideaArticles, func(i, j int) bool { return ideaArticles[i].PublishedAt > ideaArticles[j].PublishedAt })

	rfqs := make([]domain.MockRFQ, 0, len(rfqRows))
	rfqMap := map[int64]domain.MockRFQ{}
	for _, row := range rfqRows {
		item, buildErr := s.buildMockRFQ(row, categoryNameToIcon)
		if buildErr != nil {
			return nil, buildErr
		}
		rfqs = append(rfqs, item)
		rfqMap[row.ID] = item
	}

	orders := make([]domain.MockOrder, 0, len(orderRows))
	orderMap := map[int64]domain.MockOrder{}
	for _, row := range orderRows {
		item, buildErr := s.buildMockOrder(row, rfqMap[row.RFQID])
		if buildErr != nil {
			return nil, buildErr
		}
		orders = append(orders, item)
		orderMap[row.ID] = item
	}

	conversations := make([]domain.MockConversation, 0, len(threadRows))
	for index, thread := range threadRows {
		item, buildErr := s.buildMockConversation(userID, thread, rfqMap, factoryMap, index)
		if buildErr != nil {
			return nil, buildErr
		}
		conversations = append(conversations, item)
	}

	notifications := buildMockNotifications(rfqs, orders, conversations)

	return &domain.FrontendMockDataResponse{
		CurrentUser:      currentUser,
		Categories:       categories,
		Factories:        factories,
		FactoryProfiles:  factoryProfiles,
		FactoryReviews:   factoryReviews,
		IdeaArticles:     ideaArticles,
		FactoryShowcases: showcases,
		RFQs:             rfqs,
		Orders:           orders,
		Conversations:    conversations,
		Notifications:    notifications,
	}, nil
}

func (s *FrontendService) buildThreads(rows []frontendrepo.FrontendMessageThreadRow) ([]domain.FrontendMessageThread, error) {
	logger.Debug("building frontend message threads", "count", len(rows))
	items := make([]domain.FrontendMessageThread, 0, len(rows))
	for idx, item := range rows {
		logger.Debug("processing frontend message thread",
			"row_index", idx,
			"counterpart_id", item.CounterpartID,
			"reference_type", item.ReferenceType,
			"reference_id", item.ReferenceID,
		)

		// Graceful handling: use default values if data not found
		counterpartName := fmt.Sprintf("User %d", item.CounterpartID)
		userLabel, err := s.repo.GetUserLabel(item.CounterpartID)
		if err == nil {
			counterpartName = userLabel.Name
		} else {
			logger.Warn("frontend message thread user label lookup failed, using default", "counterpart_id", item.CounterpartID, "err", err)
		}

		projectName := item.ReferenceType
		hasQuote := false
		reference, err := s.repo.GetReferenceLabel(item.ReferenceType, item.ReferenceID)
		if err == nil {
			projectName = reference.ProjectName
			hasQuote = reference.HasQuote
		} else {
			logger.Warn("frontend message thread reference label lookup failed, using default",
				"reference_type", item.ReferenceType,
				"reference_id", item.ReferenceID,
				"err", err,
			)
		}

		items = append(items, domain.FrontendMessageThread{
			ReferenceType: item.ReferenceType,
			ReferenceID:   fmt.Sprintf("%d", item.ReferenceID),
			CounterpartID: item.CounterpartID,
			Counterpart:   counterpartName,
			ProjectName:   projectName,
			LastMessage:   item.LastMessage,
			LastMessageAt: item.LastMessageAt,
			Unread:        0,
			HasQuote:      hasQuote,
			Avatar:        "",
		})
	}
	return items, nil
}

func mapFactoryCard(row frontendrepo.FrontendFactoryRow) domain.FrontendFactoryCard {
	specialization := row.Specialization.String
	leadTime := strings.TrimSpace(row.LeadTimeDesc.String)
	if leadTime == "" {
		leadDays := int64(row.AverageLeadDays.Float64 + 0.5)
		if leadDays > 0 {
			leadTime = fmt.Sprintf("%d วัน", leadDays)
		}
	}

	tags := []string{}
	if specialization != "" {
		tags = append(tags, specialization)
	}
	if row.Verified {
		tags = append(tags, "Verified")
	}

	categoryScopes := []string{}
	if len(row.CategoryScopes) > 0 {
		categoryScopes = []string(row.CategoryScopes)
	}

	return domain.FrontendFactoryCard{
		ID:              row.ID,
		Name:            row.Name,
		Location:        row.Location.String,
		Rating:          row.Rating,
		Reviews:         row.ReviewCount,
		Specialization:  specialization,
		Tags:            tags,
		LeadTime:        leadTime,
		Image:           row.ImageURL.String,
		Verified:        row.Verified,
		CompletedOrders: row.CompletedOrders,
		PriceRange:      row.PriceRange.String,
		Description:     row.Description.String,
		CategoryScopes:  categoryScopes,
	}
}

func mapSessionUser(row *frontendrepo.SessionUserRow) (*domain.SessionCurrentUser, *domain.SessionWallet) {
	name := strings.TrimSpace(strings.Join([]string{
		row.FirstName.String, row.LastName.String,
	}, " "))
	if row.FactoryName.Valid && row.FactoryName.String != "" {
		name = row.FactoryName.String
	}
	if name == "" {
		name = row.Email
	}

	return &domain.SessionCurrentUser{
			ID:          row.ID,
			Role:        row.Role,
			Name:        name,
			Email:       row.Email,
			Phone:       row.Phone.String,
			MemberSince: row.MemberSince,
		}, &domain.SessionWallet{
			Balance:        row.Balance,
			PendingBalance: row.PendingBalance,
		}
}

func mapSessionOffer(row frontendrepo.SessionOfferRow) domain.SessionOffer {
	offer := domain.SessionOffer{
		QuoteID:         row.QuoteID,
		FactoryID:       row.FactoryID,
		FactoryName:     row.FactoryName,
		FactoryVerified: row.FactoryVerified,
		Status:          row.Status,
	}
	if row.FactoryImageURL.Valid {
		s := row.FactoryImageURL.String
		offer.FactoryImageURL = &s
	}
	if row.PricePerPiece.Valid {
		v := row.PricePerPiece.Float64
		offer.PricePerPiece = &v
	}
	if row.GrandTotal.Valid {
		v := row.GrandTotal.Float64
		offer.GrandTotal = &v
	}
	if row.LeadTimeDays.Valid {
		v := row.LeadTimeDays.Int64
		offer.LeadTimeDays = &v
	}
	return offer
}

func mapSessionRFQ(row frontendrepo.SessionRFQRow, offers []domain.SessionOffer) domain.SessionRFQ {
	rfq := domain.SessionRFQ{
		RFQID:        row.RFQID,
		Title:        row.Title,
		Status:       row.Status,
		CategoryName: row.CategoryName,
		CreatedAt:    row.CreatedAt,
		OfferCount:   row.OfferCount,
		Offers:       offers,
	}
	if row.Quantity.Valid {
		v := row.Quantity.Int64
		rfq.Quantity = &v
	}
	if row.TargetPrice.Valid {
		v := row.TargetPrice.Float64
		rfq.TargetPrice = &v
	}
	return rfq
}

func mapSessionOrder(row frontendrepo.SessionOrderRow) domain.SessionOrder {
	order := domain.SessionOrder{
		OrderID:     row.OrderID,
		Title:       row.Title,
		FactoryID:   row.FactoryID,
		FactoryName: row.FactoryName,
		Status:      row.Status,
		CreatedAt:   row.CreatedAt,
	}
	if row.FactoryImageURL.Valid {
		s := row.FactoryImageURL.String
		order.FactoryImageURL = &s
	}
	if row.TotalAmount.Valid {
		v := row.TotalAmount.Float64
		order.TotalAmount = &v
	}
	if row.DepositAmount.Valid {
		v := row.DepositAmount.Float64
		order.DepositAmount = &v
	}
	if row.EstimatedDelivery.Valid {
		v := row.EstimatedDelivery.String
		order.EstimatedDelivery = &v
	}
	return order
}

func mapSessionThread(row frontendrepo.SessionThreadRow) domain.SessionThread {
	thread := domain.SessionThread{
		ConvID:          row.ConvID,
		CounterpartID:   row.CounterpartID,
		CounterpartName: row.CounterpartName,
		LastMessage:     row.LastMessage.String,
		LastMessageAt:   row.LastMessageAt,
		Unread:          row.Unread,
	}
	if row.CounterpartImageURL.Valid {
		s := row.CounterpartImageURL.String
		thread.CounterpartImageURL = &s
	}
	return thread
}

func mapRFQCard(row frontendrepo.FrontendRFQRow) domain.FrontendRFQCard {
	return domain.FrontendRFQCard{
		ID:            row.ID,
		ProjectName:   row.ProjectName,
		Category:      row.Category,
		Status:        domainstatus.FrontendRFQ(row.Status, row.OfferCount),
		OfferCount:    row.OfferCount,
		AcceptedCount: row.AcceptedCount,
		PendingCount:  row.PendingCount,
		Budget:        row.Budget,
		Quantity:      row.Quantity,
		CreatedAt:     row.CreatedAt,
		Description:   row.Description,
	}
}

func mapOrderCard(row frontendrepo.FrontendOrderRow) domain.FrontendOrderCard {
	return domain.FrontendOrderCard{
		ID:                row.ID,
		ProjectName:       row.ProjectName,
		RFQID:             row.RFQID,
		FactoryID:         row.FactoryID,
		FactoryName:       row.FactoryName,
		TotalAmount:       row.TotalAmount,
		DepositPaid:       row.DepositPaid,
		Status:            domainstatus.FrontendOrder(row.Status),
		EstimatedDelivery: row.EstimatedDelivery,
		CreatedAt:         row.CreatedAt,
		CurrentStepID:     row.CurrentStepID,
	}
}

func fallbackString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

// nullStr extracts the string value from a sql.NullString, trimming whitespace.
// Returns "" when the value is NULL or empty.
func nullStr(s sql.NullString) string {
	if !s.Valid {
		return ""
	}
	return strings.TrimSpace(s.String)
}

func mapMockCurrentUser(row *frontendrepo.FrontendCurrentUserRow) *domain.MockCurrentUser {
	name := strings.TrimSpace(strings.Join([]string{row.FirstName.String, row.LastName.String}, " "))
	if name == "" {
		name = row.FactoryName.String
	}
	if name == "" {
		name = row.Email
	}

	company := row.FactoryName.String
	if company == "" {
		company = "Wemake Member"
	}

	return &domain.MockCurrentUser{
		ID:             "u" + strconv.FormatInt(row.ID, 10),
		Name:           name,
		NameEn:         name,
		Avatar:         helper.AvatarURL(name),
		Company:        company,
		Email:          row.Email,
		Phone:          row.Phone.String,
		WalletBalance:  row.WalletBalance.Float64,
		PendingBalance: row.PendingBalance.Float64,
		MemberSince:    row.MemberSince,
	}
}

func mapMockCategory(row frontendrepo.FrontendCategoryRow) domain.MockCategory {
	iconMap := map[string]string{
		"อาหารสัตว์":          "🐾",
		"อาหารเสริม":          "💊",
		"ของเล่นสัตว์เลี้ยง":  "🎾",
		"สายจูง อุปกรณ์":      "🦮",
		"เสื้อผ้าสัตว์เลี้ยง": "👕",
		"อุปกรณ์สัตว์เลี้ยง":  "🦮",
		"แพ็กเกจจิ้ง":         "📦",
	}
	colorMap := map[string]string{
		"อาหารสัตว์":          "#3B82F6",
		"อาหารเสริม":          "#8B5CF6",
		"ของเล่นสัตว์เลี้ยง":  "#22C55E",
		"สายจูง อุปกรณ์":      "#F59E0B",
		"เสื้อผ้าสัตว์เลี้ยง": "#EC4899",
		"อุปกรณ์สัตว์เลี้ยง":  "#F59E0B",
	}

	id := helper.SlugifyCategory(row.Name)
	icon := iconMap[row.Name]
	if icon == "" {
		icon = "📦"
	}
	color := colorMap[row.Name]
	if color == "" {
		color = "#6B7280"
	}

	return domain.MockCategory{
		ID:    id,
		Name:  row.Name,
		Icon:  icon,
		Color: color,
	}
}

func mapMockFactory(row frontendrepo.FrontendFactoryRow) domain.MockFactory {
	rating := 4.6 + (float64(row.ID%4) * 0.1)
	reviews := row.CompletedOrders/2 + 24
	priceRanges := []string{"฿", "฿฿", "฿฿฿"}
	priceRange := priceRanges[row.ID%int64(len(priceRanges))]
	tags := append([]string{}, row.Description.String)
	tags = []string{}
	if row.Specialization.Valid && row.Specialization.String != "" {
		tags = append(tags, row.Specialization.String)
	}
	if row.Verified {
		tags = append(tags, "Verified")
	}
	if len(tags) == 0 {
		tags = append(tags, "OEM")
	}

	return domain.MockFactory{
		ID:              fmt.Sprintf("f%d", row.ID),
		Name:            row.Name,
		Location:        fallbackString(row.Location.String, "กรุงเทพฯ"),
		Rating:          rating,
		Reviews:         reviews,
		Specialization:  fallbackString(row.Specialization.String, "โรงงานรับผลิตสินค้า"),
		Tags:            tags,
		LeadTime:        fallbackString(helper.FormatLeadTimeRange(row.AverageLeadDays.Float64), "7-14 วัน"),
		Image:           helper.FactoryImageURL(row.ID),
		Verified:        row.Verified,
		CompletedOrders: row.CompletedOrders,
		PriceRange:      priceRange,
	}
}

func buildMockFactoryProfile(row frontendrepo.FrontendFactoryDetailRow) domain.MockFactoryProfile {
	certificates := []string{}
	if row.Verified {
		certificates = append(certificates, "Verified")
	}
	if row.Specialization.Valid && row.Specialization.String != "" {
		certificates = append(certificates, row.Specialization.String)
	}

	accepted := []string{}
	if row.Specialization.Valid && row.Specialization.String != "" {
		accepted = append(accepted, row.Specialization.String)
	}

	addressParts := []string{}
	if row.AddressDetail.Valid && row.AddressDetail.String != "" {
		addressParts = append(addressParts, row.AddressDetail.String)
	}
	if row.ProvinceName.Valid && row.ProvinceName.String != "" {
		addressParts = append(addressParts, row.ProvinceName.String)
	}

	return domain.MockFactoryProfile{
		FactoryID:            fmt.Sprintf("f%d", row.ID),
		Address:              strings.Join(addressParts, ", "),
		AcceptedProductTypes: accepted,
		Certificates:         certificates,
	}
}

func buildMockFactoryReview(factory domain.MockFactory, index int) domain.MockFactoryReview {
	brands := []string{"Pawsome Brand", "HappyTail Co.", "PetTech Thailand", "Organic Paw", "MewMew Fashion"}
	comments := []string{
		"สื่อสารไวและช่วยปรับรายละเอียดงานก่อนผลิตได้ดี",
		"คุณภาพงานสม่ำเสมอ เหมาะกับแบรนด์ที่กำลังเติบโต",
		"ทีมโรงงานให้คำแนะนำเรื่องต้นทุนและ timeline ชัดเจน",
		"เอกสารและมาตรฐานพร้อม ทำให้เริ่มขายได้เร็ว",
		"เหมาะกับการทดสอบตลาดและขยายล็อตในรอบถัดไป",
	}
	return domain.MockFactoryReview{
		ID:        fmt.Sprintf("rev%d", index+1),
		FactoryID: factory.ID,
		Reviewer:  brands[index%len(brands)],
		Rating:    factory.Rating,
		Comment:   comments[index%len(comments)],
		Date:      helper.DateDaysAgo(index + 7),
	}
}

func buildMockIdeaArticle(factory domain.MockFactory, index int) domain.MockIdeaArticle {
	tag := "แนวคิดผลิต"
	if index%3 == 1 {
		tag = "โปรโมชั่น"
	}
	return domain.MockIdeaArticle{
		ID:          fmt.Sprintf("idea%d", index+1),
		FactoryID:   factory.ID,
		FactoryName: factory.Name,
		Title:       fmt.Sprintf("ไอเดียต่อยอดสินค้า %s สำหรับแบรนด์ที่เริ่มต้น", factory.Specialization),
		Excerpt:     fmt.Sprintf("แนวทางเลือกสเปก MOQ และช่วงราคาให้เหมาะกับ %s", factory.Name),
		Image:       factory.Image,
		Tag:         tag,
		PublishedAt: helper.DateDaysAgo(index + 1),
	}
}

func buildMockShowcases(factory domain.MockFactory, index int) []domain.MockShowcase {
	baseCategory := factory.Tags[0]
	product := domain.MockShowcase{
		ID:          fmt.Sprintf("show%d", index*2+1),
		FactoryID:   factory.ID,
		FactoryName: factory.Name,
		Title:       fmt.Sprintf("ตัวอย่างสินค้าเด่นจาก %s", factory.Name),
		Excerpt:     fmt.Sprintf("เหมาะกับแบรนด์ที่ต้องการเริ่มต้นในหมวด %s", baseCategory),
		Image:       factory.Image,
		ContentType: "product",
		Category:    baseCategory,
		PostedAt:    helper.DateDaysAgo(index + 1),
		Likes:       60 + int64(index*12),
		LeadTime:    factory.LeadTime,
		Tags:        factory.Tags,
	}
	secondType := "promotion"
	if index%2 == 0 {
		secondType = "idea"
	}
	second := domain.MockShowcase{
		ID:          fmt.Sprintf("show%d", index*2+2),
		FactoryID:   factory.ID,
		FactoryName: factory.Name,
		Title:       fmt.Sprintf("แนะนำจากโรงงาน %s", factory.Name),
		Excerpt:     fmt.Sprintf("สรุปจุดเด่นและแนวทางคุยสเปกกับ %s", factory.Name),
		Image:       factory.Image,
		ContentType: secondType,
		Category:    baseCategory,
		PostedAt:    helper.DateDaysAgo(index + 2),
		Likes:       42 + int64(index*9),
		LeadTime:    factory.LeadTime,
		Tags:        factory.Tags,
	}
	return []domain.MockShowcase{product, second}
}

func (s *FrontendService) buildMockRFQ(row frontendrepo.FrontendRFQRow, categoryIcons map[string]string) (domain.MockRFQ, error) {
	offerRows, err := s.repo.ListQuotationsByRFQID(row.ID)
	if err != nil {
		return domain.MockRFQ{}, err
	}
	offers := make([]domain.MockOffer, 0, len(offerRows))
	for index, offer := range offerRows {
		offers = append(offers, domain.MockOffer{
			ID:              fmt.Sprintf("off%d", offer.ID),
			FactoryID:       fmt.Sprintf("f%d", offer.FactoryID),
			FactoryName:     offer.FactoryName,
			Price:           offer.TotalPrice,
			LeadTime:        offer.LeadTime,
			Rating:          4.6 + (float64(offer.FactoryID%4) * 0.1),
			Verified:        offer.Verified,
			Recommended:     index == 0,
			AIReason:        "ราคาและระยะเวลาผลิตเหมาะสมกับคำขอ",
			CompletedOrders: offer.CompletedOrders,
			ResponseTime:    fmt.Sprintf("%d ชั่วโมง", index+1),
		})
	}
	status := domainstatus.FrontendRFQ(row.Status, row.OfferCount)
	if status == "completed" && row.OfferCount > 0 {
		status = "completed"
	}
	if status == "offers_received" && row.OfferCount >= 2 {
		status = "reviewing"
	}
	if status == "cancelled" {
		status = "cancelled"
	}
	return domain.MockRFQ{
		ID:           fmt.Sprintf("rfq%d", row.ID),
		ProjectName:  row.ProjectName,
		Category:     row.Category,
		CategoryIcon: fallbackString(categoryIcons[row.Category], "📦"),
		Status:       status,
		OfferCount:   row.OfferCount,
		Budget:       row.Budget,
		Quantity:     row.Quantity,
		Material:     "รายละเอียดวัสดุเพิ่มเติม",
		Deadline:     helper.DateDaysFromNow(domain.DefaultQuotationValidityDays),
		CreatedAt:    row.CreatedAt,
		Description:  row.Description,
		Offers:       offers,
	}, nil
}

func (s *FrontendService) buildMockOrder(row frontendrepo.FrontendOrderRow, rfq domain.MockRFQ) (domain.MockOrder, error) {
	timelineRows, err := s.repo.ListOrderTimeline(row.ID)
	if err != nil {
		return domain.MockOrder{}, err
	}
	progress := int64(0)
	timeline := make([]domain.MockOrderTimelineItem, 0, len(timelineRows))
	if len(timelineRows) > 0 {
		progress = int64((len(timelineRows) * 100) / (len(timelineRows) + 1))
	}
	for index, item := range timelineRows {
		status := "completed"
		if index == len(timelineRows)-1 && domainstatus.FrontendOrder(row.Status) != "completed" {
			status = "current"
		}
		photo := helper.OptionalString(item.Photo.String)
		timeline = append(timeline, domain.MockOrderTimelineItem{
			ID:          fmt.Sprintf("t%d", index+1),
			Title:       fallbackString(item.Title.String, "Production Update"),
			Date:        item.Date,
			Status:      status,
			Photo:       photo,
			Description: item.Description.String,
		})
	}
	return domain.MockOrder{
		ID:                fmt.Sprintf("ord%d", row.ID),
		RFQID:             fmt.Sprintf("rfq%d", row.RFQID),
		FactoryID:         fmt.Sprintf("f%d", row.FactoryID),
		FactoryName:       row.FactoryName,
		ProjectName:       row.ProjectName,
		Category:          rfq.Category,
		Status:            domainstatus.FrontendOrder(row.Status),
		Progress:          progress,
		TotalAmount:       row.TotalAmount,
		DepositPaid:       row.DepositPaid,
		Quantity:          rfq.Quantity,
		CreatedAt:         row.CreatedAt,
		EstimatedDelivery: row.EstimatedDelivery,
		Timeline:          timeline,
	}, nil
}

func (s *FrontendService) buildMockConversation(userID int64, thread frontendrepo.FrontendMessageThreadRow, rfqMap map[int64]domain.MockRFQ, factoryMap map[int64]domain.MockFactory, index int) (domain.MockConversation, error) {
	messagesRows, err := s.repo.ListMessagesByReference(thread.ReferenceType, thread.ReferenceID, userID)
	if err != nil {
		return domain.MockConversation{}, err
	}
	ref, err := s.repo.GetReferenceLabel(thread.ReferenceType, thread.ReferenceID)
	if err != nil {
		return domain.MockConversation{}, err
	}
	userLabel, err := s.repo.GetUserLabel(thread.CounterpartID)
	if err != nil {
		return domain.MockConversation{}, err
	}

	var rfqID int64
	if thread.ReferenceType == "RQ" || thread.ReferenceType == "RFQ" {
		rfqID = thread.ReferenceID
	} else {
		for _, rfq := range rfqMap {
			if rfq.ProjectName == ref.ProjectName {
				rfqID = helper.ParseRFQID(rfq.ID)
				break
			}
		}
	}
	factory := factoryMap[thread.CounterpartID]
	rfq := rfqMap[rfqID]
	if rfq.ProjectName == "" {
		rfq.ProjectName = ref.ProjectName
		rfq.ID = fmt.Sprintf("rfq%d", rfqID)
	}

	messages := make([]domain.MockConversationMessage, 0, len(messagesRows)+1)
	for idx, msg := range messagesRows {
		sender := "factory"
		if msg.SenderID == userID {
			sender = "user"
		}
		messages = append(messages, domain.MockConversationMessage{
			ID:     fmt.Sprintf("m%d", idx+1),
			Sender: sender,
			Text:   msg.Content,
			Time:   msg.CreatedAt,
			Type:   "text",
		})
	}

	if ref.HasQuote && rfq.OfferCount > 0 && len(rfq.Offers) > 0 {
		quote := rfq.Offers[0]
		messages = append(messages, domain.MockConversationMessage{
			ID:     fmt.Sprintf("m%d", len(messages)+1),
			Sender: "factory",
			Text:   "",
			Time:   fallbackString(lastFrontendMessageTime(messagesRows), "10:00"),
			Type:   "quote",
			QuoteData: &domain.MockQuoteData{
				Price:      quote.Price,
				LeadTime:   quote.LeadTime,
				ValidUntil: helper.DateDaysFromNow(7),
			},
		})
	}

	lastMessage := thread.LastMessage
	if lastMessage == "" && len(messages) > 0 {
		lastMessage = messages[len(messages)-1].Text
	}

	return domain.MockConversation{
		ID:            fmt.Sprintf("conv%d", index+1),
		FactoryID:     fmt.Sprintf("f%d", thread.CounterpartID),
		RFQID:         fallbackString(rfq.ID, fmt.Sprintf("rfq%d", thread.ReferenceID)),
		FactoryName:   fallbackString(factory.Name, userLabel.Name),
		FactoryAvatar: helper.AvatarURL(userLabel.Name),
		RFQName:       ref.ProjectName,
		LastMessage:   lastMessage,
		Time:          helper.RelativeThaiTimeFromISO(thread.LastMessageAt),
		Unread:        0,
		HasQuote:      ref.HasQuote,
		Messages:      messages,
	}, nil
}

func buildMockNotifications(rfqs []domain.MockRFQ, orders []domain.MockOrder, conversations []domain.MockConversation) []domain.MockNotification {
	items := []domain.MockNotification{}
	index := 1
	for _, rfq := range rfqs {
		if rfq.OfferCount > 0 && len(items) < 3 {
			items = append(items, domain.MockNotification{
				ID:      fmt.Sprintf("n%d", index),
				Type:    "rfq",
				Title:   "มีใบเสนอราคาใหม่",
				Message: fmt.Sprintf("โครงการ \"%s\" ได้รับใบเสนอราคา %d รายการ", rfq.ProjectName, rfq.OfferCount),
				Time:    helper.RelativeThaiTime(rfq.CreatedAt),
				Read:    false,
				LinkTo:  fmt.Sprintf("/rfqs/%s", rfq.ID),
				RFQID:   rfq.ID,
				Avatar:  "",
			})
			index++
		}
	}
	for _, order := range orders {
		if len(items) >= 6 {
			break
		}
		items = append(items, domain.MockNotification{
			ID:      fmt.Sprintf("n%d", index),
			Type:    "order",
			Title:   "อัปเดตคำสั่งซื้อ",
			Message: fmt.Sprintf("คำสั่งซื้อ \"%s\" อยู่สถานะ %s", order.ProjectName, order.Status),
			Time:    helper.RelativeThaiTime(order.CreatedAt),
			Read:    false,
			LinkTo:  fmt.Sprintf("/orders/%s", order.ID),
			OrderID: order.ID,
			Avatar:  "",
		})
		index++
	}
	for _, conversation := range conversations {
		if len(items) >= 8 {
			break
		}
		items = append(items, domain.MockNotification{
			ID:             fmt.Sprintf("n%d", index),
			Type:           "message",
			Title:          "ข้อความใหม่",
			Message:        fmt.Sprintf("%s: %s", conversation.FactoryName, conversation.LastMessage),
			Time:           conversation.Time,
			Read:           false,
			LinkTo:         fmt.Sprintf("/messages/%s", conversation.ID),
			ConversationID: conversation.ID,
			Avatar:         conversation.FactoryAvatar,
		})
		index++
	}
	return items
}

func (s *FrontendService) GetProducts(limit int, categoryID string) ([]domain.Product, error) {
	if limit <= 0 {
		limit = 8
	}
	return s.repo.GetProducts(limit, categoryID)
}

func (s *FrontendService) GetPromotions(limit int) ([]domain.Promotion, error) {
	if limit <= 0 {
		limit = 4
	}
	return s.repo.GetPromotions(limit)
}

func (s *FrontendService) GetPromoCodes() ([]domain.PromoCode, error) {
	return s.repo.GetPromoCodes()
}

func (s *FrontendService) GetExploreData(userID int64) (*domain.ExploreData, error) {
	products, err := s.GetProducts(8, "")
	if err != nil {
		products = []domain.Product{}
	}
	promotions, err := s.GetPromotions(4)
	if err != nil {
		promotions = []domain.Promotion{}
	}
	promoCodes, err := s.GetPromoCodes()
	if err != nil {
		promoCodes = []domain.PromoCode{}
	}
	if userID <= 0 {
		factoryRows, err := s.repo.ListFactories()
		if err != nil {
			return nil, err
		}
		categoryRows, err := s.repo.ListCategories()
		if err != nil {
			return nil, err
		}
		factories := make([]domain.MockFactory, 0, len(factoryRows))
		ideaArticles := make([]domain.MockIdeaArticle, 0, len(factoryRows))
		for index, row := range factoryRows {
			factory := mapMockFactory(row)
			factories = append(factories, factory)
			ideaArticles = append(ideaArticles, buildMockIdeaArticle(factory, index))
		}
		categories := make([]domain.MockCategory, 0, len(categoryRows))
		for _, item := range categoryRows {
			categories = append(categories, mapMockCategory(item))
		}
		return &domain.ExploreData{
			Products:     products,
			Promotions:   promotions,
			PromoCodes:   promoCodes,
			Factories:    factories,
			IdeaArticles: ideaArticles,
			Categories:   categories,
		}, nil
	}

	// GetMockData ต้องการ userID สำหรับ currentUser แต่ Explore ไม่ต้องใช้
	// ถ้า GetMockData fail (user not found ฯลฯ) ให้ fallback เป็น factories+categories จาก repo โดยตรง
	mockData, mockErr := s.GetMockData(userID)
	if mockErr != nil {
		factoryRows, fErr := s.repo.ListFactories()
		factories := make([]domain.MockFactory, 0)
		if fErr == nil {
			for idx, row := range factoryRows {
				factories = append(factories, mapMockFactory(row))
				_ = idx
			}
		}
		categoryRows, cErr := s.repo.ListCategories()
		categories := make([]domain.MockCategory, 0)
		if cErr == nil {
			for _, row := range categoryRows {
				categories = append(categories, mapMockCategory(row))
			}
		}
		return &domain.ExploreData{
			Products:     products,
			Promotions:   promotions,
			PromoCodes:   promoCodes,
			Factories:    factories,
			IdeaArticles: []domain.MockIdeaArticle{},
			Categories:   categories,
		}, nil
	}

	return &domain.ExploreData{
		Products:     products,
		Promotions:   promotions,
		PromoCodes:   promoCodes,
		Factories:    mockData.Factories,
		IdeaArticles: mockData.IdeaArticles,
		Categories:   mockData.Categories,
	}, nil
}

func lastFrontendMessageTime(items []frontendrepo.FrontendMessageRow) string {
	if len(items) == 0 {
		return ""
	}
	return items[len(items)-1].CreatedAt
}
