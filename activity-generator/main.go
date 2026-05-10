package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

func main() {
	bffURL := getEnv("BFF_URL", "http://localhost:8080")
	scenarioFile := getEnv("SCENARIO_FILE", "scenarios.json")
	pauseSec, err := strconv.Atoi(getEnv("PAUSE_SECONDS", "5"))
	if err != nil || pauseSec < 0 {
		log.Fatalf("invalid PAUSE_SECONDS: must be a non-negative integer")
	}

	scenarios, err := LoadScenarios(scenarioFile)
	if err != nil {
		log.Fatalf("load scenarios: %v", err)
	}

	fmt.Printf("activity-generator: loaded %d scenarios, BFF=%s\n", len(scenarios), bffURL)

	client := &http.Client{Timeout: 30 * time.Second}

	for i, s := range scenarios {
		if i > 0 && pauseSec > 0 {
			time.Sleep(time.Duration(pauseSec) * time.Second)
		}
		RunScenario(client, bffURL, s)
	}

	fmt.Println("activity-generator: done")
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
