package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/supervisor"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
)

type TransferRequest struct {
	From   string  `json:"from"`
	To     string  `json:"to"`
	Amount float64 `json:"amount"`
}

type ApprovalResponse struct {
	Approved bool `json:"approved"`
}

func main() {
	ctx := context.Background()
	model := mustCreateModel(ctx)

	transferTool := createTransferTool()

	bankerAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "banker",
		Description: "银行转账专家",
		Instruction: `你是一个银行转账专家。
帮助用户进行转账操作。
使用 transfer_money 工具来执行转账。`,
		Model: model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{transferTool},
			},
		},
	})
	if err != nil {
		panic(err)
	}

	infoAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "info_agent",
		Description: "账户信息查询专家",
		Instruction: "你是一个账户信息查询专家，帮助用户查询余额和交易记录。",
		Model:       model,
	})
	if err != nil {
		panic(err)
	}

	supervisorAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "bank_supervisor",
		Description: "银行服务总管",
		Instruction: `你是银行服务的总管。
根据用户需求将任务分配给合适的专家：
- 转账相关：banker
- 查询相关：info_agent
敏感操作会自动请求用户确认。`,
		Model: model,
	})
	if err != nil {
		panic(err)
	}

	sup, err := supervisor.New(ctx, &supervisor.Config{
		Supervisor: supervisorAgent,
		SubAgents:  []adk.Agent{bankerAgent, infoAgent},
	})
	if err != nil {
		panic(err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           sup,
		EnableStreaming: true,
	})

	fmt.Println("=== Supervisor + 审批模式示例 ===")
	fmt.Println("Supervisor 多 Agent 模式结合审批机制。")
	fmt.Println()

	runSupervisorWithApproval(ctx, runner)
}

func createTransferTool() tool.InvokableTool {
	t, err := utils.InferTool(
		"transfer_money",
		"执行银行转账（需要用户确认）",
		func(ctx context.Context, input *TransferRequest) (string, error) {
			wasInterrupted, _, _ := tool.GetInterruptState[any](ctx)
			if !wasInterrupted {
				return "", tool.StatefulInterrupt(ctx, input, input)
			}

			isResumeTarget, hasData, resumeData := tool.GetResumeContext[ApprovalResponse](ctx)
			if isResumeTarget && hasData {
				if resumeData.Approved {
					return fmt.Sprintf("转账成功：从 %s 转出 %.2f 元到 %s", input.From, input.Amount, input.To), nil
				}
				return "转账已取消", nil
			}

			return "等待用户确认", nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

func runSupervisorWithApproval(ctx context.Context, runner *adk.Runner) {
	query := "帮我从账户A转账500元到账户B"
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

				var transferReq TransferRequest
				if data, ok := info.Data.(map[string]any); ok {
					if from, ok := data["from"]; ok {
						transferReq.From = fmt.Sprint(from)
					}
					if to, ok := data["to"]; ok {
						transferReq.To = fmt.Sprint(to)
					}
					if amount, ok := data["amount"]; ok {
						fmt.Sscanf(fmt.Sprint(amount), "%f", &transferReq.Amount)
					}
				}

				fmt.Println("\n=== 转账确认 ===")
				fmt.Printf("转出账户: %s\n", transferReq.From)
				fmt.Printf("转入账户: %s\n", transferReq.To)
				fmt.Printf("转账金额: %.2f 元\n", transferReq.Amount)
				fmt.Print("\n是否确认转账？(y/n): ")

				var confirm string
				fmt.Scanln(&confirm)

				approved := confirm == "y" || confirm == "Y"
				fmt.Printf("用户选择: %v\n\n", approved)

				resumeData := ApprovalResponse{Approved: approved}

				resumeIter, err := runner.Resume(ctx, interruptID,
					adk.WithSessionValues(map[string]any{"approval": resumeData}),
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
