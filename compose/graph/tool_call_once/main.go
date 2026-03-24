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

type SearchInput struct {
	Query string `json:"query" jsonschema:"required,description=搜索查询"`
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
		"搜索互联网获取信息",
		func(ctx context.Context, input *SearchInput) (string, error) {
			return fmt.Sprintf("搜索结果: 关于 '%s' 的最新信息...", input.Query), nil
		},
	)
	if err != nil {
		panic(err)
	}

	toolsNode, err := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
		Tools: []tool.BaseTool{searchTool},
	})
	if err != nil {
		panic(err)
	}

	graph := compose.NewGraph[[]*schema.Message, []*schema.Message]()

	err = graph.AddChatModelNode("model", model)
	if err != nil {
		panic(err)
	}

	err = graph.AddToolsNode("tools", toolsNode)
	if err != nil {
		panic(err)
	}

	err = graph.AddLambdaNode("format_result", compose.InvokableLambda(
		func(ctx context.Context, msgs []*schema.Message) (*schema.Message, error) {
			if len(msgs) == 0 {
				return schema.AssistantMessage("无结果", nil), nil
			}
			return msgs[len(msgs)-1], nil
		},
	))
	if err != nil {
		panic(err)
	}

	err = graph.AddEdge(compose.START, "model")
	if err != nil {
		panic(err)
	}

	err = graph.AddEdge("model", "tools")
	if err != nil {
		panic(err)
	}

	err = graph.AddEdge("tools", "format_result")
	if err != nil {
		panic(err)
	}

	err = graph.AddEdge("format_result", compose.END)
	if err != nil {
		panic(err)
	}

	compiled, err := graph.Compile(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Println("=== 单次工具调用示例 ===")
	fmt.Println("Graph 结构: START -> model -> tools -> format_result -> END")
	fmt.Println()

	result, err := compiled.Invoke(ctx, []*schema.Message{
		schema.SystemMessage("你是一个搜索助手，使用搜索工具获取信息并回答。"),
		schema.UserMessage("搜索一下今天的新闻"),
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("结果数量: %d\n", len(result))
	if len(result) > 0 {
		fmt.Printf("最后一条: %s\n", result[len(result)-1].Content)
	}
	fmt.Println("=== 完成 ===")
}
