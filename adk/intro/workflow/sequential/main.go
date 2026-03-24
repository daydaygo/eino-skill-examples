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

	plannerAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "planner",
		Description: "任务规划者",
		Instruction: "你是任务规划者。将任务分解为3个步骤，每个步骤一句话。",
		Model:       model,
	})
	if err != nil {
		panic(err)
	}

	executorAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "executor",
		Description: "任务执行者",
		Instruction: "你是任务执行者。根据规划步骤，简要描述如何执行这些步骤。",
		Model:       model,
	})
	if err != nil {
		panic(err)
	}

	reviewerAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "reviewer",
		Description: "结果审核者",
		Instruction: "你是结果审核者。审核执行结果，给出简洁的评价和改进建议（不超过50字）。",
		Model:       model,
	})
	if err != nil {
		panic(err)
	}

	sequentialAgent, err := adk.NewSequentialAgent(ctx, &adk.SequentialAgentConfig{
		Name:        "task_workflow",
		Description: "任务处理工作流：规划 -> 执行 -> 审核",
		SubAgents:   []adk.Agent{plannerAgent, executorAgent, reviewerAgent},
	})
	if err != nil {
		panic(err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           sequentialAgent,
		EnableStreaming: true,
	})

	fmt.Println("=== SequentialAgent 示例 ===")
	fmt.Println("顺序工作流：规划 -> 执行 -> 审核")
	fmt.Println()

	iter := runner.Query(ctx, "制定学习 Go 语言的计划")

	step := 0
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
			step++
			fmt.Printf("\n=== 步骤 %d: %s ===\n", step, event.Action.TransferToAgent.DestAgentName)
		}

		if msg, _, err := adk.GetMessage(event); err == nil {
			fmt.Print(msg.Content)
		}
	}

	fmt.Println("\n\n=== 工作流完成 ===")
}
