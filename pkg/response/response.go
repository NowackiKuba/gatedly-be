package response

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// AppError is a sentinel error carrying HTTP status and message.
type AppError struct {
	Status  int
	Message string
}

func (e *AppError) Error() string {
	return e.Message
}

// BadRequest returns an AppError with status 400.
func BadRequest(msg string) *AppError {
	return &AppError{Status: http.StatusBadRequest, Message: msg}
}

// Unauthorized returns an AppError with status 401.
func Unauthorized(msg string) *AppError {
	return &AppError{Status: http.StatusUnauthorized, Message: msg}
}

// NotFound returns an AppError with status 404.
func NotFound(msg string) *AppError {
	return &AppError{Status: http.StatusNotFound, Message: msg}
}

// Conflict returns an AppError with status 409.
func Conflict(msg string) *AppError {
	return &AppError{Status: http.StatusConflict, Message: msg}
}

// JSON writes status and data as JSON to w.
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if data == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Fallback: status already written
		_, _ = w.Write([]byte(fmt.Sprintf(`{"error":"%s"}`, err.Error())))
	}
}

// Error writes the appropriate status and body for err. If err is an *AppError,
// uses its status and message; otherwise returns 500.
func Error(w http.ResponseWriter, err error) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		JSON(w, appErr.Status, map[string]string{"error": appErr.Message})
		return
	}
	JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
}

// Decode reads JSON from r into dst with DisallowUnknownFields. Returns a
// BadRequest AppError on decode failure.
func Decode(r *http.Request, dst interface{}) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return BadRequest(fmt.Sprintf("invalid JSON: %v", err))
	}
	return nil
}
