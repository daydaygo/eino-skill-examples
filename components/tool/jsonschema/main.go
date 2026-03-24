package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type CalculatorInput struct {
	Expression string `json:"expression" jsonschema:"required" jsonschema_description:"数学表达式，如 1+2*3"`
}

func main() {
	fmt.Println("=== JSON Schema Tool Demo ===")
	fmt.Println("This demo shows how to define tools using JSON Schema")
	fmt.Println()

	calculatorInfo := &schema.ToolInfo{
		Name: "calculator",
		Desc: "执行数学计算，支持加减乘除和括号",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"expression": {
				Type:     schema.String,
				Desc:     "数学表达式，如 1+2*3",
				Required: true,
			},
		}),
	}

	searchInfo := &schema.ToolInfo{
		Name: "web_search",
		Desc: "搜索网页内容",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query": {
				Type:     schema.String,
				Desc:     "搜索关键词",
				Required: true,
			},
			"limit": {
				Type:     schema.Integer,
				Desc:     "返回结果数量限制，默认10",
				Required: false,
			},
		}),
	}

	userInfoParams := schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
		"name": {
			Type:     schema.String,
			Desc:     "用户姓名",
			Required: true,
		},
		"age": {
			Type:     schema.Integer,
			Desc:     "用户年龄，范围0-150",
			Required: true,
		},
		"email": {
			Type:     schema.String,
			Desc:     "用户邮箱地址",
			Required: true,
		},
		"role": {
			Type:     schema.String,
			Desc:     "用户角色",
			Enum:     []string{"admin", "user", "guest"},
			Required: true,
		},
		"tags": {
			Type:     schema.Array,
			Desc:     "用户标签列表",
			Required: false,
			ElemInfo: &schema.ParameterInfo{
				Type: schema.String,
			},
		},
	})
	userInfo := &schema.ToolInfo{
		Name:        "create_user",
		Desc:        "创建新用户",
		ParamsOneOf: userInfoParams,
	}

	tools := []*schema.ToolInfo{calculatorInfo, searchInfo, userInfo}

	for i, info := range tools {
		fmt.Printf("%d. Tool: %s\n", i+1, info.Name)
		fmt.Printf("   Description: %s\n", info.Desc)
		fmt.Println("   Parameters (JSON Schema):")
		paramsJSON, _ := json.MarshalIndent(info.ParamsOneOf, "     ", "  ")
		fmt.Printf("     %s\n", string(paramsJSON))
		fmt.Println()
	}

	fmt.Println("=== Example Tool Calls ===")
	exampleCalls := []struct {
		toolName string
		args     string
	}{
		{"calculator", `{"expression": "2+3*4"}`},
		{"web_search", `{"query": "golang tutorial", "limit": 5}`},
		{"create_user", `{"name": "Alice", "age": 30, "email": "alice@example.com", "role": "admin", "tags": ["developer", "golang"]}`},
	}

	for _, call := range exampleCalls {
		fmt.Printf("Tool: %s\n", call.toolName)
		fmt.Printf("Arguments: %s\n\n", call.args)
	}

	fmt.Println("=== Creating and Using a Tool ===")
	ctx := context.Background()
	calculatorTool := NewCalculatorTool()

	info, err := calculatorTool.Info(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Tool info: %s - %s\n", info.Name, info.Desc)

	result, err := calculatorTool.InvokableRun(ctx, `{"expression": "10+20"}`)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Result: %s\n", result)
	}
}

var _ tool.InvokableTool = (*CalculatorTool)(nil)

type CalculatorTool struct {
	info *schema.ToolInfo
}

func NewCalculatorTool() *CalculatorTool {
	return &CalculatorTool{
		info: &schema.ToolInfo{
			Name: "calculator",
			Desc: "执行数学计算",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"expression": {
					Type:     schema.String,
					Desc:     "数学表达式",
					Required: true,
				},
			}),
		},
	}
}

func (t *CalculatorTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return t.info, nil
}

func (t *CalculatorTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var input CalculatorInput
	if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil {
		return "", fmt.Errorf("parse arguments: %w", err)
	}

	return fmt.Sprintf("Result of '%s' = [simulated]", input.Expression), nil
}
