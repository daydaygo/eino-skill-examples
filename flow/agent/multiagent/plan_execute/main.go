package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
)

type WebSearchInput struct {
	Query string `json:"query" jsonschema:"required" jsonschema_description:"搜索关键词"`
}

type APICallInput struct {
	Endpoint string            `json:"endpoint" jsonschema:"required" jsonschema_description:"API端点"`
	Params   map[string]string `json:"params" jsonschema_description:"请求参数"`
}

type CodeExecuteInput struct {
	Code string `json:"code" jsonschema:"required" jsonschema_description:"要执行的代码"`
	Lang string `json:"lang" jsonschema_description:"编程语言，如 python、javascript"`
}

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

	searchTool, err := utils.InferTool(
		"web_search",
		"搜索互联网获取信息",
		func(ctx context.Context, input *WebSearchInput) (string, error) {
			return fmt.Sprintf("搜索结果：找到关于 '%s' 的相关信息。\n1. 官方文档链接\n2. 技术博客文章\n3. 相关讨论帖", input.Query), nil
		},
	)
	if err != nil {
		panic(err)
	}

	apiTool, err := utils.InferTool(
		"api_call",
		"调用外部API",
		func(ctx context.Context, input *APICallInput) (string, error) {
			return fmt.Sprintf("API调用成功：\n端点：%s\n参数：%v\n响应：{ \"status\": \"success\", \"data\": [...] }", input.Endpoint, input.Params), nil
		},
	)
	if err != nil {
		panic(err)
	}

	codeTool, err := utils.InferTool(
		"code_execute",
		"执行代码片段",
		func(ctx context.Context, input *CodeExecuteInput) (string, error) {
			lang := input.Lang
			if lang == "" {
				lang = "python"
			}
			return fmt.Sprintf("代码执行成功（%s）：\n输出：执行完成，结果为预期值", lang), nil
		},
	)
	if err != nil {
		panic(err)
	}

	planner, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "planner",
		Description: "制定执行计划",
		Instruction: `你是一个计划制定专家。
分析用户任务，制定详细的执行步骤。

输出格式：
1. 第一步：xxx
2. 第二步：xxx
3. 第三步：xxx

确保每个步骤清晰、可执行。`,
		Model: model,
	})
	if err != nil {
		panic(err)
	}

	executor, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "executor",
		Description: "执行计划步骤",
		Instruction: `你是一个执行专家。
根据计划步骤，使用可用工具执行任务。

可用工具：
- web_search: 搜索互联网信息
- api_call: 调用外部API
- code_execute: 执行代码片段

按顺序执行每个步骤，并报告执行结果。`,
		Model: model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{searchTool, apiTool, codeTool},
			},
		},
	})
	if err != nil {
		panic(err)
	}

	replanner, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "replanner",
		Description: "评估执行结果并决定是否重规划",
		Instruction: `你是一个评估专家。
评估执行结果：
1. 如果任务完成，报告成功
2. 如果遇到问题，分析原因并决定是否需要重规划

评估结果格式：
- 状态：完成/需要重规划
- 分析：...
- 建议：...`,
		Model: model,
	})
	if err != nil {
		panic(err)
	}

	sequentialAgent, err := adk.NewSequentialAgent(ctx, &adk.SequentialAgentConfig{
		Name:        "plan_execute_workflow",
		Description: "计划-执行-评估工作流",
		SubAgents:   []adk.Agent{planner, executor, replanner},
	})
	if err != nil {
		panic(err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           sequentialAgent,
		EnableStreaming: true,
	})

	fmt.Println("=== Plan-Execute Multi-Agent 示例 ===")
	fmt.Println("演示计划-执行-评估的 Multi-Agent 协作模式")
	fmt.Println()

	tasks := []string{
		"帮我研究一下 Go 语言的并发模式，并总结最佳实践",
	}

	for i, task := range tasks {
		fmt.Printf("--- 任务 %d ---\n", i+1)
		fmt.Printf("用户: %s\n\n", task)

		iter := runner.Query(ctx, task)
		fmt.Println("执行过程：")
		stepNum := 0
		for {
			event, ok := iter.Next()
			if !ok {
				break
			}
			if event.Err != nil {
				fmt.Printf("错误: %v\n", event.Err)
				continue
			}

			if event.Action != nil {
				if event.Action.TransferToAgent != nil {
					fmt.Printf("\n[转交] -> %s\n", event.Action.TransferToAgent.DestAgentName)
				}
			}

			if msg, _, err := adk.GetMessage(event); err == nil && msg != nil && msg.Content != "" {
				stepNum++
				fmt.Printf("\n[步骤 %d 输出]\n%s\n", stepNum, msg.Content)
			}
		}
		fmt.Println("\n--- 任务完成 ---\n")
	}
}
