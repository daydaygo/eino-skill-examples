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

type CalculatorInput struct {
	Expression string `json:"expression" jsonschema:"required" jsonschema_description:"数学表达式"`
}

type TranslateInput struct {
	Text       string `json:"text" jsonschema:"required" jsonschema_description:"要翻译的文本"`
	TargetLang string `json:"target_lang" jsonschema:"required" jsonschema_description:"目标语言"`
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
			return fmt.Sprintf("搜索结果：找到关于 '%s' 的相关信息...", input.Query), nil
		},
	)
	if err != nil {
		panic(err)
	}

	calcTool, err := utils.InferTool(
		"calculate",
		"执行数学计算",
		func(ctx context.Context, input *CalculatorInput) (string, error) {
			return fmt.Sprintf("计算结果：%s = 42", input.Expression), nil
		},
	)
	if err != nil {
		panic(err)
	}

	translateTool, err := utils.InferTool(
		"translate",
		"翻译文本到指定语言",
		func(ctx context.Context, input *TranslateInput) (string, error) {
			return fmt.Sprintf("翻译结果（%s）：%s -> [已翻译内容]", input.TargetLang, input.Text), nil
		},
	)
	if err != nil {
		panic(err)
	}

	fmt.Println("=== 动态工具选择示例 ===")
	fmt.Println("演示根据场景动态选择不同的工具集")
	fmt.Println()

	fmt.Println("--- 场景1: 搜索工具 ---")
	runWithTools(ctx, model, "帮我搜索一下最新的AI技术进展", []tool.BaseTool{searchTool})

	fmt.Println("\n--- 场景2: 计算工具 ---")
	runWithTools(ctx, model, "帮我计算 123 * 456", []tool.BaseTool{calcTool})

	fmt.Println("\n--- 场景3: 翻译工具 ---")
	runWithTools(ctx, model, "请将 'Hello, World!' 翻译成中文", []tool.BaseTool{translateTool})

	fmt.Println("\n--- 场景4: 多工具组合 ---")
	runWithTools(ctx, model, "搜索并计算结果", []tool.BaseTool{searchTool, calcTool})
}

func runWithTools(ctx context.Context, model *openai.ChatModel, query string, tools []tool.BaseTool) {
	fmt.Printf("用户: %s\n", query)

	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: model,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: tools,
		},
		MessageModifier: func(ctx context.Context, input []*schema.Message) []*schema.Message {
			res := make([]*schema.Message, 0, len(input)+1)
			res = append(res, schema.SystemMessage("你是一个智能助手，根据用户需求使用可用工具完成任务。"))
			res = append(res, input...)
			return res
		},
		MaxStep: 10,
	})
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}

	sr, err := agent.Stream(ctx, []*schema.Message{schema.UserMessage(query)})
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}
	defer sr.Close()

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
	fmt.Println()
}
