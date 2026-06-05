package dto

// Auth Request DTOs
type RegisterRequest struct {
	Role           string  `json:"role" validate:"notblank"`
	Email          string  `json:"email" validate:"notblank"`
	Phone          string  `json:"phone"`
	Password       string  `json:"password" validate:"notblank"`
	FirstName      string  `json:"first_name"`
	LastName       string  `json:"last_name"`
	FactoryName    string  `json:"factory_name"`
	TaxID          string  `json:"tax_id"`
	ProvinceID     *int64  `json:"province_id"`
	CategoryIDs    []int64 `json:"category_ids"`
	SubCategoryIDs []int64 `json:"sub_category_ids"`
	// Cert fields for FT role (optional — skipped if CertID == 0 or DocumentURL == "")
	CertID         int64  `json:"cert_id"`
	DocumentURL    string `json:"document_url"`
	CertNumber     string `json:"cert_number"`
	CertExpireDate string `json:"cert_expire_date"`
	// Address fields for FT role — creates a default address row
	AddressDetail string `json:"address_detail"`
	SubDistrictID int64  `json:"sub_district_id"`
	DistrictID    int64  `json:"district_id"`
	ZipCode       string `json:"zip_code"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"notblank"`
	Password string `json:"password" validate:"notblank"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"notblank"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" validate:"notblank"`
	NewPassword string `json:"new_password" validate:"notblank"`
}

// UpgradeToFactoryRequest is used when an authenticated CT user adds a factory
// profile. No role/email/password fields — those come from the JWT session.
type UpgradeToFactoryRequest struct {
	FactoryName    string  `json:"factory_name" validate:"notblank"`
	TaxID          string  `json:"tax_id"`
	ProvinceID     *int64  `json:"province_id"`
	CategoryIDs    []int64 `json:"category_ids"`
	SubCategoryIDs []int64 `json:"sub_category_ids"`
	CertID         int64   `json:"cert_id"`
	DocumentURL    string  `json:"document_url"`
	CertNumber     string  `json:"cert_number"`
	CertExpireDate string  `json:"cert_expire_date"`
	// Address fields — creates a default address row
	AddressDetail string `json:"address_detail"`
	SubDistrictID int64  `json:"sub_district_id"`
	DistrictID    int64  `json:"district_id"`
	ZipCode       string `json:"zip_code"`
}

type UpgradeToCustomerRequest struct {
	FirstName string `json:"first_name" validate:"notblank"`
	LastName  string `json:"last_name" validate:"notblank"`
}

// SwitchRoleRequest lets a dual-profile user toggle their active role.
type SwitchRoleRequest struct {
	Role string `json:"role" validate:"notblank"`
}
