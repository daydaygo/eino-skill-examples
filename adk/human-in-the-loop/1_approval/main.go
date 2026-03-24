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

type DeleteFileInput struct {
	FilePath string `json:"file_path" jsonschema:"required" jsonschema_description:"要删除的文件路径"`
}

type ApprovalRequest struct {
	ToolName    string `json:"tool_name"`
	Description string `json:"description"`
	FilePath    string `json:"file_path"`
}

type ApprovalResponse struct {
	Approved bool `json:"approved"`
}

func main() {
	ctx := context.Background()

	model := mustCreateModel(ctx)

	deleteFileTool := createDeleteFileTool()

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "file_manager",
		Description: "文件管理助手，可以删除文件",
		Instruction: `你是一个文件管理助手。
当用户请求删除文件时，使用 delete_file 工具。
工具会自动请求用户确认。`,
		Model: model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{deleteFileTool},
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

	fmt.Println("=== 审批模式示例 ===")
	fmt.Println("这个示例演示如何在敏感操作前请求人工确认。")
	fmt.Println()

	runWithApproval(ctx, runner)
}

func createDeleteFileTool() tool.InvokableTool {
	t, err := utils.InferTool(
		"delete_file",
		"删除指定的文件（需要用户确认）",
		func(ctx context.Context, input *DeleteFileInput) (string, error) {
			wasInterrupted, _, _ := tool.GetInterruptState[any](ctx)
			if !wasInterrupted {
				approvalReq := ApprovalRequest{
					ToolName:    "delete_file",
					Description: "删除文件",
					FilePath:    input.FilePath,
				}
				return "", tool.StatefulInterrupt(ctx, approvalReq, input)
			}

			isResumeTarget, hasData, resumeData := tool.GetResumeContext[ApprovalResponse](ctx)
			if isResumeTarget && hasData {
				if resumeData.Approved {
					return fmt.Sprintf("文件 %s 已成功删除", input.FilePath), nil
				}
				return "操作已取消，文件未被删除", nil
			}

			return "等待用户确认", nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

func runWithApproval(ctx context.Context, runner *adk.Runner) {
	query := "请删除 /tmp/test.log 文件"
	fmt.Printf("用户请求: %s\n\n", query)

	iter := runner.Query(ctx, query)

	var interruptID string
	var approvalReq ApprovalRequest

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
					approvalReq = ApprovalRequest{
						ToolName:    fmt.Sprint(data["tool_name"]),
						Description: fmt.Sprint(data["description"]),
						FilePath:    fmt.Sprint(data["file_path"]),
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
		fmt.Printf("\n=== 需要审批 ===\n")
		fmt.Printf("工具: %s\n", approvalReq.ToolName)
		fmt.Printf("操作: %s\n", approvalReq.Description)
		fmt.Printf("文件: %s\n", approvalReq.FilePath)
		fmt.Printf("\n是否批准此操作？(y/n): ")

		var response string
		fmt.Scanln(&response)

		approved := response == "y" || response == "Y"
		fmt.Printf("\n用户选择: %v\n", approved)

		resumeData := ApprovalResponse{Approved: approved}

		resumeIter, err := runner.Resume(ctx, interruptID,
			adk.WithSessionValues(map[string]any{"approval": resumeData}),
		)
		if err != nil {
			fmt.Printf("恢复失败: %v\n", err)
			return
		}

		fmt.Println("\n=== 继续执行 ===")
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
