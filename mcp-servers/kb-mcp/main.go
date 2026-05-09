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
		port = "9001"
	}

	kbStoreURL := os.Getenv("KB_STORE_URL")
	if kbStoreURL == "" {
		kbStoreURL = "http://kb-store:8081"
	}

	client = NewKBClient(kbStoreURL)

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","service":"kb-mcp"}`)
	})

	addr := fmt.Sprintf(":%s", port)
	log.Printf("Starting KB MCP server on %s", addr)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

