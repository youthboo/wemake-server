package catalog

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/domainutil"
)

type CatalogRepository struct {
	db *sqlx.DB
}

func NewCatalogRepository(db *sqlx.DB) *CatalogRepository {
	return &CatalogRepository{db: db}
}

func (r *CatalogRepository) GetCategories(scope string, limit int) ([]domain.Category, error) {
	var categories []domain.Category
	scope = domainutil.NormalizeStatus(scope)
	query := `SELECT c.category_id, c.name, h.scope, c.hub_id
		FROM lbi_categories c
		JOIN lbi_hub h ON h.hub_id = c.hub_id`
	args := []interface{}{}
	if scope != "" && scope != domain.CatalogScopeAll {
		query += " WHERE h.scope = $1"
		args = append(args, scope)
	}
	query += " ORDER BY CASE WHEN h.scope = 'PD' THEN 0 ELSE 1 END, c.category_id ASC"
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", len(args)+1)
		args = append(args, limit)
	}
	err := r.db.Select(&categories, query, args...)
	return categories, err
}

func (r *CatalogRepository) GetCategoriesByHub(hubID int64) ([]domain.Category, error) {
	var categories []domain.Category
	query := `SELECT c.category_id, c.name, h.scope, c.hub_id
		FROM lbi_categories c
		JOIN lbi_hub h ON h.hub_id = c.hub_id
		WHERE c.hub_id = $1
		ORDER BY c.category_id ASC`
	err := r.db.Select(&categories, query, hubID)
	return categories, err
}

func (r *CatalogRepository) GetSubCategories(categoryID int64) ([]domain.SubCategory, error) {
	var subCategories []domain.SubCategory
	query := `
		SELECT sub_category_id, category_id, name, sort_order, status
		FROM lbi_sub_categories
		WHERE category_id = $1 AND status = '1'
		ORDER BY sub_category_id ASC, sort_order ASC
	`
	err := r.db.Select(&subCategories, query, categoryID)
	return subCategories, err
}

func (r *CatalogRepository) GetAllSubCategories(scope string) ([]domain.SubCategory, error) {
	var subs []domain.SubCategory
	query := `
		SELECT s.sub_category_id, s.category_id, s.name, s.sort_order, s.status
		FROM lbi_sub_categories s
		JOIN lbi_categories c ON c.category_id = s.category_id
		JOIN lbi_hub h ON h.hub_id = c.hub_id
		WHERE s.status = '1'
	`
	args := []interface{}{}
	if scope != "" && scope != domain.CatalogScopeAll {
		query += " AND h.scope = $1"
		args = append(args, scope)
	}
	query += " ORDER BY s.category_id ASC, s.sub_category_id ASC, s.sort_order ASC"
	err := r.db.Select(&subs, query, args...)
	return subs, err
}

func (r *CatalogRepository) GetAllSubCategoriesByHub(hubID int64) ([]domain.SubCategory, error) {
	var subs []domain.SubCategory
	query := `
		SELECT s.sub_category_id, s.category_id, s.name, s.sort_order, s.status
		FROM lbi_sub_categories s
		JOIN lbi_categories c ON c.category_id = s.category_id
		WHERE s.status = '1' AND c.hub_id = $1
		ORDER BY s.category_id ASC, s.sub_category_id ASC, s.sort_order ASC
	`
	err := r.db.Select(&subs, query, hubID)
	return subs, err
}

func (r *CatalogRepository) GetCategoriesWithSubs(scope string, limit int) ([]domain.CategoryWithSubs, error) {
	cats, err := r.GetCategories(scope, limit)
	if err != nil {
		return nil, err
	}
	allSubs, err := r.GetAllSubCategories(scope)
	if err != nil {
		return nil, err
	}
	subsByCategory := make(map[int64][]domain.SubCategory, len(cats))
	for _, sub := range allSubs {
		subsByCategory[sub.CategoryID] = append(subsByCategory[sub.CategoryID], sub)
	}
	result := make([]domain.CategoryWithSubs, 0, len(cats))
	for _, cat := range cats {
		subs := subsByCategory[cat.CategoryID]
		if subs == nil {
			subs = []domain.SubCategory{}
		}
		result = append(result, domain.CategoryWithSubs{
			CategoryID:    cat.CategoryID,
			Name:          cat.Name,
			Scope:         cat.Scope,
			HubID:         cat.HubID,
			SubCategories: subs,
		})
	}
	return result, nil
}

