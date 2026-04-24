package service

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
)

type FrontendService struct {
	repo *repository.FrontendRepository
}

func NewFrontendService(repo *repository.FrontendRepository) *FrontendService {
	return &FrontendService{repo: repo}
}

func (s *FrontendService) GetBootstrap(userID int64) (*domain.FrontendBootstrapResponse, error) {
	// anonymous request → return public data only
	if userID <= 0 {
		return s.getGuestBootstrap()
	}

	// authenticated request — currentUser อาจเป็น nil ถ้า user ไม่พบในฐานข้อมูล
	// (เช่น db.Get() fail เพราะ LEFT JOIN คืนหลาย row)
	// ในกรณีนั้นยังโหลด public data + user-specific data ต่อได้
	var currentUser *domain.FrontendCurrentUser
	{
		cu, err := s.GetCurrentUser(userID)
		if err != nil {
			if !repository.IsNotFoundError(err) {
				return nil, err
			}
			// user not found → currentUser stays nil, load data as guest
		} else {
			currentUser = cu
		}
	}

	categoryRows, err := s.repo.ListCategories()
	if err != nil {
		return nil, err
	}
	factoryRows, err := s.repo.ListFactories()
	if err != nil {
		return nil, err
	}

	var rfqRows []repository.FrontendRFQRow
	var orderRows []repository.FrontendOrderRow
	var threadRows []repository.FrontendMessageThreadRow
	if userID > 0 {
		if rows, e := s.repo.ListRFQsByUserID(userID); e == nil {
			rfqRows = rows
		}
		if rows, e := s.repo.ListOrdersByUserID(userID); e == nil {
			orderRows = rows
		}
		if rows, e := s.repo.ListMessageThreads(userID); e == nil {
			threadRows = rows
		}
	}

	response := &domain.FrontendBootstrapResponse{
		CurrentUser: currentUser,
		Categories:  make([]domain.FrontendCategory, 0, len(categoryRows)),
		Factories:   make([]domain.FrontendFactoryCard, 0, len(factoryRows)),
		RFQs:        make([]domain.FrontendRFQCard, 0, len(rfqRows)),
		Orders:      make([]domain.FrontendOrderCard, 0, len(orderRows)),
		Threads:     make([]domain.FrontendMessageThread, 0, len(threadRows)),
	}

	for _, item := range categoryRows {
		response.Categories = append(response.Categories, domain.FrontendCategory{
			ID:   item.ID,
			Name: item.Name,
		})
	}
	for _, item := range factoryRows {
		response.Factories = append(response.Factories, mapFactoryCard(item))
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

func (s *FrontendService) getGuestBootstrap() (*domain.FrontendBootstrapResponse, error) {
	categoryRows, err := s.repo.ListCategories()
	if err != nil {
		return nil, err
	}
	factoryRows, err := s.repo.ListFactories()
	if err != nil {
		return nil, err
	}
	response := &domain.FrontendBootstrapResponse{
		CurrentUser: nil,
		Categories:  make([]domain.FrontendCategory, 0, len(categoryRows)),
		Factories:   make([]domain.FrontendFactoryCard, 0, len(factoryRows)),
		RFQs:        []domain.FrontendRFQCard{},
		Orders:      []domain.FrontendOrderCard{},
		Threads:     []domain.FrontendMessageThread{},
	}
	for _, item := range categoryRows {
		response.Categories = append(response.Categories, domain.FrontendCategory{
			ID:   item.ID,
			Name: item.Name,
		})
	}
	for _, item := range factoryRows {
		response.Factories = append(response.Factories, mapFactoryCard(item))
	}
	return response, nil
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

func (s *FrontendService) ListFactories() ([]domain.FrontendFactoryCard, error) {
	rows, err := s.repo.ListFactories()
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

	addressParts := []string{}
	if row.AddressDetail.Valid && row.AddressDetail.String != "" {
		addressParts = append(addressParts, row.AddressDetail.String)
	}
	if row.ProvinceName.Valid && row.ProvinceName.String != "" {
		addressParts = append(addressParts, row.ProvinceName.String)
	}

	return &domain.FrontendFactoryDetail{
		Factory: mapFactoryCard(repository.FrontendFactoryRow{
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
			MinOrder:        row.MinOrder,
			LeadTimeDesc:    row.LeadTimeDesc,
			ImageURL:        row.ImageURL,
			PriceRange:      row.PriceRange,
		}),
		Profile: domain.FrontendFactoryProfile{
			Address:              strings.Join(addressParts, ", "),
			AcceptedProductTypes: []string{},
			Certificates:         []string{},
		},
		Reviews:  []domain.FrontendFactoryReview{},
		Products: []domain.FrontendShowcaseItem{},
		Promos:   []domain.FrontendShowcaseItem{},
		Ideas:    []domain.FrontendShowcaseItem{},
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
			Status:          mapQuotationStatus(offer.Status),
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
		factoryID := parseFactoryID(factory.ID)
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

func (s *FrontendService) buildThreads(rows []repository.FrontendMessageThreadRow) ([]domain.FrontendMessageThread, error) {
	items := make([]domain.FrontendMessageThread, 0, len(rows))
	for _, item := range rows {
		userLabel, err := s.repo.GetUserLabel(item.CounterpartID)
		if err != nil {
			return nil, err
		}
		reference, err := s.repo.GetReferenceLabel(item.ReferenceType, item.ReferenceID)
		if err != nil {
			return nil, err
		}
		items = append(items, domain.FrontendMessageThread{
			ReferenceType: item.ReferenceType,
			ReferenceID:   fmt.Sprintf("%d", item.ReferenceID),
			CounterpartID: item.CounterpartID,
			Counterpart:   userLabel.Name,
			ProjectName:   reference.ProjectName,
			LastMessage:   item.LastMessage,
			LastMessageAt: item.LastMessageAt,
			Unread:        0,
			HasQuote:      reference.HasQuote,
			Avatar:        "",
		})
	}
	return items, nil
}

func mapFactoryCard(row repository.FrontendFactoryRow) domain.FrontendFactoryCard {
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

	minOrder := int64(0)
	if row.MinOrder.Valid {
		minOrder = row.MinOrder.Int64
	}

	return domain.FrontendFactoryCard{
		ID:              row.ID,
		Name:            row.Name,
		Location:        row.Location.String,
		Rating:          row.Rating,
		Reviews:         row.ReviewCount,
		Specialization:  specialization,
		Tags:            tags,
		MinOrder:        minOrder,
		LeadTime:        leadTime,
		Image:           row.ImageURL.String,
		Verified:        row.Verified,
		CompletedOrders: row.CompletedOrders,
		PriceRange:      row.PriceRange.String,
		Description:     row.Description.String,
	}
}

func mapRFQCard(row repository.FrontendRFQRow) domain.FrontendRFQCard {
	return domain.FrontendRFQCard{
		ID:          row.ID,
		ProjectName: row.ProjectName,
		Category:    row.Category,
		Status:      mapRFQStatus(row.Status, row.OfferCount),
		OfferCount:  row.OfferCount,
		Budget:      row.Budget,
		Quantity:    row.Quantity,
		CreatedAt:   row.CreatedAt,
		Description: row.Description,
	}
}

func mapOrderCard(row repository.FrontendOrderRow) domain.FrontendOrderCard {
	return domain.FrontendOrderCard{
		ID:                row.ID,
		ProjectName:       row.ProjectName,
		RFQID:             row.RFQID,
		FactoryID:         row.FactoryID,
		FactoryName:       row.FactoryName,
		TotalAmount:       row.TotalAmount,
		DepositPaid:       row.DepositPaid,
		Status:            mapOrderStatus(row.Status),
		EstimatedDelivery: row.EstimatedDelivery,
		CreatedAt:         row.CreatedAt,
	}
}

func mapRFQStatus(status string, offerCount int64) string {
	switch status {
	case "CC":
		return "cancelled"
	case "CL":
		return "completed"
	case "OP":
		if offerCount > 0 {
			return "offers_received"
		}
		return "pending"
	default:
		return strings.ToLower(status)
	}
}

func mapOrderStatus(status string) string {
	switch status {
	case "PR", "QC":
		return "in_production"
	case "SH":
		return "shipped"
	case "CP":
		return "completed"
	default:
		return strings.ToLower(status)
	}
}

func mapQuotationStatus(status string) string {
	switch status {
	case "PD":
		return "pending"
	case "AC":
		return "accepted"
	case "RJ":
		return "rejected"
	default:
		return strings.ToLower(status)
	}
}

func fallbackString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func mapMockCurrentUser(row *repository.FrontendCurrentUserRow) *domain.MockCurrentUser {
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
		Avatar:         avatarURL(name),
		Company:        company,
		Email:          row.Email,
		Phone:          row.Phone.String,
		WalletBalance:  row.WalletBalance.Float64,
		PendingBalance: row.PendingBalance.Float64,
		MemberSince:    row.MemberSince,
	}
}

func mapMockCategory(row repository.FrontendCategoryRow) domain.MockCategory {
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

	id := slugifyCategory(row.Name)
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

func mapMockFactory(row repository.FrontendFactoryRow) domain.MockFactory {
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

	minOrder := int64(100 + ((row.ID % 5) * 100))
	if row.CompletedOrders == 0 {
		minOrder = 100
	}

	return domain.MockFactory{
		ID:              fmt.Sprintf("f%d", row.ID),
		Name:            row.Name,
		Location:        fallbackString(row.Location.String, "กรุงเทพฯ"),
		Rating:          rating,
		Reviews:         reviews,
		Specialization:  fallbackString(row.Specialization.String, "โรงงานรับผลิตสินค้า"),
		Tags:            tags,
		MinOrder:        minOrder,
		LeadTime:        fallbackString(formatLeadTimeRange(row.AverageLeadDays.Float64), "7-14 วัน"),
		Image:           factoryImageURL(row.ID),
		Verified:        row.Verified,
		CompletedOrders: row.CompletedOrders,
		PriceRange:      priceRange,
	}
}

func buildMockFactoryProfile(row repository.FrontendFactoryDetailRow) domain.MockFactoryProfile {
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
		Date:      dateDaysAgo(index + 7),
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
		PublishedAt: dateDaysAgo(index + 1),
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
		PostedAt:    dateDaysAgo(index + 1),
		Likes:       60 + int64(index*12),
		MinOrder:    factory.MinOrder,
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
		PostedAt:    dateDaysAgo(index + 2),
		Likes:       42 + int64(index*9),
		MinOrder:    factory.MinOrder,
		LeadTime:    factory.LeadTime,
		Tags:        factory.Tags,
	}
	return []domain.MockShowcase{product, second}
}

func (s *FrontendService) buildMockRFQ(row repository.FrontendRFQRow, categoryIcons map[string]string) (domain.MockRFQ, error) {
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
	status := mapRFQStatus(row.Status, row.OfferCount)
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
		Deadline:     dateDaysFromNow(14),
		CreatedAt:    row.CreatedAt,
		Description:  row.Description,
		Offers:       offers,
	}, nil
}

func (s *FrontendService) buildMockOrder(row repository.FrontendOrderRow, rfq domain.MockRFQ) (domain.MockOrder, error) {
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
		if index == len(timelineRows)-1 && mapOrderStatus(row.Status) != "completed" {
			status = "current"
		}
		photo := optionalString(item.Photo.String)
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
		Status:            mapOrderStatus(row.Status),
		Progress:          progress,
		TotalAmount:       row.TotalAmount,
		DepositPaid:       row.DepositPaid,
		Quantity:          rfq.Quantity,
		CreatedAt:         row.CreatedAt,
		EstimatedDelivery: row.EstimatedDelivery,
		Timeline:          timeline,
	}, nil
}

func (s *FrontendService) buildMockConversation(userID int64, thread repository.FrontendMessageThreadRow, rfqMap map[int64]domain.MockRFQ, factoryMap map[int64]domain.MockFactory, index int) (domain.MockConversation, error) {
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
				rfqID = parseRFQID(rfq.ID)
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
			Time:   fallbackString(lastMessageTime(messagesRows), "10:00"),
			Type:   "quote",
			QuoteData: &domain.MockQuoteData{
				Price:      quote.Price,
				LeadTime:   quote.LeadTime,
				ValidUntil: dateDaysFromNow(7),
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
		FactoryAvatar: avatarURL(userLabel.Name),
		RFQName:       ref.ProjectName,
		LastMessage:   lastMessage,
		Time:          relativeThaiTimeFromISO(thread.LastMessageAt),
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
				Time:    relativeThaiTime(rfq.CreatedAt),
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
			Time:    relativeThaiTime(order.CreatedAt),
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

func parseFactoryID(id string) int64 {
	value := strings.TrimPrefix(id, "f")
	result, _ := strconv.ParseInt(value, 10, 64)
	return result
}

func parseRFQID(id string) int64 {
	value := strings.TrimPrefix(id, "rfq")
	result, _ := strconv.ParseInt(value, 10, 64)
	return result
}

func slugifyCategory(name string) string {
	switch name {
	case "อาหารสัตว์":
		return "pet_food"
	case "อาหารเสริม":
		return "supplements"
	case "ของเล่นสัตว์เลี้ยง":
		return "pet_toys"
	case "สายจูง อุปกรณ์", "อุปกรณ์สัตว์เลี้ยง":
		return "leash_equipment"
	case "เสื้อผ้าสัตว์เลี้ยง":
		return "pet_clothes"
	default:
		return "other"
	}
}

func factoryImageURL(factoryID int64) string {
	images := []string{
		"https://images.unsplash.com/photo-1579784340946-55a7bbd51d57?w=400&h=250&fit=crop",
		"https://images.unsplash.com/photo-1684259499086-93cb3e555803?w=400&h=250&fit=crop",
		"https://images.unsplash.com/photo-1587300003388-59208cc962cb?w=400&h=250&fit=crop",
		"https://images.unsplash.com/photo-1607082348824-0a96f2a4b9da?w=400&h=250&fit=crop",
		"https://images.unsplash.com/photo-1471864190281-a93a3070b6de?w=400&h=250&fit=crop",
		"https://images.unsplash.com/photo-1517849845537-4d257902454a?w=400&h=250&fit=crop",
	}
	return images[(factoryID-1)%int64(len(images))]
}

func avatarURL(name string) string {
	return "https://ui-avatars.com/api/?background=EDE9FF&color=6C47FF&name=" + url.QueryEscape(name)
}

func formatLeadTimeRange(avg float64) string {
	if avg <= 0 {
		return ""
	}
	base := int(avg + 0.5)
	return fmt.Sprintf("%d-%d วัน", max(base-2, 1), base+2)
}

func dateDaysAgo(days int) string {
	return time.Now().AddDate(0, 0, -days).Format("2006-01-02")
}

func dateDaysFromNow(days int) string {
	return time.Now().AddDate(0, 0, days).Format("2006-01-02")
}

func relativeThaiTime(date string) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	diff := time.Since(t)
	if diff.Hours() < 24 {
		return "วันนี้"
	}
	if diff.Hours() < 48 {
		return "เมื่อวาน"
	}
	return fmt.Sprintf("%d วันที่แล้ว", int(diff.Hours()/24))
}

func relativeThaiTimeFromISO(date string) string {
	t, err := time.Parse("2006-01-02T15:04:05", date)
	if err != nil {
		return date
	}
	diff := time.Since(t)
	if diff.Minutes() < 60 {
		return fmt.Sprintf("%d นาทีที่แล้ว", int(diff.Minutes()))
	}
	if diff.Hours() < 24 {
		return fmt.Sprintf("%d ชั่วโมงที่แล้ว", int(diff.Hours()))
	}
	if diff.Hours() < 48 {
		return "เมื่อวาน"
	}
	return fmt.Sprintf("%d วันที่แล้ว", int(diff.Hours()/24))
}

func optionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	v := value
	return &v
}

func lastMessageTime(items []repository.FrontendMessageRow) string {
	if len(items) == 0 {
		return ""
	}
	return items[len(items)-1].CreatedAt
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
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
