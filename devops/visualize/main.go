package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

type WeatherInput struct {
	City string `json:"city" jsonschema:"required" jsonschema_description:"城市名称"`
}

func main() {
	fmt.Println("=== Eino 编排结构可视化工具 ===")
	fmt.Println()

	fmt.Println("1. Chain 可视化")
	fmt.Println(strings.Repeat("-", 50))
	visualizeChain()

	fmt.Println()
	fmt.Println("2. Graph 可视化 (带分支)")
	fmt.Println(strings.Repeat("-", 50))
	visualizeGraph()

	fmt.Println()
	fmt.Println("3. 复杂 Graph 可视化 (带循环)")
	fmt.Println(strings.Repeat("-", 50))
	visualizeComplexGraph()

	fmt.Println()
	fmt.Println("4. ReAct Agent 可视化")
	fmt.Println(strings.Repeat("-", 50))
	visualizeReactAgent()
}

func visualizeChain() {
	chatTpl := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是一个助手。"),
		schema.UserMessage("{query}"),
	)

	_ = compose.NewChain[map[string]any, *schema.Message]().
		AppendChatTemplate(chatTpl).
		AppendLambda(compose.InvokableLambda(
			func(ctx context.Context, msg *schema.Message) (*schema.Message, error) {
				return msg, nil
			},
		)).
		AppendPassthrough()

	fmt.Println("\nChain 结构:")
	fmt.Println(ChainToMermaid())
	fmt.Println("\n说明:")
	fmt.Println("  - START: 输入起点")
	fmt.Println("  - ChatTemplate: 消息模板处理")
	fmt.Println("  - Lambda: 自定义处理节点")
	fmt.Println("  - Passthrough: 透传节点")
	fmt.Println("  - END: 输出终点")
}

func visualizeGraph() {
	graph := compose.NewGraph[map[string]any, *schema.Message]()

	_ = graph.AddLambdaNode("preprocess", compose.InvokableLambda(
		func(ctx context.Context, input map[string]any) (map[string]any, error) {
			return input, nil
		},
	))

	_ = graph.AddLambdaNode("router", compose.InvokableLambda(
		func(ctx context.Context, input map[string]any) (string, error) {
			return "route_a", nil
		},
	))

	_ = graph.AddLambdaNode("route_a", compose.InvokableLambda(
		func(ctx context.Context, input map[string]any) (*schema.Message, error) {
			return schema.UserMessage("路径 A"), nil
		},
	))

	_ = graph.AddLambdaNode("route_b", compose.InvokableLambda(
		func(ctx context.Context, input map[string]any) (*schema.Message, error) {
			return schema.UserMessage("路径 B"), nil
		},
	))

	_ = graph.AddEdge(compose.START, "preprocess")
	_ = graph.AddEdge("preprocess", "router")

	_ = graph.AddBranch("router", compose.NewGraphBranch(
		func(ctx context.Context, input map[string]any) (string, error) {
			return "route_a", nil
		},
		map[string]bool{"route_a": true, "route_b": true},
	))

	_ = graph.AddEdge("route_a", compose.END)
	_ = graph.AddEdge("route_b", compose.END)

	fmt.Println("\nGraph 结构 (带分支):")
	fmt.Println(GraphBranchToMermaid())
	fmt.Println("\n说明:")
	fmt.Println("  - 菱形节点 (router): 分支决策点")
	fmt.Println("  - 根据条件选择不同的执行路径")
	fmt.Println("  - route_a/route_b: 分支目标节点")
}

func visualizeComplexGraph() {
	fmt.Println("\n复杂 Graph 结构 (ReAct 循环模式):")
	fmt.Println(ReActGraphToMermaid())
	fmt.Println("\n说明:")
	fmt.Println("  - 这是最常见的 Agent 循环模式")
	fmt.Println("  - model 节点生成回复或工具调用")
	fmt.Println("  - 如果有工具调用，执行 tools 后回到 model")
	fmt.Println("  - 直到生成最终回复后结束")
}

