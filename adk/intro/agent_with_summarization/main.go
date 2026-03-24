package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
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
		Name:        "assistant_with_memory",
		Description: "带记忆能力的助手",
		Instruction: "你是一个友好的助手。保持回答简洁，记住之前的对话内容。",
		Model:       model,
	})
	if err != nil {
		panic(err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	fmt.Println("=== Agent with Memory 示例 ===")
	fmt.Println("演示多轮对话中的记忆能力")
	fmt.Println()

	fmt.Println("开始对话（输入 'quit' 退出）:")
	fmt.Println()

	conversationCount := 0
	for {
		fmt.Print("You: ")
		var input string
		fmt.Scanln(&input)

		if input == "quit" {
			break
		}

		conversationCount++
		fmt.Printf("[对话轮次: %d]\n", conversationCount)
		fmt.Print("Agent: ")

		iter := runner.Query(ctx, input)

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
				if msg.Role == schema.Assistant {
					fmt.Print(msg.Content)
				}
			}
		}
		fmt.Println()
		fmt.Println()
	}

	fmt.Println("=== 对话结束 ===")
	fmt.Println("Agent 会自动管理对话历史，保持上下文连贯")
}
