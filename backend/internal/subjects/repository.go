package subjects

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"examplatform/internal/models"
	"examplatform/internal/uuidx"
)

type Repository struct{ DB *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{DB: db} }

func (r *Repository) List(ctx context.Context) ([]models.Subject, error) {
	rows, err := r.DB.Query(ctx, `SELECT id, name, created_by, created_at FROM subjects ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Subject
	for rows.Next() {
		var s models.Subject
		if err := rows.Scan(&s.ID, &s.Name, &s.CreatedBy, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *Repository) Create(ctx context.Context, name, createdBy string) (*models.Subject, error) {
	id := uuidx.New()
	s := &models.Subject{}
	const q = `INSERT INTO subjects (id, name, created_by) VALUES ($1, $2, $3)
		RETURNING id, name, created_by, created_at`
	err := r.DB.QueryRow(ctx, q, id, name, createdBy).Scan(&s.ID, &s.Name, &s.CreatedBy, &s.CreatedAt)
	return s, err
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	_, err := r.DB.Exec(ctx, `DELETE FROM subjects WHERE id = $1`, id)
	return err
}
