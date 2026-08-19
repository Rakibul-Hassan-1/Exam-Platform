// Package ai wraps an AI provider to turn study material into
// structured, schema-validated exam questions. This implementation
// calls OpenRouter's OpenAI-compatible chat completions endpoint
// (https://openrouter.ai/api/v1/chat/completions), routed to a Claude
// model. The provider is kept behind this package's interface so it
// can be swapped (e.g. back to Anthropic's native API) without
// touching callers (documents, jobs).
package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const openRouterURL = "https://openrouter.ai/api/v1/chat/completions"

// Change this if you want a different model. See https://openrouter.ai/models
// for current slugs — Claude models are listed under the "anthropic/" prefix.
const model = "anthropic/claude-sonnet-4.5"

type Client struct {
	APIKey     string
	HTTPClient *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{APIKey: apiKey, HTTPClient: &http.Client{Timeout: 90 * time.Second}}
}

type GeneratedQuestion struct {
	Question     string   `json:"question"`
	Difficulty   string   `json:"difficulty"`
	Options      []string `json:"options"`
	CorrectIndex int      `json:"correct_index"`
	Explanation  string   `json:"explanation"`
}

type DifficultyMix struct {
	Easy, Medium, Hard int
}

// --- OpenRouter (OpenAI-compatible) request/response shapes ---

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model     string        `json:"model"`
	Messages  []chatMessage `json:"messages"`
	MaxTokens int           `json:"max_tokens"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Code    any    `json:"code"`
	} `json:"error"`
}

type schemaPayload struct {
	Questions []GeneratedQuestion `json:"questions"`
}

// GenerateQuestions asks the model to produce MCQs strictly from
// sourceText, validates the structured output, and returns only
// well-formed questions matching the required schema.
func (c *Client) GenerateQuestions(subjectName, sourceText string, count int, mix DifficultyMix) ([]GeneratedQuestion, error) {
	if c.APIKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY (OpenRouter key) is not configured on the server")
	}

	system := "You are an exam question generator for an academic examination platform. " +
		"You read study material and produce multiple-choice questions strictly as JSON. " +
		"Never include any text outside the JSON object, and never wrap it in markdown fences."

	prompt := fmt.Sprintf(`Subject: %s
Source material:
"""
%s
"""

Generate exactly %d multiple-choice questions (MCQ) based ONLY on the source material above.
Difficulty distribution (approximate, out of %d total): easy=%d, medium=%d, hard=%d.

Rules:
- Each question has exactly 4 options.
- Exactly one correct answer per question (correct_index is 0-3).
- "difficulty" must be one of "easy", "medium", "hard".
- Include a short "explanation" for the correct answer.
- Base every question strictly on the provided material — do not invent facts outside it.

Respond with ONLY this JSON shape, nothing else:
{"questions":[{"question":"string","difficulty":"easy|medium|hard","options":["a","b","c","d"],"correct_index":0,"explanation":"string"}]}`,
		subjectName, truncate(sourceText, 16000), count, count, mix.Easy, mix.Medium, mix.Hard)

	reqBody := chatRequest{
		Model:     model,
		MaxTokens: 2000,
		Messages: []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: prompt},
		},
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest(http.MethodPost, openRouterURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	// Optional but recommended by OpenRouter for attribution/rate-limit tracking:
	httpReq.Header.Set("HTTP-Referer", "http://localhost")
	httpReq.Header.Set("X-Title", "Examination Hall")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("calling AI provider: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var parsed chatResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("AI provider returned an unreadable response")
	}
	if resp.StatusCode != http.StatusOK {
		if parsed.Error != nil {
			return nil, fmt.Errorf("AI provider error: %s", parsed.Error.Message)
		}
		return nil, fmt.Errorf("AI provider returned status %d: %s", resp.StatusCode, string(raw))
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("AI response contained no choices")
	}

	text := parsed.Choices[0].Message.Content
	if text == "" {
		return nil, fmt.Errorf("AI response contained no text content")
	}

	clean := strings.TrimSpace(text)
	clean = strings.TrimPrefix(clean, "```json")
	clean = strings.TrimPrefix(clean, "```")
	clean = strings.TrimSuffix(clean, "```")
	clean = strings.TrimSpace(clean)

	var schema schemaPayload
	if err := json.Unmarshal([]byte(clean), &schema); err != nil {
		return nil, fmt.Errorf("AI response was not valid JSON matching the required schema")
	}

	valid := make([]GeneratedQuestion, 0, len(schema.Questions))
	for _, q := range schema.Questions {
		if q.Question == "" || len(q.Options) != 4 {
			continue
		}
		if q.CorrectIndex < 0 || q.CorrectIndex > 3 {
			continue
		}
		if q.Difficulty != "easy" && q.Difficulty != "medium" && q.Difficulty != "hard" {
			continue
		}
		valid = append(valid, q)
	}
	if len(valid) == 0 {
		return nil, fmt.Errorf("AI response did not contain any questions matching the required schema")
	}
	return valid, nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}