func visualizeReactAgent() {
	ctx := context.Background()
	model := createModel(ctx)

	weatherTool, _ := utils.InferTool(
		"get_weather",
		"获取天气",
		func(ctx context.Context, input *WeatherInput) (string, error) {
			return fmt.Sprintf("%s 天气晴", input.City), nil
		},
	)

	agent, _ := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: model,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: []tool.BaseTool{weatherTool},
		},
	})

	_ = agent

	fmt.Println("\nReAct Agent 内部结构:")
	fmt.Println(ReActAgentToMermaid())
	fmt.Println("\n说明:")
	fmt.Println("  - ReAct = Reasoning + Acting")
	fmt.Println("  - 用户输入 -> 思考 -> 决定是否使用工具")
	fmt.Println("  - 使用工具 -> 观察结果 -> 继续思考")
	fmt.Println("  - 直到生成最终回复")
}

func ChainToMermaid() string {
	var sb strings.Builder
	sb.WriteString("```mermaid\n")
	sb.WriteString("flowchart LR\n")
	sb.WriteString("    START((START))\n")
	sb.WriteString("    A[ChatTemplate<br/>消息模板]\n")
	sb.WriteString("    B[Lambda<br/>自定义处理]\n")
	sb.WriteString("    C[Passthrough<br/>透传]\n")
	sb.WriteString("    END((END))\n")
	sb.WriteString("    \n")
	sb.WriteString("    START --> A --> B --> C --> END\n")
	sb.WriteString("```\n")
	return sb.String()
}

func GraphBranchToMermaid() string {
	var sb strings.Builder
	sb.WriteString("```mermaid\n")
	sb.WriteString("flowchart TD\n")
	sb.WriteString("    START((START))\n")
	sb.WriteString("    PRE[preprocess<br/>预处理]\n")
	sb.WriteString("    ROUTER{router<br/>路由决策}\n")
	sb.WriteString("    A[route_a<br/>路径A]\n")
	sb.WriteString("    B[route_b<br/>路径B]\n")
	sb.WriteString("    END((END))\n")
	sb.WriteString("    \n")
	sb.WriteString("    START --> PRE --> ROUTER\n")
	sb.WriteString("    ROUTER -->|条件A| A --> END\n")
	sb.WriteString("    ROUTER -->|条件B| B --> END\n")
	sb.WriteString("```\n")
	return sb.String()
}

func ReActGraphToMermaid() string {
	var sb strings.Builder
	sb.WriteString("```mermaid\n")
	sb.WriteString("flowchart TD\n")
	sb.WriteString("    START((START))\n")
	sb.WriteString("    PROMPT[prompt<br/>提示词]\n")
	sb.WriteString("    MODEL[ChatModel<br/>大模型]\n")
	sb.WriteString("    DECIDE{有工具调用?}\n")
	sb.WriteString("    TOOLS[ToolsNode<br/>工具执行]\n")
	sb.WriteString("    END((END))\n")
	sb.WriteString("    \n")
	sb.WriteString("    START --> PROMPT --> MODEL --> DECIDE\n")
	sb.WriteString("    DECIDE -->|是| TOOLS --> MODEL\n")
	sb.WriteString("    DECIDE -->|否| END\n")
	sb.WriteString("```\n")
	return sb.String()
}

func ReActAgentToMermaid() string {
	var sb strings.Builder
	sb.WriteString("```mermaid\n")
	sb.WriteString("flowchart TD\n")
	sb.WriteString("    START((START))\n")
	sb.WriteString("    INPUT[/用户输入/]\n")
	sb.WriteString("    THINK[思考: 分析问题<br/>决定下一步行动]\n")
	sb.WriteString("    DECIDE{需要工具?}\n")
	sb.WriteString("    TOOL[执行工具调用<br/>获取外部信息]\n")
	sb.WriteString("    OBSERVE[观察工具结果<br/>更新上下文]\n")
	sb.WriteString("    RESPOND[生成最终回复]\n")
	sb.WriteString("    END((END))\n")
	sb.WriteString("    \n")
	sb.WriteString("    START --> INPUT --> THINK --> DECIDE\n")
	sb.WriteString("    DECIDE -->|是| TOOL --> OBSERVE --> THINK\n")
	sb.WriteString("    DECIDE -->|否| RESPOND --> END\n")
	sb.WriteString("    \n")
	sb.WriteString("    style THINK fill:#e1f5fe\n")
	sb.WriteString("    style TOOL fill:#fff3e0\n")
	sb.WriteString("    style OBSERVE fill:#f3e5f5\n")
	sb.WriteString("```\n")
	return sb.String()
}

func createModel(ctx context.Context) *openai.ChatModel {
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
	return model
}
