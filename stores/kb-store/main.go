package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	seedPath := os.Getenv("SEED_PATH")
	if seedPath == "" {
		seedPath = "seed-data/kb.json"
	}

	var err error
	store, err = NewStore(seedPath)
	if err != nil {
		log.Fatalf("Failed to initialize store: %v", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("GET /articles", handleArticles)
	mux.HandleFunc("POST /articles", handleArticles)
	mux.HandleFunc("GET /articles/{id}", handleArticleByID)
	mux.HandleFunc("PUT /articles/{id}", handleArticleByID)
	mux.HandleFunc("DELETE /articles/{id}", handleArticleByID)
	mux.HandleFunc("POST /reset", handleReset)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("Starting KB store on %s", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
