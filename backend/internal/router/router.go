// Package router wires every module's HTTP handlers onto a single
// net/http.ServeMux, using Go 1.22's method+wildcard routing patterns
// so the project needs no external router dependency.
package router

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"examplatform/internal/auth"
	"examplatform/internal/config"
	"examplatform/internal/documents"
	"examplatform/internal/exams"
	"examplatform/internal/httpx"
	"examplatform/internal/jobs"
	"examplatform/internal/middleware"
	"examplatform/internal/models"
	"examplatform/internal/questions"
	"examplatform/internal/subjects"
)

func New(cfg config.Config, db *pgxpool.Pool) http.Handler {
	mux := http.NewServeMux()

	// --- wire repositories & handlers ---
	authRepo := auth.NewRepository(db)
	authH := auth.NewHandlers(authRepo, cfg.JWTSecret)

	subjectsRepo := subjects.NewRepository(db)
	subjectsH := subjects.NewHandlers(subjectsRepo)

	questionsRepo := questions.NewRepository(db)
	questionsH := questions.NewHandlers(questionsRepo)

	jobsRepo := jobs.NewRepository(db)
	documentsRepo := documents.NewRepository(db)
	documentsH := documents.NewHandlers(documentsRepo, jobsRepo, cfg.DocsStoragePath)

	examsRepo := exams.NewRepository(db)
	examsH := exams.NewHandlers(examsRepo, questionsRepo)

	authed := middleware.RequireAuth(cfg.JWTSecret)
	teacherOnly := middleware.RequireRole(models.RoleTeacher, models.RoleAdmin)
	studentOnly := middleware.RequireRole(models.RoleStudent)

	// --- health ---
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, httpx.Envelope{"status": "ok"})
	})

	// --- auth ---
	mux.HandleFunc("POST /api/v1/auth/register", authH.Register)
	mux.HandleFunc("POST /api/v1/auth/login", authH.Login)
	mux.Handle("GET /api/v1/auth/me", middleware.Chain(http.HandlerFunc(authH.Me), authed))

	// --- subjects ---
	mux.Handle("GET /api/v1/subjects", middleware.Chain(http.HandlerFunc(subjectsH.List), authed))
	mux.Handle("POST /api/v1/subjects", middleware.Chain(http.HandlerFunc(subjectsH.Create), authed, teacherOnly))
	mux.Handle("DELETE /api/v1/subjects/{id}", middleware.Chain(http.HandlerFunc(subjectsH.Delete), authed, teacherOnly))

	// --- questions (question bank) ---
	mux.Handle("GET /api/v1/questions", middleware.Chain(http.HandlerFunc(questionsH.List), authed, teacherOnly))
	mux.Handle("POST /api/v1/questions", middleware.Chain(http.HandlerFunc(questionsH.Create), authed, teacherOnly))
	mux.Handle("PUT /api/v1/questions/{id}", middleware.Chain(http.HandlerFunc(questionsH.Update), authed, teacherOnly))
	mux.Handle("DELETE /api/v1/questions/{id}", middleware.Chain(http.HandlerFunc(questionsH.Delete), authed, teacherOnly))

	// --- documents & AI generation ---
	mux.Handle("POST /api/v1/documents/upload", middleware.Chain(http.HandlerFunc(documentsH.Upload), authed, teacherOnly))
	mux.Handle("GET /api/v1/documents", middleware.Chain(http.HandlerFunc(documentsH.List), authed, teacherOnly))
	mux.Handle("POST /api/v1/documents/{id}/generate-questions", middleware.Chain(http.HandlerFunc(documentsH.GenerateFromDocument), authed, teacherOnly))
	mux.Handle("POST /api/v1/documents/generate-from-text", middleware.Chain(http.HandlerFunc(documentsH.GenerateFromText), authed, teacherOnly))
	mux.Handle("GET /api/v1/generation-jobs/{id}", middleware.Chain(http.HandlerFunc(documentsH.JobStatus), authed, teacherOnly))

	// --- exams ---
	mux.Handle("POST /api/v1/exams", middleware.Chain(http.HandlerFunc(examsH.Create), authed, teacherOnly))
	mux.Handle("GET /api/v1/exams", middleware.Chain(http.HandlerFunc(examsH.List), authed))
	mux.Handle("GET /api/v1/exams/{id}", middleware.Chain(http.HandlerFunc(examsH.Get), authed))
	mux.Handle("PATCH /api/v1/exams/{id}/publish", middleware.Chain(http.HandlerFunc(examsH.SetPublished), authed, teacherOnly))
	mux.Handle("DELETE /api/v1/exams/{id}", middleware.Chain(http.HandlerFunc(examsH.Delete), authed, teacherOnly))

	// --- exam attempts (student flow) ---
	mux.Handle("POST /api/v1/exams/{id}/start", middleware.Chain(http.HandlerFunc(examsH.Start), authed, studentOnly))
	mux.Handle("POST /api/v1/exams/{id}/submit", middleware.Chain(http.HandlerFunc(examsH.Submit), authed, studentOnly))

	// --- results ---
	mux.Handle("GET /api/v1/results", middleware.Chain(http.HandlerFunc(examsH.ResultsList), authed))

	return middleware.Chain(mux, middleware.Logger, middleware.CORS(cfg.AllowedOrigin))
}
