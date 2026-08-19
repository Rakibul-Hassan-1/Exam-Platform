package questions

import (
	"net/http"

	"examplatform/internal/auth"
	"examplatform/internal/httpx"
	"examplatform/internal/models"
)

type Handlers struct{ Repo *Repository }

func NewHandlers(repo *Repository) *Handlers { return &Handlers{Repo: repo} }

// List handles GET /api/v1/questions?subject_id=&status=
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	f := ListFilter{
		SubjectID: r.URL.Query().Get("subject_id"),
		Status:    r.URL.Query().Get("status"),
	}
	list, err := h.Repo.List(r.Context(), f)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to load questions")
		return
	}
	httpx.JSON(w, http.StatusOK, httpx.Envelope{"questions": list})
}

type createRequest struct {
	SubjectID    string   `json:"subject_id"`
	Question     string   `json:"question"`
	Options      []string `json:"options"`
	CorrectIndex int      `json:"correct_index"`
	Difficulty   string   `json:"difficulty"`
}

// Create handles POST /api/v1/questions — manual question creation by a teacher.
func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.SubjectID == "" || req.Question == "" || len(req.Options) != 4 ||
		req.CorrectIndex < 0 || req.CorrectIndex > 3 {
		httpx.Error(w, http.StatusBadRequest, "subject_id, question, exactly 4 options, and a valid correct_index are required")
		return
	}
	diff := models.Difficulty(req.Difficulty)
	if diff != models.Easy && diff != models.Medium && diff != models.Hard {
		diff = models.Medium
	}
	claims, _ := auth.ClaimsFromContext(r.Context())

	q, err := h.Repo.Create(r.Context(), CreateInput{
		SubjectID: req.SubjectID, Question: req.Question, Options: req.Options,
		CorrectIndex: req.CorrectIndex, Difficulty: diff, Source: models.SourceManual,
		Status: models.StatusApproved, CreatedBy: claims.Sub,
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to create question")
		return
	}
	httpx.JSON(w, http.StatusCreated, q)
}

type updateRequest struct {
	Question     *string   `json:"question"`
	Options      *[]string `json:"options"`
	CorrectIndex *int      `json:"correct_index"`
	Difficulty   *string   `json:"difficulty"`
	Status       *string   `json:"status"`
}

// Update handles PUT /api/v1/questions/{id} — edit text, options, difficulty, or status
// (used for both manual edits and the approve/reject review workflow).
func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req updateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	in := UpdateInput{Question: req.Question, Options: req.Options, CorrectIndex: req.CorrectIndex}
	if req.Difficulty != nil {
		d := models.Difficulty(*req.Difficulty)
		in.Difficulty = &d
	}
	if req.Status != nil {
		s := models.QuestionStatus(*req.Status)
		in.Status = &s
	}

	q, err := h.Repo.Update(r.Context(), id, in)
	if err != nil || q == nil {
		httpx.Error(w, http.StatusNotFound, "question not found")
		return
	}
	httpx.JSON(w, http.StatusOK, q)
}

// Delete handles DELETE /api/v1/questions/{id}
func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.Repo.Delete(r.Context(), id); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to delete question")
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}
