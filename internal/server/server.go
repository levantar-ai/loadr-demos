// Package server wires the HTTP routes for the demo storefront API.
package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/levantar-ai/loadr-demos/internal/store"
)

// Server holds the dependencies shared by the handlers.
type Server struct {
	store  *store.Store
	logger *slog.Logger
}

// New builds the HTTP handler (router + middleware).
func New(st *store.Store, logger *slog.Logger) http.Handler {
	s := &Server{store: st, logger: logger}

	mux := http.NewServeMux()

	// Probes.
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /readyz", s.ready)

	// Catalog (reads).
	mux.HandleFunc("GET /api/products", s.listProducts)
	mux.HandleFunc("GET /api/products/{id}", s.getProduct)

	// Catalog (writes — require auth).
	mux.Handle("POST /api/products", s.requireAuth(http.HandlerFunc(s.createProduct)))

	// Orders.
	mux.HandleFunc("POST /api/orders", s.createOrder)
	mux.HandleFunc("GET /api/orders/{id}", s.getOrder)

	// Auth.
	mux.HandleFunc("POST /api/auth/login", s.login)

	// Synthetic endpoints so load tests can probe CPU + latency behaviour.
	mux.HandleFunc("GET /api/compute", s.compute)
	mux.HandleFunc("GET /api/slow", s.slow)

	return s.recoverer(s.logRequests(mux))
}

// logRequests logs one line per request with status and latency.
func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		s.logger.Info("request",
			"method", r.Method, "path", r.URL.Path,
			"status", sw.status, "dur_ms", time.Since(start).Milliseconds())
	})
}

// recoverer turns a panic into a 500 instead of dropping the connection.
func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				s.logger.Error("panic", "err", rec, "path", r.URL.Path)
				writeError(w, http.StatusInternalServerError, "internal error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
