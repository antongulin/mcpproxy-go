package server

import (
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/smart-mcp-proxy/mcpproxy-go/internal/config"
)

// Parse the management argument strictly even when a caller bypasses schema
// validation. GetString alone would silently discard a non-string setting.
func argumentErrorModeFromRequest(request mcp.CallToolRequest) (string, error) {
	raw, present := request.GetArguments()["tool_argument_error_mode"]
	if !present {
		return "", nil
	}
	mode, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("tool_argument_error_mode must be a string")
	}
	if err := config.ValidateToolArgumentErrorMode(mode); err != nil {
		return "", err
	}
	return mode, nil
}
