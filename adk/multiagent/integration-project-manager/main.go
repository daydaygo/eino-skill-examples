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

	writeCodeTool, _ := utils.InferTool(
		"write_code",
		"Write code in a specified language",
		func(ctx context.Context, input *struct {
			Language string `json:"language" jsonschema:"required,description=Programming language"`
			Code     string `json:"code" jsonschema:"required,description=The code to write"`
		}) (string, error) {
			return fmt.Sprintf("代码已写入 (%s):\n```\n%s\n```", input.Language, input.Code), nil
		},
	)

	runCodeTool, _ := utils.InferTool(
		"run_code",
		"Execute code and return the result",
		func(ctx context.Context, input *struct {
			Language string `json:"language" jsonschema:"required,description=Programming language"`
			Code     string `json:"code" jsonschema:"required,description=The code to execute"`
		}) (string, error) {
			return fmt.Sprintf("代码执行成功\n输出: Hello, World!"), nil
		},
	)

	searchDocsTool, _ := utils.InferTool(
		"search_docs",
		"Search documentation and technical resources",
		func(ctx context.Context, input *struct {
			Query string `json:"query" jsonschema:"required,description=Search query for documentation"`
		}) (string, error) {
			return fmt.Sprintf("文档搜索结果: 找到关于 '%s' 的相关文档。", input.Query), nil
		},
	)

	searchWebTool, _ := utils.InferTool(
		"search_web",
		"Search the web for information",
		func(ctx context.Context, input *struct {
			Query string `json:"query" jsonschema:"required,description=Web search query"`
		}) (string, error) {
			return fmt.Sprintf("网络搜索结果: 找到关于 '%s' 的最新信息。", input.Query), nil
		},
	)

	reviewCodeTool, _ := utils.InferTool(
		"review_code",
		"Review code for issues and improvements",
		func(ctx context.Context, input *struct {
			Code     string `json:"code" jsonschema:"required,description=Code to review"`
			Criteria string `json:"criteria" jsonschema:"optional,description=Review criteria"`
		}) (string, error) {
			return "代码审查完成:\n- 代码结构清晰\n- 命名规范\n- 建议添加更多注释", nil
		},
	)

	runTestsTool, _ := utils.InferTool(
		"run_tests",
		"Run unit tests for the code",
		func(ctx context.Context, input *struct {
			TestFile string `json:"test_file" jsonschema:"required,description=Test file to run"`
		}) (string, error) {
			return "测试运行完成:\n- 通过: 5\n- 失败: 0\n- 覆盖率: 85%", nil
		},
	)

	coderAgent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "coder",
		Description: "程序员 - 编写和修改代码",
		Instruction: `你是一个专业的程序员。你的职责是：
1. 编写高质量的代码
2. 修复 bug
3. 优化代码性能

使用 write_code 工具编写代码，使用 run_code 工具测试运行。`,
		Model: model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{writeCodeTool, runCodeTool},
			},
		},
	})

	researcherAgent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "researcher",
		Description: "研究员 - 搜索和整理信息",
		Instruction: `你是一个专业的研究员。你的职责是：
1. 搜索相关文档和技术资料
2. 整理信息并提供摘要
3. 回答技术问题

使用 search_docs 和 search_web 工具获取信息。`,
		Model: model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{searchDocsTool, searchWebTool},
			},
		},
	})

	reviewerAgent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "reviewer",
		Description: "审查员 - 审查代码和运行测试",
		Instruction: `你是一个专业的代码审查员。你的职责是：
1. 审查代码质量
2. 检查最佳实践
3. 运行测试验证

使用 review_code 和 run_tests 工具完成审查。`,
		Model: model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{reviewCodeTool, runTestsTool},
			},
		},
	})

	supervisorAgent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "project_manager",
		Description: "项目经理 - 协调团队完成任务",
		Instruction: `你是一个项目经理，负责协调以下团队成员：

1. coder - 程序员
   - 编写代码
   - 修复 bug
   - 代码优化

2. researcher - 研究员
   - 搜索文档
   - 技术调研
   - 信息整理

3. reviewer - 审查员
   - 代码审查
   - 运行测试
   - 质量把关

根据任务需求，合理分配工作给团队成员。
对于编程任务，典型流程：
1. researcher 调研需求
2. coder 编写代码
3. reviewer 审查测试
4. coder 修复问题（如有）
`,
		Model: model,
	})

	agent, _ := supervisor.New(ctx, &supervisor.Config{
		Supervisor: supervisorAgent,
		SubAgents:  []adk.Agent{coderAgent, researcherAgent, reviewerAgent},
	})

	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})

	fmt.Println("=== 项目管理器示例 ===")
	fmt.Println("团队成员:")
	fmt.Println("  - project_manager: 项目经理 (协调员)")
	fmt.Println("  - coder: 程序员 (编码)")
	fmt.Println("  - researcher: 研究员 (调研)")
	fmt.Println("  - reviewer: 审查员 (审查)")
	fmt.Println()

	query := `请帮我完成以下任务：
1. 调研 Go 语言的错误处理最佳实践
2. 写一个简单的错误处理示例代码
3. 审查代码质量`

	fmt.Printf("用户需求:\n%s\n\n", query)

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
			dest := event.Action.TransferToAgent.DestAgentName
			role := ""
			switch dest {
			case "coder":
				role = "程序员"
			case "researcher":
				role = "研究员"
			case "reviewer":
				role = "审查员"
			case "project_manager":
				role = "项目经理"
			}
			fmt.Printf("\n[%s 正在工作]\n", role)
		}

		if msg, _, err := adk.GetMessage(event); err == nil && msg.Content != "" {
			fmt.Print(msg.Content)
		}
	}
	fmt.Println("\n\n=== 任务完成 ===")
}
