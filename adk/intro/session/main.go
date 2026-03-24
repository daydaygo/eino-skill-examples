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
		Name:        "session_demo",
		Description: "Session 管理演示",
		Instruction: `你是助手。记住用户的偏好和信息。
在对话中引用用户之前提到的信息，展示记忆能力。`,
		Model: model,
	})
	if err != nil {
		panic(err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	fmt.Println("=== Session 管理示例 ===")
	fmt.Println("演示多轮对话中的 Session 持久化")
	fmt.Println()

	sessionValues := map[string]any{
		"user_id":   "user_12345",
		"user_name": "张三",
		"topic":     "人工智能",
	}

	fmt.Printf("Session 数据: %+v\n", sessionValues)
	fmt.Println()

	fmt.Println("--- 第一轮对话 ---")
	fmt.Println("User: 你好，我是张三")

	iter := runner.Query(ctx, "你好，我是张三",
		adk.WithSessionValues(sessionValues),
	)

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
				fmt.Print("Agent: ")
				fmt.Print(msg.Content)
			}
		}
	}
	fmt.Println()

	fmt.Println("\n--- 第二轮对话 ---")
	fmt.Println("User: 记得我的名字吗？")

	iter2 := runner.Query(ctx, "记得我的名字吗？")

	for {
		event, ok := iter2.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			fmt.Printf("Error: %v\n", event.Err)
			continue
		}

		if msg, _, err := adk.GetMessage(event); err == nil {
			if msg.Role == schema.Assistant {
				fmt.Print("Agent: ")
				fmt.Print(msg.Content)
			}
		}
	}
	fmt.Println()

	fmt.Println("\n=== Session 演示完成 ===")
}
