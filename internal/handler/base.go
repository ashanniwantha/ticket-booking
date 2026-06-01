package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// BaseHandler encapsulates all shared dependencies and utility behaviors
type BaseHandler struct {
	logger *slog.Logger
}

// NewBaseHandler intialize the shared handler base components.
func NewBaseHandler(logger *slog.Logger) *BaseHandler {
	return &BaseHandler{logger: logger}
}

// Respond handles standard JSON serialization and HTTP status headers
func (b *BaseHandler) Respond(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// Check serialization error to handle
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			b.logger.Error("failed to encode respond payload", "err", err)
		}
	}
}

// RespondError marshals a clean, standardized JSON error to the client
func (b *BaseHandler) RespondError(w http.ResponseWriter, status int, msg string) {
	b.Respond(w, status, map[string]string{"error": msg})
}

// ParseIDParam extracts and validate and int64 path parameter router key
func (b *BaseHandler) ParseIDParam(r *http.Request, key string) (int64, error) {
	idStr := chi.URLParam(r, key)

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid identifier format")
	}

	return id, nil
}

// RespondDecodeError translates low-level JSON parsing failures into descriptive client messages.
func (b *BaseHandler) RespondDecodeError(w http.ResponseWriter, err error) {
	var syntaxError *json.SyntaxError
	var unmarshalTypeError *json.UnmarshalTypeError

	switch {
	case errors.As(err, &syntaxError):
		b.RespondError(w, http.StatusBadRequest, "malformed json syntax")
	case errors.As(err, &unmarshalTypeError):
		b.RespondError(w, http.StatusBadRequest, "invalid field type for field: "+unmarshalTypeError.Field)
	case errors.Is(err, io.EOF):
		b.RespondError(w, http.StatusBadRequest, "request body cannot be empty")
	default:
		b.RespondError(w, http.StatusBadRequest, "invalid request configuration")
	}
}

// DecodeJSON limits the request body to 1MB
// And automatically handles client error respond if decoding fails
// It returns true if decoding succeeded, and false if it failed
func (b *BaseHandler) DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields() // Reject extra JSON keys

	if err := dec.Decode(dst); err != nil {
		b.RespondDecodeError(w, err) // Automatically maps and writes the descriptive error
		return false
	}
	return true
}
