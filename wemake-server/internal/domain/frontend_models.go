package domain

type FrontendBootstrapResponse struct {
	CurrentUser *FrontendCurrentUser    `json:"currentUser"`
	Categories  []FrontendCategory      `json:"categories"`
	Factories   []FrontendFactoryCard   `json:"factories"`
	RFQs        []FrontendRFQCard       `json:"rfqs"`
	Orders      []FrontendOrderCard     `json:"orders"`
	Threads     []FrontendMessageThread `json:"threads"`
}

type FrontendCurrentUser struct {
	ID             int64   `json:"id"`
	Role           string  `json:"role"`
	Name           string  `json:"name"`
	Company        string  `json:"company"`
	Email          string  `json:"email"`
	Phone          string  `json:"phone"`
	Avatar         string  `json:"avatar"`
	WalletBalance  float64 `json:"walletBalance"`
	PendingBalance float64 `json:"pendingBalance"`
	MemberSince    string  `json:"memberSince"`
}

type FrontendCategory struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type FrontendFactoryCard struct {
	ID              int64    `json:"id"`
	Name            string   `json:"name"`
	Location        string   `json:"location"`
	Rating          float64  `json:"rating"`
	Reviews         int64    `json:"reviews"`
	Specialization  string   `json:"specialization"`
	Tags            []string `json:"tags"`
	MinOrder        int64    `json:"minOrder"`
	LeadTime        string   `json:"leadTime"`
	Image           string   `json:"image"`
	Verified        bool     `json:"verified"`
	CompletedOrders int64    `json:"completedOrders"`
	PriceRange      string   `json:"priceRange"`
	Description     string   `json:"description,omitempty"`
}

type FrontendFactoryDetail struct {
	Factory  FrontendFactoryCard     `json:"factory"`
	Profile  FrontendFactoryProfile  `json:"profile"`
	Reviews  []FrontendFactoryReview `json:"reviews"`
	Products []FrontendShowcaseItem  `json:"products"`
	Promos   []FrontendShowcaseItem  `json:"promotions"`
	Ideas    []FrontendShowcaseItem  `json:"ideas"`
}

type FrontendFactoryProfile struct {
	Address              string   `json:"address"`
	AcceptedProductTypes []string `json:"acceptedProductTypes"`
	Certificates         []string `json:"certificates"`
}

type FrontendFactoryReview struct {
	ID       string  `json:"id"`
	Reviewer string  `json:"reviewer"`
	Rating   float64 `json:"rating"`
	Comment  string  `json:"comment"`
	Date     string  `json:"date"`
}

type FrontendShowcaseItem struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	ContentType string `json:"contentType"`
	Excerpt     string `json:"excerpt"`
	Image       string `json:"image"`
}

type FrontendRFQCard struct {
	ID          int64                   `json:"id"`
	ProjectName string                  `json:"projectName"`
	Category    string                  `json:"category"`
	Status      string                  `json:"status"`
	OfferCount  int64                   `json:"offerCount"`
	Budget      float64                 `json:"budget"`
	Quantity    int64                   `json:"quantity"`
	CreatedAt   string                  `json:"createdAt"`
	Description string                  `json:"description"`
	Offers      []FrontendQuotationCard `json:"offers,omitempty"`
	Images      []string                `json:"images,omitempty"`
}

type FrontendQuotationCard struct {
	ID              int64   `json:"id"`
	FactoryID       int64   `json:"factoryId"`
	FactoryName     string  `json:"factoryName"`
	Price           float64 `json:"price"`
	LeadTime        int64   `json:"leadTime"`
	Verified        bool    `json:"verified"`
	Recommended     bool    `json:"recommended"`
	CompletedOrders int64   `json:"completedOrders"`
	Status          string  `json:"status"`
}

type FrontendOrderCard struct {
	ID                int64   `json:"id"`
	ProjectName       string  `json:"projectName"`
	RFQID             int64   `json:"rfqId"`
	FactoryID         int64   `json:"factoryId"`
	FactoryName       string  `json:"factoryName"`
	TotalAmount       float64 `json:"totalAmount"`
	DepositPaid       float64 `json:"depositPaid"`
	Status            string  `json:"status"`
	EstimatedDelivery string  `json:"estimatedDelivery"`
	CreatedAt         string  `json:"createdAt"`
}

