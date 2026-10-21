package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/momerefe/Kabine101/internal/api"
)

type Readiness func(context.Context) error

func Router(ready Readiness) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Recoverer)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			next.ServeHTTP(w, r)
		})
	})
	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) { respond(w, http.StatusOK, api.Ok) })
	r.Get("/api/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := ready(ctx); err != nil {
			respond(w, http.StatusServiceUnavailable, api.Unavailable)
			return
		}
		respond(w, http.StatusOK, api.Ok)
	})
	return r
}

func respond(w http.ResponseWriter, status int, state api.HealthStatus) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(api.Health{Status: state})
}
