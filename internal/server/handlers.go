package server

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/levantar-ai/loadr-demos/internal/store"
)

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	if err := s.store.Ping(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) listProducts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	limit := clampInt(queryInt(r, "limit", 20), 1, 100)
	offset := clampInt(queryInt(r, "offset", 0), 0, 1_000_000)

	products, err := s.store.ListProducts(r.Context(), q, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"count":    len(products),
		"limit":    limit,
		"offset":   offset,
		"products": products,
	})
}

func (s *Server) getProduct(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	p, err := s.store.GetProduct(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "product not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) createProduct(w http.ResponseWriter, r *http.Request) {
	var p store.Product
	if err := decodeJSON(r, &p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if p.SKU == "" || p.Name == "" {
		writeError(w, http.StatusUnprocessableEntity, "sku and name are required")
		return
	}
	id, err := s.store.CreateProduct(r.Context(), p)
	if err != nil {
		writeError(w, http.StatusConflict, "could not create product (duplicate sku?)")
		return
	}
	p.ID = id
	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) createOrder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Customer string            `json:"customer"`
		Items    []store.OrderItem `json:"items"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.Customer == "" || len(req.Items) == 0 {
		writeError(w, http.StatusUnprocessableEntity, "customer and items are required")
		return
	}
	order, err := s.store.CreateOrder(r.Context(), req.Customer, req.Items)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusConflict, "one or more items are unavailable")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not place order")
		return
	}
	writeJSON(w, http.StatusCreated, order)
}

func (s *Server) getOrder(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	o, err := s.store.GetOrder(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	writeJSON(w, http.StatusOK, o)
}

// topSellers returns an expensive aggregation, cached in Redis. A cold cache
// means every concurrent request runs the heavy query (a "thundering herd"),
// which is exactly what the impulse test exercises: high concurrency from t=0
// with nothing warm. The X-Cache header reports HIT/MISS.
func (s *Server) topSellers(w http.ResponseWriter, r *http.Request) {
	const key = "report:top-sellers"

	if cached, ok := s.cache.Get(r.Context(), key); ok {
		w.Header().Set("X-Cache", "HIT")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(cached))
		return
	}

	rows, err := s.store.TopSellers(r.Context(), 10)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "report query failed")
		return
	}
	body, err := json.Marshal(map[string]any{"cached": false, "top_sellers": rows})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "encode failed")
		return
	}
	// Cache the warm payload (with cached:true) for the next 30s.
	warm, _ := json.Marshal(map[string]any{"cached": true, "top_sellers": rows})
	s.cache.Set(r.Context(), key, string(warm), 30*time.Second)

	w.Header().Set("X-Cache", "MISS")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

// login is a deliberately simple demo auth endpoint: any username with a
// non-empty password gets a bearer token. It exists so load tests can show a
// realistic login -> extract token -> authenticated request flow.
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &req); err != nil || req.Username == "" || req.Password == "" {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	sum := sha256.Sum256([]byte(req.Username + ":" + req.Password))
	token := "tok_" + hex(sum[:8])
	writeJSON(w, http.StatusOK, map[string]string{"token": token, "user": req.Username})
}

// requireAuth rejects requests without a non-empty Bearer token.
func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") || strings.TrimSpace(auth[len("Bearer "):]) == "" {
			writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// compute does ?n iterations of SHA-256 hashing — a CPU-bound endpoint whose
// latency rises predictably with concurrency, so stress tests find the knee.
func (s *Server) compute(w http.ResponseWriter, r *http.Request) {
	n := clampInt(queryInt(r, "n", 5000), 1, 2_000_000)
	buf := []byte("loadr")
	for i := 0; i < n; i++ {
		sum := sha256.Sum256(buf)
		buf = sum[:]
	}
	writeJSON(w, http.StatusOK, map[string]any{"n": n, "digest": hex(buf[:8])})
}

// slow sleeps for ?ms milliseconds — a controllable-latency endpoint.
func (s *Server) slow(w http.ResponseWriter, r *http.Request) {
	ms := clampInt(queryInt(r, "ms", 100), 0, 10_000)
	select {
	case <-time.After(time.Duration(ms) * time.Millisecond):
		writeJSON(w, http.StatusOK, map[string]any{"slept_ms": ms})
	case <-r.Context().Done():
		writeError(w, http.StatusRequestTimeout, "client gone")
	}
}

// --- helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decodeJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func queryInt(r *http.Request, key string, def int) int {
	if v := r.URL.Query().Get(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func hex(b []byte) string {
	const digits = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, c := range b {
		out[i*2] = digits[c>>4]
		out[i*2+1] = digits[c&0x0f]
	}
	return string(out)
}
