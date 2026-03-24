package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
)

type DeployInput struct {
	Environment string `json:"environment" jsonschema:"required" jsonschema_description:"部署环境"`
	Version     string `json:"version" jsonschema:"required" jsonschema_description:"版本号"`
}

type ReviewResponse struct {
	Approved bool   `json:"approved"`
	Comment  string `json:"comment,omitempty"`
}

func main() {
	ctx := context.Background()
	model := mustCreateModel(ctx)

	deployTool := createDeployTool()

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
				Tools: []tool.BaseTool{deployTool},
			},
		},
	})
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

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           planExecute,
		EnableStreaming: true,
	})

	fmt.Println("=== Plan-Execute-Replan + 参数审核示例 ===")
	fmt.Println("计划执行模式结合参数审核机制。")
	fmt.Println()

	runPlanExecute(ctx, runner)
}

func createDeployTool() tool.InvokableTool {
	t, err := utils.InferTool(
		"deploy",
		"执行部署（需要用户审核参数）",
		func(ctx context.Context, input *DeployInput) (string, error) {
			wasInterrupted, _, _ := tool.GetInterruptState[any](ctx)
			if !wasInterrupted {
				return "", tool.StatefulInterrupt(ctx, input, input)
			}

			isResumeTarget, hasData, resumeData := tool.GetResumeContext[ReviewResponse](ctx)
			if isResumeTarget && hasData {
				if resumeData.Approved {
					return fmt.Sprintf("部署成功: 环境=%s, 版本=%s", input.Environment, input.Version), nil
				}
				return fmt.Sprintf("部署已取消: %s", resumeData.Comment), nil
			}

			return "等待用户审核", nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

func runPlanExecute(ctx context.Context, runner *adk.Runner) {
	query := "请部署 v2.0.1 版本到测试环境"
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

				var deployReq DeployInput
				if data, ok := info.Data.(map[string]any); ok {
					deployReq.Environment = fmt.Sprint(data["environment"])
					deployReq.Version = fmt.Sprint(data["version"])
				}

				fmt.Println("\n=== 部署参数审核 ===")
				fmt.Printf("环境: %s\n", deployReq.Environment)
				fmt.Printf("版本: %s\n", deployReq.Version)
				fmt.Print("\n是否批准部署？(y/n): ")

				var confirm string
				fmt.Scanln(&confirm)

				approved := confirm == "y" || confirm == "Y"
				var comment string
				if !approved {
					fmt.Print("请输入拒绝原因: ")
					fmt.Scanln(&comment)
				}

				fmt.Println()

				resumeData := ReviewResponse{
					Approved: approved,
					Comment:  comment,
				}

				resumeIter, err := runner.Resume(ctx, interruptID,
					adk.WithSessionValues(map[string]any{"review": resumeData}),
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
