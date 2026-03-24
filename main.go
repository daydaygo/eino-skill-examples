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

type WeatherInput struct {
	City string `json:"city" jsonschema:"required" jsonschema_description:"城市名称"`
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
			return fmt.Sprintf("%s 今天天气晴朗，温度 25°C", input.City), nil
		},
	)
	if err != nil {
		panic(err)
	}

	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: model,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: []tool.BaseTool{weatherTool},
		},
		MessageModifier: func(ctx context.Context, input []*schema.Message) []*schema.Message {
			res := make([]*schema.Message, 0, len(input)+1)
			res = append(res, schema.SystemMessage("你是一个天气助手。"))
			res = append(res, input...)
			return res
		},
		MaxStep: 20,
	})
	if err != nil {
		panic(err)
	}

	sr, err := agent.Stream(ctx, []*schema.Message{
		schema.UserMessage("北京今天天气怎么样？"),
	})
	if err != nil {
		panic(err)
	}
	defer sr.Close()

	fmt.Println("=== Eino ReAct Agent 验证测试 ===")
	for {
		msg, err := sr.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			panic(err)
		}
		fmt.Print(msg.Content)
	}
	fmt.Println("\n=== 验证成功 ===")
}
