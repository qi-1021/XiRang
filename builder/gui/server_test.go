package gui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleBuild_InvalidMethod(t *testing.T) {
	server := NewForgeServer(".")
	req := httptest.NewRequest(http.MethodGet, "/api/build", nil)
	rec := httptest.NewRecorder()

	server.handleBuild(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d Method Not Allowed, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

func TestHandleBuild_InvalidJSON(t *testing.T) {
	server := NewForgeServer(".")
	invalidJSON := `{"target_os": "windows", "target_arch":`
	req := httptest.NewRequest(http.MethodPost, "/api/build", strings.NewReader(invalidJSON))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	server.handleBuild(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d Bad Request, got %d", http.StatusBadRequest, rec.Code)
	}
}
