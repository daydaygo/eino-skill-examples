package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
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

	chatTemplate := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是一个友好的助手，请简洁回答问题。"),
		schema.MessagesPlaceholder("query", true),
	)

	chain, err := compose.NewChain[map[string]any, *schema.Message]().
		AppendChatTemplate(chatTemplate).
		AppendChatModel(model).
		Compile(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Println("=== Chain 基础示例 ===")
	fmt.Println("展示 Chain 顺序编排: Prompt -> ChatModel")

	input := map[string]any{
		"query": []*schema.Message{schema.UserMessage("什么是 Eino 框架？")},
	}

	stream, err := chain.Stream(ctx, input)
	if err != nil {
		panic(err)
	}
	defer stream.Close()

	fmt.Print("回答: ")
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
		fmt.Print(chunk.Content)
	}
	fmt.Println("\n=== 完成 ===")
}
