package showcase

import (
	"database/sql"

	"github.com/yourusername/wemake/internal/domain"
)

func (r *ShowcaseRepository) Create(showcase *domain.FactoryShowcase) error {
	query := `
		INSERT INTO factory_showcases
			(factory_id, content_type, title,
			 category_id, sub_category_id, moq, unit_id, lead_time_days,
			 base_price, promo_price, start_date, end_date,
			 content, linked_showcases, status,
			 published_at, updated_at)
		VALUES
			(:factory_id, :content_type, :title,
			 :category_id, :sub_category_id, :moq, :unit_id, :lead_time_days,
			 :base_price, :promo_price, :start_date, :end_date,
			 :content, :linked_showcases,
			 COALESCE(NULLIF(:status, ''), 'DR'),
			 CASE WHEN COALESCE(NULLIF(:status, ''), 'DR') = 'AC' THEN NOW() ELSE NULL END,
			 NOW())
		RETURNING showcase_id, created_at, updated_at, published_at, likes_count, status
	`
	rows, err := r.db.NamedQuery(query, showcase)
	if err != nil {
		return err
	}
	if rows.Next() {
		err = rows.Scan(&showcase.ShowcaseID, &showcase.CreatedAt, &showcase.UpdatedAt, &showcase.PublishedAt, &showcase.LikesCount, &showcase.Status)
	}
	rows.Close()
	return err
}

func (r *ShowcaseRepository) Update(s *domain.FactoryShowcase) error {
	query := `
		UPDATE factory_showcases
		SET content_type    = :content_type,
		    title           = :title,
		    category_id     = :category_id,
		    sub_category_id = :sub_category_id,
		    moq             = :moq,
		    unit_id         = :unit_id,
		    lead_time_days  = :lead_time_days,
		    base_price      = :base_price,
		    promo_price     = :promo_price,
		    start_date      = :start_date,
		    end_date        = :end_date,
		    content         = :content,
		    linked_showcases = :linked_showcases,
		    status          = CASE WHEN :status = '' THEN status ELSE :status END,
		    published_at    = CASE
		                        WHEN :status = 'AC' AND published_at IS NULL THEN NOW()
		                        ELSE published_at
		                      END,
		    updated_at      = NOW()
		WHERE showcase_id = :showcase_id AND factory_id = :factory_id
	`
	res, err := r.db.NamedExec(query, s)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *ShowcaseRepository) UpdateStatus(showcaseID, factoryID int64, status string) error {
	res, err := r.db.Exec(`
		UPDATE factory_showcases
		SET status = $1,
		    published_at = CASE WHEN $1 = 'AC' AND published_at IS NULL THEN NOW() ELSE published_at END,
		    updated_at = NOW()
		WHERE showcase_id = $2 AND factory_id = $3
	`, status, showcaseID, factoryID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *ShowcaseRepository) IncrementViewCount(showcaseID int64) error {
	_, err := r.db.Exec(`UPDATE factory_showcases SET view_count = view_count + 1 WHERE showcase_id = $1`, showcaseID)
	return err
}

func (r *ShowcaseRepository) Delete(showcaseID, factoryID int64) error {
	res, err := r.db.Exec(`DELETE FROM factory_showcases WHERE showcase_id = $1 AND factory_id = $2`, showcaseID, factoryID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
