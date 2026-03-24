package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
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

	graph := compose.NewGraph[[]*schema.Message, *schema.Message]()

	err = graph.AddChatModelNode("model", model)
	if err != nil {
		panic(err)
	}

	err = graph.AddEdge(compose.START, "model")
	if err != nil {
		panic(err)
	}

	err = graph.AddEdge("model", compose.END)
	if err != nil {
		panic(err)
	}

	compiled, err := graph.Compile(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Println("=== 简单 Graph 示例 ===")
	fmt.Println("Graph 结构: START -> model -> END")

	result, err := compiled.Invoke(ctx, []*schema.Message{
		schema.SystemMessage("你是一个友好的助手。"),
		schema.UserMessage("用一句话介绍 Go 语言。"),
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("回答: %s\n", result.Content)
	fmt.Println("=== 完成 ===")
}
