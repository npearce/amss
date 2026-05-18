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

	// Configure auth from environment.
	// Refresh mode: all KEYCLOAK_* vars set — token fetched and refreshed each cycle.
	// Static mode:  only KEYCLOAK_TOKEN set — used as-is, no refresh.
	// No auth:      neither set — requests proceed without Authorization header.
	auth := &AuthConfig{
		StaticToken:          getEnv("KEYCLOAK_TOKEN", ""),
		KeycloakURL:          getEnv("KEYCLOAK_URL", ""),
		KeycloakClientID:     getEnv("KEYCLOAK_CLIENT_ID", ""),
		KeycloakClientSecret: getEnv("KEYCLOAK_CLIENT_SECRET", ""),
		KeycloakUsername:     getEnv("KEYCLOAK_USERNAME", ""),
		KeycloakPassword:     getEnv("KEYCLOAK_PASSWORD", ""),
	}
	switch auth.mode() {
	case "refresh":
		fmt.Println("activity-generator: Using Keycloak auth (token refresh enabled)")
	case "static":
		fmt.Println("activity-generator: Using static auth token")
	default:
		fmt.Println("activity-generator: No auth configured")
	}

	fmt.Printf("activity-generator: loaded %d scenarios, BFF=%s, cycle=%dm\n", len(scenarios), bffURL, cycleMins)

	client := &http.Client{Timeout: 30 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	for ctx.Err() == nil {
		// Refresh the token at the start of each cycle (same cadence as store resets).
		if auth.mode() == "refresh" {
			if err := auth.Refresh(client); err != nil {
				logf("WARNING: token refresh failed: %v", err)
			}
		}

		cycleCtx, cancel := context.WithTimeout(ctx, time.Duration(cycleMins)*time.Minute)
		RunCycle(cycleCtx, client, bffURL, scenarios, auth)
		cancel()

		if ctx.Err() != nil {
			break
		}

		if err := ResetStores(ctx, client, bffURL, auth); err != nil {
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
