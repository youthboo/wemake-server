package order

import (
	"strings"
	"time"

	"github.com/yourusername/wemake/internal/domain"
	domainstatus "github.com/yourusername/wemake/internal/domain/status"
	"github.com/yourusername/wemake/internal/helper"
	orderrepo "github.com/yourusername/wemake/internal/repository/order"
	"github.com/yourusername/wemake/internal/domainutil"
)

func (s *OrderService) List(userID int64, role string, status string, rfqID *int64, requestKind string) ([]domain.OrderListItem, error) {
	st := domainutil.NormalizeStatus(status)
	kinds := normalizeOrderRequestKinds(requestKind)
	if role == domain.RoleFactory {
		return s.repo.ListEnrichedByFactoryID(userID, st, rfqID, kinds)
	}
	return s.repo.ListEnrichedByUserID(userID, st, rfqID, kinds)
}

func normalizeOrderRequestKinds(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		item := domainutil.NormalizeStatus(part)
		switch item {
		case domain.RequestKindProduction, domain.RequestKindProductSample, domain.RequestKindMaterialSample:
		default:
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func (s *OrderService) GetByID(orderID, userID int64, role string) (*domain.Order, error) {
	detail, err := s.GetDetailByID(orderID, userID, role)
	if err != nil {
		return nil, err
	}
	return &domain.Order{
		OrderID:           detail.OrderID,
		QuotationID:       detail.QuotationID,
		UserID:            detail.CustomerUserID,
		FactoryID:         detail.FactoryID,
		TotalAmount:       detail.TotalAmount,
		DepositAmount:     detail.DepositAmount,
		Status:            detail.Status,
		EstimatedDelivery: detail.EstimatedDelivery,
		TrackingNo:        detail.TrackingNo,
		Courier:           detail.Courier,
		ShippedAt:         detail.ShippedAt,
		CreatedAt:         detail.CreatedAt,
		UpdatedAt:         detail.UpdatedAt,
	}, nil
}

func (s *OrderService) GetDetailByID(orderID, userID int64, role string) (*domain.OrderDetailResponse, error) {
	row, err := s.repo.GetDetailByParticipant(orderID, userID, role)
	if err != nil {
		return nil, err
	}
	if row.UserID != userID && row.FactoryID != userID {
		return nil, domain.ErrForbidden
	}
	return s.buildOrderDetailResponse(row)
}

func (s *OrderService) GetAdminDetailByID(orderID int64) (*domain.OrderDetailResponse, error) {
	row, err := s.repo.GetDetailByParticipant(orderID, 0, "")
	if err != nil {
		return nil, err
	}
	return s.buildOrderDetailResponse(row)
}

func (s *OrderService) ListActivity(orderID int64) ([]domain.OrderActivityEntry, error) {
	return s.repo.ListActivity(orderID)
}

func (s *OrderService) buildOrderDetailResponse(row *orderrepo.OrderDetailRow) (*domain.OrderDetailResponse, error) {
	images, err := s.repo.GetRfqImages(row.RFQID)
	if err != nil {
		return nil, err
	}
	depositDueDate := deriveDepositDueDate(row)
	nowTH := time.Now().In(thailandLocation)
	depositPaidAt := s.depositPaidAt(row.OrderID)
	finalPaidAt := s.finalPaymentPaidAt(row.OrderID)
	statusCode := domainstatus.NormalizeOrder(row.Status)
	rfqDetails := ""
	if row.RFQDetails != nil {
		rfqDetails = *row.RFQDetails
	}
	rfqCategoryName := ""
	if row.RFQCategoryName != nil {
		rfqCategoryName = *row.RFQCategoryName
	}
	shippingDays := getShippingDays(s.db)
	leadTimeDays := int(row.LeadTimeDays)

	// Build delivery address nested object (only if we have address data)
	var rfqAddress *domain.RfqAddressNested
	if row.RFQAddrDetail != nil || row.RFQAddrProvince != nil {
		rfqAddress = &domain.RfqAddressNested{
			AddressDetail:   derefStr(row.RFQAddrDetail),
			SubDistrictName: derefStr(row.RFQAddrSubDistrict),
			DistrictName:    derefStr(row.RFQAddrDistrict),
			ProvinceName:    derefStr(row.RFQAddrProvince),
			ZipCode:         derefStr(row.RFQAddrZipCode),
		}
	}

	// Build certifications list from pq.StringArray
	certs := make(domain.StringArray, len(row.RFQCertifications))
	copy(certs, row.RFQCertifications)

	return &domain.OrderDetailResponse{
		OrderID:           row.OrderID,
		QuotationID:       row.QuotationID,
		UserID:            row.UserID,
		FactoryID:         row.FactoryID,
		TotalAmount:       helper.MoneyDecimal(row.TotalAmount),
		DepositAmount:     helper.MoneyDecimal(row.DepositAmount),
		Status:            statusCode,
		StatusLabelTH:     domainstatus.OrderLabelTH(statusCode),
		PaymentType:       row.PaymentType,
		Currency:          "THB",
		Factory:           domain.OrderFactorySummary{FactoryID: row.FactoryID, Name: row.FactoryName, Phone: row.FactoryPhone, Address: row.FactoryProvince},
		CustomerUserID:    row.UserID,
		CustomerName:      row.CustomerName,
		CustomerPhone:     row.CustomerPhone,
		EstimatedDelivery: timePtrInTH(row.EstimatedDelivery),
		ShippingDays:      shippingDays,
		LeadTimeDays:      &leadTimeDays,
		TrackingNo:        row.TrackingNo,
		Courier:           row.Courier,
		ShippedAt:         timePtrInTH(row.ShippedAt),
		CompletedAt:       timePtrInTH(row.CompletedAt),
		CreatedAt:         row.CreatedAt.In(thailandLocation),
		UpdatedAt:         row.UpdatedAt.In(thailandLocation),
		NextAction:        buildNextAction(row, statusCode, depositDueDate, depositPaidAt, finalPaidAt, nowTH),
		PaymentSchedule:   buildPaymentSchedule(row, statusCode, depositDueDate, depositPaidAt, finalPaidAt),
		RFQ: domain.RfqNested{
			RfqID:               row.RFQID,
			Title:               row.RFQTitle,
			Details:             rfqDetails,
			Quantity:            row.RFQQuantity,
			UnitName:            "",
			BudgetPerPiece:      helper.MoneyDecimal(row.RFQBudget),
			CategoryID:          row.RFQCategoryID,
			CategoryName:        rfqCategoryName,
			RequestKind:         row.RFQRequestKind,
			SubCategoryID:       row.RFQSubCategoryID,
			SubCategoryName:     row.RFQSubCategoryName,
			ShippingMethodName:  row.RFQShippingMethodName,
			MaterialGrade:       row.RFQMaterialGrade,
			Certifications:      certs,
			TargetLeadTimeDays:  row.RFQTargetLeadTimeDays,
			TargetPrice:         row.RFQTargetPrice,
			Address:             rfqAddress,
			CreatedAt:           row.RFQCreatedAt.In(thailandLocation),
			Images:              images,
		},
		Quotation: domain.QuoteNested{
			QuoteID:          row.QuotationID,
			PricePerPiece:    helper.MoneyDecimal(row.PricePerPiece),
			MoldCost:         helper.MoneyDecimal(row.MoldCost),
			ToolingMoldCost:  helper.MoneyDecimal(row.MoldCost),
			LeadTimeDays:     row.LeadTimeDays,
			GrandTotal:       helper.MoneyDecimal(row.QuoteGrandTotal),
			Subtotal:         helper.MoneyDecimal(row.QuoteSubtotal),
			DiscountAmount:   helper.MoneyDecimal(row.QuoteDiscountAmount),
			ShippingCost:     helper.MoneyDecimal(row.QuoteShippingCost),
			PackagingCost:    helper.MoneyDecimal(row.QuotePackagingCost),
			VatRate:          helper.MoneyDecimal(row.QuoteVatRate),
			VatAmount:        helper.MoneyDecimal(row.QuoteVatAmount),
			ValidityDays:     row.QuoteValidityDays,
			ValidUntil:       row.QuoteValidUntil,
			PaymentTerms:     row.QuotePaymentTerms,
			ImageURLs:        domain.StringArray(row.QuoteImageURLs),
			FactoryHighlight: row.QuoteFactoryHighlight,
			FactoryNote:      row.QuoteFactoryNote,
		},
	}, nil
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
