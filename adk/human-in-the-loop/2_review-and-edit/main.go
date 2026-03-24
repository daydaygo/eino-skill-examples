package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
)

type SendEmailInput struct {
	To      string `json:"to" jsonschema:"required" jsonschema_description:"收件人邮箱地址"`
	Subject string `json:"subject" jsonschema:"required" jsonschema_description:"邮件主题"`
	Body    string `json:"body" jsonschema:"required" jsonschema_description:"邮件正文"`
}

type ReviewRequest struct {
	ToolName string         `json:"tool_name"`
	Args     SendEmailInput `json:"args"`
}

type ReviewResponse struct {
	Action   string          `json:"action"`
	Modified *SendEmailInput `json:"modified,omitempty"`
}

func main() {
	ctx := context.Background()
	model := mustCreateModel(ctx)

	sendEmailTool := createSendEmailTool()

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "email_assistant",
		Description: "邮件助手，可以发送邮件",
		Instruction: `你是一个邮件助手。
当用户要求发送邮件时，使用 send_email 工具。
工具参数会展示给用户审核，用户可以批准、修改或拒绝。`,
		Model: model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{sendEmailTool},
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

	fmt.Println("=== 审核编辑模式示例 ===")
	fmt.Println("这个示例演示如何让用户审核和修改工具调用参数。")
	fmt.Println()

	runWithReview(ctx, runner)
}

func createSendEmailTool() tool.InvokableTool {
	t, err := utils.InferTool(
		"send_email",
		"发送邮件（需要用户审核）",
		func(ctx context.Context, input *SendEmailInput) (string, error) {
			wasInterrupted, _, _ := tool.GetInterruptState[any](ctx)
			if !wasInterrupted {
				reviewReq := ReviewRequest{
					ToolName: "send_email",
					Args:     *input,
				}
				return "", tool.StatefulInterrupt(ctx, reviewReq, input)
			}

			isResumeTarget, hasData, resumeData := tool.GetResumeContext[ReviewResponse](ctx)
			if isResumeTarget && hasData {
				switch resumeData.Action {
				case "approved":
					return fmt.Sprintf("邮件已发送至 %s", input.To), nil
				case "modified":
					if resumeData.Modified != nil {
						return fmt.Sprintf("邮件已发送至 %s（修改后）", resumeData.Modified.To), nil
					}
				case "rejected":
					return "邮件发送已取消", nil
				}
			}

			return "等待用户审核", nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

func runWithReview(ctx context.Context, runner *adk.Runner) {
	query := "给张三发送邮件，主题是项目进度，内容是项目已完成80%"
	fmt.Printf("用户请求: %s\n\n", query)

	iter := runner.Query(ctx, query)

	var interruptID string
	var reviewReq ReviewRequest

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
				interruptID = info.InterruptContexts[0].ID
				if data, ok := info.Data.(map[string]any); ok {
					if args, ok := data["args"].(map[string]any); ok {
						reviewReq = ReviewRequest{
							ToolName: fmt.Sprint(data["tool_name"]),
							Args: SendEmailInput{
								To:      fmt.Sprint(args["to"]),
								Subject: fmt.Sprint(args["subject"]),
								Body:    fmt.Sprint(args["body"]),
							},
						}
					}
				}
			}
			break processLoop
		}

		if event.Output != nil {
			if msg, _, err := adk.GetMessage(event); err == nil && msg != nil {
				fmt.Printf("助手: %s\n", msg.Content)
			}
		}
	}

	if interruptID != "" {
		fmt.Printf("\n=== 邮件审核 ===\n")
		fmt.Printf("收件人: %s\n", reviewReq.Args.To)
		fmt.Printf("主题: %s\n", reviewReq.Args.Subject)
		fmt.Printf("正文: %s\n", reviewReq.Args.Body)
		fmt.Println()
		fmt.Println("选项: (1)批准 (2)修改 (3)拒绝")
		fmt.Print("请选择: ")

		var choice string
		fmt.Scanln(&choice)

		var resumeData ReviewResponse
		switch choice {
		case "1":
			resumeData = ReviewResponse{Action: "approved"}
			fmt.Println("\n已批准发送")
		case "2":
			fmt.Print("请输入新的收件人（直接回车保持原值）: ")
			var newTo string
			fmt.Scanln(&newTo)
			if newTo == "" {
				newTo = reviewReq.Args.To
			}

			fmt.Print("请输入新的主题（直接回车保持原值）: ")
			var newSubject string
			fmt.Scanln(&newSubject)
			if newSubject == "" {
				newSubject = reviewReq.Args.Subject
			}

			fmt.Println("请输入新的正文（输入 END 结束，直接输入 END 保持原值）:")
			var bodyLines []string
			for {
				var line string
				fmt.Scanln(&line)
				if line == "END" {
					break
				}
				bodyLines = append(bodyLines, line)
			}
			newBody := reviewReq.Args.Body
			if len(bodyLines) > 0 {
				newBody = strings.Join(bodyLines, "\n")
			}

			resumeData = ReviewResponse{
				Action: "modified",
				Modified: &SendEmailInput{
					To:      newTo,
					Subject: newSubject,
					Body:    newBody,
				},
			}
			fmt.Println("\n已修改并批准发送")
		case "3":
			resumeData = ReviewResponse{Action: "rejected"}
			fmt.Println("\n已拒绝发送")
		default:
			fmt.Println("\n无效选择，已取消")
			return
		}

		resumeIter, err := runner.Resume(ctx, interruptID,
			adk.WithSessionValues(map[string]any{"review": resumeData}),
		)
		if err != nil {
			fmt.Printf("恢复失败: %v\n", err)
			return
		}

		fmt.Println("\n=== 执行结果 ===")
		for {
			event, ok := resumeIter.Next()
			if !ok {
				break
			}

			if event.Err != nil {
				fmt.Printf("错误: %v\n", event.Err)
				continue
			}

			if msg, _, err := adk.GetMessage(event); err == nil && msg != nil {
				fmt.Printf("助手: %s\n", msg.Content)
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
