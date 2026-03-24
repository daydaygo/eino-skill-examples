package main

import (
	"context"
	"fmt"
	"io"
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
		Query string
	}

	type Output struct {
		Response string
	}

	workflow := compose.NewWorkflow[Input, Output]()

	workflow.AddLambdaNode("format_query", compose.InvokableLambda(
		func(ctx context.Context, input Input) (*schema.Message, error) {
			return schema.UserMessage(input.Query), nil
		},
	), compose.WithOutputKey("formatted_query"))

	workflow.AddChatModelNode("model", model,
		compose.WithInputKey("formatted_query"),
		compose.WithOutputKey("model_output"))

	workflow.AddLambdaNode("extract_content", compose.InvokableLambda(
		func(ctx context.Context, msg *schema.Message) (Output, error) {
			return Output{Response: msg.Content}, nil
		},
	))

	workflow.End().AddDependency("extract_content")

	compiled, err := workflow.Compile(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Println("=== 流式字段映射 Workflow 示例 ===")
	fmt.Println("展示流式处理中的字段映射")
	fmt.Println()

	stream, err := compiled.Stream(ctx, Input{
		Query: "用一句话介绍 Eino 框架",
	})
	if err != nil {
		panic(err)
	}
	defer stream.Close()

	fmt.Print("流式输出: ")
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
		fmt.Print(chunk.Response)
	}
	fmt.Println("\n=== 完成 ===")
}
