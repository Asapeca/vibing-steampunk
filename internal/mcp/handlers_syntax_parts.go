package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/oisee/vibing-steampunk/pkg/adt"
)

// checkArtifacts reads SyntaxCheck's artifacts: given reports whether the
// argument is there at all, so a malformed one is an error, not a fallback
// to content.
func checkArtifacts(v any) (parts []adt.CheckArtifact, given bool, err error) {
	if v == nil {
		return nil, false, nil
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, true, fmt.Errorf("artifacts: %w", err)
	}
	if s, ok := v.(string); ok { // a client that sends the array as JSON text
		raw = []byte(s)
	}
	if err := json.Unmarshal(raw, &parts); err != nil {
		return nil, true, fmt.Errorf("artifacts must be an array of {uri, content}: %w", err)
	}
	return parts, true, nil
}

// syntaxCheckParts checks the parts of one class in one check run.
func (s *Server) syntaxCheckParts(ctx context.Context, classURL string, parts []adt.CheckArtifact) (*mcp.CallToolResult, error) {
	results, err := s.adtClient.SyntaxCheckParts(ctx, classURL, parts)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Syntax check failed: %v", err)), nil
	}
	output, _ := json.MarshalIndent(results, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}
