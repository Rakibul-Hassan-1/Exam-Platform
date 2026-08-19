package questions

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"

	"examplatform/internal/models"
	"examplatform/internal/uuidx"
)

type Repository struct{ DB *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{DB: db} }

type ListFilter struct {
	SubjectID string
	Status    string
}

func (r *Repository) List(ctx context.Context, f ListFilter) ([]models.Question, error) {
	query := `SELECT id, subject_id, question, options, correct_index, difficulty, explanation,
		source, status, created_by, generation_job_id, created_at FROM questions WHERE 1=1`
	args := []any{}
	if f.SubjectID != "" {
		args = append(args, f.SubjectID)
		query += " AND subject_id = $" + strconv.Itoa(len(args))
	}
	if f.Status != "" {
		args = append(args, f.Status)
		query += " AND status = $" + strconv.Itoa(len(args))
	}
	query += " ORDER BY created_at DESC"

	rows, err := r.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Question
	for rows.Next() {
		q, err := scanQuestion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, q)
	}
	return out, rows.Err()
}

func (r *Repository) GetByIDs(ctx context.Context, ids []string) ([]models.Question, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.DB.Query(ctx, `SELECT id, subject_id, question, options, correct_index, difficulty,
		explanation, source, status, created_by, generation_job_id, created_at
		FROM questions WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Question
	for rows.Next() {
		q, err := scanQuestion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, q)
	}
	return out, rows.Err()
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanQuestion(row rowScanner) (models.Question, error) {
	var q models.Question
	var optionsJSON []byte
	var genJobID *string
	err := row.Scan(&q.ID, &q.SubjectID, &q.Question, &optionsJSON, &q.CorrectIndex, &q.Difficulty,
		&q.Explanation, &q.Source, &q.Status, &q.CreatedBy, &genJobID, &q.CreatedAt)
	if err != nil {
		return q, err
	}
	q.GenerationJobID = genJobID
	if err := json.Unmarshal(optionsJSON, &q.Options); err != nil {
		return q, err
	}
	return q, nil
}

type CreateInput struct {
	SubjectID       string
	Question        string
	Options         []string
	CorrectIndex    int
	Difficulty      models.Difficulty
	Explanation     string
	Source          models.QuestionSource
	Status          models.QuestionStatus
	CreatedBy       string
	GenerationJobID *string
}

func (r *Repository) Create(ctx context.Context, in CreateInput) (*models.Question, error) {
	id := uuidx.New()
	optionsJSON, err := json.Marshal(in.Options)
	if err != nil {
		return nil, err
	}
	const q = `INSERT INTO questions
		(id, subject_id, question, options, correct_index, difficulty, explanation, source, status, created_by, generation_job_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id, subject_id, question, options, correct_index, difficulty, explanation, source, status, created_by, generation_job_id, created_at`
	row := r.DB.QueryRow(ctx, q, id, in.SubjectID, in.Question, optionsJSON, in.CorrectIndex, in.Difficulty,
		in.Explanation, in.Source, in.Status, in.CreatedBy, in.GenerationJobID)

	question, err := scanQuestion(row)
	return &question, err
}

type UpdateInput struct {
	Question     *string
	Options      *[]string
	CorrectIndex *int
	Difficulty   *models.Difficulty
	Status       *models.QuestionStatus
}

func (r *Repository) Update(ctx context.Context, id string, in UpdateInput) (*models.Question, error) {
	// Simple, explicit partial update — keeps the SQL readable over a
	// fully dynamic query builder for a handful of optional fields.
	current, err := r.GetByIDs(ctx, []string{id})
	if err != nil || len(current) == 0 {
		return nil, err
	}
	q := current[0]
	if in.Question != nil {
		q.Question = *in.Question
	}
	if in.Options != nil {
		q.Options = *in.Options
	}
	if in.CorrectIndex != nil {
		q.CorrectIndex = *in.CorrectIndex
	}
	if in.Difficulty != nil {
		q.Difficulty = *in.Difficulty
	}
	if in.Status != nil {
		q.Status = *in.Status
	}

	optionsJSON, err := json.Marshal(q.Options)
	if err != nil {
		return nil, err
	}
	const upd = `UPDATE questions SET question=$2, options=$3, correct_index=$4, difficulty=$5, status=$6
		WHERE id=$1
		RETURNING id, subject_id, question, options, correct_index, difficulty, explanation, source, status, created_by, generation_job_id, created_at`
	row := r.DB.QueryRow(ctx, upd, id, q.Question, optionsJSON, q.CorrectIndex, q.Difficulty, q.Status)
	updated, err := scanQuestion(row)
	return &updated, err
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	_, err := r.DB.Exec(ctx, `DELETE FROM questions WHERE id = $1`, id)
	return err
}