func (r *CatalogRepository) GetHubs(scope string) ([]domain.HubWithCategories, error) {
	scope = domainutil.NormalizeStatus(scope)

	var hubs []domain.Hub
	hubQuery := "SELECT hub_id, name, scope FROM lbi_hub"
	hubArgs := []interface{}{}
	if scope != "" && scope != domain.CatalogScopeAll {
		hubQuery += " WHERE scope = $1"
		hubArgs = append(hubArgs, scope)
	}
	hubQuery += " ORDER BY sort_order ASC, hub_id ASC"
	if err := r.db.Select(&hubs, hubQuery, hubArgs...); err != nil {
		return nil, err
	}

	type catRow struct {
		HubID        int64  `db:"hub_id"`
		CategoryID   int64  `db:"category_id"`
		Name         string `db:"name"`
		FactoryCount int64  `db:"factory_count"`
	}
	catQuery := `
		SELECT c.hub_id, c.category_id, c.name,
			COUNT(DISTINCT CASE WHEN fp.approval_status = 'AP' AND u.is_active = TRUE THEN mfc.factory_id END) AS factory_count
		FROM lbi_categories c
		JOIN lbi_hub h ON h.hub_id = c.hub_id
		LEFT JOIN map_factory_categories mfc ON mfc.category_id = c.category_id
		LEFT JOIN factory_profiles fp ON fp.user_id = mfc.factory_id
		LEFT JOIN users u ON u.user_id = mfc.factory_id
	`
	catArgs := []interface{}{}
	if scope != "" && scope != domain.CatalogScopeAll {
		catQuery += " WHERE h.scope = $1"
		catArgs = append(catArgs, scope)
	}
	catQuery += " GROUP BY c.hub_id, c.category_id, c.name, h.sort_order ORDER BY h.sort_order ASC, c.hub_id ASC, c.category_id ASC"
	var catRows []catRow
	if err := r.db.Select(&catRows, catQuery, catArgs...); err != nil {
		return nil, err
	}

	type subRow struct {
		CategoryID int64  `db:"category_id"`
		Name       string `db:"name"`
		SortOrder  int64  `db:"sort_order"`
	}
	subQuery := `
		SELECT s.category_id, s.name, s.sort_order
		FROM lbi_sub_categories s
		JOIN lbi_categories c ON c.category_id = s.category_id
		JOIN lbi_hub h ON h.hub_id = c.hub_id
		WHERE s.status = '1' AND s.name <> 'ทั้งหมด'
	`
	subArgs := []interface{}{}
	if scope != "" && scope != domain.CatalogScopeAll {
		subQuery += " AND h.scope = $1"
		subArgs = append(subArgs, scope)
	}
	subQuery += " ORDER BY s.category_id ASC, s.sort_order ASC, s.sub_category_id ASC"
	var subRows []subRow
	if err := r.db.Select(&subRows, subQuery, subArgs...); err != nil {
		return nil, err
	}

	// Build sub_preview map (category_id → first 3 sub names)
	subPreviewMap := make(map[int64][]string)
	for _, s := range subRows {
		prev := subPreviewMap[s.CategoryID]
		if len(prev) < 3 {
			subPreviewMap[s.CategoryID] = append(prev, s.Name)
		}
	}

	// Build categories per hub
	catsByHub := make(map[int64][]domain.CategoryForHub)
	for _, cr := range catRows {
		sp := subPreviewMap[cr.CategoryID]
		if sp == nil {
			sp = []string{}
		}
		catsByHub[cr.HubID] = append(catsByHub[cr.HubID], domain.CategoryForHub{
			CategoryID:   cr.CategoryID,
			Name:         cr.Name,
			FactoryCount: cr.FactoryCount,
			SubPreview:   sp,
		})
	}

	result := make([]domain.HubWithCategories, 0, len(hubs))
	for _, h := range hubs {
		cats := catsByHub[h.HubID]
		if cats == nil {
			cats = []domain.CategoryForHub{}
		}
		result = append(result, domain.HubWithCategories{
			HubID:      h.HubID,
			Name:       h.Name,
			Scope:      h.Scope,
			Categories: cats,
		})
	}
	return result, nil
}

func (r *CatalogRepository) GetUnits() ([]domain.Unit, error) {
	var units []domain.Unit
	query := `
		SELECT unit_id, unit_name_th AS name, unit_name_en
		FROM lbi_units
		WHERE status = '1'
		ORDER BY unit_id ASC
	`
	err := r.db.Select(&units, query)
	return units, err
}
