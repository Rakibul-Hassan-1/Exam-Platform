# AI-Powered Online Examination & Question Generation Platform

Production-style scaffold matching the original design doc: a **Go** REST
API, **PostgreSQL** database, a **background worker** for asynchronous AI
question generation, and a **Next.js** frontend, wired together with JWT
auth and role-based access control.

```
exam-platform/
├── backend/           Go API + worker + Postgres migrations
│   ├── cmd/api/        HTTP server entrypoint
│   ├── cmd/worker/     Background AI-generation worker entrypoint
│   ├── internal/       auth, subjects, questions, documents, exams, ai, jobs...
│   └── migrations/     SQL schema (up/down)
├── frontend/           Next.js 14 (App Router) + TypeScript + Tailwind
└── docker-compose.yml  Postgres + api + worker + frontend, wired together
```

## Quick start (Docker Compose)

```bash
cp backend/.env.example backend/.env   # optional — compose sets most vars itself
export ANTHROPIC_API_KEY=sk-ant-...    # required for AI question generation
export JWT_SECRET=$(openssl rand -hex 32)

docker compose up --build
```

This starts:
- **postgres** on `5432`, schema auto-applied from `backend/migrations/0001_init.up.sql`
- **api** on `8080`
- **worker** — polls `generation_jobs` and calls the AI provider
- **frontend** on `3000`

Visit `http://localhost:3000`, register a teacher account, add a subject,
generate some questions, build and publish an exam, then register a
student account (or open an incognito tab) to take it.

## Running without Docker

**Backend:**
```bash
cd backend
go mod tidy                 # needs network access to proxy.golang.org
createdb examplatform       # or run migrations/0001_init.up.sql manually via psql
psql examplatform < migrations/0001_init.up.sql
cp .env.example .env        # edit DATABASE_URL / JWT_SECRET / ANTHROPIC_API_KEY
go run ./cmd/api            # terminal 1
go run ./cmd/worker         # terminal 2
```

**Frontend:**
```bash
cd frontend
npm install
cp .env.local.example .env.local
npm run dev
```

> This sandbox has no outbound network access to `proxy.golang.org` or
> `registry.npmjs.org` for arbitrary packages, so `go mod tidy` / `npm install`
> could not be run here to produce lockfiles. The code was written and
> reviewed by hand for import correctness; run both commands in your own
> environment (which has normal internet access) before building.

## Architecture notes

- **Auth**: HS256 JWT implemented with the Go standard library only (no
  external JWT dependency) — see `internal/auth/jwt.go`. Passwords are
  hashed with bcrypt. Role checks (`teacher` / `student` / `admin`) are
  enforced per-route in `internal/router/router.go` via
  `middleware.RequireRole`.
- **IDs**: UUID v4 generated with `crypto/rand`, no external UUID library.
- **Router**: Go 1.22's `net/http.ServeMux` method+wildcard patterns
  (`"POST /api/v1/exams/{id}/submit"`), so the project needs no external
  router dependency either. The only third-party Go modules are
  `github.com/jackc/pgx/v5` (Postgres driver/pool) and
  `golang.org/x/crypto` (bcrypt).
- **AI generation is asynchronous**: `POST /api/v1/documents/generate-from-text`
  (or `/documents/{id}/generate-questions`) inserts a row into
  `generation_jobs` with status `PENDING` and returns immediately. The
  `worker` binary polls with `SELECT ... FOR UPDATE SKIP LOCKED` (safe to
  run multiple worker replicas), calls the Anthropic API
  (`internal/ai/client.go`), validates the returned JSON against the
  required schema, and inserts questions with status `REVIEWING` — never
  directly into an exam. A teacher must explicitly approve them
  (`internal/questions` review endpoints) before they can go into an exam.
- **Exam timing is server-authoritative**: `POST /exams/{id}/start` records
  `started_at` in Postgres and returns a `deadline` the client renders as a
  countdown; the client's clock is never trusted. `POST /exams/{id}/submit`
  is graded server-side against the stored `correct_index` — a compromised
  or modified frontend cannot inflate a score. Late submissions are still
  accepted but recorded as `AUTO_SUBMITTED` rather than `SUBMITTED`.
- **One attempt per student per exam** is enforced with a
  `UNIQUE (exam_id, student_id)` constraint on `exam_attempts`.
- **Question review workflow**: `GENERATED`/`REVIEWING` → teacher
  edits/approves/rejects → `APPROVED` (usable in exams) or `REJECTED`.
  This matches the human-in-the-loop requirement from the design doc —
  AI output is treated as untrusted until a teacher signs off on it.

## Known scaffold limitations (by design, clearly marked in code)

- **PDF text extraction is a documented TODO**
  (`internal/documents/extract.go`). `.txt` and `.md` uploads are fully
  supported end-to-end. For PDFs, wire in a library such as
  `github.com/ledongthuc/pdf`, `unidoc/unipdf`, or an external
  extraction/OCR service at the marked extension point — until then, use
  the "generate from pasted text" endpoint/page for PDF content.
- **No refresh tokens / token revocation** — JWTs are 24h bearer tokens.
  Add a refresh-token table or short-lived access + long-lived refresh
  pair before shipping this to real users.
- **No rate limiting, audit logging, or CORS allow-list beyond a single
  origin** — the design doc calls for both; they're natural additions to
  `internal/middleware`.
- **No automated tests** — the design doc's non-functional requirements
  call for a test suite; this scaffold prioritized a working, coherent
  system across the full stack within scope. Recommended next step: table-
  driven tests per `internal/*` package plus an integration test against a
  throwaway Postgres container.
- **Chunking is size-based, not semantic** — `internal/documents/extract.go`
  splits by character count. For very large books, the design doc's own
  section 29 suggests embeddings/vector search for relevance-based chunk
  selection as a future enhancement.

## API summary

See `internal/router/router.go` for the definitive route list. Highlights:

```
POST   /api/v1/auth/register
POST   /api/v1/auth/login
GET    /api/v1/auth/me

GET    /api/v1/subjects
POST   /api/v1/subjects                         (teacher)

POST   /api/v1/questions                         (teacher, manual)
PUT    /api/v1/questions/{id}                    (teacher, edit/approve/reject)
GET    /api/v1/questions?subject_id=&status=     (teacher)

POST   /api/v1/documents/upload                  (teacher, multipart)
POST   /api/v1/documents/generate-from-text       (teacher, async job)
POST   /api/v1/documents/{id}/generate-questions  (teacher, async job)
GET    /api/v1/generation-jobs/{id}               (poll job status)

POST   /api/v1/exams                              (teacher)
GET    /api/v1/exams                              (published-only for students)
GET    /api/v1/exams/{id}                          (no answer key for students)
PATCH  /api/v1/exams/{id}/publish                 (teacher)
POST   /api/v1/exams/{id}/start                   (student)
POST   /api/v1/exams/{id}/submit                  (student, server-graded)

GET    /api/v1/results                            (own results for students, all for teachers)
```
# Exam-Platform
