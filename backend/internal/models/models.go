package models

import "time"

type Role string

const (
	RoleTeacher Role = "teacher"
	RoleStudent Role = "student"
	RoleAdmin   Role = "admin"
)

type User struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         Role      `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

type Subject struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type Difficulty string

const (
	Easy   Difficulty = "easy"
	Medium Difficulty = "medium"
	Hard   Difficulty = "hard"
)

type QuestionStatus string

const (
	StatusGenerated QuestionStatus = "GENERATED"
	StatusReviewing QuestionStatus = "REVIEWING"
	StatusApproved  QuestionStatus = "APPROVED"
	StatusRejected  QuestionStatus = "REJECTED"
	StatusArchived  QuestionStatus = "ARCHIVED"
)

type QuestionSource string

const (
	SourceManual QuestionSource = "manual"
	SourceAI     QuestionSource = "ai"
)

type Question struct {
	ID              string         `json:"id"`
	SubjectID       string         `json:"subject_id"`
	Question        string         `json:"question"`
	Options         []string       `json:"options"`
	CorrectIndex    int            `json:"correct_index"`
	Difficulty      Difficulty     `json:"difficulty"`
	Explanation     string         `json:"explanation"`
	Source          QuestionSource `json:"source"`
	Status          QuestionStatus `json:"status"`
	CreatedBy       string         `json:"created_by"`
	GenerationJobID *string        `json:"generation_job_id,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
}

// QuestionPublic strips the answer key — used when serving a live exam to a student.
type QuestionPublic struct {
	ID         string     `json:"id"`
	Question   string     `json:"question"`
	Options    []string   `json:"options"`
	Difficulty Difficulty `json:"difficulty"`
}

type Document struct {
	ID          string    `json:"id"`
	TeacherID   string    `json:"teacher_id"`
	SubjectID   *string   `json:"subject_id,omitempty"`
	Filename    string    `json:"filename"`
	StoragePath string    `json:"storage_path"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type JobStatus string

const (
	JobPending    JobStatus = "PENDING"
	JobProcessing JobStatus = "PROCESSING"
	JobCompleted  JobStatus = "COMPLETED"
	JobFailed     JobStatus = "FAILED"
)

type GenerationJob struct {
	ID              string    `json:"id"`
	DocumentID      *string   `json:"document_id,omitempty"`
	TeacherID       string    `json:"teacher_id"`
	SubjectID       string    `json:"subject_id"`
	SourceText      string    `json:"-"`
	QuestionCount   int       `json:"question_count"`
	DifficultyEasy  int       `json:"difficulty_easy"`
	DifficultyMed   int       `json:"difficulty_medium"`
	DifficultyHard  int       `json:"difficulty_hard"`
	Status          JobStatus `json:"status"`
	Error           string    `json:"error,omitempty"`
	CreatedQuestion []string  `json:"created_question_ids,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Exam struct {
	ID           string     `json:"id"`
	Title        string     `json:"title"`
	SubjectID    string     `json:"subject_id"`
	DurationMin  int        `json:"duration_min"`
	TotalMarks   int        `json:"total_marks"`
	PassingMarks int        `json:"passing_marks"`
	Published    bool       `json:"published"`
	CreatedBy    string     `json:"created_by"`
	CreatedAt    time.Time  `json:"created_at"`
	QuestionIDs  []string   `json:"question_ids,omitempty"`
}

type AttemptStatus string

const (
	AttemptInProgress   AttemptStatus = "IN_PROGRESS"
	AttemptSubmitted    AttemptStatus = "SUBMITTED"
	AttemptAutoSubmitted AttemptStatus = "AUTO_SUBMITTED"
)

type ExamAttempt struct {
	ID            string        `json:"id"`
	ExamID        string        `json:"exam_id"`
	StudentID     string        `json:"student_id"`
	Status        AttemptStatus `json:"status"`
	StartedAt     time.Time     `json:"started_at"`
	SubmittedAt   *time.Time    `json:"submitted_at,omitempty"`
	CorrectCount  int           `json:"correct_count"`
	TotalCount    int           `json:"total_count"`
	ObtainedMarks int           `json:"obtained_marks"`
	TotalMarks    int           `json:"total_marks"`
	Percentage    float64       `json:"percentage"`
}

type StudentAnswer struct {
	ID             string `json:"id"`
	AttemptID      string `json:"attempt_id"`
	QuestionID     string `json:"question_id"`
	SelectedIndex  *int   `json:"selected_index"`
	IsCorrect      bool   `json:"is_correct"`
}
