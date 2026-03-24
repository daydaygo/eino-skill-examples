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

type FeedbackRequest struct {
	Content       string `json:"content"`
	Iteration     int    `json:"iteration"`
	MaxIterations int    `json:"max_iterations"`
}

type FeedbackResponse struct {
	Approved bool   `json:"approved"`
	Feedback string `json:"feedback,omitempty"`
}

func main() {
	ctx := context.Background()
	model := mustCreateModel(ctx)

	writerAgent := createWriterAgent(ctx, model)
	reviewerAgent := createReviewerAgent(ctx, model)

	loopAgent, err := adk.NewLoopAgent(ctx, &adk.LoopAgentConfig{
		Name:          "content_creation_loop",
		Description:   "迭代创作内容直到满意",
		SubAgents:     []adk.Agent{writerAgent, reviewerAgent},
		MaxIterations: 3,
	})
	if err != nil {
		panic(err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           loopAgent,
		EnableStreaming: true,
	})

	fmt.Println("=== 反馈循环模式示例 ===")
	fmt.Println("Writer 生成内容，Reviewer 收集人工反馈，迭代优化。")
	fmt.Println()

	runFeedbackLoop(ctx, runner)
}

func createWriterAgent(ctx context.Context, model *openai.ChatModel) *adk.ChatModelAgent {
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "writer",
		Description: "内容创作者",
		Instruction: `你是一个专业的内容创作者。
根据用户的主题创作高质量的内容。
如果收到反馈意见，请根据反馈改进内容。
直接输出创作的内容，不要有多余的说明。`,
		Model: model,
	})
	if err != nil {
		panic(err)
	}
	return agent
}

func createReviewerAgent(ctx context.Context, model *openai.ChatModel) *adk.ChatModelAgent {
	submitTool := createSubmitTool()

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "reviewer",
		Description: "内容审核者，收集用户反馈",
		Instruction: `你是一个内容审核者。
你的职责是展示 Writer 创作的内容，并收集用户的反馈。
使用 submit_feedback 工具来提交用户反馈。
如果用户满意，设置 approved=true。
如果用户有修改意见，设置 approved=false 并在 feedback 中说明。`,
		Model: model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{submitTool},
			},
		},
	})
	if err != nil {
		panic(err)
	}
	return agent
}

func createSubmitTool() tool.InvokableTool {
	t, err := utils.InferTool(
		"submit_feedback",
		"提交用户反馈",
		func(ctx context.Context, input *struct {
			Approved bool   `json:"approved" jsonschema_description:"是否满意"`
			Feedback string `json:"feedback" jsonschema_description:"修改意见（如果不满意）"`
		}) (string, error) {
			wasInterrupted, _, _ := tool.GetInterruptState[any](ctx)
			if !wasInterrupted {
				return "", tool.Interrupt(ctx, map[string]any{
					"message": "请审核内容并提供反馈",
				})
			}

			isResumeTarget, hasData, resumeData := tool.GetResumeContext[FeedbackResponse](ctx)
			if isResumeTarget && hasData {
				if resumeData.Approved {
					return "用户已满意，内容通过审核", nil
				}
				return fmt.Sprintf("用户反馈: %s", resumeData.Feedback), nil
			}

			return "等待用户反馈", nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

func runFeedbackLoop(ctx context.Context, runner *adk.Runner) {
	query := "写一篇关于人工智能未来发展的短文"
	fmt.Printf("用户请求: %s\n\n", query)

	currentContent := ""
	iteration := 0

	iter := runner.Query(ctx, query)

processLoop:
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

				fmt.Printf("\n=== 第 %d 次迭代 ===\n", iteration+1)
				fmt.Printf("当前内容:\n%s\n", currentContent)
				fmt.Println()

				fmt.Print("是否满意？(y/n): ")
				var approve string
				fmt.Scanln(&approve)

				approved := approve == "y" || approve == "Y"

				var feedback string
				if !approved {
					fmt.Print("请输入修改意见: ")
					fmt.Scanln(&feedback)
				}

				iteration++

				resumeData := FeedbackResponse{
					Approved: approved,
					Feedback: feedback,
				}

				resumeIter, err := runner.Resume(ctx, interruptID,
					adk.WithSessionValues(map[string]any{"feedback": resumeData}),
				)
				if err != nil {
					fmt.Printf("恢复失败: %v\n", err)
					break processLoop
				}
				iter = resumeIter
				continue
			}
		}

		if event.Output != nil {
			if msg, _, err := adk.GetMessage(event); err == nil && msg != nil {
				content := msg.Content
				if content != "" {
					currentContent = content
				}
			}
		}
	}

	fmt.Printf("\n=== 最终内容 ===\n%s\n", currentContent)
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
