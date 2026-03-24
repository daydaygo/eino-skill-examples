package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
)

type WeatherInput struct {
	City string `json:"city" jsonschema:"required" jsonschema_description:"城市名称"`
}

type CalcInput struct {
	Expression string `json:"expression" jsonschema:"required" jsonschema_description:"数学表达式"`
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
		"获取指定城市的天气",
		func(ctx context.Context, input *WeatherInput) (string, error) {
			return fmt.Sprintf("%s 天气：晴朗，25°C", input.City), nil
		},
	)
	if err != nil {
		panic(err)
	}

	calcTool, err := utils.InferTool(
		"calculate",
		"执行数学计算",
		func(ctx context.Context, input *CalcInput) (string, error) {
			return fmt.Sprintf("计算结果: %s = (请使用计算器)", input.Expression), nil
		},
	)
	if err != nil {
		panic(err)
	}

	weatherAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "weather_expert",
		Description: "天气专家，负责处理天气相关的查询",
		Instruction: "你是天气专家。使用 get_weather 工具回答天气问题。",
		Model:       model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{weatherTool},
			},
		},
	})
	if err != nil {
		panic(err)
	}

	mathAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "math_expert",
		Description: "数学专家，负责处理数学计算",
		Instruction: "你是数学专家。使用 calculate 工具进行计算。",
		Model:       model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{calcTool},
			},
		},
	})
	if err != nil {
		panic(err)
	}

	routerAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "router",
		Description: "任务路由器，将任务分发给合适的专家",
		Instruction: `你是任务路由器。根据用户问题类型，将任务转移给合适的专家：
- weather_expert: 处理天气相关问题
- math_expert: 处理数学计算问题
如果问题不在这两个领域，直接回答。`,
		Model: model,
	})
	if err != nil {
		panic(err)
	}

	_, err = adk.SetSubAgents(ctx, routerAgent, []adk.Agent{weatherAgent, mathAgent})
	if err != nil {
		panic(err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           routerAgent,
		EnableStreaming: true,
	})

	fmt.Println("=== Agent Transfer 示例 ===")
	fmt.Println("展示 ChatModelAgent 的 Transfer 能力")
	fmt.Println()

	queries := []string{
		"北京今天天气怎么样？",
		"计算 123 + 456 等于多少？",
	}

	for i, query := range queries {
		fmt.Printf("=== 查询 %d: %s ===\n", i+1, query)
		fmt.Println()

		iter := runner.Query(ctx, query)

		for {
			event, ok := iter.Next()
			if !ok {
				break
			}

			if event.Err != nil {
				fmt.Printf("Error: %v\n", event.Err)
				continue
			}

			if event.Action != nil && event.Action.TransferToAgent != nil {
				fmt.Printf("[Transfer] -> %s\n", event.Action.TransferToAgent.DestAgentName)
			}

			if msg, _, err := adk.GetMessage(event); err == nil {
				content := msg.Content
				if content != "" {
					fmt.Printf("回复: %s\n", content)
				}
			}
		}

		fmt.Println()
	}

	fmt.Println("=== Transfer 演示完成 ===")
}
