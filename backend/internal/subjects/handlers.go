package subjects

import (
	"net/http"
	"strings"

	"examplatform/internal/auth"
	"examplatform/internal/httpx"
)

type Handlers struct{ Repo *Repository }

func NewHandlers(repo *Repository) *Handlers { return &Handlers{Repo: repo} }

// List handles GET /api/v1/subjects — available to any authenticated user.
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	subjects, err := h.Repo.List(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to load subjects")
		return
	}
	httpx.JSON(w, http.StatusOK, httpx.Envelope{"subjects": subjects})
}

type createRequest struct {
	Name string `json:"name"`
}

// Create handles POST /api/v1/subjects — teachers/admins only.
func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := httpx.DecodeJSON(r, &req); err != nil || strings.TrimSpace(req.Name) == "" {
		httpx.Error(w, http.StatusBadRequest, "a non-empty 'name' is required")
		return
	}
	claims, _ := auth.ClaimsFromContext(r.Context())

	subject, err := h.Repo.Create(r.Context(), strings.TrimSpace(req.Name), claims.Sub)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to create subject")
		return
	}
	httpx.JSON(w, http.StatusCreated, subject)
}

// Delete handles DELETE /api/v1/subjects/{id} — teachers/admins only.
func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.Repo.Delete(r.Context(), id); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to delete subject")
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}
