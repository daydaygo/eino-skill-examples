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

	type Input struct {
		Question string
	}

	type Output struct {
		Answer      string
		ModelUsed   string
		Temperature string
	}

	workflow := compose.NewWorkflow[Input, Output]()

	workflow.AddLambdaNode("inject_static", compose.InvokableLambda(
		func(ctx context.Context, input Input) (string, error) {
			return input.Question, nil
		},
	), compose.WithOutputKey("question"))

	workflow.AddChatModelNode("model", model,
		compose.WithInputKey("question"),
		compose.WithOutputKey("answer"))

	workflow.AddLambdaNode("finalize", compose.InvokableLambda(
		func(ctx context.Context, msg *schema.Message) (Output, error) {
			return Output{
				Answer:      msg.Content,
				ModelUsed:   "gpt-4",
				Temperature: "0.7",
			}, nil
		},
	))

	workflow.End().AddDependency("finalize")

	compiled, err := workflow.Compile(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Println("=== 静态值注入 Workflow 示例 ===")
	fmt.Println("展示在工作流中注入静态配置值")
	fmt.Println()

	result, err := compiled.Invoke(ctx, Input{
		Question: "什么是 Go 语言的并发模型？",
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("回答: %s\n", result.Answer)
	fmt.Printf("使用的模型: %s\n", result.ModelUsed)
	fmt.Printf("温度参数: %s\n", result.Temperature)
	fmt.Println("=== 完成 ===")
}
