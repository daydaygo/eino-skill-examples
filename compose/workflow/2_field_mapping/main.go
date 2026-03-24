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
		UserQuestion string
		Context      string
	}

	type Output struct {
		Answer     string
		SourceUsed string
	}

	workflow := compose.NewWorkflow[Input, Output]()

	workflow.AddLambdaNode("enrich_query", compose.InvokableLambda(
		func(ctx context.Context, input Input) (string, error) {
			return fmt.Sprintf("上下文: %s\n问题: %s", input.Context, input.UserQuestion), nil
		},
	), compose.WithOutputKey("enriched"))

	workflow.AddChatModelNode("model", model,
		compose.WithInputKey("enriched"),
		compose.WithOutputKey("answer"))

	workflow.AddLambdaNode("format_output", compose.InvokableLambda(
		func(ctx context.Context, msg *schema.Message) (Output, error) {
			return Output{
				Answer:     msg.Content,
				SourceUsed: "知识库",
			}, nil
		},
	))

	workflow.End().AddDependency("format_output")

	compiled, err := workflow.Compile(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Println("=== 字段映射 Workflow 示例 ===")
	fmt.Println("展示节点间的字段映射:")
	fmt.Println("  Input.UserQuestion + Context -> enrich_query -> enriched")
	fmt.Println("  enriched -> model -> answer")
	fmt.Println("  answer -> format_output -> Output")
	fmt.Println()

	result, err := compiled.Invoke(ctx, Input{
		UserQuestion: "什么是微服务？",
		Context:      "微服务是一种架构风格",
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("回答: %s\n", result.Answer)
	fmt.Printf("来源: %s\n", result.SourceUsed)
	fmt.Println("=== 完成 ===")
}
