package showcase

import (
	"database/sql"
	"errors"
	"net/url"
	"strconv"
	"strings"

	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/domainutil"
	factoryrepo "github.com/yourusername/wemake/internal/repository/factory"
	showcaserepo "github.com/yourusername/wemake/internal/repository/showcase"
)

var (
	ErrShowcaseInvalidSectionType      = errors.New("INVALID_SECTION_TYPE")
	ErrShowcaseSectionTitleRequired    = errors.New("SECTION_TITLE_REQUIRED")
	ErrShowcaseMaxItemsPerSection      = errors.New("MAX_20_ITEMS_PER_SECTION")
	ErrShowcaseItemDescriptionRequired = errors.New("ITEM_DESCRIPTION_REQUIRED")
)

type ShowcaseService struct {
	repo        *showcaserepo.ShowcaseRepository
	factoryRepo *factoryrepo.FactoryRepository
}

func NewShowcaseService(repo *showcaserepo.ShowcaseRepository, factoryRepo *factoryrepo.FactoryRepository) *ShowcaseService {
	return &ShowcaseService{repo: repo, factoryRepo: factoryRepo}
}

func (s *ShowcaseService) upsertFactoryCategoryMap(factoryID int64, item *domain.FactoryShowcase) {
	if item.CategoryID != nil && *item.CategoryID > 0 {
		err := s.factoryRepo.AddFactoryCategory(factoryID, *item.CategoryID)
		if err != nil && !errors.Is(err, factoryrepo.ErrDuplicateFactoryCategory) {
			// best-effort; ignore errors
			_ = err
		}
	}
	if item.SubCategoryID != nil && *item.SubCategoryID > 0 {
		err := s.factoryRepo.AddFactorySubCategory(factoryID, *item.SubCategoryID)
		if err != nil && !errors.Is(err, factoryrepo.ErrDuplicateFactorySubCategory) {
			_ = err
		}
	}
}

func (s *ShowcaseService) ValidateSectionInputs(sections []domain.ShowcaseSectionInput) error {
	for _, sec := range sections {
		if sec.SectionType != "highlight" && sec.SectionType != "checklist" {
			return ErrShowcaseInvalidSectionType
		}
		if strings.TrimSpace(sec.SectionTitle) == "" {
			return ErrShowcaseSectionTitleRequired
		}
		if len(sec.Items) > 20 {
			return ErrShowcaseMaxItemsPerSection
		}
		for _, item := range sec.Items {
			if strings.TrimSpace(item.Description) == "" {
				return ErrShowcaseItemDescriptionRequired
			}
		}
	}
	return nil
}

func (s *ShowcaseService) ListExplore(contentType string) ([]domain.ShowcaseExploreItem, error) {
	return s.repo.ListExplore(contentType)
}

func (s *ShowcaseService) ListExploreByFactory(factoryID int64, contentType string) ([]domain.ShowcaseExploreItem, error) {
	return s.repo.ListExploreByFactory(factoryID, contentType)
}

func (s *ShowcaseService) ListStructured(filter domain.ShowcaseListFilter) ([]domain.ShowcaseExploreItem, error) {
	return s.repo.ListStructured(filter)
}

func (s *ShowcaseService) GetShowcasesByFactory(factoryID int64, contentType string, callerID int64) ([]domain.ShowcaseByFactoryItem, error) {
	return s.repo.GetShowcasesByFactory(factoryID, contentType, callerID)
}

func (s *ShowcaseService) GetDetail(showcaseID int64) (*domain.ShowcaseDetail, error) {
	detail, err := s.repo.GetDetail(showcaseID)
	if err != nil {
		return nil, err
	}
	// Embed factory reviews (soft-fail — missing reviews should not block detail)
	if reviews, rErr := s.repo.GetFactoryReviewsForShowcase(detail.FactoryID); rErr == nil {
		detail.Reviews = reviews
	}
	return detail, nil
}

func (s *ShowcaseService) Create(showcase *domain.FactoryShowcase) error {
	return s.repo.Create(showcase)
}

