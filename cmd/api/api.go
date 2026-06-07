package main

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/keelwave/keelwave/internal/batch"
	"github.com/keelwave/keelwave/internal/store"
)

type application struct {
	config   config
	pool     *pgxpool.Pool
	store    store.Storage
	batchers *batch.Batchers
	srv      *http.Server
	logger   *zap.SugaredLogger
}

type config struct {
	addr            string
	env             string
	db              dbConfig
	rateLimit       rateLimitConfig
	batch           batchConfig
	shutdownTimeout time.Duration
}

type dbConfig struct {
	addr     string
	maxConns int32
}

type rateLimitConfig struct {
	ingestIPPerMinute  int
	ingestKeyPerMinute int
	ingestWindow       time.Duration
}

type batchConfig struct {
	flushInterval time.Duration
	maxRows       int
	queueDepth    int
}

func (app *application) mount() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Route("/v1", func(r chi.Router) {
		r.Get("/health", app.healthHandler)
		r.Get("/openapi.json", app.openAPIHandler)
		r.Get("/docs", app.docsHandler)

		r.Route("/ingest", func(r chi.Router) {
			r.Use(app.ingestIPRateLimit())
			r.Use(app.apiKeyAuth)
			r.Use(app.ingestKeyRateLimit())
			r.Post("/ai", app.ingestAIHandler)
			r.Post("/events", app.ingestEventHandler)
			r.Post("/metrics", app.ingestMetricHandler)

			r.Route("/agent", func(r chi.Router) {
				r.Post("/runs", app.ingestAgentRunStartHandler)
				r.Post("/runs/{runID}/finish", app.ingestAgentRunFinishHandler)
				r.Post("/steps", app.ingestAgentStepHandler)
			})
		})

		r.Route("/admin", func(r chi.Router) {
			r.Route("/projects", func(r chi.Router) {
				r.Post("/", app.createProjectHandler)
				r.Get("/", app.listProjectsHandler)

				r.Route("/{projectID}", func(r chi.Router) {
					r.Get("/", app.getProjectHandler)
					r.Delete("/", app.deleteProjectHandler)

					r.Route("/keys", func(r chi.Router) {
						r.Post("/", app.createKeyHandler)
						r.Get("/", app.listKeysHandler)
					})
				})
			})

			r.Delete("/keys/{keyID}", app.deleteKeyHandler)
		})
	})

	return r
}

func (app *application) run(mux http.Handler) error {
	app.srv = &http.Server{
		Addr:         app.config.addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  time.Minute,
	}

	app.logger.Infow("server started", "addr", app.config.addr, "env", app.config.env, "version", version)
	return app.srv.ListenAndServe()
}
