package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func main() {
	bffURL := getEnv("BFF_URL", "http://localhost:8080")
	scenarioFile := getEnv("SCENARIO_FILE", "scenarios.json")
	cycleMins, err := strconv.Atoi(getEnv("CYCLE_MINUTES", "22"))
	if err != nil || cycleMins <= 0 {
		log.Fatalf("invalid CYCLE_MINUTES: must be a positive integer")
	}

	scenarios, err := LoadScenarios(scenarioFile)
	if err != nil {
		log.Fatalf("load scenarios: %v", err)
	}

	fmt.Printf("activity-generator: loaded %d scenarios, BFF=%s, cycle=%dm\n", len(scenarios), bffURL, cycleMins)

	client := &http.Client{Timeout: 30 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	for ctx.Err() == nil {
		cycleCtx, cancel := context.WithTimeout(ctx, time.Duration(cycleMins)*time.Minute)
		RunCycle(cycleCtx, client, bffURL, scenarios)
		cancel()

		if ctx.Err() != nil {
			break
		}

		if err := ResetStores(ctx, client, bffURL); err != nil {
			logf("WARNING: reset stores failed: %v", err)
		}
		logf("Cycle complete — resetting stores for next cycle...")
	}

	logf("Shutting down activity generator")
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
