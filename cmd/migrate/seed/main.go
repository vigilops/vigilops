package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/keelwave/keelwave/internal/auth"
	"github.com/keelwave/keelwave/internal/db"
	"github.com/keelwave/keelwave/internal/env"
	"github.com/keelwave/keelwave/internal/store"
)

func main() {
	ctx := context.Background()

	addr := env.GetString("DB_ADDR", "postgres://keelwave:keelwave@localhost:5432/keelwave?sslmode=disable")
	projectName := env.GetString("SEED_PROJECT_NAME", "dev")
	keyName := env.GetString("SEED_KEY_NAME", "seed")

	pool, err := db.New(ctx, addr, 5)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	s := store.NewStorage(pool)

	p := &store.Project{Name: projectName}
	if err := s.Projects.Create(ctx, p); err != nil {
		log.Fatalf("create project: %v", err)
	}

	plaintext, hash, err := auth.Generate()
	if err != nil {
		log.Fatalf("generate api key: %v", err)
	}

	k := &store.APIKey{ProjectID: p.ID, KeyHash: hash, Name: keyName}
	if err := s.APIKeys.Create(ctx, k); err != nil {
		log.Fatalf("create api key: %v", err)
	}

	fmt.Fprintf(os.Stdout,
		"project_id:   %s\nproject_name: %s\napi_key_id:   %s\napi_key:      %s\n",
		p.ID, p.Name, k.ID, plaintext,
	)
	fmt.Fprintln(os.Stderr, "store the api_key now — server only keeps the SHA-256 hash")
}
