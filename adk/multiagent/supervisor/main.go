package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/supervisor"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
)

func main() {
	ctx := context.Background()

	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	model, _ := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: baseURL,
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		Model:   os.Getenv("OPENAI_MODEL"),
	})

	addTool, _ := utils.InferTool(
		"add",
		"Add two numbers",
		func(ctx context.Context, input *struct {
			A int `json:"a" jsonschema:"required,description=First number"`
			B int `json:"b" jsonschema:"required,description=Second number"`
		}) (string, error) {
			result := input.A + input.B
			return fmt.Sprintf("计算结果: %d + %d = %d", input.A, input.B, result), nil
		},
	)

	multiplyTool, _ := utils.InferTool(
		"multiply",
		"Multiply two numbers",
		func(ctx context.Context, input *struct {
			A int `json:"a" jsonschema:"required,description=First number"`
			B int `json:"b" jsonschema:"required,description=Second number"`
		}) (string, error) {
			result := input.A * input.B
			return fmt.Sprintf("计算结果: %d * %d = %d", input.A, input.B, result), nil
		},
	)

	searchTool, _ := utils.InferTool(
		"search",
		"Search for information on the web",
		func(ctx context.Context, input *struct {
			Query string `json:"query" jsonschema:"required,description=Search query"`
		}) (string, error) {
			return fmt.Sprintf("搜索结果: 找到了关于 '%s' 的相关信息...", input.Query), nil
		},
	)

	mathAgent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "math_agent",
		Description: "数学专家 - 执行数学计算任务，包括加法、乘法等运算",
		Instruction: "你是一个数学专家。使用可用的工具来执行精确的数学计算。始终给出清晰的计算过程和结果。",
		Model:       model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{addTool, multiplyTool},
			},
		},
	})

	researchAgent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "research_agent",
		Description: "研究专家 - 搜索和收集信息，回答知识性问题",
		Instruction: "你是一个研究专家。使用搜索工具查找信息，并提供准确、详细的回答。",
		Model:       model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{searchTool},
			},
		},
	})

	supervisorAgent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "supervisor",
		Description: "协调员 - 分析用户请求并分配给合适的专家",
		Instruction: `你是一个协调员。分析用户的请求，决定应该由哪个专家来处理：

1. math_agent: 处理数学计算、数字运算相关的问题
2. research_agent: 处理信息搜索、知识查询相关的问题

你可以：
- 将任务转交给合适的子 Agent
- 或者直接回答简单问题
- 必要时可以协调多个专家合作完成任务`,
		Model: model,
	})

	agent, _ := supervisor.New(ctx, &supervisor.Config{
		Supervisor: supervisorAgent,
		SubAgents:  []adk.Agent{mathAgent, researchAgent},
	})

	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})

	fmt.Println("=== Supervisor Agent 示例 ===")
	fmt.Println("这个示例展示了 Supervisor 模式：")
	fmt.Println("- Supervisor 作为协调者分析任务")
	fmt.Println("- 将任务分配给合适的专家 Agent")
	fmt.Println("- 支持数学计算和信息搜索两种任务")
	fmt.Println()

	query := "请帮我计算 123 + 456，然后再搜索一下 Go 语言的发展历史"
	fmt.Printf("用户问题: %s\n\n", query)

	iter := runner.Query(ctx, query)

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			fmt.Printf("[错误] %v\n", event.Err)
			continue
		}

		if event.Action != nil && event.Action.TransferToAgent != nil {
			fmt.Printf("[任务转移] -> %s\n", event.Action.TransferToAgent.DestAgentName)
		}

		if msg, _, err := adk.GetMessage(event); err == nil && msg.Content != "" {
			fmt.Print(msg.Content)
		}
	}
	fmt.Println("\n\n=== 执行完成 ===")
}
