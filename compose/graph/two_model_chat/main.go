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

	modelA, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: baseURL,
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		Model:   os.Getenv("OPENAI_MODEL"),
	})
	if err != nil {
		panic(err)
	}

	modelB, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: baseURL,
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		Model:   os.Getenv("OPENAI_MODEL"),
	})
	if err != nil {
		panic(err)
	}

	graph := compose.NewGraph[[]*schema.Message, *schema.Message]()

	err = graph.AddChatModelNode("model_a", modelA)
	if err != nil {
		panic(err)
	}

	err = graph.AddChatModelNode("model_b", modelB)
	if err != nil {
		panic(err)
	}

	err = graph.AddLambdaNode("print_exchange", compose.InvokableLambda(
		func(ctx context.Context, msg *schema.Message) ([]*schema.Message, error) {
			fmt.Printf("Model A: %s\n", msg.Content)
			return []*schema.Message{
				schema.SystemMessage("你是一个辩论家，请对对方观点进行反驳（一句话）。"),
				schema.AssistantMessage(msg.Content, nil),
			}, nil
		},
	))
	if err != nil {
		panic(err)
	}

	err = graph.AddEdge(compose.START, "model_a")
	if err != nil {
		panic(err)
	}

	err = graph.AddEdge("model_a", "print_exchange")
	if err != nil {
		panic(err)
	}

	err = graph.AddEdge("print_exchange", "model_b")
	if err != nil {
		panic(err)
	}

	err = graph.AddEdge("model_b", compose.END)
	if err != nil {
		panic(err)
	}

	compiled, err := graph.Compile(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Println("=== 双模型对话示例 ===")
	fmt.Println("Graph 结构: START -> model_a -> print_exchange -> model_b -> END")
	fmt.Println()

	stream, err := compiled.Stream(ctx, []*schema.Message{
		schema.SystemMessage("你是一个哲学家，请提出一个关于人工智能的观点（一句话）。"),
		schema.UserMessage("请开始"),
	})
	if err != nil {
		panic(err)
	}
	defer stream.Close()

	fmt.Print("Model B 响应: ")
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
