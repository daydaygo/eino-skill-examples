package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

type WeatherInput struct {
	City string `json:"city" jsonschema:"required,description=城市名称"`
}

type CalculatorInput struct {
	Expression string `json:"expression" jsonschema:"required,description=数学表达式"`
}

func main() {
	ctx := context.Background()

	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	model, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: baseURL,
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		Model:   os.Getenv("OPENAI_MODEL"),
	})
	if err != nil {
		panic(err)
	}

	weatherTool, err := utils.InferTool(
		"get_weather",
		"获取指定城市的天气信息",
		func(ctx context.Context, input *WeatherInput) (string, error) {
			return fmt.Sprintf("%s 今天天气晴朗，温度 25°C，适合户外活动", input.City), nil
		},
	)
	if err != nil {
		panic(err)
	}

	calculatorTool, err := utils.InferTool(
		"calculator",
		"计算数学表达式",
		func(ctx context.Context, input *CalculatorInput) (string, error) {
			return fmt.Sprintf("计算结果: %s = 42", input.Expression), nil
		},
	)
	if err != nil {
		panic(err)
	}

	toolsNode, err := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
		Tools: []tool.BaseTool{weatherTool, calculatorTool},
	})
	if err != nil {
		panic(err)
	}

	graph := compose.NewGraph[[]*schema.Message, *schema.Message]()

	err = graph.AddChatModelNode("chat_model", model)
	if err != nil {
		panic(err)
	}

	err = graph.AddToolsNode("tools", toolsNode)
	if err != nil {
		panic(err)
	}

	err = graph.AddLambdaNode("should_continue", compose.InvokableLambda(
		func(ctx context.Context, msg *schema.Message) (bool, error) {
			return len(msg.ToolCalls) > 0, nil
		},
	))
	if err != nil {
		panic(err)
	}

	err = graph.AddEdge(compose.START, "chat_model")
	if err != nil {
		panic(err)
	}

	err = graph.AddEdge("chat_model", "should_continue")
	if err != nil {
		panic(err)
	}

	branch := compose.NewGraphBranch(
		func(ctx context.Context, continueCall bool) (string, error) {
			if continueCall {
				return "tools", nil
			}
			return compose.END, nil
		},
		map[string]bool{"tools": true, compose.END: true},
	)
	err = graph.AddBranch("should_continue", branch)
	if err != nil {
		panic(err)
	}

	err = graph.AddEdge("tools", "chat_model")
	if err != nil {
		panic(err)
	}

	compiled, err := graph.Compile(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Println("=== Tool Call Agent Graph 示例 ===")
	fmt.Println("Graph 结构: START -> chat_model -> should_continue -> tools -> chat_model (循环)")
	fmt.Println()

	result, err := compiled.Invoke(ctx, []*schema.Message{
		schema.SystemMessage("你是一个助手，可以使用工具来回答问题。"),
		schema.UserMessage("北京今天天气怎么样？"),
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("最终回答: %s\n", result.Content)
	fmt.Println("=== 完成 ===")
}
