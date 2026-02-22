package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pantry/internal/core"
	"pantry/internal/models"
)

// pantryService is the subset of core.Service used by MCP tool handlers.
// Defining it here allows tests to inject stubs without depending on core.Service.
type pantryService interface {
	Store(raw models.RawItemInput, project string) (map[string]any, error)
	Search(query string, limit int, project *string, source *string, useVectors bool) ([]models.SearchResult, error)
	GetContext(limit int, project *string, source *string, query *string, semanticMode string, topupRecent bool) ([]models.SearchResult, int64, error)
	Close() error
}

// RunServer starts the MCP server with stdio transport.
func RunServer() error {
	svc, err := core.NewService("")
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	defer func() { _ = svc.Close() }()

	// Create MCP server
	mcpServer := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    "pantry",
		Version: "0.1.0",
	}, nil)

	// Register tools
	if err := registerTools(mcpServer, svc); err != nil {
		return fmt.Errorf("failed to register tools: %w", err)
	}

	// Run server with stdio transport
	return mcpServer.Run(context.Background(), &mcpsdk.StdioTransport{})
}

// registerTools registers all pantry tools with the MCP server.
//
//nolint:unparam
func registerTools(s *mcpsdk.Server, svc pantryService) error {
	// Register pantry_store tool
	//nolint:revive
	storeHandler := func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]any) (*mcpsdk.CallToolResult, map[string]any, error) {
		result, err := HandlePantryStore(svc, input)
		if err != nil {
			return &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{
					&mcpsdk.TextContent{Text: fmt.Sprintf("Error: %v", err)},
				},
				IsError: true,
			}, nil, nil
		}

		return nil, result, nil
	}
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "pantry_store",
		Description: "Store a note for future sessions. You MUST call this before ending any session where you made changes, fixed bugs, made decisions, or learned something.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"title":         map[string]any{"type": "string", "description": "Short descriptive title"},
				"what":          map[string]any{"type": "string", "description": "What happened or was decided"},
				"why":           map[string]any{"type": "string", "description": "Reasoning behind it"},
				"impact":        map[string]any{"type": "string", "description": "What changed as a result"},
				"tags":          map[string]any{"type": []any{"string", "array"}, "items": map[string]any{"type": "string"}, "description": "Comma-separated string or array of tags"},
				"category":      map[string]any{"type": "string", "enum": []string{"decision", "pattern", "bug", "context", "learning"}},
				"related_files": map[string]any{"type": []any{"string", "array"}, "items": map[string]any{"type": "string"}, "description": "Comma-separated string or array of file paths"},
				"details":       map[string]any{"type": "string", "description": "Full context with all important details"},
				"source":        map[string]any{"type": "string", "description": "Source agent name"},
				"project":       map[string]any{"type": "string", "description": "Project name (defaults to current directory)"},
			},
			"required": []string{"title", "what"},
		},
	}, storeHandler)

	// Register pantry_search tool
	//nolint:revive
	searchHandler := func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]any) (*mcpsdk.CallToolResult, map[string]any, error) {
		results, err := HandlePantrySearch(svc, input)
		if err != nil {
			return &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{
					&mcpsdk.TextContent{Text: fmt.Sprintf("Error: %v", err)},
				},
				IsError: true,
			}, nil, nil
		}

		return nil, map[string]any{"results": results}, nil
	}
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "pantry_search",
		Description: "Search notes using keyword and semantic search. Returns matching notes ranked by relevance.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query":   map[string]any{"type": "string", "description": "Search query"},
				"limit":   map[string]any{"type": "integer", "description": "Maximum number of notes", "default": 5},
				"project": map[string]any{"type": "string", "description": "Filter by project"},
				"source":  map[string]any{"type": "string", "description": "Filter by source"},
			},
			"required": []string{"query"},
		},
	}, searchHandler)

	// Register pantry_context tool
	//nolint:revive
	contextHandler := func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]any) (*mcpsdk.CallToolResult, map[string]any, error) {
		result, err := HandlePantryContext(svc, input)
		if err != nil {
			return &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{
					&mcpsdk.TextContent{Text: fmt.Sprintf("Error: %v", err)},
				},
				IsError: true,
			}, nil, nil
		}

		return nil, result, nil
	}
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "pantry_context",
		Description: "Get notes for the current project. Returns prior decisions, bugs, and context.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"limit":   map[string]any{"type": "integer", "description": "Maximum number of notes", "default": 10},
				"project": map[string]any{"type": "string", "description": "Project name (defaults to current directory)"},
				"source":  map[string]any{"type": "string", "description": "Filter by source"},
			},
		},
	}, contextHandler)

	return nil
}

