package exams

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"examplatform/internal/models"
	"examplatform/internal/uuidx"
)

var ErrNotFound = errors.New("not found")
var ErrAlreadyAttempted = errors.New("you have already attempted this exam")
var ErrAttemptNotFound = errors.New("no in-progress attempt found for this exam")

type Repository struct{ DB *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{DB: db} }

type CreateInput struct {
	Title       string
	SubjectID   string
	DurationMin int
	CreatedBy   string
	QuestionIDs []string
}

func (r *Repository) Create(ctx context.Context, in CreateInput) (*models.Exam, error) {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	id := uuidx.New()
	totalMarks := len(in.QuestionIDs)
	const q = `INSERT INTO exams (id, title, subject_id, duration_min, total_marks, passing_marks, published, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,false,$7)
		RETURNING id, title, subject_id, duration_min, total_marks, passing_marks, published, created_by, created_at`
	e := &models.Exam{}
	passingMarks := (totalMarks + 1) / 2 // default: 50% to pass
	err = tx.QueryRow(ctx, q, id, in.Title, in.SubjectID, in.DurationMin, totalMarks, passingMarks, in.CreatedBy).
		Scan(&e.ID, &e.Title, &e.SubjectID, &e.DurationMin, &e.TotalMarks, &e.PassingMarks, &e.Published, &e.CreatedBy, &e.CreatedAt)
	if err != nil {
		return nil, err
	}

	for i, qid := range in.QuestionIDs {
		if _, err := tx.Exec(ctx,
			`INSERT INTO exam_questions (exam_id, question_id, position) VALUES ($1,$2,$3)`,
			id, qid, i); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	e.QuestionIDs = in.QuestionIDs
	return e, nil
}

func (r *Repository) List(ctx context.Context, publishedOnly bool) ([]models.Exam, error) {
	query := `SELECT id, title, subject_id, duration_min, total_marks, passing_marks, published, created_by, created_at
		FROM exams`
	if publishedOnly {
		query += ` WHERE published = true`
	}
	query += ` ORDER BY created_at DESC`

	rows, err := r.DB.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Exam
	for rows.Next() {
		var e models.Exam
		if err := rows.Scan(&e.ID, &e.Title, &e.SubjectID, &e.DurationMin, &e.TotalMarks, &e.PassingMarks,
			&e.Published, &e.CreatedBy, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *Repository) Get(ctx context.Context, id string) (*models.Exam, error) {
	e := &models.Exam{}
	const q = `SELECT id, title, subject_id, duration_min, total_marks, passing_marks, published, created_by, created_at
		FROM exams WHERE id = $1`
	err := r.DB.QueryRow(ctx, q, id).
		Scan(&e.ID, &e.Title, &e.SubjectID, &e.DurationMin, &e.TotalMarks, &e.PassingMarks, &e.Published, &e.CreatedBy, &e.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	rows, err := r.DB.Query(ctx, `SELECT question_id FROM exam_questions WHERE exam_id = $1 ORDER BY position`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var qid string
		if err := rows.Scan(&qid); err != nil {
			return nil, err
		}
		e.QuestionIDs = append(e.QuestionIDs, qid)
	}
	return e, rows.Err()
}

func (r *Repository) SetPublished(ctx context.Context, id string, published bool) error {
	_, err := r.DB.Exec(ctx, `UPDATE exams SET published = $2 WHERE id = $1`, id, published)
	return err
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	_, err := r.DB.Exec(ctx, `DELETE FROM exams WHERE id = $1`, id)
	return err
}

// --- Attempts ---

// StartAttempt creates an IN_PROGRESS attempt if one does not already
// exist for this student+exam. The backend records started_at itself —
// it remains the authoritative source for exam timing, not the client.
func (r *Repository) StartAttempt(ctx context.Context, examID, studentID string, totalCount, totalMarks int) (*models.ExamAttempt, error) {
	var existing int
	err := r.DB.QueryRow(ctx, `SELECT count(*) FROM exam_attempts WHERE exam_id = $1 AND student_id = $2`, examID, studentID).Scan(&existing)
	if err != nil {
		return nil, err
	}
	if existing > 0 {
		return nil, ErrAlreadyAttempted
	}

	id := uuidx.New()
	a := &models.ExamAttempt{}
	const q = `INSERT INTO exam_attempts (id, exam_id, student_id, status, started_at, total_count, total_marks)
		VALUES ($1,$2,$3,'IN_PROGRESS', now(), $4, $5)
		RETURNING id, exam_id, student_id, status, started_at, total_count, total_marks`
	err = r.DB.QueryRow(ctx, q, id, examID, studentID, totalCount, totalMarks).
		Scan(&a.ID, &a.ExamID, &a.StudentID, &a.Status, &a.StartedAt, &a.TotalCount, &a.TotalMarks)
	return a, err
}

func (r *Repository) GetInProgressAttempt(ctx context.Context, examID, studentID string) (*models.ExamAttempt, error) {
	a := &models.ExamAttempt{}
	const q = `SELECT id, exam_id, student_id, status, started_at, total_count, total_marks
		FROM exam_attempts WHERE exam_id = $1 AND student_id = $2 AND status = 'IN_PROGRESS'`
	err := r.DB.QueryRow(ctx, q, examID, studentID).
		Scan(&a.ID, &a.ExamID, &a.StudentID, &a.Status, &a.StartedAt, &a.TotalCount, &a.TotalMarks)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAttemptNotFound
	}
	return a, err
}

type AnswerInput struct {
	QuestionID    string
	SelectedIndex *int
}

// SubmitAttempt grades objective answers server-side against the
// stored correct_index for each question — the client never supplies
// (or can influence) the score.
func (r *Repository) SubmitAttempt(ctx context.Context, attempt *models.ExamAttempt, answers []AnswerInput, status models.AttemptStatus) (*models.ExamAttempt, error) {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	correctCount := 0
	for _, ans := range answers {
		var correctIndex int
		if err := tx.QueryRow(ctx, `SELECT correct_index FROM questions WHERE id = $1`, ans.QuestionID).Scan(&correctIndex); err != nil {
			continue
		}
		isCorrect := ans.SelectedIndex != nil && *ans.SelectedIndex == correctIndex
		if isCorrect {
			correctCount++
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO student_answers (id, attempt_id, question_id, selected_index, is_correct)
			 VALUES ($1,$2,$3,$4,$5)`,
			uuidx.New(), attempt.ID, ans.QuestionID, ans.SelectedIndex, isCorrect); err != nil {
			return nil, err
		}
	}

	percentage := 0.0
	if attempt.TotalCount > 0 {
		percentage = (float64(correctCount) / float64(attempt.TotalCount)) * 100
	}
	now := time.Now()

	const upd = `UPDATE exam_attempts
		SET status = $2, submitted_at = $3, correct_count = $4, obtained_marks = $4, percentage = $5
		WHERE id = $1
		RETURNING id, exam_id, student_id, status, started_at, submitted_at, correct_count, total_count, obtained_marks, total_marks, percentage`
	updated := &models.ExamAttempt{}
	err = tx.QueryRow(ctx, upd, attempt.ID, status, now, correctCount, percentage).
		Scan(&updated.ID, &updated.ExamID, &updated.StudentID, &updated.Status, &updated.StartedAt, &updated.SubmittedAt,
			&updated.CorrectCount, &updated.TotalCount, &updated.ObtainedMarks, &updated.TotalMarks, &updated.Percentage)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return updated, nil
}

func (r *Repository) ListResultsForStudent(ctx context.Context, studentID string) ([]models.ExamAttempt, error) {
	return r.queryAttempts(ctx, `WHERE student_id = $1 AND status != 'IN_PROGRESS' ORDER BY submitted_at DESC`, studentID)
}

func (r *Repository) ListAllResults(ctx context.Context) ([]models.ExamAttempt, error) {
	return r.queryAttempts(ctx, `WHERE status != 'IN_PROGRESS' ORDER BY submitted_at DESC`)
}

func (r *Repository) queryAttempts(ctx context.Context, whereAndOrder string, args ...any) ([]models.ExamAttempt, error) {
	rows, err := r.DB.Query(ctx, `SELECT id, exam_id, student_id, status, started_at, submitted_at,
		correct_count, total_count, obtained_marks, total_marks, percentage
		FROM exam_attempts `+whereAndOrder, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.ExamAttempt
	for rows.Next() {
		var a models.ExamAttempt
		if err := rows.Scan(&a.ID, &a.ExamID, &a.StudentID, &a.Status, &a.StartedAt, &a.SubmittedAt,
			&a.CorrectCount, &a.TotalCount, &a.ObtainedMarks, &a.TotalMarks, &a.Percentage); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
