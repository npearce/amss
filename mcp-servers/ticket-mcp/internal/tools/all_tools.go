package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/npearce/amss/mcp-servers/ticket-mcp/internal/client"
)

// ClientInterface is the subset of TicketClient that tool handlers use.
type ClientInterface interface {
	Search(mission, severity, status, category, search string, limit, offset int) ([]*client.Ticket, int, error)
	GetTicket(id string) (*client.Ticket, error)
	CreateTicket(title, description, severity, category, reportedBy, mission, assignedTo string, kbArticles []string) (*client.Ticket, error)
	UpdateTicket(id string, updates map[string]interface{}) (*client.Ticket, error)
	AddComment(ticketID, author, text string) (*client.Comment, error)
}

var ticketClient ClientInterface

// SetClient injects the ticket store client before starting the server.
func SetClient(c ClientInterface) {
	ticketClient = c
}

func AddToolsToServer(server *mcp.Server) {
	for _, addToolFunc := range toolsToAdd {
		addToolFunc(server)
	}
}

var toolsToAdd []func(server *mcp.Server)

func registerTool[I, O any](tool MCPTool[I, O]) {
	toolsToAdd = append(toolsToAdd, func(server *mcp.Server) {
		mcp.AddTool(server, &mcp.Tool{Name: tool.Name, Description: tool.Description}, tool.Handler)
	})
}

type MCPTool[I, O any] struct {
	Name        string
	Description string
	Handler     func(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[I]) (*mcp.CallToolResultFor[O], error)
}
