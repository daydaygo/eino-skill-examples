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
		Question string
		Mode     string
	}

	type Output struct {
		Answer string
	}

	workflow := compose.NewWorkflow[Input, Output]()

	workflow.AddLambdaNode("router", compose.InvokableLambda(
		func(ctx context.Context, input Input) (string, error) {
			return input.Mode, nil
		},
	), compose.WithOutputKey("route"))

	workflow.AddChatModelNode("technical_model", model,
		compose.WithInputKey("Question"),
		compose.WithOutputKey("Answer"))

	workflow.AddChatModelNode("casual_model", model,
		compose.WithInputKey("Question"),
		compose.WithOutputKey("Answer"))

	branch := compose.NewGraphBranch(
		func(ctx context.Context, route string) (string, error) {
			if route == "technical" {
				return "technical_model", nil
			}
			return "casual_model", nil
		},
		map[string]bool{"technical_model": true, "casual_model": true},
	)
	workflow.AddBranch("router", branch)

	workflow.End().AddDependency("technical_model")
	workflow.End().AddDependency("casual_model")

	compiled, err := workflow.Compile(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Println("=== 控制流分支 Workflow 示例 ===")
	fmt.Println("根据输入字段选择不同路径:")
	fmt.Println("  Mode=technical -> technical_model")
	fmt.Println("  Mode=casual -> casual_model")
	fmt.Println()

	result1, err := compiled.Invoke(ctx, Input{
		Question: "什么是并发？",
		Mode:     "technical",
	})
	if err != nil {
		panic(err)
	}
	fmt.Printf("技术模式回答: %s\n", result1.Answer)

	result2, err := compiled.Invoke(ctx, Input{
		Question: "今天天气怎么样？",
		Mode:     "casual",
	})
	if err != nil {
		panic(err)
	}
	fmt.Printf("休闲模式回答: %s\n", result2.Answer)

	fmt.Println("=== 完成 ===")
}
