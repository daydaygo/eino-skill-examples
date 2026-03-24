package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type MCPToolResult struct {
	Content []MCPContent `json:"content"`
	IsError bool         `json:"isError"`
}

type MCPContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type CallResultHandler func(ctx context.Context, result *MCPToolResult) (string, error)

type MCPTool struct {
	info    *schema.ToolInfo
	handler CallResultHandler
}

func NewMCPTool(name, description string, params map[string]*schema.ParameterInfo, handler CallResultHandler) *MCPTool {
	return &MCPTool{
		info: &schema.ToolInfo{
			Name:        name,
			Desc:        description,
			ParamsOneOf: schema.NewParamsOneOfByParams(params),
		},
		handler: handler,
	}
}

func (t *MCPTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return t.info, nil
}

func (t *MCPTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	fmt.Printf("[MCP Tool] Received call with arguments: %s\n", argumentsInJSON)

	mcpResult := &MCPToolResult{
		Content: []MCPContent{
			{Type: "text", Text: fmt.Sprintf("Raw MCP response for input: %s", argumentsInJSON)},
		},
		IsError: false,
	}

	if t.handler != nil {
		return t.handler(ctx, mcpResult)
	}

	return t.defaultHandler(ctx, mcpResult)
}

func (t *MCPTool) defaultHandler(ctx context.Context, result *MCPToolResult) (string, error) {
	if result.IsError {
		return "", fmt.Errorf("MCP tool returned error")
	}

	var textContent string
	for _, content := range result.Content {
		if content.Type == "text" {
			textContent += content.Text
		}
	}
	return textContent, nil
}

func summarizeHandler(ctx context.Context, result *MCPToolResult) (string, error) {
	fmt.Println("[Handler] Using summarize handler")

	if result.IsError {
		return fmt.Sprintf("Error: %v", result.Content), nil
	}

	var texts []string
	for _, content := range result.Content {
		if content.Type == "text" && len(content.Text) > 0 {
			if len(content.Text) > 100 {
				texts = append(texts, content.Text[:100]+"...")
			} else {
				texts = append(texts, content.Text)
			}
		}
	}

	if len(texts) == 0 {
		return "No content available", nil
	}

	return fmt.Sprintf("Summary: %s", texts[0]), nil
}

func structuredHandler(ctx context.Context, result *MCPToolResult) (string, error) {
	fmt.Println("[Handler] Using structured output handler")

	output := map[string]any{
		"success": !result.IsError,
		"content": result.Content,
		"count":   len(result.Content),
	}

	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

func retryHandler(ctx context.Context, result *MCPToolResult) (string, error) {
	fmt.Println("[Handler] Using retry handler")

	if result.IsError {
		return "", fmt.Errorf("tool call failed, retry recommended")
	}

	var textContent string
	for _, content := range result.Content {
		if content.Type == "text" {
			textContent += content.Text
		}
	}
	return textContent, nil
}

func getEnvOrDefault(key, defaultVal string) string {
	return defaultVal
}

func main() {
	fmt.Println("=== MCP Tool Call Result Handler Demo ===")
	fmt.Println("This demo shows how to customize MCP tool result handling")
	fmt.Println()

	ctx := context.Background()

	searchTool := NewMCPTool(
		"web_search",
		"搜索网页内容",
		map[string]*schema.ParameterInfo{
			"query": {Type: schema.String, Desc: "搜索关键词", Required: true},
		},
		summarizeHandler,
	)

	databaseTool := NewMCPTool(
		"query_database",
		"查询数据库",
		map[string]*schema.ParameterInfo{
			"sql": {Type: schema.String, Desc: "SQL查询语句", Required: true},
		},
		structuredHandler,
	)

	apiTool := NewMCPTool(
		"call_api",
		"调用外部API",
		map[string]*schema.ParameterInfo{
			"endpoint": {Type: schema.String, Desc: "API端点", Required: true},
			"method":   {Type: schema.String, Desc: "HTTP方法", Required: true},
		},
		nil,
	)

	tools := []struct {
		name string
		tool *MCPTool
		args string
	}{
		{"web_search", searchTool, `{"query": "golang best practices"}`},
		{"query_database", databaseTool, `{"sql": "SELECT * FROM users LIMIT 10"}`},
		{"call_api", apiTool, `{"endpoint": "/api/users", "method": "GET"}`},
	}

	for _, tc := range tools {
		fmt.Printf("\n--- Testing %s ---\n", tc.name)

		info, _ := tc.tool.Info(ctx)
		fmt.Printf("Tool: %s\n", info.Name)
		fmt.Printf("Description: %s\n", info.Desc)

		result, err := tc.tool.InvokableRun(ctx, tc.args)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Printf("Result: %s\n", result)
	}

	fmt.Println("\n=== Custom Handler Examples ===")

	fmt.Println("\n1. Error handling with custom error message:")
	errorResult := &MCPToolResult{
		Content: []MCPContent{{Type: "text", Text: "Connection timeout"}},
		IsError: true,
	}
	result, err := summarizeHandler(ctx, errorResult)
	fmt.Printf("Result: %s, Error: %v\n", result, err)

	fmt.Println("\n2. Large content truncation:")
	largeResult := &MCPToolResult{
		Content: []MCPContent{{
			Type: "text",
			Text: "This is a very long text content that should be truncated when using the summarize handler because it exceeds the 100 character limit that we have set in our handler implementation.",
		}},
		IsError: false,
	}
	result, _ = summarizeHandler(ctx, largeResult)
	fmt.Printf("Result: %s\n", result)
}
