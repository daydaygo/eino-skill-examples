package main

import (
	"context"
	"fmt"
	"os"
	"time"

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

	asyncNode := compose.InvokableLambda(
		func(ctx context.Context, msgs []*schema.Message) ([]*schema.Message, error) {
			start := time.Now()
			fmt.Printf("[%s] 异步节点开始处理...\n", start.Format("15:04:05"))

			time.Sleep(2 * time.Second)

			elapsed := time.Since(start)
			fmt.Printf("[%s] 异步节点处理完成，耗时: %v\n", time.Now().Format("15:04:05"), elapsed)

			return msgs, nil
		},
	)

	err = graph.AddLambdaNode("async_preprocess", asyncNode)
	if err != nil {
		panic(err)
	}

	err = graph.AddChatModelNode("model", model)
	if err != nil {
		panic(err)
	}

	err = graph.AddLambdaNode("async_postprocess", compose.InvokableLambda(
		func(ctx context.Context, msg *schema.Message) (*schema.Message, error) {
			start := time.Now()
			fmt.Printf("[%s] 后处理节点开始...\n", start.Format("15:04:05"))

			time.Sleep(1 * time.Second)

			fmt.Printf("[%s] 后处理节点完成\n", time.Now().Format("15:04:05"))
			return msg, nil
		},
	))
	if err != nil {
		panic(err)
	}

	err = graph.AddEdge(compose.START, "async_preprocess")
	if err != nil {
		panic(err)
	}

	err = graph.AddEdge("async_preprocess", "model")
	if err != nil {
		panic(err)
	}

	err = graph.AddEdge("model", "async_postprocess")
	if err != nil {
		panic(err)
	}

	err = graph.AddEdge("async_postprocess", compose.END)
	if err != nil {
		panic(err)
	}

	compiled, err := graph.Compile(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Println("=== 异步节点示例 ===")
	fmt.Println("Graph 结构: START -> async_preprocess -> model -> async_postprocess -> END")
	fmt.Println("展示 Lambda 节点处理异步逻辑")
	fmt.Println()

	start := time.Now()
	result, err := compiled.Invoke(ctx, []*schema.Message{
		schema.SystemMessage("你是一个友好的助手。"),
		schema.UserMessage("你好！"),
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("\n最终回答: %s\n", result.Content)
	fmt.Printf("总耗时: %v\n", time.Since(start))
	fmt.Println("=== 完成 ===")
}
