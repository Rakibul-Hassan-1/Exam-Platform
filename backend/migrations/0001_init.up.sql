-- Initial schema for the AI-Powered Online Examination & Question
-- Generation Platform. IDs are app-generated UUID v4 strings (TEXT),
-- so no pgcrypto/uuid-ossp extension is required.

CREATE TABLE users (
    id            TEXT PRIMARY KEY,
    name          TEXT NOT NULL,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL CHECK (role IN ('teacher', 'student', 'admin')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE subjects (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    created_by TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE documents (
    id           TEXT PRIMARY KEY,
    teacher_id   TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subject_id   TEXT REFERENCES subjects(id) ON DELETE SET NULL,
    filename     TEXT NOT NULL,
    storage_path TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'UPLOADED'
                 CHECK (status IN ('UPLOADED', 'PROCESSED', 'EXTRACTION_FAILED')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE document_chunks (
    id          TEXT PRIMARY KEY,
    document_id TEXT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    chunk_index INT NOT NULL,
    content     TEXT NOT NULL
);
CREATE INDEX idx_document_chunks_document_id ON document_chunks(document_id);

CREATE TABLE generation_jobs (
    id                TEXT PRIMARY KEY,
    document_id       TEXT REFERENCES documents(id) ON DELETE SET NULL,
    teacher_id        TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subject_id        TEXT NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    source_text       TEXT NOT NULL,
    question_count    INT NOT NULL,
    difficulty_easy   INT NOT NULL DEFAULT 0,
    difficulty_medium INT NOT NULL DEFAULT 0,
    difficulty_hard   INT NOT NULL DEFAULT 0,
    status            TEXT NOT NULL DEFAULT 'PENDING'
                      CHECK (status IN ('PENDING', 'PROCESSING', 'COMPLETED', 'FAILED')),
    error             TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_generation_jobs_status ON generation_jobs(status, created_at);

CREATE TABLE questions (
    id                 TEXT PRIMARY KEY,
    subject_id         TEXT NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    question           TEXT NOT NULL,
    options            JSONB NOT NULL,
    correct_index      INT NOT NULL CHECK (correct_index BETWEEN 0 AND 3),
    difficulty         TEXT NOT NULL CHECK (difficulty IN ('easy', 'medium', 'hard')),
    explanation        TEXT NOT NULL DEFAULT '',
    source             TEXT NOT NULL CHECK (source IN ('manual', 'ai')),
    status             TEXT NOT NULL DEFAULT 'GENERATED'
                       CHECK (status IN ('GENERATED', 'REVIEWING', 'APPROVED', 'REJECTED', 'ARCHIVED')),
    created_by         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    generation_job_id  TEXT REFERENCES generation_jobs(id) ON DELETE SET NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_questions_subject_status ON questions(subject_id, status);

CREATE TABLE exams (
    id            TEXT PRIMARY KEY,
    title         TEXT NOT NULL,
    subject_id    TEXT NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    duration_min  INT NOT NULL,
    total_marks   INT NOT NULL,
    passing_marks INT NOT NULL DEFAULT 0,
    published     BOOLEAN NOT NULL DEFAULT false,
    created_by    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_exams_published ON exams(published);

CREATE TABLE exam_questions (
    exam_id     TEXT NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
    question_id TEXT NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    position    INT NOT NULL,
    PRIMARY KEY (exam_id, question_id)
);

CREATE TABLE exam_attempts (
    id             TEXT PRIMARY KEY,
    exam_id        TEXT NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
    student_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status         TEXT NOT NULL DEFAULT 'IN_PROGRESS'
                   CHECK (status IN ('IN_PROGRESS', 'SUBMITTED', 'AUTO_SUBMITTED')),
    started_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    submitted_at   TIMESTAMPTZ,
    correct_count  INT NOT NULL DEFAULT 0,
    total_count    INT NOT NULL DEFAULT 0,
    obtained_marks INT NOT NULL DEFAULT 0,
    total_marks    INT NOT NULL DEFAULT 0,
    percentage     NUMERIC(5,2) NOT NULL DEFAULT 0,
    UNIQUE (exam_id, student_id)
);
CREATE INDEX idx_exam_attempts_student ON exam_attempts(student_id);

CREATE TABLE student_answers (
    id             TEXT PRIMARY KEY,
    attempt_id     TEXT NOT NULL REFERENCES exam_attempts(id) ON DELETE CASCADE,
    question_id    TEXT NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    selected_index INT,
    is_correct     BOOLEAN NOT NULL DEFAULT false
);
CREATE INDEX idx_student_answers_attempt ON student_answers(attempt_id);
