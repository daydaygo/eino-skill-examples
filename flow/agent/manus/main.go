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

type FileReadInput struct {
	Path string `json:"path" jsonschema:"required" jsonschema_description:"文件路径"`
}

type FileWriteInput struct {
	Path    string `json:"path" jsonschema:"required" jsonschema_description:"文件路径"`
	Content string `json:"content" jsonschema:"required" jsonschema_description:"文件内容"`
}

type ShellExecuteInput struct {
	Command string `json:"command" jsonschema:"required" jsonschema_description:"要执行的命令"`
}

type WebSearchInput struct {
	Query string `json:"query" jsonschema:"required" jsonschema_description:"搜索关键词"`
}

type BrowserInput struct {
	URL string `json:"url" jsonschema:"required" jsonschema_description:"要访问的URL"`
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

	fileReadTool, err := utils.InferTool(
		"file_read",
		"读取文件内容",
		func(ctx context.Context, input *FileReadInput) (string, error) {
			return fmt.Sprintf("文件 %s 内容：\n[模拟文件内容]\npackage main\n\nfunc main() {\n\tprintln(\"Hello\")\n}", input.Path), nil
		},
	)
	if err != nil {
		panic(err)
	}

	fileWriteTool, err := utils.InferTool(
		"file_write",
		"写入文件内容",
		func(ctx context.Context, input *FileWriteInput) (string, error) {
			return fmt.Sprintf("成功写入文件：%s，共 %d 字节", input.Path, len(input.Content)), nil
		},
	)
	if err != nil {
		panic(err)
	}

	shellTool, err := utils.InferTool(
		"shell_execute",
		"执行 shell 命令",
		func(ctx context.Context, input *ShellExecuteInput) (string, error) {
			return fmt.Sprintf("执行命令：%s\n输出：\n[命令执行成功]\n结果：操作完成", input.Command), nil
		},
	)
	if err != nil {
		panic(err)
	}

	webSearchTool, err := utils.InferTool(
		"web_search",
		"搜索互联网信息",
		func(ctx context.Context, input *WebSearchInput) (string, error) {
			return fmt.Sprintf("搜索 '%s' 的结果：\n1. 相关文章链接\n2. 官方文档\n3. 社区讨论", input.Query), nil
		},
	)
	if err != nil {
		panic(err)
	}

	browserTool, err := utils.InferTool(
		"browser_navigate",
		"打开浏览器访问网页",
		func(ctx context.Context, input *BrowserInput) (string, error) {
			return fmt.Sprintf("访问：%s\n页面标题：示例页面\n主要内容：[页面内容摘要]", input.URL), nil
		},
	)
	if err != nil {
		panic(err)
	}

	plannerAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "planner",
		Description: "任务规划者，负责分析和拆解复杂任务",
		Instruction: `你是一个任务规划专家。
分析用户的复杂请求，制定详细的执行计划。
将大任务拆分为可执行的小步骤。

输出格式：
## 任务分析
[分析用户需求]

## 执行计划
1. [步骤1]
2. [步骤2]
3. [步骤3]

## 预期结果
[描述预期的最终结果]`,
		Model: model,
	})
	if err != nil {
		panic(err)
	}

	coderAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "coder",
		Description: "代码编写专家，负责编写和修改代码",
		Instruction: `你是一个代码编写专家。
根据任务需求编写高质量的代码。
使用 file_read 读取现有文件，使用 file_write 创建或修改文件。

代码规范：
1. 代码简洁、可读性强
2. 添加必要的注释
3. 遵循语言最佳实践`,
		Model: model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{fileReadTool, fileWriteTool},
			},
		},
	})
	if err != nil {
		panic(err)
	}

	researcherAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "researcher",
		Description: "研究专家，负责搜索和整理信息",
		Instruction: `你是一个研究专家。
使用 web_search 搜索相关信息，使用 browser_navigate 访问网页获取详细内容。

研究流程：
1. 理解研究目标
2. 搜索相关资料
3. 整理关键信息
4. 生成研究报告`,
		Model: model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{webSearchTool, browserTool},
			},
		},
	})
	if err != nil {
		panic(err)
	}

	executorAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "executor",
		Description: "执行专家，负责执行各种操作",
		Instruction: `你是一个执行专家。
使用 shell_execute 执行系统命令，完成各种操作任务。

注意事项：
1. 执行前确认操作安全性
2. 处理可能的错误
3. 报告执行结果`,
		Model: model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{shellTool, fileWriteTool},
			},
		},
	})
	if err != nil {
		panic(err)
	}

	manusAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "manus",
		Description: "Manus 主控 Agent，协调各个专业 Agent 完成复杂任务",
		Instruction: `你是 Manus，一个强大的自主 Agent。
你可以协调多个专业 Agent 来完成复杂任务：

- planner: 任务规划，拆解复杂任务
- coder: 代码编写和修改
- researcher: 信息搜索和整理
- executor: 执行系统操作

根据用户需求，选择合适的 Agent 来完成任务。
你可以同时调度多个 Agent 并行工作。

工作流程：
1. 理解用户需求
2. 制定执行策略
3. 分配任务给专业 Agent
4. 整合结果
5. 报告完成情况`,
		Model: model,
	})
	if err != nil {
		panic(err)
	}

	manusAgentWithSubAgents, err := adk.SetSubAgents(ctx, manusAgent, []adk.Agent{
		plannerAgent,
		coderAgent,
		researcherAgent,
		executorAgent,
	})
	if err != nil {
		panic(err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           manusAgentWithSubAgents,
		EnableStreaming: true,
	})

	fmt.Println("=== Manus Agent 示例 ===")
	fmt.Println("基于 Eino 实现的自主 Agent，参考 OpenManus 项目")
	fmt.Println("支持：任务规划、代码编写、信息研究、系统执行")
	fmt.Println()

	tasks := []string{
		"帮我创建一个简单的 Go HTTP 服务器，监听 8080 端口",
		"研究一下 Go 语言的并发编程最佳实践，并总结要点",
	}

	for i, task := range tasks {
		fmt.Printf("--- 任务 %d ---\n", i+1)
		fmt.Printf("用户: %s\n\n", task)

		iter := runner.Query(ctx, task)

		fmt.Println("执行过程：")
		agentCalls := 0
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
					agentCalls++
					fmt.Printf("\n[调用 Agent] -> %s\n", event.Action.TransferToAgent.DestAgentName)
				}
			}

			if msg, _, err := adk.GetMessage(event); err == nil && msg != nil && msg.Content != "" {
				fmt.Printf("%s", msg.Content)
			}
		}
		fmt.Printf("\n\n--- 任务完成 (共调用 %d 次 Agent) ---\n\n", agentCalls)
	}
}
