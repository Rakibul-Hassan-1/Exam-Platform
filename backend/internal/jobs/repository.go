package jobs

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"examplatform/internal/models"
	"examplatform/internal/uuidx"
)

type Repository struct{ DB *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{DB: db} }

type CreateInput struct {
	DocumentID     *string
	TeacherID      string
	SubjectID      string
	SourceText     string
	QuestionCount  int
	DifficultyEasy int
	DifficultyMed  int
	DifficultyHard int
}

func (r *Repository) Create(ctx context.Context, in CreateInput) (*models.GenerationJob, error) {
	id := uuidx.New()
	j := &models.GenerationJob{}
	const q = `INSERT INTO generation_jobs
		(id, document_id, teacher_id, subject_id, source_text, question_count,
		 difficulty_easy, difficulty_medium, difficulty_hard, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'PENDING')
		RETURNING id, document_id, teacher_id, subject_id, question_count,
			difficulty_easy, difficulty_medium, difficulty_hard, status, created_at, updated_at`
	err := r.DB.QueryRow(ctx, q, id, in.DocumentID, in.TeacherID, in.SubjectID, in.SourceText, in.QuestionCount,
		in.DifficultyEasy, in.DifficultyMed, in.DifficultyHard).
		Scan(&j.ID, &j.DocumentID, &j.TeacherID, &j.SubjectID, &j.QuestionCount,
			&j.DifficultyEasy, &j.DifficultyMed, &j.DifficultyHard, &j.Status, &j.CreatedAt, &j.UpdatedAt)
	return j, err
}

func (r *Repository) Get(ctx context.Context, id string) (*models.GenerationJob, error) {
	j := &models.GenerationJob{}
	const q = `SELECT id, document_id, teacher_id, subject_id, question_count,
		difficulty_easy, difficulty_medium, difficulty_hard, status, COALESCE(error, ''), created_at, updated_at
		FROM generation_jobs WHERE id = $1`
	err := r.DB.QueryRow(ctx, q, id).Scan(&j.ID, &j.DocumentID, &j.TeacherID, &j.SubjectID, &j.QuestionCount,
		&j.DifficultyEasy, &j.DifficultyMed, &j.DifficultyHard, &j.Status, &j.Error, &j.CreatedAt, &j.UpdatedAt)
	return j, err
}

// claimedJob carries the fields the worker needs to execute a job.
type claimedJob struct {
	ID             string
	DocumentID     *string
	TeacherID      string
	SubjectID      string
	SourceText     string
	QuestionCount  int
	DifficultyEasy int
	DifficultyMed  int
	DifficultyHard int
}

// ClaimNextPending atomically picks the oldest PENDING job and marks it
// PROCESSING using SELECT ... FOR UPDATE SKIP LOCKED, so multiple worker
// replicas can run safely without double-processing a job.
func (r *Repository) ClaimNextPending(ctx context.Context) (*claimedJob, error) {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, `
		SELECT id, document_id, teacher_id, subject_id, source_text, question_count,
		       difficulty_easy, difficulty_medium, difficulty_hard
		FROM generation_jobs
		WHERE status = 'PENDING'
		ORDER BY created_at
		LIMIT 1
		FOR UPDATE SKIP LOCKED`)

	var j claimedJob
	err = row.Scan(&j.ID, &j.DocumentID, &j.TeacherID, &j.SubjectID, &j.SourceText, &j.QuestionCount,
		&j.DifficultyEasy, &j.DifficultyMed, &j.DifficultyHard)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if _, err := tx.Exec(ctx, `UPDATE generation_jobs SET status = 'PROCESSING', updated_at = now() WHERE id = $1`, j.ID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &j, nil
}

func (r *Repository) MarkCompleted(ctx context.Context, id string) error {
	_, err := r.DB.Exec(ctx, `UPDATE generation_jobs SET status = 'COMPLETED', updated_at = now() WHERE id = $1`, id)
	return err
}

func (r *Repository) MarkFailed(ctx context.Context, id, reason string) error {
	_, err := r.DB.Exec(ctx, `UPDATE generation_jobs SET status = 'FAILED', error = $2, updated_at = now() WHERE id = $1`, id, reason)
	return err
}
