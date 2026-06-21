// Command seeddemo generates a realistic demo project: ~200 agent runs with
// steps carrying latency, tokens, cost, failures, and loops — enough to make
// every dashboard panel render believable data without a live SDK.
package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/google/uuid"

	"github.com/keelwave/keelwave/internal/auth"
	"github.com/keelwave/keelwave/internal/db"
	"github.com/keelwave/keelwave/internal/env"
	"github.com/keelwave/keelwave/internal/store"
)

var (
	agents = []string{
		"research-agent", "bug-triage", "code-reviewer",
		"data-pipeline", "support-bot", "looper-demo",
	}
	tools = []string{
		"search", "read_file", "run_tests", "web_fetch",
		"summarize", "sql_query", "write_file",
	}
	thoughts = []string{
		"Plan the next step.", "Need more context.", "Verify the result.",
		"Refine the query.", "Check the failing case.", "Compile the answer.",
	}
)

// costRate ≈ blended $3 / 1M tokens → realistic demo $ figures.
const costRate = 0.000003

func fingerprint(tool, input string) []byte {
	h := sha256.New()
	h.Write([]byte(tool))
	h.Write([]byte{0})
	h.Write([]byte(input))
	return h.Sum(nil)
}

func main() {
	ctx := context.Background()

	addr := env.GetString("DB_ADDR", "postgres://keelwave:keelwave@localhost:5432/keelwave?sslmode=disable")
	projectName := env.GetString("SEED_PROJECT_NAME", "demo")
	runCount := env.GetInt("SEED_RUNS", 200)

	pool, err := db.New(ctx, addr, 5)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	s := store.NewStorage(pool)
	rng := rand.New(rand.NewSource(42)) // fixed seed → reproducible dataset

	p := &store.Project{Name: projectName}
	if err := s.Projects.Create(ctx, p); err != nil {
		log.Fatalf("create project: %v", err)
	}
	plaintext, hash, err := auth.Generate()
	if err != nil {
		log.Fatalf("generate api key: %v", err)
	}
	k := &store.APIKey{ProjectID: p.ID, KeyHash: hash, Name: "demo"}
	if err := s.APIKeys.Create(ctx, k); err != nil {
		log.Fatalf("create api key: %v", err)
	}

	now := time.Now()
	for i := 0; i < runCount; i++ {
		if err := genRun(ctx, s, p.ID, rng, now); err != nil {
			log.Fatalf("gen run %d: %v", i, err)
		}
	}

	fmt.Fprintf(os.Stdout,
		"project_id:   %s\nproject_name: %s\napi_key:      %s\nruns:         %d\n",
		p.ID, p.Name, plaintext, runCount,
	)
	fmt.Fprintln(os.Stderr, "store the api_key now — server only keeps the SHA-256 hash")
}

func genRun(ctx context.Context, s store.Storage, projectID uuid.UUID, rng *rand.Rand, now time.Time) error {
	agent := agents[rng.Intn(len(agents))]
	// spread over the last 30 days
	ts := now.Add(-time.Duration(rng.Intn(30*24*60)) * time.Minute)

	roll := rng.Float64()
	isLoop := roll < 0.05
	isFailed := !isLoop && roll < 0.15
	nSteps := 3 + rng.Intn(13) // 3..15

	run := &store.AgentRun{
		ProjectID: projectID,
		AgentName: agent,
		Status:    "running",
		Timestamp: ts,
		Input:     new(fmt.Sprintf("task for %s", agent)),
	}
	if err := s.AgentRuns.Insert(ctx, run); err != nil {
		return err
	}

	// For a loop run, all tool_calls reuse one fingerprint so the loop view fires.
	loopTool := tools[rng.Intn(len(tools))]
	loopInput := `{"q":"same query"}`

	var (
		totalTokens   int
		totalCost     float64
		elapsedMs     int
		loopStepIndex *int
	)
	for i := 0; i < nSteps; i++ {
		stepIdx := i + 1
		stepTS := ts.Add(time.Duration(elapsedMs) * time.Millisecond)
		// cycle think → tool_call → tool_result
		kind := i % 3

		step := &store.AgentStep{
			ProjectID:  projectID,
			AgentRunID: run.ID,
			StepIndex:  stepIdx,
			Timestamp:  stepTS,
		}

		switch kind {
		case 0:
			step.StepType = "think"
			step.Content = new(thoughts[rng.Intn(len(thoughts))])
			tok := 50 + rng.Intn(150)
			step.Tokens = new(tok)
			step.CostUSD = new(float64(tok) * costRate)
			elapsedMs += 10 + rng.Intn(40)
		case 1:
			step.StepType = "tool_call"
			tool := tools[rng.Intn(len(tools))]
			input := fmt.Sprintf(`{"arg":"%d"}`, rng.Intn(1000))
			if isLoop {
				tool, input = loopTool, loopInput
				if loopStepIndex == nil {
					loopStepIndex = new(stepIdx)
				}
			}
			step.ToolName = new(tool)
			step.ToolInput = []byte(input)
			step.InputFingerprint = fingerprint(tool, input)
			lat := 5 + rng.Intn(1995)
			step.ToolLatencyMs = new(lat)
			// last step fails on a failed run
			ok := !(isFailed && i >= nSteps-2) && rng.Float64() > 0.08
			step.ToolSuccess = new(ok)
			tok := 20 + rng.Intn(80)
			step.Tokens = new(tok)
			step.CostUSD = new(float64(tok) * costRate)
			elapsedMs += lat
		default:
			step.StepType = "tool_result"
			step.ToolOutput = []byte(`{"results":[]}`)
			step.ToolSuccess = new(true)
			tok := 100 + rng.Intn(700)
			step.Tokens = new(tok)
			step.CostUSD = new(float64(tok) * costRate)
			elapsedMs += 5 + rng.Intn(30)
		}

		if step.Tokens != nil {
			totalTokens += *step.Tokens
			totalCost += *step.CostUSD
		}
		if err := s.AgentSteps.Insert(ctx, step); err != nil {
			return err
		}
	}

	status := "completed"
	reason := "clean"
	if isFailed {
		status, reason = "failed", "error"
	} else if isLoop {
		// A run halted by the loop guard did not finish its task — it's a failure.
		status, reason = "failed", "loop_detected"
	}

	return s.AgentRuns.Finish(ctx, run.ID, run.Timestamp, store.AgentRunFinish{
		Status:            status,
		TerminationReason: new(reason),
		LoopDetected:      isLoop,
		LoopStepIndex:     loopStepIndex,
		TotalSteps:        nSteps,
		TotalTokens:       totalTokens,
		TotalCostUSD:      new(totalCost),
		DurationMs:        new(elapsedMs),
		Output:            new("done"),
	})
}
