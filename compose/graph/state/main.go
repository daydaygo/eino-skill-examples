package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

type ConversationState struct {
	Messages []*schema.Message
	Turn     int
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

	graph := compose.NewGraph[*ConversationState, *schema.Message]()

	preHandler := func(ctx context.Context, in []*schema.Message, state *ConversationState) ([]*schema.Message, error) {
		state.Turn++
		fmt.Printf("=== 第 %d 轮对话 ===\n", state.Turn)
		return in, nil
	}

	err = graph.AddLambdaNode("preprocess", compose.InvokableLambda(
		func(ctx context.Context, state *ConversationState) ([]*schema.Message, error) {
			state.Turn++
			fmt.Printf("=== 第 %d 轮对话 ===\n", state.Turn)
			return state.Messages, nil
		},
	))
	if err != nil {
		panic(err)
	}

	err = graph.AddChatModelNode("model", model)
	if err != nil {
		panic(err)
	}

	err = graph.AddEdge(compose.START, "preprocess")
	if err != nil {
		panic(err)
	}

	err = graph.AddEdge("preprocess", "model")
	if err != nil {
		panic(err)
	}

	err = graph.AddEdge("model", compose.END)
	if err != nil {
		panic(err)
	}

	compiled, err := graph.Compile(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Println("=== State Graph 示例 ===")
	fmt.Println("展示状态管理: 使用 Lambda 节点处理状态")

	state := &ConversationState{
		Messages: []*schema.Message{
			schema.SystemMessage("你是一个友好的助手，请简洁回答。"),
			schema.UserMessage("你好！"),
		},
		Turn: 0,
	}

	result, err := compiled.Invoke(ctx, state)
	if err != nil {
		panic(err)
	}

	fmt.Printf("回答: %s\n", result.Content)
	fmt.Printf("最终状态: Turn=%d\n", state.Turn)
	fmt.Println("=== 完成 ===")

	_ = preHandler
}
