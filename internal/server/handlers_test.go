package server

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestServer builds the handler with a nil store; the routes exercised here
// never touch the database.
func newTestServer() http.Handler {
	return New(nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestHealthz(t *testing.T) {
	rr := httptest.NewRecorder()
	newTestServer().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"ok"`) {
		t.Fatalf("unexpected body: %s", rr.Body.String())
	}
}

func TestLoginReturnsToken(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login",
		strings.NewReader(`{"username":"ada","password":"secret"}`))
	newTestServer().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"token":"tok_`) {
		t.Fatalf("expected a token, got: %s", rr.Body.String())
	}
}

func TestLoginRejectsEmpty(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login",
		strings.NewReader(`{"username":"","password":""}`))
	newTestServer().ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want 401", rr.Code)
	}
}

func TestCreateProductRequiresAuth(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/products",
		strings.NewReader(`{"sku":"X-1","name":"Test"}`))
	newTestServer().ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want 401 (no bearer token)", rr.Code)
	}
}

func TestCompute(t *testing.T) {
	rr := httptest.NewRecorder()
	newTestServer().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/compute?n=100", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"digest"`) {
		t.Fatalf("unexpected body: %s", rr.Body.String())
	}
}
