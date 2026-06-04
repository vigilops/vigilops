package main

import (
	"context"

	"go.uber.org/zap"

	"github.com/keelwave/keelwave/internal/db"
	"github.com/keelwave/keelwave/internal/env"
)

const version = "0.0.1"

func main() {
	cfg := config{
		addr: env.GetString("ADDR", ":8080"),
		env:  env.GetString("ENV", "development"),
		db: dbConfig{
			addr:     env.GetString("DB_ADDR", "postgres://vigil:vigil@localhost:5432/vigil?sslmode=disable"),
			maxConns: int32(env.GetInt("DB_MAX_CONNS", 30)),
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
		logger: logger,
	}

	mux := app.mount()
	logger.Fatal(app.run(mux))
}