func (s *ShowcaseService) CreateStructured(factoryID int64, input domain.ShowcaseWriteInput) (*domain.FactoryShowcase, error) {
	item := &domain.FactoryShowcase{
		FactoryID:       factoryID,
		ContentType:     "PD",
		Status:          "DR",
		LinkedShowcases: domain.JSONLinkArray{},
		Tags:            domain.JSONStringArray{},
	}
	mergeShowcaseInput(item, input)
	if err := s.validateShowcase(item); err != nil {
		return nil, err
	}
	if err := s.repo.Create(item); err != nil {
		return nil, err
	}
	s.upsertFactoryCategoryMap(factoryID, item)
	return item, nil
}

func (s *ShowcaseService) GetByID(showcaseID, factoryID int64) (*domain.FactoryShowcase, error) {
	return s.repo.GetByID(showcaseID, factoryID)
}

func (s *ShowcaseService) GetAnalytics(showcaseID, factoryID int64) (*domain.ShowcaseAnalytics, error) {
	return s.repo.GetAnalytics(showcaseID, factoryID)
}

func (s *ShowcaseService) Update(showcase *domain.FactoryShowcase) error {
	return s.repo.Update(showcase)
}

func (s *ShowcaseService) UpdateStructured(showcaseID, factoryID int64, input domain.ShowcaseWriteInput, replace bool) (*domain.FactoryShowcase, error) {
	existing, err := s.repo.GetByID(showcaseID, factoryID)
	if err != nil {
		return nil, err
	}
	if existing.Status == "AR" {
		onlyStatus := input.Status != nil && input.ContentType == nil && input.Title == nil && input.CategoryID == nil &&
			input.SubCategoryID == nil && input.MOQ == nil && input.BasePrice == nil && input.PromoPrice == nil &&
			input.StartDate == nil && input.EndDate == nil &&
			input.Content == nil && input.LinkedShowcases == nil && input.Excerpt == nil &&
			input.ImageURL == nil && input.Tags == nil
		if !onlyStatus {
			return nil, domain.ShowcaseValidationError{Details: []domain.ShowcaseValidationDetail{{Field: "status", Message: "archived showcase is read-only except status transitions"}}}
		}
	}

	item := existing
	if replace {
		item = &domain.FactoryShowcase{
			ShowcaseID:      showcaseID,
			FactoryID:       factoryID,
			ContentType:     existing.ContentType,
			Status:          existing.Status,
			LinkedShowcases: domain.JSONLinkArray{},
			Tags:            domain.JSONStringArray{},
		}
	}
	mergeShowcaseInput(item, input)
	if err := s.validateShowcase(item); err != nil {
		return nil, err
	}
	if err := s.repo.Update(item); err != nil {
		return nil, err
	}
	s.upsertFactoryCategoryMap(factoryID, item)
	return s.repo.GetByID(showcaseID, factoryID)
}

func (s *ShowcaseService) UpdateStatus(showcaseID, factoryID int64, status string) error {
	status = domainutil.NormalizeStatus(status)
	if !validShowcaseStatus(status) {
		return domain.ShowcaseValidationError{Details: []domain.ShowcaseValidationDetail{{Field: "status", Message: "must be one of DR, AC, HI, AR"}}}
	}
	item, err := s.repo.GetByID(showcaseID, factoryID)
	if err != nil {
		return err
	}
	item.Status = status
	if status == "AC" {
		if err := s.validateShowcase(item); err != nil {
			return err
		}
	}
	return s.repo.UpdateStatus(showcaseID, factoryID, status)
}

func (s *ShowcaseService) Delete(showcaseID, factoryID int64) error {
	return s.repo.Delete(showcaseID, factoryID)
}

