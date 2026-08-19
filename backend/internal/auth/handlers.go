package auth

import (
	"errors"
	"net/http"
	"strings"

	"examplatform/internal/httpx"
	"examplatform/internal/models"
)

type Handlers struct {
	Repo      *Repository
	JWTSecret string
}

func NewHandlers(repo *Repository, jwtSecret string) *Handlers {
	return &Handlers{Repo: repo, JWTSecret: jwtSecret}
}

type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type authResponse struct {
	Token string      `json:"token"`
	User  models.User `json:"user"`
}

// Register handles POST /api/v1/auth/register
func (h *Handlers) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	if req.Name == "" || req.Email == "" || len(req.Password) < 8 {
		httpx.Error(w, http.StatusBadRequest, "name, email, and a password of at least 8 characters are required")
		return
	}
	role := models.Role(req.Role)
	if role != models.RoleTeacher && role != models.RoleStudent {
		httpx.Error(w, http.StatusBadRequest, "role must be 'teacher' or 'student'")
		return
	}

	hash, err := HashPassword(req.Password)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to process password")
		return
	}

	user, err := h.Repo.CreateUser(r.Context(), req.Name, req.Email, hash, role)
	if err != nil {
		if errors.Is(err, ErrEmailTaken) {
			httpx.Error(w, http.StatusConflict, err.Error())
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	token, err := IssueToken(h.JWTSecret, user.ID, user.Name, user.Role)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to issue token")
		return
	}

	httpx.JSON(w, http.StatusCreated, authResponse{Token: token, User: *user})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login handles POST /api/v1/auth/login
func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	user, err := h.Repo.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	if !CheckPassword(user.PasswordHash, req.Password) {
		httpx.Error(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	token, err := IssueToken(h.JWTSecret, user.ID, user.Name, user.Role)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to issue token")
		return
	}

	httpx.JSON(w, http.StatusOK, authResponse{Token: token, User: *user})
}

// Me handles GET /api/v1/auth/me — returns the authenticated user's identity.
func (h *Handlers) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	httpx.JSON(w, http.StatusOK, httpx.Envelope{
		"id": claims.Sub, "name": claims.Name, "role": claims.Role,
	})
}
