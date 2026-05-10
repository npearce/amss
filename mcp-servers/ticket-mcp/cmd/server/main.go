package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/npearce/amss/mcp-servers/ticket-mcp/internal/client"
	"github.com/npearce/amss/mcp-servers/ticket-mcp/internal/tools"
)

var httpAddr = flag.String("http", "", "if set, serve MCP over streamable HTTP on this address instead of stdin/stdout")

func main() {
	flag.Parse()
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ticketStoreURL := os.Getenv("TICKET_STORE_URL")
	if ticketStoreURL == "" {
		ticketStoreURL = "http://ticket-store:8082"
	}

	tools.SetClient(client.New(ticketStoreURL))

	server := mcp.NewServer(&mcp.Implementation{Name: "ticket-mcp", Version: "0.1.0"}, nil)
	tools.AddToolsToServer(server)

	if *httpAddr != "" {
		handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
			return server
		}, nil)
		log.Printf("ticket-mcp listening at %s", *httpAddr)
		return http.ListenAndServe(*httpAddr, handler)
	}

	t := mcp.NewLoggingTransport(mcp.NewStdioTransport(), os.Stderr)
	return server.Run(context.Background(), t)
}