type FrontendOrderDetail struct {
	Order    FrontendOrderCard           `json:"order"`
	Timeline []FrontendOrderTimelineItem `json:"timeline"`
}

type FrontendOrderTimelineItem struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Date        string `json:"date"`
	Status      string `json:"status"`
	Description string `json:"description"`
	Photo       string `json:"photo"`
}

type FrontendMessageThread struct {
	ReferenceType string `json:"referenceType"`
	ReferenceID   string `json:"referenceId"`
	CounterpartID int64  `json:"counterpartId"`
	Counterpart   string `json:"counterpart"`
	ProjectName   string `json:"projectName"`
	LastMessage   string `json:"lastMessage"`
	LastMessageAt string `json:"lastMessageAt"`
	Unread        int64  `json:"unread"`
	HasQuote      bool   `json:"hasQuote"`
	Avatar        string `json:"avatar"`
}

type FrontendMockDataResponse struct {
	CurrentUser      *MockCurrentUser     `json:"currentUser"`
	Categories       []MockCategory       `json:"categories"`
	Factories        []MockFactory        `json:"factories"`
	FactoryProfiles  []MockFactoryProfile `json:"factoryProfiles"`
	FactoryReviews   []MockFactoryReview  `json:"factoryReviews"`
	IdeaArticles     []MockIdeaArticle    `json:"ideaArticles"`
	FactoryShowcases []MockShowcase       `json:"factoryShowcases"`
	RFQs             []MockRFQ            `json:"rfqs"`
	Orders           []MockOrder          `json:"orders"`
	Conversations    []MockConversation   `json:"conversations"`
	Notifications    []MockNotification   `json:"notifications"`
}

type MockCurrentUser struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	NameEn         string  `json:"nameEn"`
	Avatar         string  `json:"avatar"`
	Company        string  `json:"company"`
	Email          string  `json:"email"`
	Phone          string  `json:"phone"`
	WalletBalance  float64 `json:"walletBalance"`
	PendingBalance float64 `json:"pendingBalance"`
	MemberSince    string  `json:"memberSince"`
}

type MockCategory struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Icon  string `json:"icon"`
	Color string `json:"color"`
}

type MockFactory struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Location        string   `json:"location"`
	Rating          float64  `json:"rating"`
	Reviews         int64    `json:"reviews"`
	Specialization  string   `json:"specialization"`
	Tags            []string `json:"tags"`
	MinOrder        int64    `json:"minOrder"`
	LeadTime        string   `json:"leadTime"`
	Image           string   `json:"image"`
	Verified        bool     `json:"verified"`
	CompletedOrders int64    `json:"completedOrders"`
	PriceRange      string   `json:"priceRange"`
}

type MockFactoryProfile struct {
	FactoryID            string   `json:"factoryId"`
	Address              string   `json:"address"`
	AcceptedProductTypes []string `json:"acceptedProductTypes"`
	Certificates         []string `json:"certificates"`
}

type MockFactoryReview struct {
	ID        string  `json:"id"`
	FactoryID string  `json:"factoryId"`
	Reviewer  string  `json:"reviewer"`
	Rating    float64 `json:"rating"`
	Comment   string  `json:"comment"`
	Date      string  `json:"date"`
}

type MockIdeaArticle struct {
	ID          string `json:"id"`
	FactoryID   string `json:"factoryId"`
	FactoryName string `json:"factoryName"`
	Title       string `json:"title"`
	Excerpt     string `json:"excerpt"`
	Image       string `json:"image"`
	Tag         string `json:"tag"`
	PublishedAt string `json:"publishedAt"`
}

