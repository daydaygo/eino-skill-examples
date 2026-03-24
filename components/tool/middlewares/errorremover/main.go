package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type ErrorRemoverMiddleware struct {
	baseTool   tool.InvokableTool
	logErrors  bool
	errHandler func(err error) string
}

type ErrorRemoverConfig struct {
	BaseTool   tool.InvokableTool
	LogErrors  bool
	ErrHandler func(err error) string
}

func NewErrorRemoverMiddleware(config *ErrorRemoverConfig) *ErrorRemoverMiddleware {
	m := &ErrorRemoverMiddleware{
		baseTool:   config.BaseTool,
		logErrors:  config.LogErrors,
		errHandler: config.ErrHandler,
	}

	if m.errHandler == nil {
		m.errHandler = defaultErrorHandler
	}

	return m
}

func defaultErrorHandler(err error) string {
	return fmt.Sprintf("Operation could not be completed: %s", err.Error())
}

func (m *ErrorRemoverMiddleware) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return m.baseTool.Info(ctx)
}

func (m *ErrorRemoverMiddleware) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	result, err := m.baseTool.InvokableRun(ctx, argumentsInJSON, opts...)
	if err != nil {
		if m.logErrors {
			fmt.Printf("[ErrorRemover] Tool error: %v\n", err)
		}
		return m.errHandler(err), nil
	}
	return result, nil
}

type MockTool struct {
	name        string
	shouldError bool
}

func NewMockTool(name string, shouldError bool) *MockTool {
	return &MockTool{name: name, shouldError: shouldError}
}

func (t *MockTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: t.name,
		Desc: "A mock tool for testing",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"input": {Type: schema.String, Desc: "Input parameter", Required: true},
		}),
	}, nil
}

func (t *MockTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var input struct {
		Input string `json:"input"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil {
		return "", err
	}

	if t.shouldError {
		return "", fmt.Errorf("simulated error from %s", t.name)
	}

	return fmt.Sprintf("Success from %s: %s", t.name, input.Input), nil
}

func main() {
	fmt.Println("=== Error Remover Middleware Demo ===")
	fmt.Println("This middleware catches tool errors and returns user-friendly messages")
	fmt.Println()

	ctx := context.Background()

	successTool := NewMockTool("success_tool", false)
	errorTool := NewMockTool("error_tool", true)

	customErrorHandler := func(err error) string {
		return fmt.Sprintf("⚠️ 操作失败: %s，请稍后重试", err.Error())
	}

	wrappedErrorTool := NewErrorRemoverMiddleware(&ErrorRemoverConfig{
		BaseTool:   errorTool,
		LogErrors:  true,
		ErrHandler: customErrorHandler,
	})

	wrappedSuccessTool := NewErrorRemoverMiddleware(&ErrorRemoverConfig{
		BaseTool:  successTool,
		LogErrors: true,
	})

	testCases := []struct {
		name string
		tool tool.InvokableTool
		args string
	}{
		{"Success tool (wrapped)", wrappedSuccessTool, `{"input": "hello"}`},
		{"Error tool (wrapped)", wrappedErrorTool, `{"input": "test"}`},
		{"Success tool (unwrapped)", successTool, `{"input": "direct"}`},
		{"Error tool (unwrapped)", errorTool, `{"input": "will fail"}`},
	}

	for _, tc := range testCases {
		fmt.Printf("\n--- Testing: %s ---\n", tc.name)
		fmt.Printf("Arguments: %s\n", tc.args)

		result, err := tc.tool.InvokableRun(ctx, tc.args)

		fmt.Printf("Result: %s\n", result)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Println("Error: nil (caught by middleware)")
		}
	}

	fmt.Println("\n\n=== Batch Tool Execution Demo ===")

	tools := []tool.InvokableTool{
		NewErrorRemoverMiddleware(&ErrorRemoverConfig{
			BaseTool:  NewMockTool("tool_1", false),
			LogErrors: true,
		}),
		NewErrorRemoverMiddleware(&ErrorRemoverConfig{
			BaseTool:  NewMockTool("tool_2", true),
			LogErrors: true,
		}),
		NewErrorRemoverMiddleware(&ErrorRemoverConfig{
			BaseTool:  NewMockTool("tool_3", false),
			LogErrors: true,
		}),
	}

	args := `{"input": "batch test"}`
	results := make([]string, 0)

	for i, t := range tools {
		result, _ := t.InvokableRun(ctx, args)
		results = append(results, result)
		fmt.Printf("Tool %d result: %s\n", i+1, result)
	}

	fmt.Println("\nAll tools completed without interruption (errors were caught)")
}
