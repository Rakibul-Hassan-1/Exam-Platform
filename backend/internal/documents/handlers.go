package documents

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"examplatform/internal/auth"
	"examplatform/internal/httpx"
	"examplatform/internal/jobs"
	"examplatform/internal/uuidx"
)

const maxUploadSize = 25 << 20 // 25 MB

type Handlers struct {
	Repo        *Repository
	Jobs        *jobs.Repository
	StoragePath string
}

func NewHandlers(repo *Repository, jobsRepo *jobs.Repository, storagePath string) *Handlers {
	return &Handlers{Repo: repo, Jobs: jobsRepo, StoragePath: storagePath}
}

// Upload handles POST /api/v1/documents/upload (multipart/form-data:
// file, subject_id). Validates file type and size before storing.
func (h *Handlers) Upload(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFromContext(r.Context())

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		httpx.Error(w, http.StatusBadRequest, "file too large or malformed upload (25MB max)")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "missing 'file' field")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".txt" && ext != ".md" && ext != ".pdf" {
		httpx.Error(w, http.StatusUnprocessableEntity, "unsupported file type — upload a .txt, .md, or .pdf file")
		return
	}

	if err := os.MkdirAll(h.StoragePath, 0o755); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to prepare storage")
		return
	}
	storedName := uuidx.New() + ext
	destPath := filepath.Join(h.StoragePath, storedName)

	dest, err := os.Create(destPath)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to store file")
		return
	}
	defer dest.Close()
	if _, err := io.Copy(dest, file); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to store file")
		return
	}

	subjectID := r.FormValue("subject_id")
	doc, err := h.Repo.Create(r.Context(), claims.Sub, subjectID, header.Filename, destPath, "UPLOADED")
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to record document")
		return
	}

	// Extract + chunk synchronously (fast, local operation); the AI
	// generation step itself is what runs asynchronously via the job queue.
	chunks, err := ExtractAndChunk(destPath)
	if err != nil {
		h.Repo.UpdateStatus(r.Context(), doc.ID, "EXTRACTION_FAILED")
		httpx.JSON(w, http.StatusCreated, httpx.Envelope{
			"document": doc,
			"warning":  err.Error(),
		})
		return
	}
	if err := h.Repo.SaveChunks(r.Context(), doc.ID, chunks); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to store extracted content")
		return
	}
	h.Repo.UpdateStatus(r.Context(), doc.ID, "PROCESSED")

	httpx.JSON(w, http.StatusCreated, httpx.Envelope{"document": doc, "chunks": len(chunks)})
}

// List handles GET /api/v1/documents — the current teacher's uploads.
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFromContext(r.Context())
	docs, err := h.Repo.List(r.Context(), claims.Sub)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to load documents")
		return
	}
	httpx.JSON(w, http.StatusOK, httpx.Envelope{"documents": docs})
}

type difficultyInput struct {
	Easy   int `json:"easy"`
	Medium int `json:"medium"`
	Hard   int `json:"hard"`
}

type generateRequest struct {
	SubjectID     string          `json:"subject_id"`
	QuestionCount int             `json:"question_count"`
	Difficulty    difficultyInput `json:"difficulty"`
}

// GenerateFromDocument handles POST /api/v1/documents/{id}/generate-questions
// — queues an async AI generation job from a previously uploaded (and
// extracted) document.
func (h *Handlers) GenerateFromDocument(w http.ResponseWriter, r *http.Request) {
	docID := r.PathValue("id")
	claims, _ := auth.ClaimsFromContext(r.Context())

	var req generateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if !validGenerateRequest(w, req) {
		return
	}

	doc, err := h.Repo.Get(r.Context(), docID)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "document not found")
		return
	}
	text, err := h.Repo.ConcatenatedText(r.Context(), docID)
	if err != nil || strings.TrimSpace(text) == "" {
		httpx.Error(w, http.StatusUnprocessableEntity, "document has no extracted content to generate from")
		return
	}

	job, err := h.Jobs.Create(r.Context(), jobs.CreateInput{
		DocumentID: &doc.ID, TeacherID: claims.Sub, SubjectID: req.SubjectID, SourceText: text,
		QuestionCount: req.QuestionCount, DifficultyEasy: req.Difficulty.Easy,
		DifficultyMed: req.Difficulty.Medium, DifficultyHard: req.Difficulty.Hard,
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to queue generation job")
		return
	}
	httpx.JSON(w, http.StatusAccepted, httpx.Envelope{"job": job})
}

type generateFromTextRequest struct {
	SubjectID     string          `json:"subject_id"`
	SourceText    string          `json:"source_text"`
	QuestionCount int             `json:"question_count"`
	Difficulty    difficultyInput `json:"difficulty"`
}

// GenerateFromText handles POST /api/v1/documents/generate-from-text —
// a convenience path for generating questions directly from pasted
// study material without a separate upload step.
func (h *Handlers) GenerateFromText(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFromContext(r.Context())

	var req generateFromTextRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.SourceText) == "" || len(req.SourceText) < 200 {
		httpx.Error(w, http.StatusBadRequest, "source_text must be at least 200 characters so the AI has enough context")
		return
	}
	if !validGenerateRequest(w, generateRequest{
		SubjectID: req.SubjectID, QuestionCount: req.QuestionCount, Difficulty: req.Difficulty,
	}) {
		return
	}

	job, err := h.Jobs.Create(r.Context(), jobs.CreateInput{
		TeacherID: claims.Sub, SubjectID: req.SubjectID, SourceText: req.SourceText,
		QuestionCount: req.QuestionCount, DifficultyEasy: req.Difficulty.Easy,
		DifficultyMed: req.Difficulty.Medium, DifficultyHard: req.Difficulty.Hard,
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to queue generation job")
		return
	}
	httpx.JSON(w, http.StatusAccepted, httpx.Envelope{"job": job})
}

// JobStatus handles GET /api/v1/generation-jobs/{id} for polling job progress.
func (h *Handlers) JobStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	job, err := h.Jobs.Get(r.Context(), id)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "job not found")
		return
	}
	httpx.JSON(w, http.StatusOK, job)
}

func validGenerateRequest(w http.ResponseWriter, req generateRequest) bool {
	if req.SubjectID == "" {
		httpx.Error(w, http.StatusBadRequest, "subject_id is required")
		return false
	}
	if req.QuestionCount < 1 || req.QuestionCount > 50 {
		httpx.Error(w, http.StatusBadRequest, "question_count must be between 1 and 50")
		return false
	}
	sum := req.Difficulty.Easy + req.Difficulty.Medium + req.Difficulty.Hard
	if sum <= 0 {
		httpx.Error(w, http.StatusBadRequest, "difficulty distribution must sum to more than 0")
		return false
	}
	return true
}