func mergeShowcaseInput(item *domain.FactoryShowcase, input domain.ShowcaseWriteInput) {
	if input.ContentType != nil {
		item.ContentType = domainutil.NormalizeStatus(*input.ContentType)
	}
	if input.Status != nil {
		item.Status = domainutil.NormalizeStatus(*input.Status)
	}
	if input.Title != nil {
		item.Title = strings.TrimSpace(*input.Title)
	}
	if input.CategoryID != nil {
		item.CategoryID = input.CategoryID
	}
	if input.SubCategoryID != nil {
		item.SubCategoryID = input.SubCategoryID
	}
	if input.MOQ != nil {
		item.MOQ = input.MOQ
	}
	if input.UnitID != nil {
		item.UnitID = input.UnitID
	}
	if input.LeadTimeDays != nil {
		item.LeadTimeDays = input.LeadTimeDays
	}
	if input.BasePrice != nil {
		item.BasePrice = input.BasePrice
	}
	if input.PromoPrice != nil {
		item.PromoPrice = input.PromoPrice
	}
	if input.StartDate != nil {
		item.StartDate = input.StartDate
	}
	if input.EndDate != nil {
		item.EndDate = input.EndDate
	}
	if input.Content != nil {
		v := strings.TrimSpace(*input.Content)
		item.Content = &v
	}
	if input.LinkedShowcases != nil {
		item.LinkedShowcases = domain.JSONLinkArray(normalizeLinkedShowcases(*input.LinkedShowcases))
	}
	if input.Excerpt != nil {
		v := strings.TrimSpace(*input.Excerpt)
		item.Excerpt = &v
	}
	if input.ImageURL != nil {
		v := strings.TrimSpace(*input.ImageURL)
		item.ImageURL = &v
	}
	if input.Tags != nil {
		item.Tags = domain.JSONStringArray(*input.Tags)
	}
}

func normalizeLinkedShowcases(values []string) []string {
	out := make([]string, 0, len(values))
	for _, raw := range values {
		v := strings.TrimSpace(raw)
		if v == "" {
			continue
		}
		out = append(out, v)
	}
	return out
}

func validShowcaseStatus(status string) bool {
	switch status {
	case "DR", "AC", "HI", "AR":
		return true
	default:
		return false
	}
}

type showcaseValidationCollector struct {
	details []domain.ShowcaseValidationDetail
}

func (v *showcaseValidationCollector) add(field, message string) {
	v.details = append(v.details, domain.ShowcaseValidationDetail{Field: field, Message: message})
}

func (v *showcaseValidationCollector) err() error {
	if len(v.details) == 0 {
		return nil
	}
	return domain.ShowcaseValidationError{Details: v.details}
}

func (s *ShowcaseService) validateShowcase(item *domain.FactoryShowcase) error {
	var validation showcaseValidationCollector
	validation.validateBasicFields(item)
	validation.applyDefaultCoverImage(item)
	validation.ensureDefaults(item)
	if err := s.validateShowcaseCategoryRefs(item, validation.add); err != nil {
		return err
	}
	validation.validateContentLength(item)
	if item.Status == "AC" {
		validation.validateActiveRequirements(item)
		validation.validateActiveFieldRules(item)
	}
	if err := s.validateLinkedShowcases(item, validation.add); err != nil {
		return err
	}
	return validation.err()
}

func (v *showcaseValidationCollector) validateBasicFields(item *domain.FactoryShowcase) {
	if item.ContentType == "" {
		v.add("content_type", "must be one of PD, PM, ID, MT")
	}
	switch item.ContentType {
	case "PD", "PM", "ID", "MT":
	default:
		v.add("content_type", "must be one of PD, PM, ID, MT")
	}
	if !validShowcaseStatus(item.Status) {
		v.add("status", "must be one of DR, AC, HI, AR")
	}
	if strings.TrimSpace(item.Title) == "" {
		v.add("title", "must be non-empty")
	} else if len([]rune(strings.TrimSpace(item.Title))) > 200 {
		v.add("title", "must be 200 characters or fewer")
	}
	if item.ImageURL != nil && strings.TrimSpace(*item.ImageURL) != "" && !isShowcaseHTTPSURL(*item.ImageURL) {
		v.add("image_url", "image URL must be HTTPS")
	}
	if len(item.LinkedShowcases) > 5 {
		v.add("linked_showcases", "maximum 5 linked showcases allowed")
	}
	for _, ref := range item.LinkedShowcases {
		if !isShowcaseHTTPSURL(ref) {
			if _, err := strconv.ParseInt(ref, 10, 64); err != nil {
				v.add("linked_showcases", "all entries must be HTTPS URLs or numeric showcase IDs")
				break
			}
		}
	}
}

func (v *showcaseValidationCollector) applyDefaultCoverImage(item *domain.FactoryShowcase) {
	if (item.ImageURL == nil || strings.TrimSpace(*item.ImageURL) == "") && len(item.LinkedShowcases) > 0 {
		if isShowcaseHTTPSURL(item.LinkedShowcases[0]) {
			cover := item.LinkedShowcases[0]
			item.ImageURL = &cover
		}
	}
}