type MockShowcase struct {
	ID          string   `json:"id"`
	FactoryID   string   `json:"factoryId"`
	FactoryName string   `json:"factoryName"`
	Title       string   `json:"title"`
	Excerpt     string   `json:"excerpt"`
	Image       string   `json:"image"`
	ContentType string   `json:"contentType"`
	Category    string   `json:"category"`
	PostedAt    string   `json:"postedAt"`
	Likes       int64    `json:"likes"`
	MinOrder    int64    `json:"minOrder"`
	LeadTime    string   `json:"leadTime"`
	Tags        []string `json:"tags"`
}

type MockOffer struct {
	ID              string  `json:"id"`
	FactoryID       string  `json:"factoryId"`
	FactoryName     string  `json:"factoryName"`
	Price           float64 `json:"price"`
	LeadTime        int64   `json:"leadTime"`
	Rating          float64 `json:"rating"`
	Verified        bool    `json:"verified"`
	Recommended     bool    `json:"recommended"`
	AIReason        string  `json:"aiReason"`
	CompletedOrders int64   `json:"completedOrders"`
	ResponseTime    string  `json:"responseTime"`
}

type MockRFQ struct {
	ID           string      `json:"id"`
	ProjectName  string      `json:"projectName"`
	Category     string      `json:"category"`
	CategoryIcon string      `json:"categoryIcon"`
	Status       string      `json:"status"`
	OfferCount   int64       `json:"offerCount"`
	Budget       float64     `json:"budget"`
	Quantity     int64       `json:"quantity"`
	Material     string      `json:"material"`
	Deadline     string      `json:"deadline"`
	CreatedAt    string      `json:"createdAt"`
	Description  string      `json:"description"`
	Offers       []MockOffer `json:"offers"`
}

type MockOrderTimelineItem struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Date        string  `json:"date"`
	Status      string  `json:"status"`
	Photo       *string `json:"photo"`
	Description string  `json:"description"`
}

type MockOrder struct {
	ID                string                  `json:"id"`
	RFQID             string                  `json:"rfqId"`
	FactoryID         string                  `json:"factoryId"`
	FactoryName       string                  `json:"factoryName"`
	ProjectName       string                  `json:"projectName"`
	Category          string                  `json:"category"`
	Status            string                  `json:"status"`
	Progress          int64                   `json:"progress"`
	TotalAmount       float64                 `json:"totalAmount"`
	DepositPaid       float64                 `json:"depositPaid"`
	Quantity          int64                   `json:"quantity"`
	CreatedAt         string                  `json:"createdAt"`
	EstimatedDelivery string                  `json:"estimatedDelivery"`
	Timeline          []MockOrderTimelineItem `json:"timeline"`
}

type MockQuoteData struct {
	Price      float64 `json:"price"`
	LeadTime   int64   `json:"leadTime"`
	ValidUntil string  `json:"validUntil"`
}

type MockConversationMessage struct {
	ID        string         `json:"id"`
	Sender    string         `json:"sender"`
	Text      string         `json:"text"`
	Time      string         `json:"time"`
	Type      string         `json:"type"`
	QuoteData *MockQuoteData `json:"quoteData,omitempty"`
}

type MockConversation struct {
	ID            string                    `json:"id"`
	FactoryID     string                    `json:"factoryId"`
	RFQID         string                    `json:"rfqId"`
	FactoryName   string                    `json:"factoryName"`
	FactoryAvatar string                    `json:"factoryAvatar"`
	RFQName       string                    `json:"rfqName"`
	LastMessage   string                    `json:"lastMessage"`
	Time          string                    `json:"time"`
	Unread        int64                     `json:"unread"`
	HasQuote      bool                      `json:"hasQuote"`
	Messages      []MockConversationMessage `json:"messages"`
}

type MockNotification struct {
	ID             string `json:"id"`
	Type           string `json:"type"`
	Title          string `json:"title"`
	Message        string `json:"message"`
	Time           string `json:"time"`
	Read           bool   `json:"read"`
	LinkTo         string `json:"linkTo"`
	RFQID          string `json:"rfqId,omitempty"`
	OrderID        string `json:"orderId,omitempty"`
	ConversationID string `json:"conversationId,omitempty"`
	Avatar         string `json:"avatar"`
}
