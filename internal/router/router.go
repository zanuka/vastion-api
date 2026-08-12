package router

import (
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/zanuka/com-scan-api/internal/handler"
	mw "github.com/zanuka/com-scan-api/internal/middleware"
	"github.com/zanuka/com-scan-api/internal/service"
)

type Deps struct {
	Health      *service.Health
	CORSOrigins []string
	Logger      *slog.Logger
}

func New(deps Deps) http.Handler {
	mux := chi.NewMux()
	mux.Use(chimw.RequestID)
	mux.Use(chimw.Recoverer)
	mux.Use(mw.RequestLog(deps.Logger))
	mux.Use(cors.Handler(cors.Options{
		AllowedOrigins:   deps.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-Id", "X-Operator-Role", "X-Operator-Name"},
		ExposedHeaders:   []string{"X-Request-Id"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	api := humachi.New(mux, huma.DefaultConfig("com-scan-api", "0.1.0"))
	handler.NewHealth(deps.Health).Register(api)
	return mux
}
