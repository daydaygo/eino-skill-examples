package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"

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

	fmt.Println("=== BatchNode 批处理示例 ===")
	fmt.Println("展示并发处理多个请求的能力")
	fmt.Println()

	queries := []string{
		"什么是 Go 语言？",
		"什么是并发编程？",
		"什么是微服务？",
	}

	var processedCount int32
	var mu sync.Mutex
	results := make([]string, len(queries))

	var wg sync.WaitGroup
	for i, query := range queries {
		wg.Add(1)
		go func(idx int, q string) {
			defer wg.Done()

			graph := compose.NewGraph[[]*schema.Message, *schema.Message]()

			err := graph.AddChatModelNode("model", model)
			if err != nil {
				fmt.Printf("Error adding node: %v\n", err)
				return
			}

			err = graph.AddEdge(compose.START, "model")
			if err != nil {
				fmt.Printf("Error adding edge: %v\n", err)
				return
			}

			err = graph.AddEdge("model", compose.END)
			if err != nil {
				fmt.Printf("Error adding edge: %v\n", err)
				return
			}

			compiled, err := graph.Compile(ctx)
			if err != nil {
				fmt.Printf("Error compiling: %v\n", err)
				return
			}

			result, err := compiled.Invoke(ctx, []*schema.Message{
				schema.SystemMessage("请用一句话简洁回答问题。"),
				schema.UserMessage(q),
			})
			if err != nil {
				fmt.Printf("Error for query %d: %v\n", idx, err)
				return
			}

			mu.Lock()
			results[idx] = result.Content
			mu.Unlock()

			atomic.AddInt32(&processedCount, 1)
			fmt.Printf("[%d/%d] 完成: %s\n", processedCount, len(queries), q)
		}(i, query)
	}
	wg.Wait()

	fmt.Println("\n=== 所有结果 ===")
	for i, query := range queries {
		fmt.Printf("\n问题 %d: %s\n", i+1, query)
		fmt.Printf("回答: %s\n", results[i])
	}

	fmt.Println("\n=== 批处理完成 ===")
}
