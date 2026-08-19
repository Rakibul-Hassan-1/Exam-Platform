package documents

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"examplatform/internal/models"
	"examplatform/internal/uuidx"
)

type Repository struct{ DB *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{DB: db} }

func (r *Repository) Create(ctx context.Context, teacherID, subjectID, filename, storagePath, status string) (*models.Document, error) {
	id := uuidx.New()
	var subjPtr *string
	if subjectID != "" {
		subjPtr = &subjectID
	}
	d := &models.Document{}
	const q = `INSERT INTO documents (id, teacher_id, subject_id, filename, storage_path, status)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id, teacher_id, subject_id, filename, storage_path, status, created_at`
	err := r.DB.QueryRow(ctx, q, id, teacherID, subjPtr, filename, storagePath, status).
		Scan(&d.ID, &d.TeacherID, &d.SubjectID, &d.Filename, &d.StoragePath, &d.Status, &d.CreatedAt)
	return d, err
}

func (r *Repository) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := r.DB.Exec(ctx, `UPDATE documents SET status = $2 WHERE id = $1`, id, status)
	return err
}

func (r *Repository) Get(ctx context.Context, id string) (*models.Document, error) {
	d := &models.Document{}
	const q = `SELECT id, teacher_id, subject_id, filename, storage_path, status, created_at
		FROM documents WHERE id = $1`
	err := r.DB.QueryRow(ctx, q, id).
		Scan(&d.ID, &d.TeacherID, &d.SubjectID, &d.Filename, &d.StoragePath, &d.Status, &d.CreatedAt)
	return d, err
}

func (r *Repository) List(ctx context.Context, teacherID string) ([]models.Document, error) {
	rows, err := r.DB.Query(ctx, `SELECT id, teacher_id, subject_id, filename, storage_path, status, created_at
		FROM documents WHERE teacher_id = $1 ORDER BY created_at DESC`, teacherID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Document
	for rows.Next() {
		var d models.Document
		if err := rows.Scan(&d.ID, &d.TeacherID, &d.SubjectID, &d.Filename, &d.StoragePath, &d.Status, &d.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *Repository) SaveChunks(ctx context.Context, documentID string, chunks []string) error {
	batch := &strings.Builder{}
	_ = batch
	for i, c := range chunks {
		if _, err := r.DB.Exec(ctx,
			`INSERT INTO document_chunks (id, document_id, chunk_index, content) VALUES ($1,$2,$3,$4)`,
			uuidx.New(), documentID, i, c); err != nil {
			return err
		}
	}
	return nil
}

// ConcatenatedText returns all chunks for a document joined back into
// one text blob for AI processing.
func (r *Repository) ConcatenatedText(ctx context.Context, documentID string) (string, error) {
	rows, err := r.DB.Query(ctx,
		`SELECT content FROM document_chunks WHERE document_id = $1 ORDER BY chunk_index`, documentID)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var sb strings.Builder
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return "", err
		}
		sb.WriteString(c)
		sb.WriteString("\n")
	}
	return sb.String(), rows.Err()
}
