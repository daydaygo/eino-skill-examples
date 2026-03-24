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

type RestaurantSearchInput struct {
	Cuisine    string `json:"cuisine" jsonschema:"required" jsonschema_description:"菜系类型，如川菜、粤菜、日料等"`
	Location   string `json:"location" jsonschema:"required" jsonschema_description:"位置或区域"`
	PriceRange string `json:"price_range" jsonschema_description:"价格区间，如低档、中档、高档"`
}

type RestaurantDetailInput struct {
	Name string `json:"name" jsonschema:"required" jsonschema_description:"餐厅名称"`
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
		"search_restaurants",
		"根据菜系、位置和价格区间搜索餐厅",
		func(ctx context.Context, input *RestaurantSearchInput) (string, error) {
			priceFilter := ""
			if input.PriceRange != "" {
				priceFilter = fmt.Sprintf("，价格区间：%s", input.PriceRange)
			}
			return fmt.Sprintf(
				"找到 %s 附近的 %s 餐厅%s：\n1. 川味轩 - 评分4.5，人均80元\n2. 蜀香楼 - 评分4.8，人均120元\n3. 满庭芳 - 评分4.6，人均95元",
				input.Location, input.Cuisine, priceFilter,
			), nil
		},
	)
	if err != nil {
		panic(err)
	}

	detailTool, err := utils.InferTool(
		"get_restaurant_detail",
		"获取餐厅详细信息，包括地址、电话、营业时间等",
		func(ctx context.Context, input *RestaurantDetailInput) (string, error) {
			restaurantDetails := map[string]string{
				"川味轩": "地址：朝阳区三里屯路19号\n电话：010-12345678\n营业时间：11:00-22:00\n特色菜：水煮鱼、麻婆豆腐、回锅肉",
				"蜀香楼": "地址：海淀区中关村大街1号\n电话：010-87654321\n营业时间：10:30-21:30\n特色菜：宫保鸡丁、酸菜鱼、毛血旺",
				"满庭芳": "地址：西城区金融街8号\n电话：010-11112222\n营业时间：11:00-22:00\n特色菜：东坡肉、龙井虾仁、西湖醋鱼",
			}
			if detail, ok := restaurantDetails[input.Name]; ok {
				return detail, nil
			}
			return fmt.Sprintf("未找到餐厅 %s 的详细信息", input.Name), nil
		},
	)
	if err != nil {
		panic(err)
	}

	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: model,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: []tool.BaseTool{searchTool, detailTool},
		},
		MessageModifier: func(ctx context.Context, input []*schema.Message) []*schema.Message {
			res := make([]*schema.Message, 0, len(input)+1)
			res = append(res, schema.SystemMessage(`你是一个专业的餐厅推荐助手。
你的任务是帮助用户找到合适的餐厅，并提供详细信息。
请根据用户的需求，先搜索餐厅，然后提供详细信息。
回答时要礼貌、专业，并给出具体的推荐理由。`))
			res = append(res, input...)
			return res
		},
		MaxStep: 20,
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("=== 餐厅推荐助手 ===")
	fmt.Println("示例：帮我找一家朝阳区的川菜餐厅")
	fmt.Println()

	sr, err := agent.Stream(ctx, []*schema.Message{
		schema.UserMessage("我想在朝阳区找一家川菜餐厅，请推荐一家好评的并告诉我详细信息"),
	})
	if err != nil {
		panic(err)
	}
	defer sr.Close()

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
	fmt.Println()
}
