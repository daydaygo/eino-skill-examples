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

	researcherAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "researcher",
		Description: "研究员，负责调研分析",
		Instruction: "你是研究员。对给定主题进行简短分析（不超过50字）。",
		Model:       model,
	})
	if err != nil {
		panic(err)
	}

	reviewerAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "reviewer",
		Description: "审核员，负责审核和改进",
		Instruction: "你是审核员。审核研究结果，提出简短改进建议（不超过30字）。如果满意，回复'通过'。",
		Model:       model,
	})
	if err != nil {
		panic(err)
	}

	loopAgent, err := adk.NewLoopAgent(ctx, &adk.LoopAgentConfig{
		Name:          "research_loop",
		Description:   "研究循环：研究 -> 审核 -> 改进",
		SubAgents:     []adk.Agent{researcherAgent, reviewerAgent},
		MaxIterations: 3,
	})
	if err != nil {
		panic(err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           loopAgent,
		EnableStreaming: true,
	})

	fmt.Println("=== LoopAgent 示例 ===")
	fmt.Println("循环执行直到满意或达到最大迭代次数")
	fmt.Println()

	iter := runner.Query(ctx, "分析人工智能的发展趋势")

	iteration := 0
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			fmt.Printf("Error: %v\n", event.Err)
			continue
		}

		if event.Action != nil {
			if event.Action.TransferToAgent != nil {
				if event.Action.TransferToAgent.DestAgentName == "researcher" {
					iteration++
					fmt.Printf("\n=== 第 %d 次迭代 ===\n", iteration)
				}
				fmt.Printf("[%s] ", event.Action.TransferToAgent.DestAgentName)
			}
		}

		if msg, _, err := adk.GetMessage(event); err == nil {
			fmt.Print(msg.Content)
		}
	}

	fmt.Println("\n\n=== 循环结束 ===")
}
