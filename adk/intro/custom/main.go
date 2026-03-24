package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
)

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

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "custom_assistant",
		Description: "自定义 Agent 示例",
		Instruction: "你是一个友好的助手。简洁回答问题。展示你的个性特点。",
		Model:       model,
	})
	if err != nil {
		panic(err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	fmt.Println("=== Custom Agent 示例 ===")
	fmt.Println("使用 ChatModelAgent 配置自定义行为")
	fmt.Println()

	iter := runner.Query(ctx, "介绍一下你自己")

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			fmt.Printf("Error: %v\n", event.Err)
			continue
		}

		if msg, _, err := adk.GetMessage(event); err == nil {
			fmt.Print(msg.Content)
		}
	}

	fmt.Println("\n=== End ===")
}
