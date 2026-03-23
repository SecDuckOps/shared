package transport

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/SecDuckOps/shared/types"
)

// --- Shared Helpers ---

// parsePagination parses safely bounded offset and limit metrics.
func parsePagination(r *http.Request) (offset int, limit int, err error) {
	offsetStr := r.URL.Query().Get("offset")
	limitStr := r.URL.Query().Get("limit")

	if offsetStr == "" {
		offset = 0
	} else {
		offset, err = strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			return 0, 0, types.New(types.ErrCodeInvalidInput, "offset must be a non-negative integer")
		}
	}

	if limitStr == "" {
		limit = 20
	} else {
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit <= 0 || limit > 100 {
			return 0, 0, types.New(types.ErrCodeInvalidInput, "limit must be a positive integer <= 100")
		}
	}

	return offset, limit, nil
}

// decodeJSON securely reads JSON bodies.
func decodeJSON[T any](w http.ResponseWriter, r *http.Request, dest *T) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB limit max
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields() // Reject unmapped JSON keys immediately
	return decoder.Decode(dest)
}

// writeJSON securely serializes responses. 204 HTTP status handles immediately.
func writeJSON(w http.ResponseWriter, status int, data any) {
	if status == http.StatusNoContent {
		w.WriteHeader(status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

// writeError evaluates standard shared/types.AppError instances returning correctly mapped HTTP statuses.
func writeError(w http.ResponseWriter, err error) {
	if appErr, ok := err.(*types.AppError); ok {
		status := http.StatusInternalServerError

		switch appErr.Code {
		case types.ErrCodeNotFound:
			status = http.StatusNotFound
		case types.ErrCodeAlreadyExists, types.ErrCodeVersionConflict:
			status = http.StatusConflict
		case types.ErrCodeInvalidInput:
			status = http.StatusBadRequest
		case types.ErrCodeTooManyRequests:
			status = http.StatusTooManyRequests
		}

		writeJSON(w, status, appErr.ToMap())
		return
	}

	// Unmapped fatal errors map to 500
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
}
