package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
)

type QuestionRequest struct {
	Question string `json:"question"`
	Reason   string `json:"reason"`
}

type QuestionResponse struct {
	Answer string `json:"answer"`
}

func main() {
	ctx := context.Background()
	model := mustCreateModel(ctx)

	askTool := createAskTool()

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "smart_assistant",
		Description: "智能助手，会主动追问以收集完整信息",
		Instruction: `你是一个智能助手。
当用户的请求信息不完整时，使用 ask_user 工具进行追问。
一次只问一个问题，确保问题清晰明确。
收集到足够信息后，再给出完整的回答或执行操作。`,
		Model: model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{askTool},
			},
		},
	})
	if err != nil {
		panic(err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	fmt.Println("=== 追问模式示例 ===")
	fmt.Println("智能识别信息缺失，通过多轮追问收集用户需求。")
	fmt.Println()

	runFollowUp(ctx, runner)
}

func createAskTool() tool.InvokableTool {
	t, err := utils.InferTool(
		"ask_user",
		"向用户提问以获取更多信息",
		func(ctx context.Context, input *struct {
			Question string `json:"question" jsonschema:"required" jsonschema_description:"要问用户的问题"`
			Reason   string `json:"reason" jsonschema:"required" jsonschema_description:"为什么需要这个信息"`
		}) (string, error) {
			wasInterrupted, _, _ := tool.GetInterruptState[any](ctx)
			if !wasInterrupted {
				return "", tool.Interrupt(ctx, QuestionRequest{
					Question: input.Question,
					Reason:   input.Reason,
				})
			}

			isResumeTarget, hasData, resumeData := tool.GetResumeContext[QuestionResponse](ctx)
			if isResumeTarget && hasData {
				return resumeData.Answer, nil
			}

			return "等待用户回答", nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

func runFollowUp(ctx context.Context, runner *adk.Runner) {
	query := "帮我订一张机票"
	fmt.Printf("用户请求: %s\n\n", query)

	iter := runner.Query(ctx, query)

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			fmt.Printf("错误: %v\n", event.Err)
			break
		}

		if event.Action != nil && event.Action.Interrupted != nil {
			info := event.Action.Interrupted
			if len(info.InterruptContexts) > 0 {
				interruptID := info.InterruptContexts[0].ID

				var questionReq QuestionRequest
				if data, ok := info.Data.(map[string]any); ok {
					questionReq.Question = fmt.Sprint(data["question"])
					questionReq.Reason = fmt.Sprint(data["reason"])
				} else if data, ok := info.Data.(QuestionRequest); ok {
					questionReq = data
				}

				fmt.Printf("\n助手需要更多信息:\n")
				if questionReq.Reason != "" {
					fmt.Printf("原因: %s\n", questionReq.Reason)
				}
				fmt.Printf("问题: %s\n", questionReq.Question)
				fmt.Print("请回答: ")

				var answer string
				fmt.Scanln(&answer)

				resumeData := QuestionResponse{Answer: answer}
				fmt.Printf("用户回答: %s\n\n", answer)

				resumeIter, err := runner.Resume(ctx, interruptID,
					adk.WithSessionValues(map[string]any{"answer": resumeData}),
				)
				if err != nil {
					fmt.Printf("恢复失败: %v\n", err)
					break
				}
				iter = resumeIter
				continue
			}
		}

		if event.Output != nil {
			if msg, _, err := adk.GetMessage(event); err == nil && msg != nil {
				if msg.Content != "" {
					fmt.Printf("助手: %s\n", msg.Content)
				}
			}
		}
	}
}

func mustCreateModel(ctx context.Context) *openai.ChatModel {
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
