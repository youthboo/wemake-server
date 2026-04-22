package service

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
)

type ShowcaseService struct {
	repo *repository.ShowcaseRepository
}

func NewShowcaseService(repo *repository.ShowcaseRepository) *ShowcaseService {
	return &ShowcaseService{repo: repo}
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
	return s.repo.GetDetail(showcaseID)
}

func (s *ShowcaseService) Create(showcase *domain.FactoryShowcase) error {
	return s.repo.Create(showcase)
}

func (s *ShowcaseService) CreateStructured(factoryID int64, input domain.ShowcaseWriteInput) (*domain.FactoryShowcase, error) {
	item := &domain.FactoryShowcase{
		FactoryID:       factoryID,
		ContentType:     "PD",
		Status:          "DR",
		Images:          domain.JSONStringArray{},
		LinkedShowcases: domain.JSONInt64Array{},
	}
	mergeShowcaseInput(item, input)
	if item.MOQ != nil {
		item.MinOrder = item.MOQ
	}
	if item.Content != nil {
		item.Description = item.Content
	}
	if len(item.Images) > 0 {
		first := string(item.Images[0])
		item.ImageURL = &first
	}
	if err := s.validateShowcase(item); err != nil {
		return nil, err
	}
	if err := s.repo.Create(item); err != nil {
		return nil, err
	}
	item.Type = item.ContentType
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
		onlyStatus := input.Status != nil && input.Type == nil && input.Title == nil && input.CategoryID == nil &&
			input.SubCategoryID == nil && input.MOQ == nil && input.ProductionCapacity == nil &&
			input.LeadTimeDays == nil && input.BasePrice == nil && input.PromoPrice == nil &&
			input.StartDate == nil && input.EndDate == nil && input.SampleAvailable == nil &&
			input.Content == nil && input.Images == nil && input.LinkedShowcases == nil
		if !onlyStatus {
			return nil, domain.ShowcaseValidationError{Details: []domain.ShowcaseValidationDetail{{Field: "status", Message: "archived showcase is read-only except status transitions"}}}
		}
	}

	item := existing
	if replace {
		item = &domain.FactoryShowcase{
			ShowcaseID:      showcaseID,
			FactoryID:       factoryID,
			Status:          existing.Status,
			Images:          domain.JSONStringArray{},
			LinkedShowcases: domain.JSONInt64Array{},
		}
	}
	mergeShowcaseInput(item, input)
	if item.MOQ != nil {
		item.MinOrder = item.MOQ
	}
	if item.Content != nil {
		item.Description = item.Content
	}
	if len(item.Images) > 0 {
		first := string(item.Images[0])
		item.ImageURL = &first
	} else {
		item.ImageURL = nil
	}
	if err := s.validateShowcase(item); err != nil {
		return nil, err
	}
	if err := s.repo.Update(item); err != nil {
		return nil, err
	}
	return s.repo.GetByID(showcaseID, factoryID)
}

