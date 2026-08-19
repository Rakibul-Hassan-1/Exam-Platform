package exams

import (
	"errors"
	"net/http"
	"time"

	"examplatform/internal/auth"
	"examplatform/internal/httpx"
	"examplatform/internal/models"
	"examplatform/internal/questions"
)

type Handlers struct {
	Repo      *Repository
	Questions *questions.Repository
}

func NewHandlers(repo *Repository, questionsRepo *questions.Repository) *Handlers {
	return &Handlers{Repo: repo, Questions: questionsRepo}
}

type createRequest struct {
	Title       string   `json:"title"`
	SubjectID   string   `json:"subject_id"`
	DurationMin int      `json:"duration_min"`
	QuestionIDs []string `json:"question_ids"`
}

// Create handles POST /api/v1/exams — teachers only.
func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Title == "" || req.SubjectID == "" || req.DurationMin < 1 || len(req.QuestionIDs) == 0 {
		httpx.Error(w, http.StatusBadRequest, "title, subject_id, duration_min, and at least one question_id are required")
		return
	}
	claims, _ := auth.ClaimsFromContext(r.Context())

	exam, err := h.Repo.Create(r.Context(), CreateInput{
		Title: req.Title, SubjectID: req.SubjectID, DurationMin: req.DurationMin,
		CreatedBy: claims.Sub, QuestionIDs: req.QuestionIDs,
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to create exam")
		return
	}
	httpx.JSON(w, http.StatusCreated, exam)
}

// List handles GET /api/v1/exams — students see only published exams; teachers see all.
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFromContext(r.Context())
	publishedOnly := claims.Role == models.RoleStudent

	list, err := h.Repo.List(r.Context(), publishedOnly)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to load exams")
		return
	}
	httpx.JSON(w, http.StatusOK, httpx.Envelope{"exams": list})
}

// Get handles GET /api/v1/exams/{id}. Students receive questions
// without the answer key; teachers receive the full record.
func (h *Handlers) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	claims, _ := auth.ClaimsFromContext(r.Context())

	exam, err := h.Repo.Get(r.Context(), id)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "exam not found")
		return
	}
	if claims.Role == models.RoleStudent && !exam.Published {
		httpx.Error(w, http.StatusNotFound, "exam not found")
		return
	}

	qs, err := h.Questions.GetByIDs(r.Context(), exam.QuestionIDs)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to load exam questions")
		return
	}

	if claims.Role == models.RoleStudent {
		public := make([]models.QuestionPublic, 0, len(qs))
		for _, q := range qs {
			public = append(public, models.QuestionPublic{ID: q.ID, Question: q.Question, Options: q.Options, Difficulty: q.Difficulty})
		}
		httpx.JSON(w, http.StatusOK, httpx.Envelope{"exam": exam, "questions": public})
		return
	}
	httpx.JSON(w, http.StatusOK, httpx.Envelope{"exam": exam, "questions": qs})
}

type publishRequest struct {
	Published bool `json:"published"`
}

// SetPublished handles PATCH /api/v1/exams/{id}/publish — teachers only.
func (h *Handlers) SetPublished(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req publishRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.Repo.SetPublished(r.Context(), id, req.Published); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to update exam")
		return
	}
	httpx.JSON(w, http.StatusOK, httpx.Envelope{"id": id, "published": req.Published})
}

// Delete handles DELETE /api/v1/exams/{id} — teachers only.
func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.Repo.Delete(r.Context(), id); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to delete exam")
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

// Start handles POST /api/v1/exams/{id}/start — students only. The
// server records started_at and returns it so the client can render a
// countdown; the server, not the client, remains authoritative for timing.
func (h *Handlers) Start(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	claims, _ := auth.ClaimsFromContext(r.Context())

	exam, err := h.Repo.Get(r.Context(), id)
	if err != nil || !exam.Published {
		httpx.Error(w, http.StatusNotFound, "exam not found")
		return
	}

	attempt, err := h.Repo.StartAttempt(r.Context(), id, claims.Sub, len(exam.QuestionIDs), exam.TotalMarks)
	if err != nil {
		if errors.Is(err, ErrAlreadyAttempted) {
			httpx.Error(w, http.StatusConflict, "you have already attempted this exam")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "failed to start exam")
		return
	}

	deadline := attempt.StartedAt.Add(time.Duration(exam.DurationMin) * time.Minute)
	httpx.JSON(w, http.StatusCreated, httpx.Envelope{
		"attempt": attempt, "started_at": attempt.StartedAt, "deadline": deadline,
	})
}

type submitAnswer struct {
	QuestionID    string `json:"question_id"`
	SelectedIndex *int   `json:"selected_index"`
}
type submitRequest struct {
	Answers []submitAnswer `json:"answers"`
}

// Submit handles POST /api/v1/exams/{id}/submit — students only. Grades
// server-side; if the submission arrives after duration + grace period,
// the attempt is recorded as AUTO_SUBMITTED rather than rejected outright.
func (h *Handlers) Submit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	claims, _ := auth.ClaimsFromContext(r.Context())

	var req submitRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	exam, err := h.Repo.Get(r.Context(), id)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "exam not found")
		return
	}
	attempt, err := h.Repo.GetInProgressAttempt(r.Context(), id, claims.Sub)
	if err != nil {
		httpx.Error(w, http.StatusConflict, "no in-progress attempt found — start the exam first")
		return
	}

	deadline := attempt.StartedAt.Add(time.Duration(exam.DurationMin)*time.Minute + 30*time.Second)
	status := models.AttemptSubmitted
	if time.Now().After(deadline) {
		status = models.AttemptAutoSubmitted
	}

	answers := make([]AnswerInput, 0, len(req.Answers))
	for _, a := range req.Answers {
		answers = append(answers, AnswerInput{QuestionID: a.QuestionID, SelectedIndex: a.SelectedIndex})
	}

	result, err := h.Repo.SubmitAttempt(r.Context(), attempt, answers, status)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to submit exam")
		return
	}
	httpx.JSON(w, http.StatusOK, result)
}

// ResultsList handles GET /api/v1/results — students see only their own
// results, teachers and admins see every submission.
func (h *Handlers) ResultsList(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFromContext(r.Context())

	var (
		results []models.ExamAttempt
		err     error
	)
	if claims.Role == models.RoleStudent {
		results, err = h.Repo.ListResultsForStudent(r.Context(), claims.Sub)
	} else {
		results, err = h.Repo.ListAllResults(r.Context())
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to load results")
		return
	}
	httpx.JSON(w, http.StatusOK, httpx.Envelope{"results": results})
}
