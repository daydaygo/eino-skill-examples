package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

type BookTicketInput struct {
	From      string `json:"from" jsonschema:"required,description=出发地"`
	To        string `json:"to" jsonschema:"required,description=目的地"`
	Date      string `json:"date" jsonschema:"required,description=出发日期"`
	Passenger string `json:"passenger" jsonschema:"required,description=乘客姓名"`
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

	bookTicketTool, err := utils.InferTool(
		"book_ticket",
		"预订火车票",
		func(ctx context.Context, input *BookTicketInput) (string, error) {
			return fmt.Sprintf("已预订: %s 从 %s 到 %s，乘客: %s",
				input.Date, input.From, input.To, input.Passenger), nil
		},
	)
	if err != nil {
		panic(err)
	}

	toolsNode, err := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
		Tools: []tool.BaseTool{bookTicketTool},
	})
	if err != nil {
		panic(err)
	}

	graph := compose.NewGraph[[]*schema.Message, *schema.Message]()

	err = graph.AddChatModelNode("model", model)
	if err != nil {
		panic(err)
	}

	err = graph.AddToolsNode("tools", toolsNode)
	if err != nil {
		panic(err)
	}

	err = graph.AddLambdaNode("check_tool_call", compose.InvokableLambda(
		func(ctx context.Context, msg *schema.Message) (*schema.Message, error) {
			if len(msg.ToolCalls) > 0 {
				return schema.AssistantMessage(fmt.Sprintf(
					"[需要人工确认] 工具调用: %s, 参数: %+v",
					msg.ToolCalls[0].Function.Name,
					msg.ToolCalls[0].Function.Arguments,
				), nil), nil
			}
			return msg, nil
		},
	))
	if err != nil {
		panic(err)
	}

	err = graph.AddLambdaNode("interrupt_for_approval", compose.InvokableLambda(
		func(ctx context.Context, msg *schema.Message) (bool, error) {
			fmt.Printf("\n=== 中断点 ===\n")
			fmt.Printf("消息: %s\n", msg.Content)
			fmt.Println("等待人工审批... (模拟自动批准)")
			fmt.Println("==============\n")
			return true, nil
		},
	))
	if err != nil {
		panic(err)
	}

	err = graph.AddEdge(compose.START, "model")
	if err != nil {
		panic(err)
	}

	err = graph.AddEdge("model", "check_tool_call")
	if err != nil {
		panic(err)
	}

	branch := compose.NewGraphBranch(
		func(ctx context.Context, msg *schema.Message) (string, error) {
			if len(msg.ToolCalls) > 0 {
				return "interrupt_for_approval", nil
			}
			return compose.END, nil
		},
		map[string]bool{"interrupt_for_approval": true, compose.END: true},
	)
	err = graph.AddBranch("check_tool_call", branch)
	if err != nil {
		panic(err)
	}

	err = graph.AddEdge("interrupt_for_approval", "tools")
	if err != nil {
		panic(err)
	}

	err = graph.AddEdge("tools", "model")
	if err != nil {
		panic(err)
	}

	compiled, err := graph.Compile(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Println("=== ReAct + 中断示例 ===")
	fmt.Println("场景: 票务预订，需要人工确认后才执行工具调用")
	fmt.Println("Graph 结构: START -> model -> check_tool_call -> [interrupt_for_approval -> tools -> model] 或 [END]")
	fmt.Println()

	result, err := compiled.Invoke(ctx, []*schema.Message{
		schema.SystemMessage("你是一个票务助手。当用户要订票时，使用 book_ticket 工具。"),
		schema.UserMessage("帮我订一张明天从北京到上海的票，乘客是张三"),
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("最终结果: %s\n", result.Content)
	fmt.Println("=== 完成 ===")
}
