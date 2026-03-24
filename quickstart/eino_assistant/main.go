package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

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

	store := NewMemoryStore()

	fmt.Println("=== 索引知识库 ===")
	docs := []string{
		"Eino 是 CloudWeGo 开源的 Go 语言大模型应用开发框架。",
		"Eino 提供组件抽象：ChatModel、Tool、Retriever、Embedding 等。",
		"Eino 支持编排能力：Chain（链式）、Graph（有向图）、Workflow（字段映射）。",
		"Eino ADK 提供 ChatModelAgent、Supervisor、Plan-Execute 等智能体模式。",
		"Eino 支持流处理的四种范式：Invoke、Stream、Collect、Transform。",
	}
	for i, doc := range docs {
		store.Index(ctx, doc, map[string]any{"id": i + 1})
		fmt.Printf("  [%d] %s\n", i+1, doc)
	}

	searchTool, err := NewSearchTool(store)
	if err != nil {
		panic(err)
	}

	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: model,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: []tool.BaseTool{searchTool},
		},
		MessageModifier: func(ctx context.Context, input []*schema.Message) []*schema.Message {
			res := make([]*schema.Message, 0, len(input)+1)
			res = append(res, schema.SystemMessage(`你是 Eino 框架助手。使用 search_knowledge 工具检索知识库来回答问题。
如果知识库中没有相关信息，请说明并提供你的知识。`))
			res = append(res, input...)
			return res
		},
		MaxStep: 10,
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("\n=== Eino 助手已就绪 ===")
	fmt.Println("输入问题进行对话，输入 'quit' 退出")
	fmt.Println("示例问题：")
	fmt.Println("  - Eino 是什么？")
	fmt.Println("  - Eino 支持哪些编排方式？")
	fmt.Println("  - Eino ADK 提供哪些智能体模式？")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		query := strings.TrimSpace(line)
		if query == "" {
			continue
		}
		if query == "quit" || query == "exit" {
			fmt.Println("再见！")
			break
		}

		sr, err := agent.Stream(ctx, []*schema.Message{schema.UserMessage(query)})
		if err != nil {
			fmt.Printf("错误: %v\n", err)
			continue
		}

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