// HandlePantryStore handles the pantry_store tool call.
func HandlePantryStore(svc pantryService, params map[string]any) (map[string]any, error) {
	title, _ := params["title"].(string)
	what, _ := params["what"].(string)
	why, _ := getStringFromMap(params, "why")
	impact, _ := getStringFromMap(params, "impact")
	tags, _ := getStringSliceFromMap(params, "tags")
	category, _ := getStringFromMap(params, "category")
	relatedFiles, _ := getStringSliceFromMap(params, "related_files")
	details, _ := getStringFromMap(params, "details")
	source, _ := getStringFromMap(params, "source")
	project, _ := getStringFromMap(params, "project")

	if project == "" {
		project = filepath.Base(getCurrentDir())
	}

	raw := models.RawItemInput{
		Title: title,
		What:  what,
	}

	if why != "" {
		raw.Why = &why
	}

	if impact != "" {
		raw.Impact = &impact
	}

	if category != "" {
		raw.Category = &category
	}

	if source != "" {
		raw.Source = &source
	}

	if details != "" {
		raw.Details = &details
	}

	raw.Tags = tags
	raw.RelatedFiles = relatedFiles

	result, err := svc.Store(raw, project)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// HandlePantrySearch handles the pantry_search tool call.
func HandlePantrySearch(svc pantryService, params map[string]any) ([]map[string]any, error) {
	query, _ := params["query"].(string)

	limit := 5
	if l, ok := params["limit"].(float64); ok {
		limit = int(l)
	}

	var project *string
	if p, ok := params["project"].(string); ok && p != "" {
		project = &p
	}

	results, err := svc.Search(query, limit, project, nil, true)
	if err != nil {
		return nil, err
	}

	clean := make([]map[string]any, len(results))
	for i, r := range results {
		clean[i] = map[string]any{
			"id":          r.ID,
			"title":       r.Title,
			"what":        r.What,
			"why":         r.Why,
			"impact":      r.Impact,
			"category":    r.Category,
			"tags":        r.Tags,
			"project":     r.Project,
			"source":      r.Source,
			"created_at":  r.CreatedAt[:10],
			"score":       r.Score,
			"has_details": r.HasDetails,
		}
	}

	return clean, nil
}

// HandlePantryContext handles the pantry_context tool call.
func HandlePantryContext(svc pantryService, params map[string]any) (map[string]any, error) {
	limit := 10
	if l, ok := params["limit"].(float64); ok {
		limit = int(l)
	}

	var project *string
	if p, ok := params["project"].(string); ok && p != "" {
		project = &p
	} else {
		proj := filepath.Base(getCurrentDir())
		project = &proj
	}

	results, total, err := svc.GetContext(limit, project, nil, nil, "never", false)
	if err != nil {
		return nil, err
	}

	notes := make([]map[string]any, len(results))

	for i, r := range results {
		dateStr := r.CreatedAt[:10]
		notes[i] = map[string]any{
			"id":       r.ID,
			"title":    r.Title,
			"category": r.Category,
			"tags":     r.Tags,
			"date":     dateStr,
		}
	}

	return map[string]any{
		"total":   total,
		"showing": len(notes),
		"notes":   notes,
	}, nil
}

// Helper functions.
//
//nolint:unparam
func getStringFromMap(m map[string]any, key string) (string, bool) {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str, true
		}
	}

	return "", false
}

func getStringSliceFromMap(m map[string]any, key string) ([]string, bool) {
	//nolint:nestif
	if val, ok := m[key]; ok {
		if arr, ok := val.([]any); ok {
			result := make([]string, len(arr))

			for i, v := range arr {
				if str, ok := v.(string); ok {
					result[i] = str
				}
			}

			return result, true
		}

		if str, ok := val.(string); ok {
			// Try to parse as JSON array
			var arr []string
			if err := json.Unmarshal([]byte(str), &arr); err == nil {
				return arr, true
			}
			// Fallback: comma-separated string
			parts := strings.Split(str, ",")

			result := make([]string, 0, len(parts))

			for _, p := range parts {
				if t := strings.TrimSpace(p); t != "" {
					result = append(result, t)
				}
			}

			if len(result) > 0 {
				return result, true
			}
		}
	}

	return nil, false
}

// getCurrentDir returns the current working directory, or "unknown" if it
// cannot be determined. This prevents filepath.Base("") returning "." which
// would silently be stored as a project name.
func getCurrentDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return "unknown"
	}

	return dir
}