func (v *showcaseValidationCollector) ensureDefaults(item *domain.FactoryShowcase) {
	if item.Tags == nil {
		item.Tags = domain.JSONStringArray{}
	}
}

func (s *ShowcaseService) validateShowcaseCategoryRefs(item *domain.FactoryShowcase, add func(field, message string)) error {
	if item.CategoryID != nil {
		ok, err := s.repo.CategoryExists(*item.CategoryID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if !ok {
			add("category_id", "category not found")
		}
	}
	if item.SubCategoryID != nil {
		if item.CategoryID == nil {
			add("sub_category_id", "category_id is required when sub_category_id is present")
		} else {
			ok, err := s.repo.SubCategoryBelongsToCategory(*item.SubCategoryID, *item.CategoryID)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return err
			}
			if !ok {
				add("sub_category_id", "must belong to category_id")
			}
		}
	}
	return nil
}

func (v *showcaseValidationCollector) validateContentLength(item *domain.FactoryShowcase) {
	contentLen := 0
	if item.Content != nil {
		contentLen = len([]rune(*item.Content))
	}
	switch item.ContentType {
	case "PD", "PM", "MT":
		if contentLen > 50000 {
			v.add("content", "content exceeds max length")
		}
	case "ID":
		if contentLen > 20000 {
			v.add("content", "content exceeds max length")
		}
	}
}

func (v *showcaseValidationCollector) validateActiveRequirements(item *domain.FactoryShowcase) {
	if item.ContentType != "ID" && (item.ImageURL == nil || strings.TrimSpace(*item.ImageURL) == "") {
		v.add("image_url", "is required when showcase is active")
	}
	switch item.ContentType {
	case "PD", "MT":
		// base_price and lead_time_days are optional
	case "PM":
		if item.PromoPrice == nil {
			v.add("promo_price", "is required for PM")
		}
		if item.BasePrice != nil && item.PromoPrice != nil && *item.PromoPrice > *item.BasePrice {
			v.add("promo_price", "must not exceed base_price")
		}
		if item.StartDate == nil {
			v.add("start_date", "is required for PM")
		}
		if item.EndDate == nil {
			v.add("end_date", "is required for PM")
		}
	case "ID":
		if item.Content == nil || strings.TrimSpace(*item.Content) == "" {
			v.add("content", "is required for ID")
		}
	}
}

func (v *showcaseValidationCollector) validateActiveFieldRules(item *domain.FactoryShowcase) {
	switch item.ContentType {
	case "PD", "MT":
		if item.PromoPrice != nil {
			v.add("promo_price", "must be null for "+item.ContentType)
		}
		if item.StartDate != nil {
			v.add("start_date", "must be null for "+item.ContentType)
		}
		if item.EndDate != nil {
			v.add("end_date", "must be null for "+item.ContentType)
		}
	case "PM":
		if item.StartDate != nil && item.EndDate != nil && item.EndDate.Before(*item.StartDate) {
			v.add("end_date", "must be greater than or equal to start_date")
		}
	case "ID":
		if item.MOQ != nil {
			v.add("moq", "must be null for ID")
		}
		if item.BasePrice != nil {
			v.add("base_price", "must be null for ID")
		}
		if item.PromoPrice != nil {
			v.add("promo_price", "must be null for ID")
		}
		if item.StartDate != nil {
			v.add("start_date", "must be null for ID")
		}
		if item.EndDate != nil {
			v.add("end_date", "must be null for ID")
		}
	}
}

func (s *ShowcaseService) validateLinkedShowcases(item *domain.FactoryShowcase, add func(field, message string)) error {
	if len(item.LinkedShowcases) > 0 {
		ids := make([]int64, 0, len(item.LinkedShowcases))
		for _, ref := range item.LinkedShowcases {
			id, err := strconv.ParseInt(strings.TrimSpace(ref), 10, 64)
			if err == nil && id > 0 {
				ids = append(ids, id)
			}
		}
		if len(ids) > 0 {
			rows, err := s.repo.CheckLinkedShowcases(ids)
			if err != nil {
				return err
			}
			byID := map[int64]showcaserepo.LinkedShowcaseCheckRow{}
			for _, row := range rows {
				byID[row.ShowcaseID] = row
			}
			for _, ref := range item.LinkedShowcases {
				id, err := strconv.ParseInt(strings.TrimSpace(ref), 10, 64)
				if err != nil || id <= 0 {
					continue
				}
				row, ok := byID[id]
				if !ok {
					add("linked_showcases", "all linked showcases must exist")
					break
				}
				if row.FactoryID != item.FactoryID {
					add("linked_showcases", "all linked showcases must belong to the same factory")
					break
				}
				if row.Status != "AC" && row.ShowcaseID != item.ShowcaseID {
					add("linked_showcases", "all linked showcases must be active")
					break
				}
			}
		}
	}
	return nil
}

