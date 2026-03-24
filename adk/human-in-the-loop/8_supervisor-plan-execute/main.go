package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/adk/prebuilt/supervisor"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
)

type ApprovalRequest struct {
	TaskName string `json:"task_name"`
	Details  string `json:"details"`
}

type ApprovalResponse struct {
	Approved bool `json:"approved"`
}

func main() {
	ctx := context.Background()
	model := mustCreateModel(ctx)

	approvalTool := createApprovalTool()

	taskAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "task_agent",
		Description: "任务执行者",
		Instruction: `你是任务执行者。
执行分配给你的具体任务。
执行重要任务前使用 request_approval 工具请求用户确认。`,
		Model: model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{approvalTool},
			},
		},
	})
	if err != nil {
		panic(err)
	}

	planner, err := planexecute.NewPlanner(ctx, &planexecute.PlannerConfig{
		ToolCallingChatModel: model,
	})
	if err != nil {
		panic(err)
	}

	executor, err := planexecute.NewExecutor(ctx, &planexecute.ExecutorConfig{
		Model: model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{approvalTool},
			},
		},
	})
	if err != nil {
		panic(err)
	}

	_, err = adk.SetSubAgents(ctx, executor, []adk.Agent{taskAgent})
	if err != nil {
		panic(err)
	}

	replanner, err := planexecute.NewReplanner(ctx, &planexecute.ReplannerConfig{
		ChatModel: model,
	})
	if err != nil {
		panic(err)
	}

	planExecute, err := planexecute.New(ctx, &planexecute.Config{
		Planner:   planner,
		Executor:  executor,
		Replanner: replanner,
	})
	if err != nil {
		panic(err)
	}

	topSupervisor, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "top_supervisor",
		Description: "顶层协调者",
		Instruction: `你是顶层协调者。
根据用户需求分配任务：
- 复杂任务需要规划：plan_execute_agent
直接与用户交互，收集需求。`,
		Model: model,
	})
	if err != nil {
		panic(err)
	}

	sup, err := supervisor.New(ctx, &supervisor.Config{
		Supervisor: topSupervisor,
		SubAgents:  []adk.Agent{planExecute},
	})
	if err != nil {
		panic(err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           sup,
		EnableStreaming: true,
	})

	fmt.Println("=== 嵌套多 Agent 示例 ===")
	fmt.Println("Supervisor 嵌套 Plan-Execute-Replan 子 Agent。")
	fmt.Println()

	runNestedAgents(ctx, runner)
}

func createApprovalTool() tool.InvokableTool {
	t, err := utils.InferTool(
		"request_approval",
		"请求用户审批",
		func(ctx context.Context, input *ApprovalRequest) (string, error) {
			wasInterrupted, _, _ := tool.GetInterruptState[any](ctx)
			if !wasInterrupted {
				return "", tool.StatefulInterrupt(ctx, input, input)
			}

			isResumeTarget, hasData, resumeData := tool.GetResumeContext[ApprovalResponse](ctx)
			if isResumeTarget && hasData {
				if resumeData.Approved {
					return fmt.Sprintf("任务 '%s' 已批准执行", input.TaskName), nil
				}
				return fmt.Sprintf("任务 '%s' 已被拒绝", input.TaskName), nil
			}

			return "等待用户审批", nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

func runNestedAgents(ctx context.Context, runner *adk.Runner) {
	query := "帮我完成服务器迁移任务"
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

				var approvalReq ApprovalRequest
				if data, ok := info.Data.(map[string]any); ok {
					approvalReq.TaskName = fmt.Sprint(data["task_name"])
					approvalReq.Details = fmt.Sprint(data["details"])
				}

				fmt.Println("\n=== 任务审批 ===")
				fmt.Printf("任务: %s\n", approvalReq.TaskName)
				fmt.Printf("详情: %s\n", approvalReq.Details)
				fmt.Print("\n是否批准执行？(y/n): ")

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

		if event.Action != nil && event.Action.TransferToAgent != nil {
			fmt.Printf("\n[任务转移至: %s]\n", event.Action.TransferToAgent.DestAgentName)
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
