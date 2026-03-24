package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/compose"
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

	type Input struct {
		Query string
	}

	type Output struct {
		Result string
	}

	workflow := compose.NewWorkflow[Input, Output]()

	node := workflow.AddChatModelNode("model", model,
		compose.WithInputKey("Query"),
		compose.WithOutputKey("Result"))

	workflow.End().AddDependency("model")
	_ = node

	compiled, err := workflow.Compile(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Println("=== 简单 Workflow 示例 ===")
	fmt.Println("Workflow 结构: START -> model -> END")
	fmt.Println("输入字段: Query -> 输出字段: Result")

	result, err := compiled.Invoke(ctx, Input{
		Query: "用一句话介绍 Go 语言",
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("结果: %s\n", result.Result)
	fmt.Println("=== 完成 ===")
}
