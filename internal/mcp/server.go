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

// RunServer starts the MCP server with stdio transport
func RunServer() error {
	svc, err := core.NewService("")
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}
	defer svc.Close()

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

// registerTools registers all pantry tools with the MCP server
func registerTools(s *mcpsdk.Server, svc *core.Service) error {
	// Register pantry_store tool
	storeHandler := func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]interface{}) (*mcpsdk.CallToolResult, map[string]interface{}, error) {
		result, err := HandlePantryStore(svc, input)
		if err != nil {
			return &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{
					&mcpsdk.TextContent{Text: fmt.Sprintf("Error: %v", err)},
				},
				IsError: true,
			}, nil, nil
		}
		// Convert map[string]string to map[string]interface{}
		resultMap := make(map[string]interface{})
		for k, v := range result {
			resultMap[k] = v
		}
		return nil, resultMap, nil
	}
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "pantry_store",
		Description: "Save a memory for future sessions. You MUST call this before ending any session where you made changes, fixed bugs, made decisions, or learned something.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"title":        map[string]interface{}{"type": "string", "description": "Short descriptive title"},
				"what":          map[string]interface{}{"type": "string", "description": "What happened or was decided"},
				"why":           map[string]interface{}{"type": "string", "description": "Reasoning behind it"},
				"impact":       map[string]interface{}{"type": "string", "description": "What changed as a result"},
				"tags":          map[string]interface{}{"type": []interface{}{"string", "array"}, "items": map[string]interface{}{"type": "string"}, "description": "Comma-separated string or array of tags"},
				"category":      map[string]interface{}{"type": "string", "enum": []string{"decision", "pattern", "bug", "context", "learning"}},
				"related_files": map[string]interface{}{"type": []interface{}{"string", "array"}, "items": map[string]interface{}{"type": "string"}, "description": "Comma-separated string or array of file paths"},
				"details":       map[string]interface{}{"type": "string", "description": "Full context with all important details"},
				"source":        map[string]interface{}{"type": "string", "description": "Source agent name"},
				"project":       map[string]interface{}{"type": "string", "description": "Project name (defaults to current directory)"},
			},
			"required": []string{"title", "what"},
		},
	}, storeHandler)

	// Register pantry_search tool
	searchHandler := func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]interface{}) (*mcpsdk.CallToolResult, map[string]interface{}, error) {
		results, err := HandlePantrySearch(svc, input)
		if err != nil {
			return &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{
					&mcpsdk.TextContent{Text: fmt.Sprintf("Error: %v", err)},
				},
				IsError: true,
			}, nil, nil
		}
		return nil, map[string]interface{}{"results": results}, nil
	}
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "pantry_search",
		Description: "Search memories using keyword and semantic search. Returns matching memories ranked by relevance.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query":   map[string]interface{}{"type": "string", "description": "Search query"},
				"limit":   map[string]interface{}{"type": "integer", "description": "Maximum number of results", "default": 5},
				"project": map[string]interface{}{"type": "string", "description": "Filter by project"},
				"source":  map[string]interface{}{"type": "string", "description": "Filter by source"},
			},
			"required": []string{"query"},
		},
	}, searchHandler)

	// Register pantry_context tool
	contextHandler := func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]interface{}) (*mcpsdk.CallToolResult, map[string]interface{}, error) {
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
		Description: "Get memory context for the current project. Returns prior decisions, bugs, and context.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"limit":   map[string]interface{}{"type": "integer", "description": "Maximum number of items", "default": 10},
				"project": map[string]interface{}{"type": "string", "description": "Project name (defaults to current directory)"},
				"source":  map[string]interface{}{"type": "string", "description": "Filter by source"},
			},
		},
	}, contextHandler)

	return nil
}

// HandlePantryStore handles the pantry_store tool call
func HandlePantryStore(svc *core.Service, params map[string]interface{}) (map[string]interface{}, error) {
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

	// Convert map[string]string to map[string]interface{}
	resultMap := make(map[string]interface{})
	for k, v := range result {
		resultMap[k] = v
	}

	return resultMap, nil
}

// HandlePantrySearch handles the pantry_search tool call
func HandlePantrySearch(svc *core.Service, params map[string]interface{}) ([]map[string]interface{}, error) {
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

	clean := make([]map[string]interface{}, len(results))
	for i, r := range results {
		clean[i] = map[string]interface{}{
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

// HandlePantryContext handles the pantry_context tool call
func HandlePantryContext(svc *core.Service, params map[string]interface{}) (map[string]interface{}, error) {
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

	memories := make([]map[string]interface{}, len(results))
	for i, r := range results {
		dateStr := r.CreatedAt[:10]
		memories[i] = map[string]interface{}{
			"id":       r.ID,
			"title":    r.Title,
			"category": r.Category,
			"tags":     r.Tags,
			"date":     dateStr,
		}
	}

	return map[string]interface{}{
		"total":    total,
		"showing":  len(memories),
		"memories": memories,
	}, nil
}

// Helper functions
func getStringFromMap(m map[string]interface{}, key string) (string, bool) {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str, true
		}
	}
	return "", false
}

func getStringSliceFromMap(m map[string]interface{}, key string) ([]string, bool) {
	if val, ok := m[key]; ok {
		if arr, ok := val.([]interface{}); ok {
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

func getCurrentDir() string {
	dir, _ := os.Getwd()
	return dir
}
