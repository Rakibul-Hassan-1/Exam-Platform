package jobs

import (
	"context"
	"fmt"
	"log"
	"time"

	"examplatform/internal/ai"
	"examplatform/internal/models"
	"examplatform/internal/questions"
)

// Worker polls generation_jobs for PENDING work, calls the AI provider,
// validates the structured output, and writes REVIEWING questions into
// the question bank. This keeps large-document processing off the
// request/response path, per the platform's asynchronous generation
// requirement (PENDING -> PROCESSING -> COMPLETED/FAILED).
type Worker struct {
	Jobs        *Repository
	Questions   *questions.Repository
	AI          *ai.Client
	SubjectName func(ctx context.Context, subjectID string) (string, error)
	PollInterval time.Duration
}

func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.PollInterval)
	defer ticker.Stop()

	log.Printf("worker: polling every %s", w.PollInterval)
	for {
		select {
		case <-ctx.Done():
			log.Println("worker: shutting down")
			return
		case <-ticker.C:
			if err := w.processOne(ctx); err != nil {
				log.Printf("worker: error processing job: %v", err)
			}
		}
	}
}

func (w *Worker) processOne(ctx context.Context) error {
	job, err := w.Jobs.ClaimNextPending(ctx)
	if err != nil {
		return fmt.Errorf("claim job: %w", err)
	}
	if job == nil {
		return nil // nothing pending right now
	}
	log.Printf("worker: processing job %s (%d questions)", job.ID, job.QuestionCount)

	subjectName, err := w.SubjectName(ctx, job.SubjectID)
	if err != nil {
		subjectName = "General"
	}

	generated, err := w.AI.GenerateQuestions(subjectName, job.SourceText, job.QuestionCount, ai.DifficultyMix{
		Easy: job.DifficultyEasy, Medium: job.DifficultyMed, Hard: job.DifficultyHard,
	})
	if err != nil {
		log.Printf("worker: job %s failed: %v", job.ID, err)
		return w.Jobs.MarkFailed(ctx, job.ID, err.Error())
	}

	for _, gq := range generated {
		jobID := job.ID
		_, err := w.Questions.Create(ctx, questions.CreateInput{
			SubjectID:       job.SubjectID,
			Question:        gq.Question,
			Options:         gq.Options,
			CorrectIndex:    gq.CorrectIndex,
			Difficulty:      models.Difficulty(gq.Difficulty),
			Explanation:     gq.Explanation,
			Source:          models.SourceAI,
			Status:          models.StatusReviewing,
			CreatedBy:       job.TeacherID,
			GenerationJobID: &jobID,
		})
		if err != nil {
			log.Printf("worker: failed to save generated question for job %s: %v", job.ID, err)
		}
	}

	log.Printf("worker: job %s completed, %d questions saved", job.ID, len(generated))
	return w.Jobs.MarkCompleted(ctx, job.ID)
}
