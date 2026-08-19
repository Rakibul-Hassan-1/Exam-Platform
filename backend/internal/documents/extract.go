package documents

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const chunkSize = 4000 // characters per chunk, keeps AI requests manageable

// ExtractAndChunk reads a stored document from disk and splits its text
// into content chunks for AI processing, per the platform's document
// processing pipeline (validate -> extract -> clean -> chunk -> AI).
//
// TXT files are fully supported. PDF extraction is intentionally left
// as an integration point: wire in a library such as
// github.com/ledongthuc/pdf or an external OCR/extraction service here.
// Until then, uploading a .pdf stores the file but returns an error
// when question generation is requested, rather than silently
// fabricating content.
func ExtractAndChunk(storagePath string) ([]string, error) {
	ext := strings.ToLower(filepath.Ext(storagePath))

	switch ext {
	case ".txt", ".md":
		raw, err := os.ReadFile(storagePath)
		if err != nil {
			return nil, fmt.Errorf("reading file: %w", err)
		}
		return chunk(cleanText(string(raw))), nil
	case ".pdf":
		return nil, fmt.Errorf("PDF text extraction is not wired up in this scaffold yet — " +
			"integrate a PDF text-extraction library in internal/documents/extract.go, " +
			"or use the 'generate from pasted text' endpoint in the meantime")
	default:
		return nil, fmt.Errorf("unsupported file type %q — supported: .txt, .md (PDF extraction is a documented TODO)", ext)
	}
}

func cleanText(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return strings.Join(out, "\n")
}

func chunk(text string) []string {
	if text == "" {
		return nil
	}
	var chunks []string
	runes := []rune(text)
	for i := 0; i < len(runes); i += chunkSize {
		end := i + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[i:end]))
	}
	return chunks
}
