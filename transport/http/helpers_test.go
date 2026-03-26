package transport

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSONRejectsUnknownFields(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions", strings.NewReader(`{"name":"duckops","unknown":true}`))
	rec := httptest.NewRecorder()

	var payload struct {
		Name string `json:"name"`
	}
	err := decodeJSON(rec, req, &payload)
	if err == nil {
		t.Fatal("expected decodeJSON to reject unknown fields")
	}
}

func TestDecodeJSONRejectsTrailingPayload(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions", strings.NewReader(`{"name":"duckops"}{"extra":true}`))
	rec := httptest.NewRecorder()

	var payload struct {
		Name string `json:"name"`
	}
	err := decodeJSON(rec, req, &payload)
	if err == nil {
		t.Fatal("expected decodeJSON to reject trailing JSON payload")
	}
}
