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

type MemoryStore interface {
	Get(sessionID string) []*schema.Message
	Set(sessionID string, messages []*schema.Message)
	Clear(sessionID string)
}

type InMemoryStore struct {
	sessions map[string][]*schema.Message
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		sessions: make(map[string][]*schema.Message),
	}
}

func (s *InMemoryStore) Get(sessionID string) []*schema.Message {
	return s.sessions[sessionID]
}

func (s *InMemoryStore) Set(sessionID string, messages []*schema.Message) {
	s.sessions[sessionID] = messages
}

func (s *InMemoryStore) Clear(sessionID string) {
	delete(s.sessions, sessionID)
}

type WeatherInput struct {
	City string `json:"city" jsonschema:"required" jsonschema_description:"城市名称"`
}

type TimeInput struct {
	Timezone string `json:"timezone" jsonschema:"description:"时区，如 Asia/Shanghai"`
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

	weatherTool, err := utils.InferTool(
		"get_weather",
		"获取指定城市的天气信息",
		func(ctx context.Context, input *WeatherInput) (string, error) {
			weatherData := map[string]string{
				"北京": "晴，温度 18°C，空气质量良好",
				"上海": "多云，温度 22°C，有轻微雾霾",
				"广州": "小雨，温度 26°C，湿度较高",
				"深圳": "阴天，温度 25°C",
			}
			if weather, ok := weatherData[input.City]; ok {
				return weather, nil
			}
			return fmt.Sprintf("%s 天气：晴，温度 20°C", input.City), nil
		},
	)
	if err != nil {
		panic(err)
	}

	timeTool, err := utils.InferTool(
		"get_current_time",
		"获取当前时间",
		func(ctx context.Context, input *TimeInput) (string, error) {
			return "当前时间：2024年3月24日 14:30:00 (Asia/Shanghai)", nil
		},
	)
	if err != nil {
		panic(err)
	}

	store := NewInMemoryStore()
	sessionID := "session_001"

	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: model,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: []tool.BaseTool{weatherTool, timeTool},
		},
		MessageModifier: func(ctx context.Context, input []*schema.Message) []*schema.Message {
			res := make([]*schema.Message, 0, len(input)+1)
			res = append(res, schema.SystemMessage(`你是一个智能助手，具有短期记忆能力。
你可以记住之前对话的内容，并在后续对话中引用之前的信息。
请保持友好的对话风格。`))
			res = append(res, input...)
			return res
		},
		MessageRewriter: func(ctx context.Context, input []*schema.Message) []*schema.Message {
			if len(input) > 20 {
				summarized := schema.SystemMessage("[之前对话的摘要：用户询问了多个城市的天气信息]")
				return append([]*schema.Message{summarized}, input[len(input)-10:]...)
			}
			return input
		},
		MaxStep: 20,
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("=== 短期记忆对话示例 ===")
	fmt.Println("支持多轮对话，可以记住之前的对话内容")
	fmt.Println()

	conversations := []string{
		"今天北京的天气怎么样？",
		"那上海呢？",
		"刚才我问了哪两个城市的天气？",
	}

	for i, query := range conversations {
		fmt.Printf("--- 第 %d 轮对话 ---\n", i+1)
		fmt.Printf("用户: %s\n", query)

		history := store.Get(sessionID)
		messages := append(history, schema.UserMessage(query))

		sr, err := agent.Stream(ctx, messages)
		if err != nil {
			fmt.Printf("错误: %v\n", err)
			continue
		}

		fmt.Print("助手: ")
		var responseContent string
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
			responseContent += msg.Content
		}
		fmt.Println()

		store.Set(sessionID, append(messages, schema.AssistantMessage(responseContent, nil)))
		fmt.Println()
	}
}
