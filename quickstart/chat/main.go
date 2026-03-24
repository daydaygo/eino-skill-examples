package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
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

	fmt.Println("=== 1. Generate 非流式调用 ===")
	generateDemo(ctx, model)

	fmt.Println("\n=== 2. Stream 流式输出 ===")
	streamDemo(ctx, model)

	fmt.Println("\n=== 3. 多轮对话示例 ===")
	chatDemo(ctx, model)
}

func generateDemo(ctx context.Context, model *openai.ChatModel) {
	messages := []*schema.Message{
		schema.SystemMessage("你是一个有帮助的助手，回答要简洁。"),
		schema.UserMessage("用一句话介绍 Go 语言。"),
	}

	msg, err := model.Generate(ctx, messages)
	if err != nil {
		panic(err)
	}

	fmt.Printf("回复: %s\n", msg.Content)
}

func streamDemo(ctx context.Context, model *openai.ChatModel) {
	messages := []*schema.Message{
		schema.SystemMessage("你是一个有帮助的助手。"),
		schema.UserMessage("写一首关于编程的四行小诗。"),
	}

	sr, err := model.Stream(ctx, messages)
	if err != nil {
		panic(err)
	}
	defer sr.Close()

	fmt.Print("流式回复: ")
	for {
		chunk, err := sr.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			panic(err)
		}
		fmt.Print(chunk.Content)
	}
	fmt.Println()
}

func chatDemo(ctx context.Context, model *openai.ChatModel) {
	history := []*schema.Message{
		schema.SystemMessage("你是一个编程助手，回答简洁专业。"),
	}

	questions := []string{
		"什么是并发？",
		"Go 如何实现并发？",
	}

	for i, q := range questions {
		fmt.Printf("\n[用户 %d]: %s\n", i+1, q)
		history = append(history, schema.UserMessage(q))

		msg, err := model.Generate(ctx, history)
		if err != nil {
			panic(err)
		}

		fmt.Printf("[助手]: %s\n", msg.Content)
		history = append(history, msg)
	}
}