func isShowcaseHTTPSURL(raw string) bool {
	v := strings.TrimSpace(raw)
	if v == "" || len(v) > 2048 {
		return false
	}
	u, err := url.Parse(v)
	if err != nil || u.Host == "" {
		return false
	}
	if strings.EqualFold(u.Scheme, "https") {
		return true
	}
	// Dev-friendly: allow HTTP for localhost asset URLs.
	if strings.EqualFold(u.Scheme, "http") {
		host := strings.ToLower(u.Hostname())
		return host == "localhost" || host == "127.0.0.1"
	}
	return false
}

func (s *ShowcaseService) RecordView(showcaseID int64) error {
	return s.repo.IncrementViewCount(showcaseID)
}

func (s *ShowcaseService) GetHomeShowcases(types []string, limitPerType int) (map[string][]domain.ShowcaseExploreItem, error) {
	return s.repo.GetHomeShowcases(types, limitPerType)
}

func (s *ShowcaseService) GetHubShowcases(hubID int64, limitPerHub int) ([]domain.HubShowcaseGroup, error) {
	return s.repo.GetHubShowcases(hubID, limitPerHub)
}

func (s *ShowcaseService) ListPaginated(filter domain.ShowcasePaginatedFilter) (*domain.ShowcasePaginatedResponse, error) {
	items, total, err := s.repo.ListPaginated(filter)
	if err != nil {
		return nil, err
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	if items == nil {
		items = []domain.ShowcaseExploreItem{}
	}
	return &domain.ShowcasePaginatedResponse{
		Total: total,
		Page:  page,
		Limit: limit,
		Items: items,
	}, nil
}

func (s *ShowcaseService) ListHomePromoSlides(limit int) ([]domain.HomePromoSlide, error) {
	return s.repo.ListHomePromoSlides(limit)
}

func (s *ShowcaseService) ListPromoSlides() ([]domain.PromoSlide, error) {
	return s.repo.ListPromoSlides()
}

func (s *ShowcaseService) CreateImage(img *domain.ShowcaseImage, factoryID int64) error {
	return s.repo.CreateImage(img, factoryID)
}

func (s *ShowcaseService) ListImages(showcaseID, callerID int64) ([]domain.ShowcaseImage, error) {
	return s.repo.ListImages(showcaseID, callerID)
}

func (s *ShowcaseService) DeleteImage(showcaseID, imageID, factoryID int64) error {
	return s.repo.DeleteImage(showcaseID, imageID, factoryID)
}

func (s *ShowcaseService) GetSections(showcaseID, factoryID int64) ([]domain.ShowcaseSection, error) {
	return s.repo.GetSections(showcaseID, factoryID)
}

func (s *ShowcaseService) BulkReplaceSections(showcaseID, factoryID int64, inputs []domain.ShowcaseSectionInput) error {
	return s.repo.BulkReplaceSections(showcaseID, factoryID, inputs)
}

func (s *ShowcaseService) GetSpecs(showcaseID, factoryID int64) ([]domain.ShowcaseSpec, error) {
	return s.repo.GetSpecs(showcaseID, factoryID)
}

func (s *ShowcaseService) BulkReplaceSpecs(showcaseID, factoryID int64, inputs []domain.ShowcaseSpecInput) error {
	return s.repo.BulkReplaceSpecs(showcaseID, factoryID, inputs)
}

func (s *ShowcaseService) PatchImage(showcaseID, imageID, factoryID int64, sortOrder *int, caption *string) (*domain.ShowcaseImage, error) {
	return s.repo.PatchImage(showcaseID, imageID, factoryID, sortOrder, caption)
}

func (s *ShowcaseService) DeleteSection(showcaseID, sectionID, factoryID int64) error {
	return s.repo.DeleteSection(showcaseID, sectionID, factoryID)
}
