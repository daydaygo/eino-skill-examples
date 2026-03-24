package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
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

	prosAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "pros_analyst",
		Description: "优势分析师",
		Instruction: "你负责分析话题的优势。用2-3句话列出主要优势。",
		Model:       model,
	})
	if err != nil {
		panic(err)
	}

	consAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "cons_analyst",
		Description: "劣势分析师",
		Instruction: "你负责分析话题的劣势。用2-3句话列出主要劣势。",
		Model:       model,
	})
	if err != nil {
		panic(err)
	}

	parallelAgent, err := adk.NewParallelAgent(ctx, &adk.ParallelAgentConfig{
		Name:        "parallel_analysis",
		Description: "并行分析：同时分析优势与劣势",
		SubAgents:   []adk.Agent{prosAgent, consAgent},
	})
	if err != nil {
		panic(err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           parallelAgent,
		EnableStreaming: true,
	})

	fmt.Println("=== ParallelAgent 示例 ===")
	fmt.Println("并行执行多个 Agent，结果合并")
	fmt.Println()

	iter := runner.Query(ctx, "分析远程工作的利弊")

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			fmt.Printf("Error: %v\n", event.Err)
			continue
		}

		if event.Action != nil && event.Action.TransferToAgent != nil {
			fmt.Printf("\n[%s]\n", event.Action.TransferToAgent.DestAgentName)
		}

		if msg, _, err := adk.GetMessage(event); err == nil {
			fmt.Print(msg.Content)
		}
	}

	fmt.Println("\n\n=== 并行分析完成 ===")
}
