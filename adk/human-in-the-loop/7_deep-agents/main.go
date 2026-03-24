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
}

type QuestionResponse struct {
	Answer string `json:"answer"`
}

func main() {
	ctx := context.Background()
	model := mustCreateModel(ctx)

	askTool := createAskTool()

	coordinatorAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "coordinator",
		Description: "任务协调者",
		Instruction: `你是任务协调者。
接收用户的任务请求，拆解为子任务。
协调 research_agent 和 writer_agent 完成任务。
当信息不足时使用 ask_user 工具追问用户。`,
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

	researchAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "research_agent",
		Description: "研究分析专家",
		Instruction: "你是研究分析专家，负责收集和分析信息。",
		Model:       model,
	})
	if err != nil {
		panic(err)
	}

	writerAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "writer_agent",
		Description: "内容撰写专家",
		Instruction: "你是内容撰写专家，负责根据研究结果撰写内容。",
		Model:       model,
	})
	if err != nil {
		panic(err)
	}

	_, err = adk.SetSubAgents(ctx, coordinatorAgent, []adk.Agent{researchAgent, writerAgent})
	if err != nil {
		panic(err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           coordinatorAgent,
		EnableStreaming: true,
	})

	fmt.Println("=== Deep Agents + 追问模式示例 ===")
	fmt.Println("Deep Agents 模式结合追问机制。")
	fmt.Println()

	runDeepAgents(ctx, runner)
}

func createAskTool() tool.InvokableTool {
	t, err := utils.InferTool(
		"ask_user",
		"向用户提问",
		func(ctx context.Context, input *QuestionRequest) (string, error) {
			wasInterrupted, _, _ := tool.GetInterruptState[any](ctx)
			if !wasInterrupted {
				return "", tool.Interrupt(ctx, input)
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

func runDeepAgents(ctx context.Context, runner *adk.Runner) {
	query := "帮我写一份报告"
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
				} else if data, ok := info.Data.(QuestionRequest); ok {
					questionReq = data
				}

				fmt.Printf("\n助手追问: %s\n", questionReq.Question)
				fmt.Print("请回答: ")

				var answer string
				fmt.Scanln(&answer)
				fmt.Println()

				resumeData := QuestionResponse{Answer: answer}

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

		if event.Action != nil && event.Action.TransferToAgent != nil {
			dest := event.Action.TransferToAgent.DestAgentName
			fmt.Printf("\n[任务转移至: %s]\n", dest)
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
