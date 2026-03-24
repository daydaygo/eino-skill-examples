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
			return fmt.Sprintf("%s 今天天气晴朗，温度 25°C", input.City), nil
		},
	)
	if err != nil {
		panic(err)
	}

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "weather_assistant",
		Description: "天气助手，可以查询城市天气",
		Instruction: "你是一个天气助手。使用 get_weather 工具查询天气信息。",
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

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	fmt.Println("=== ChatModelAgent with Tools ===")
	fmt.Println("示例: 北京今天天气怎么样？")
	fmt.Println()

	iter := runner.Query(ctx, "北京今天天气怎么样？")

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			fmt.Printf("Error: %v\n", event.Err)
			continue
		}

		if event.Action != nil {
			if event.Action.TransferToAgent != nil {
				fmt.Printf("[Transfer] -> %s\n", event.Action.TransferToAgent.DestAgentName)
			}
			if event.Action.Interrupted != nil {
				fmt.Printf("[Interrupt] Checkpoint\n")
			}
		}

		if msg, _, err := adk.GetMessage(event); err == nil {
			fmt.Print(msg.Content)
		}
	}

	fmt.Println("\n=== End ===")
}
