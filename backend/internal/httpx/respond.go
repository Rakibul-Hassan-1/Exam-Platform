// Package httpx provides small helpers for consistent JSON responses
// and error handling across all handlers, per the API design principle
// that "all APIs should use consistent response structures and HTTP
// status codes."
package httpx

import (
	"encoding/json"
	"log"
	"net/http"
)

type Envelope map[string]any

func JSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("httpx: failed to encode response: %v", err)
	}
}

func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, Envelope{"error": message})
}

func DecodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}
