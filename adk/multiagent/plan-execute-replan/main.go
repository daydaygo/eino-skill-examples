package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
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

	chatModel, _ := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: baseURL,
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		Model:   os.Getenv("OPENAI_MODEL"),
	})

	searchTool, _ := utils.InferTool(
		"search",
		"Search for information on a given topic",
		func(ctx context.Context, input *struct {
			Query string `json:"query" jsonschema:"required,description=The search query"`
		}) (string, error) {
			return fmt.Sprintf("搜索结果: 关于 '%s' 的详细信息已找到。", input.Query), nil
		},
	)

	calculateTool, _ := utils.InferTool(
		"calculate",
		"Perform mathematical calculations",
		func(ctx context.Context, input *struct {
			Expression string `json:"expression" jsonschema:"required,description=The mathematical expression to evaluate"`
		}) (string, error) {
			return fmt.Sprintf("计算结果: %s = (需要数学引擎计算)", input.Expression), nil
		},
	)

	writeFileTool, _ := utils.InferTool(
		"write_file",
		"Write content to a file",
		func(ctx context.Context, input *struct {
			Filename string `json:"filename" jsonschema:"required,description=The name of the file"`
			Content  string `json:"content" jsonschema:"required,description=The content to write"`
		}) (string, error) {
			return fmt.Sprintf("文件 '%s' 已成功写入 %d 字符。", input.Filename, len(input.Content)), nil
		},
	)

	planner, _ := planexecute.NewPlanner(ctx, &planexecute.PlannerConfig{
		ChatModelWithFormattedOutput: chatModel,
	})

	executor, _ := planexecute.NewExecutor(ctx, &planexecute.ExecutorConfig{
		Model: chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{searchTool, calculateTool, writeFileTool},
			},
		},
	})

	replanner, _ := planexecute.NewReplanner(ctx, &planexecute.ReplannerConfig{
		ChatModel: chatModel,
	})

	agent, _ := planexecute.New(ctx, &planexecute.Config{
		Planner:   planner,
		Executor:  executor,
		Replanner: replanner,
	})

	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})

	fmt.Println("=== Plan-Execute-Replan 示例 ===")
	fmt.Println("工作流程:")
	fmt.Println("  Planner -> Executor -> Replanner -> (循环直到完成)")
	fmt.Println()

	query := "帮我研究一下 Go 语言的并发模型，并写一个简单的总结报告"
	fmt.Printf("用户问题: %s\n\n", query)

	iter := runner.Query(ctx, query)

	var currentPhase string
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
			dest := event.Action.TransferToAgent.DestAgentName
			if dest != currentPhase {
				currentPhase = dest
				fmt.Printf("\n[阶段: %s]\n", dest)
				switch dest {
				case "planner":
					fmt.Println("正在制定执行计划...")
				case "executor":
					fmt.Println("正在执行计划...")
				case "replanner":
					fmt.Println("正在评估和调整...")
				}
			}
		}

		if msg, _, err := adk.GetMessage(event); err == nil && msg.Content != "" {
			content := strings.TrimSpace(msg.Content)
			if content != "" {
				fmt.Println(content)
			}
		}
	}
	fmt.Println("\n\n=== 执行完成 ===")
}
