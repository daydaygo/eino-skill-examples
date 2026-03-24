package main

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type JSONFixMiddleware struct {
	baseTool      tool.InvokableTool
	maxAttempts   int
	logFixes      bool
	preProcessors []JSONPreProcessor
}

type JSONPreProcessor func(jsonStr string) string

type JSONFixConfig struct {
	BaseTool      tool.InvokableTool
	MaxAttempts   int
	LogFixes      bool
	PreProcessors []JSONPreProcessor
}

func NewJSONFixMiddleware(config *JSONFixConfig) *JSONFixMiddleware {
	m := &JSONFixMiddleware{
		baseTool:      config.BaseTool,
		maxAttempts:   config.MaxAttempts,
		logFixes:      config.LogFixes,
		preProcessors: config.PreProcessors,
	}

	if m.maxAttempts <= 0 {
		m.maxAttempts = 3
	}

	return m
}

func (m *JSONFixMiddleware) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return m.baseTool.Info(ctx)
}

func (m *JSONFixMiddleware) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	fixedJSON := argumentsInJSON

	for _, preprocessor := range m.preProcessors {
		fixedJSON = preprocessor(fixedJSON)
	}

	fixedJSON = m.fixCommonIssues(fixedJSON)

	if fixedJSON != argumentsInJSON && m.logFixes {
		fmt.Printf("[JSONFix] Original: %s\n", argumentsInJSON)
		fmt.Printf("[JSONFix] Fixed:    %s\n", fixedJSON)
	}

	return m.baseTool.InvokableRun(ctx, fixedJSON, opts...)
}

func (m *JSONFixMiddleware) fixCommonIssues(jsonStr string) string {
	result := jsonStr

	result = strings.TrimSpace(result)

	result = strings.TrimPrefix(result, "```json")
	result = strings.TrimPrefix(result, "```")
	result = strings.TrimSuffix(result, "```")
	result = strings.TrimSpace(result)

	result = fixUnquotedKeys(result)

	result = fixSingleQuotes(result)

	result = fixTrailingCommas(result)

	result = fixMissingQuotes(result)

	return result
}

func fixUnquotedKeys(jsonStr string) string {
	re := regexp.MustCompile(`(\{|,)\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*:`)
	return re.ReplaceAllStringFunc(jsonStr, func(match string) string {
		if strings.Contains(match, `"`) {
			return match
		}
		parts := regexp.MustCompile(`([a-zA-Z_][a-zA-Z0-9_]*)`).FindStringSubmatch(match)
		if len(parts) > 1 {
			return strings.Replace(match, parts[1], `"`+parts[1]+`"`, 1)
		}
		return match
	})
}

func fixSingleQuotes(jsonStr string) string {
	if !strings.Contains(jsonStr, "'") {
		return jsonStr
	}

	var buf strings.Builder
	inString := false
	escape := false

	for _, r := range jsonStr {
		if escape {
			buf.WriteRune(r)
			escape = false
			continue
		}

		if r == '\\' {
			buf.WriteRune(r)
			escape = true
			continue
		}

		if r == '"' {
			inString = !inString
			buf.WriteRune(r)
			continue
		}

		if r == '\'' && !inString {
			buf.WriteRune('"')
			continue
		}

		buf.WriteRune(r)
	}

	return buf.String()
}

func fixTrailingCommas(jsonStr string) string {
	re := regexp.MustCompile(`,\s*([}\]])`)
	return re.ReplaceAllString(jsonStr, "$1")
}

func fixMissingQuotes(jsonStr string) string {
	re := regexp.MustCompile(`:\s*([a-zA-Z_][a-zA-Z0-9_]*)(\s*[,}\]])`)
	return re.ReplaceAllString(jsonStr, `: "$1"$2`)
}

type MockJSONTool struct{}

func NewMockJSONTool() *MockJSONTool {
	return &MockJSONTool{}
}

func (t *MockJSONTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "mock_tool",
		Desc: "A tool that expects valid JSON",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"name":  {Type: schema.String, Desc: "Name", Required: true},
			"value": {Type: schema.Integer, Desc: "Value", Required: true},
		}),
	}, nil
}

func (t *MockJSONTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var input struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}

	return fmt.Sprintf("Successfully parsed: name=%s, value=%d", input.Name, input.Value), nil
}

func main() {
	fmt.Println("=== JSON Fix Middleware Demo ===")
	fmt.Println("This middleware fixes common JSON formatting issues from LLM output")
	fmt.Println()

	ctx := context.Background()

	fixMiddleware := NewJSONFixMiddleware(&JSONFixConfig{
		BaseTool:    NewMockJSONTool(),
		MaxAttempts: 3,
		LogFixes:    true,
		PreProcessors: []JSONPreProcessor{
			func(s string) string {
				return strings.TrimSpace(s)
			},
		},
	})

	testCases := []struct {
		name string
		json string
	}{
		{
			name: "Valid JSON",
			json: `{"name": "test", "value": 42}`,
		},
		{
			name: "Unquoted keys",
			json: `{name: "test", value: 42}`,
		},
		{
			name: "Single quotes",
			json: `{'name': 'test', 'value': 42}`,
		},
		{
			name: "Trailing comma",
			json: `{"name": "test", "value": 42,}`,
		},
		{
			name: "Markdown code block",
			json: "```json\n{\"name\": \"test\", \"value\": 42}\n```",
		},
		{
			name: "Multiple issues",
			json: `{name: 'test', value: 42,}`,
		},
	}

	for _, tc := range testCases {
		fmt.Printf("\n--- Test: %s ---\n", tc.name)
		fmt.Printf("Input:    %s\n", tc.json)

		result, err := fixMiddleware.InvokableRun(ctx, tc.json)

		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Printf("Result:   %s\n", result)
		}
	}

	fmt.Println("\n\n=== JSON Fix Details ===")

	examples := []struct {
		description string
		input       string
		transform   func(string) string
	}{
		{"Remove markdown code block", "```json\n{\"a\":1}\n```", func(s string) string {
			s = strings.TrimPrefix(s, "```json\n")
			s = strings.TrimSuffix(s, "\n```")
			return s
		}},
		{"Fix single quotes", `{'key': 'value'}`, fixSingleQuotes},
		{"Fix trailing comma", `{"a": 1,}`, fixTrailingCommas},
		{"Fix unquoted keys", `{key: "value"}`, fixUnquotedKeys},
	}

	for _, ex := range examples {
		fmt.Printf("\n%s:\n", ex.description)
		fmt.Printf("  Before: %s\n", ex.input)
		fmt.Printf("  After:  %s\n", ex.transform(ex.input))
	}
}
