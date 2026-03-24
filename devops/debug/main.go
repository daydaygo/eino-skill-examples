package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== Eino DevOps 调试工具示例 ===")
	fmt.Println()

	fmt.Println("1. Chain 调试示例")
	fmt.Println(strings.Repeat("-", 40))
	debugChain(ctx)

	fmt.Println()
	fmt.Println("2. Graph 调试示例")
	fmt.Println(strings.Repeat("-", 40))
	debugGraph(ctx)

	fmt.Println()
	fmt.Println("3. 流式调试示例")
	fmt.Println(strings.Repeat("-", 40))
	streamDebugExample(ctx)
}

func debugChain(ctx context.Context) {
	model := createModel(ctx)

	debugHandler := callbacks.NewHandlerBuilder().
		OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
			fmt.Printf("[DEBUG] 开始执行: %s (组件类型: %s)\n", info.Name, info.Component)
			return ctx
		}).
		OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
			fmt.Printf("[DEBUG] 执行完成: %s\n", info.Name)
			return ctx
		}).
		OnErrorFn(func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			fmt.Printf("[DEBUG] 执行错误: %s, 错误: %v\n", info.Name, err)
			return ctx
		}).
		Build()

	chatTpl := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是一个助手。"),
		schema.UserMessage("请简洁回答: {query}"),
	)

	chain, err := compose.NewChain[map[string]any, *schema.Message]().
		AppendChatTemplate(chatTpl).
		AppendChatModel(model).
		Compile(ctx)
	if err != nil {
		fmt.Printf("创建 Chain 失败: %v\n", err)
		return
	}

	fmt.Println("\n[DEBUG] Chain 结构:")
	fmt.Println("  START -> ChatTemplate -> ChatModel -> END")
	fmt.Println()

	result, err := chain.Invoke(ctx, map[string]any{
		"query": "什么是 Eino?",
	}, compose.WithCallbacks(debugHandler))
	if err != nil {
		fmt.Printf("执行失败: %v\n", err)
		return
	}

	fmt.Printf("\n[RESULT] %s\n", truncateContent(result.Content))
}

func debugGraph(ctx context.Context) {
	model := createModel(ctx)

	stepCount := 0
	stepHandler := callbacks.NewHandlerBuilder().
		OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
			stepCount++
			fmt.Printf("[STEP %d] 执行节点: %s\n", stepCount, info.Name)
			return ctx
		}).
		OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
			fmt.Printf("[STEP %d] 节点完成: %s\n", stepCount, info.Name)
			return ctx
		}).
		Build()

	graph := compose.NewGraph[map[string]any, *schema.Message]()

	chatTpl := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是一个分析助手。"),
		schema.UserMessage("分析以下内容: {query}"),
	)

	_ = graph.AddChatTemplateNode("prompt", chatTpl)
	_ = graph.AddChatModelNode("model", model)
	_ = graph.AddLambdaNode("formatter", compose.InvokableLambda(
		func(ctx context.Context, msg *schema.Message) (*schema.Message, error) {
			fmt.Println("[DEBUG] Lambda 节点处理中...")
			msg.Content = fmt.Sprintf("【分析结果】\n%s", msg.Content)
			return msg, nil
		},
	))

	_ = graph.AddEdge(compose.START, "prompt")
	_ = graph.AddEdge("prompt", "model")
	_ = graph.AddEdge("model", "formatter")
	_ = graph.AddEdge("formatter", compose.END)

	fmt.Println("\n[DEBUG] Graph 结构:")
	fmt.Println("  START -> prompt -> model -> formatter -> END")
	fmt.Println()

	compiled, err := graph.Compile(ctx)
	if err != nil {
		fmt.Printf("编译 Graph 失败: %v\n", err)
		return
	}

	start := time.Now()
	result, err := compiled.Invoke(ctx, map[string]any{
		"query": "Eino 框架的优点",
	}, compose.WithCallbacks(stepHandler))
	if err != nil {
		fmt.Printf("执行失败: %v\n", err)
		return
	}

	fmt.Printf("\n[DEBUG] 总耗时: %v\n", time.Since(start))
	fmt.Printf("[DEBUG] 总步骤: %d\n", stepCount)
	fmt.Printf("\n[RESULT]\n%s\n", truncateContent(result.Content))
}

func streamDebugExample(ctx context.Context) {
	model := createModel(ctx)

	chunkCount := 0

	sr, err := model.Stream(ctx, []*schema.Message{
		schema.SystemMessage("你是一个简洁的助手"),
		schema.UserMessage("用一句话介绍 Go 语言"),
	})
	if err != nil {
		fmt.Printf("流式调用失败: %v\n", err)
		return
	}
	defer sr.Close()

	fmt.Println("\n[流式输出]")
	for {
		msg, err := sr.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			fmt.Printf("接收错误: %v\n", err)
			break
		}
		fmt.Print(msg.Content)
	}
	fmt.Printf("\n\n[DEBUG] 总块数: %d\n", chunkCount)
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

func truncateContent(content string) string {
	if len(content) > 200 {
		return content[:200] + "..."
	}
	return content
}
