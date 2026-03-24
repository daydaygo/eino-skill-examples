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
			return fmt.Sprintf("%d + %d = %d", input.A, input.B, input.A+input.B), nil
		},
	)

	subtractTool, _ := utils.InferTool(
		"subtract",
		"Subtract two numbers",
		func(ctx context.Context, input *struct {
			A int `json:"a" jsonschema:"required,description=First number"`
			B int `json:"b" jsonschema:"required,description=Second number"`
		}) (string, error) {
			return fmt.Sprintf("%d - %d = %d", input.A, input.B, input.A-input.B), nil
		},
	)

	multiplyTool, _ := utils.InferTool(
		"multiply",
		"Multiply two numbers",
		func(ctx context.Context, input *struct {
			A int `json:"a" jsonschema:"required,description=First number"`
			B int `json:"b" jsonschema:"required,description=Second number"`
		}) (string, error) {
			return fmt.Sprintf("%d * %d = %d", input.A, input.B, input.A*input.B), nil
		},
	)

	divideTool, _ := utils.InferTool(
		"divide",
		"Divide two numbers",
		func(ctx context.Context, input *struct {
			A int `json:"a" jsonschema:"required,description=Dividend"`
			B int `json:"b" jsonschema:"required,description=Divisor"`
		}) (string, error) {
			if input.B == 0 {
				return "错误: 除数不能为0", nil
			}
			return fmt.Sprintf("%d / %d = %.2f", input.A, input.B, float64(input.A)/float64(input.B)), nil
		},
	)

	addAgent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "add_agent",
		Description: "加法专家 - 执行加法运算",
		Instruction: "你是加法专家。使用 add 工具执行加法运算。",
		Model:       model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{addTool},
			},
		},
	})

	subtractAgent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "subtract_agent",
		Description: "减法专家 - 执行减法运算",
		Instruction: "你是减法专家。使用 subtract 工具执行减法运算。",
		Model:       model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{subtractTool},
			},
		},
	})

	addSubSupervisor, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "add_sub_supervisor",
		Description: "加减法协调员 - 处理加法和减法运算",
		Instruction: "你是加减法协调员。将加减法任务分配给对应的专家。",
		Model:       model,
	})

	addSubLayer, _ := supervisor.New(ctx, &supervisor.Config{
		Supervisor: addSubSupervisor,
		SubAgents:  []adk.Agent{addAgent, subtractAgent},
	})

	multiplyAgent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "multiply_agent",
		Description: "乘法专家 - 执行乘法运算",
		Instruction: "你是乘法专家。使用 multiply 工具执行乘法运算。",
		Model:       model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{multiplyTool},
			},
		},
	})

	divideAgent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "divide_agent",
		Description: "除法专家 - 执行除法运算",
		Instruction: "你是除法专家。使用 divide 工具执行除法运算。",
		Model:       model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{divideTool},
			},
		},
	})

	mulDivSupervisor, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "mul_div_supervisor",
		Description: "乘除法协调员 - 处理乘法和除法运算",
		Instruction: "你是乘除法协调员。将乘除法任务分配给对应的专家。",
		Model:       model,
	})

	mulDivLayer, _ := supervisor.New(ctx, &supervisor.Config{
		Supervisor: mulDivSupervisor,
		SubAgents:  []adk.Agent{multiplyAgent, divideAgent},
	})

	topSupervisor, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "top_supervisor",
		Description: "顶层协调员 - 分析任务类型并路由",
		Instruction: `你是顶层协调员。分析用户的计算任务：

1. 加法/减法任务 -> 转交给 add_sub_supervisor
2. 乘法/除法任务 -> 转交给 mul_div_supervisor
3. 混合运算 -> 按顺序协调处理

确保任务被正确分配到对应的层级。`,
		Model: model,
	})

	agent, _ := supervisor.New(ctx, &supervisor.Config{
		Supervisor: topSupervisor,
		SubAgents:  []adk.Agent{addSubLayer, mulDivLayer},
	})

	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})

	fmt.Println("=== 分层 Supervisor 示例 ===")
	fmt.Println("层级结构:")
	fmt.Println("  top_supervisor (顶层)")
	fmt.Println("  ├── add_sub_supervisor (加减法层)")
	fmt.Println("  │   ├── add_agent")
	fmt.Println("  │   └── subtract_agent")
	fmt.Println("  └── mul_div_supervisor (乘除法层)")
	fmt.Println("      ├── multiply_agent")
	fmt.Println("      └── divide_agent")
	fmt.Println()

	query := "请计算: (100 + 50) 和 (200 / 4)"
	fmt.Printf("用户问题: %s\n\n", query)

	iter := runner.Query(ctx, query)

	transferDepth := 0
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
			transferDepth++
			indent := ""
			for i := 0; i < transferDepth; i++ {
				indent += "  "
			}
			fmt.Printf("%s[任务转移] -> %s\n", indent, event.Action.TransferToAgent.DestAgentName)
		}

		if msg, _, err := adk.GetMessage(event); err == nil && msg.Content != "" {
			fmt.Print(msg.Content)
		}
	}
	fmt.Println("\n\n=== 执行完成 ===")
}
