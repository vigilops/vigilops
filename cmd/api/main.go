package main

import (
	"context"
	"time"

	"go.uber.org/zap"

	_ "github.com/keelwave/keelwave/docs"
	"github.com/keelwave/keelwave/internal/db"
	"github.com/keelwave/keelwave/internal/env"
	"github.com/keelwave/keelwave/internal/store"
)

const version = "0.0.1"

//	@title			Keelwave API
//	@description	Unified observability platform for AI agents, APIs, and infrastructure.
//	@version		0.0.1

// @BasePath					/v1
//
// @securityDefinitions.apikey	ApiKeyAuth
// @in							header
// @name						Authorization
// @description				Format: "Bearer vgl_<token>"
func main() {
	cfg := config{
		addr: env.GetString("ADDR", ":8080"),
		env:  env.GetString("ENV", "development"),
		db: dbConfig{
			addr:     env.GetString("DB_ADDR", "postgres://vigil:vigil@localhost:5432/vigil?sslmode=disable"),
			maxConns: int32(env.GetInt("DB_MAX_CONNS", 30)),
		},
		rateLimit: rateLimitConfig{
			ingestIPPerMinute:  env.GetInt("RATE_LIMIT_INGEST_IP_PER_MINUTE", 100),
			ingestKeyPerMinute: env.GetInt("RATE_LIMIT_INGEST_KEY_PER_MINUTE", 1000),
			ingestWindow:       parseDurationOr("RATE_LIMIT_INGEST_WINDOW", time.Minute),
		},
	}

	logger := zap.Must(zap.NewProduction()).Sugar()
	defer logger.Sync()

	ctx := context.Background()
	pool, err := db.New(ctx, cfg.db.addr, cfg.db.maxConns)
	if err != nil {
		logger.Fatalw("db connect failed", "err", err)
	}
	defer pool.Close()
	logger.Info("db connected")

	app := &application{
		config: cfg,
		pool:   pool,
		store:  store.NewStorage(pool),
		logger: logger,
	}

	mux := app.mount()
	logger.Fatal(app.run(mux))
}

func parseDurationOr(key string, def time.Duration) time.Duration {
	s := env.GetString(key, "")
	if s == "" {
		return def
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return def
	}
	return d
}
