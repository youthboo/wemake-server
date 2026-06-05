package dto

// Factory Request DTOs
type CreateFactoryRequest struct {
	FactoryName    string  `json:"factory_name" validate:"notblank"`
	TaxID          string  `json:"tax_id"`
	ProvinceID     *int64  `json:"province_id"`
	CategoryIDs    []int64 `json:"category_ids"`
	SubCategoryIDs []int64 `json:"sub_category_ids"`
	// Cert fields (optional — skipped if CertID == 0 or DocumentURL == "")
	CertID         int64  `json:"cert_id"`
	DocumentURL    string `json:"document_url"`
	CertNumber     string `json:"cert_number"`
	CertExpireDate string `json:"cert_expire_date"` // "YYYY-MM-DD" or ""
}

type PatchFactoryProfileRequest struct {
	FactoryName        *string `json:"factory_name"`
	TaxID              *string `json:"tax_id"`
	Description        *string `json:"description"`
	ImageURL           *string `json:"image_url"`
	BackgroundImageURL *string `json:"background_image_url"`
	LeadTimeDesc       *string `json:"lead_time_desc"`
}

type SaveProfileRequest struct {
	FactoryName        string  `json:"factory_name" validate:"notblank"`
	TaxID              *string `json:"tax_id"`
	Description        *string `json:"description"`
	ImageURL           *string `json:"image_url"`
	BackgroundImageURL *string `json:"background_image_url"`
	LeadTimeDesc       *string `json:"lead_time_desc"`
	CategoryIDs        []int64 `json:"category_ids"`
	SubCategoryIDs     []int64 `json:"sub_category_ids"`
}

type AddCategoryRequest struct {
	CategoryID    int64 `json:"category_id" validate:"gt=0"`
	SubCategoryID *int64 `json:"sub_category_id"`
}

type ReplaceCategoriesRequest struct {
	CategoryIDs []int64 `json:"category_ids"`
}

type AddSubCategoryRequest struct {
	CategoryID    int64 `json:"category_id" validate:"gt=0"`
	SubCategoryID int64 `json:"sub_category_id" validate:"gt=0"`
}

type ReplaceSubCategoriesRequest struct {
	SubCategoryIDs []int64 `json:"sub_category_ids"`
}

type AssignFactoryConfigRequest struct {
	ConfigID int64 `json:"config_id" validate:"gt=0"`
}
