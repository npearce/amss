package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/npearce/amss/mcp-servers/kb-mcp/internal/client"
)

// ClientInterface is the subset of KBClient that tool handlers use.
// Keeping it here lets tests inject a mock without importing the real client.
type ClientInterface interface {
	Search(category, tags, search string, limit, offset int) ([]*client.Article, int, error)
	GetArticle(id string) (*client.Article, error)
	CreateArticle(title, body, category, createdBy string, tags []string) (*client.Article, error)
	UpdateArticle(id string, updates map[string]interface{}) (*client.Article, error)
}

var kbClient ClientInterface

// SetClient injects the KB store client before starting the server.
func SetClient(c ClientInterface) {
	kbClient = c
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