func (s *ShowcaseService) UpdateStatus(showcaseID, factoryID int64, status string) error {
	status = strings.TrimSpace(strings.ToUpper(status))
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
	if input.Type != nil {
		item.ContentType = strings.TrimSpace(strings.ToUpper(*input.Type))
		item.Type = item.ContentType
	}
	if input.Status != nil {
		item.Status = strings.TrimSpace(strings.ToUpper(*input.Status))
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
	if input.ProductionCapacity != nil {
		item.ProductionCapacity = input.ProductionCapacity
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
	if input.SampleAvailable != nil {
		item.SampleAvailable = *input.SampleAvailable
	}
	if input.Content != nil {
		v := strings.TrimSpace(*input.Content)
		item.Content = &v
	}
	if input.Images != nil {
		item.Images = domain.JSONStringArray(*input.Images)
	}
	if input.LinkedShowcases != nil {
		item.LinkedShowcases = domain.JSONInt64Array(uniquePositiveIDs(*input.LinkedShowcases))
	}
	if input.Excerpt != nil {
		v := strings.TrimSpace(*input.Excerpt)
		item.Excerpt = &v
	}
	if input.Description != nil {
		v := strings.TrimSpace(*input.Description)
		item.Description = &v
	}
	if input.ImageURL != nil {
		v := strings.TrimSpace(*input.ImageURL)
		item.ImageURL = &v
	}
	if input.PriceRange != nil {
		v := strings.TrimSpace(*input.PriceRange)
		item.PriceRange = &v
	}
}

func uniquePositiveIDs(ids []int64) []int64 {
	seen := map[int64]struct{}{}
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			out = append(out, id)
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
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

func (s *ShowcaseService) validateShowcase(item *domain.FactoryShowcase) error {
	var details []domain.ShowcaseValidationDetail
	add := func(field, message string) {
		details = append(details, domain.ShowcaseValidationDetail{Field: field, Message: message})
	}

	if item.ContentType == "" {
		add("type", "must be one of PD, PM, ID")
	}
	switch item.ContentType {
	case "PD", "PM", "ID":
	default:
		add("type", "must be one of PD, PM, ID")
	}
	if !validShowcaseStatus(item.Status) {
		add("status", "must be one of DR, AC, HI, AR")
	}
	if strings.TrimSpace(item.Title) == "" {
		add("title", "must be non-empty")
	}
	if len(item.Images) > 5 {
		add("images", "maximum 5 images allowed")
	}
	for _, image := range item.Images {
		if strings.TrimSpace(image) == "" {
			add("images", "image URL must be non-empty")
			break
		}
	}
	if len(item.LinkedShowcases) > 5 {
		add("linked_showcases", "maximum 5 linked showcases allowed")
	}
	for _, id := range item.LinkedShowcases {
		if id <= 0 {
			add("linked_showcases", "all IDs must be positive integers")
			break
		}
	}

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

	fullValidation := item.Status == "AC"
	if fullValidation {
		switch item.ContentType {
		case "PD":
			if item.MOQ == nil {
				add("moq", "is required for PD")
			}
			if item.LeadTimeDays == nil {
				add("lead_time_days", "is required for PD")
			}
			if item.BasePrice == nil {
				add("base_price", "is required for PD")
			}
		case "PM":
			if item.MOQ == nil {
				add("moq", "is required for PM")
			}
			if item.LeadTimeDays == nil {
				add("lead_time_days", "is required for PM")
			}
			if item.BasePrice == nil {
				add("base_price", "is required for PM")
			}
			if item.PromoPrice == nil {
				add("promo_price", "is required for PM")
			}
			if item.StartDate == nil {
				add("start_date", "is required for PM")
			}
			if item.EndDate == nil {
				add("end_date", "is required for PM")
			}
		case "ID":
			if item.Content == nil || strings.TrimSpace(*item.Content) == "" {
				add("content", "is required for ID")
			}
		}
	}

	switch item.ContentType {
	case "PD":
		if item.PromoPrice != nil {
			add("promo_price", "must be null for PD")
		}
		if item.StartDate != nil {
			add("start_date", "must be null for PD")
		}
		if item.EndDate != nil {
			add("end_date", "must be null for PD")
		}
		if len(item.LinkedShowcases) > 0 {
			add("linked_showcases", "must be empty for PD")
		}
	case "PM":
		if item.StartDate != nil && item.EndDate != nil && item.EndDate.Before(*item.StartDate) {
			add("end_date", "must be greater than or equal to start_date")
		}
		if item.BasePrice != nil && item.PromoPrice != nil && *item.PromoPrice > *item.BasePrice {
			add("promo_price", "must be less than or equal to base_price")
		}
		if len(item.LinkedShowcases) > 0 {
			add("linked_showcases", "must be empty for PM")
		}
	case "ID":
		if item.MOQ != nil {
			add("moq", "must be null for ID")
		}
		if item.LeadTimeDays != nil {
			add("lead_time_days", "must be null for ID")
		}
		if item.BasePrice != nil {
			add("base_price", "must be null for ID")
		}
		if item.PromoPrice != nil {
			add("promo_price", "must be null for ID")
		}
		if item.StartDate != nil {
			add("start_date", "must be null for ID")
		}
		if item.EndDate != nil {
			add("end_date", "must be null for ID")
		}
		if len(item.LinkedShowcases) > 0 {
			rows, err := s.repo.CheckLinkedShowcases([]int64(item.LinkedShowcases))
			if err != nil {
				return err
			}
			byID := map[int64]repository.LinkedShowcaseCheckRow{}
			for _, row := range rows {
				byID[row.ShowcaseID] = row
			}
			for _, id := range item.LinkedShowcases {
				row, ok := byID[id]
				if !ok {
					add("linked_showcases", "all linked showcases must exist")
					break
				}
				if row.FactoryID != item.FactoryID {
					add("linked_showcases", "all linked showcases must belong to the same factory")
					break
				}
				if row.Type != "PD" {
					add("linked_showcases", "all linked showcases must be type PD")
					break
				}
			}
		}
	}

	if len(details) > 0 {
		return domain.ShowcaseValidationError{Details: details}
	}
	return nil
}

func (s *ShowcaseService) RecordView(showcaseID int64) error {
	return s.repo.IncrementViewCount(showcaseID)
}

func (s *ShowcaseService) ListPromoSlides() ([]domain.PromoSlide, error) {
	return s.repo.ListPromoSlides()
}

func (s *ShowcaseService) CreateImage(img *domain.ShowcaseImage, factoryID int64) error {
	return s.repo.CreateImage(img, factoryID)
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
