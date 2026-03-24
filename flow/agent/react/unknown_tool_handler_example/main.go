package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

type SearchInput struct {
	Query string `json:"query" jsonschema:"required" jsonschema_description:"搜索关键词"`
}

func handleUnknownToolCall(toolName string, args map[string]any) (string, error) {
	fmt.Printf("[未知工具处理] 检测到幻觉工具调用: %s, 参数: %v\n", toolName, args)

	commonHallucinations := map[string]string{
		"get_user_info":    "获取用户信息功能暂未实现，请让用户提供相关信息",
		"send_email":       "发送邮件功能暂未实现，请使用其他方式通知用户",
		"check_database":   "数据库查询功能暂未实现，请尝试其他方式获取信息",
		"call_api":         "外部API调用功能暂未实现",
		"translate_text":   "请使用内置的翻译能力而非工具",
		"get_current_user": "无法获取当前用户信息",
	}

	if suggestion, ok := commonHallucinations[toolName]; ok {
		return fmt.Sprintf("工具 '%s' 不存在。%s", toolName, suggestion), nil
	}

	return fmt.Sprintf("工具 '%s' 不存在。请检查是否有其他可用的工具，或者使用其他方式完成任务。", toolName), nil
}

func createToolsInterceptor(baseTools []tool.BaseTool, unknownHandler func(string, map[string]any) (string, error)) *ToolsInterceptor {
	return &ToolsInterceptor{
		tools:          baseTools,
		unknownHandler: unknownHandler,
		knownToolNames: make(map[string]bool),
	}
}

type ToolsInterceptor struct {
	tools          []tool.BaseTool
	unknownHandler func(string, map[string]any) (string, error)
	knownToolNames map[string]bool
}

func (t *ToolsInterceptor) Init(ctx context.Context) {
	for _, tool := range t.tools {
		if info, err := tool.Info(ctx); err == nil {
			t.knownToolNames[info.Name] = true
		}
	}
}

func (t *ToolsInterceptor) IsKnownTool(name string) bool {
	return t.knownToolNames[name]
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
		"search",
		"搜索互联网信息",
		func(ctx context.Context, input *SearchInput) (string, error) {
			return fmt.Sprintf("搜索 '%s' 的结果：找到相关信息...", input.Query), nil
		},
	)
	if err != nil {
		panic(err)
	}

	tools := []tool.BaseTool{searchTool}
	interceptor := createToolsInterceptor(tools, handleUnknownToolCall)
	interceptor.Init(ctx)

	wrappedTools := make([]tool.BaseTool, len(tools))
	copy(wrappedTools, tools)

	fallbackTool, err := utils.InferTool(
		"fallback_handler",
		"处理未知工具调用的回退处理器",
		func(ctx context.Context, input *struct {
			OriginalTool string         `json:"original_tool" jsonschema_description:"原始调用的工具名称"`
			Args         map[string]any `json:"args" jsonschema_description:"原始调用的参数"`
		}) (string, error) {
			result, _ := interceptor.unknownHandler(input.OriginalTool, input.Args)
			return result, nil
		},
	)
	if err != nil {
		panic(err)
	}
	wrappedTools = append(wrappedTools, fallbackTool)

	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: model,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: wrappedTools,
		},
		MessageModifier: func(ctx context.Context, input []*schema.Message) []*schema.Message {
			res := make([]*schema.Message, 0, len(input)+1)
			res = append(res, schema.SystemMessage(`你是一个智能助手。
重要提示：
1. 只使用明确提供的工具
2. 如果用户请求的功能没有对应工具，请直接说明并建议替代方案
3. 不要臆造不存在的工具
4. 可用工具：search（搜索信息）

如果模型尝试调用不存在的工具，系统会捕获并返回友好的错误信息。`))
			res = append(res, input...)
			return res
		},
		MaxStep: 20,
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("=== 未知工具处理示例 ===")
	fmt.Println("演示如何处理模型产生的幻觉工具调用")
	fmt.Println()

	testCases := []string{
		"帮我搜索一下今天的新闻",
		"请发送一封邮件给 test@example.com",
		"查询用户ID为123的用户信息",
		"翻译 'Hello' 到中文",
	}

	for i, query := range testCases {
		fmt.Printf("--- 测试 %d ---\n", i+1)
		fmt.Printf("用户: %s\n", query)

		sr, err := agent.Stream(ctx, []*schema.Message{schema.UserMessage(query)})
		if err != nil {
			fmt.Printf("错误: %v\n", err)
			continue
		}

		fmt.Print("助手: ")
		for {
			msg, err := sr.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				fmt.Printf("错误: %v\n", err)
				break
			}
			fmt.Print(msg.Content)
		}
		fmt.Println("\n")
	}
}
