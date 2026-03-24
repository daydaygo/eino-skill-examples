package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/adk/prebuilt/supervisor"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
)

func main() {
	ctx := context.Background()

	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	model, _ := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: baseURL,
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		Model:   os.Getenv("OPENAI_MODEL"),
	})

	readExcelTool, _ := utils.InferTool(
		"read_excel",
		"Read and parse Excel file",
		func(ctx context.Context, input *struct {
			FilePath string `json:"file_path" jsonschema:"required,description=Path to the Excel file"`
			Sheet    string `json:"sheet" jsonschema:"optional,description=Sheet name to read"`
		}) (string, error) {
			return fmt.Sprintf(`Excel 文件读取成功
文件: %s
工作表: %s
行数: 50
列: [日期, 产品, 销量, 收入, 成本, 利润]`, input.FilePath, input.Sheet), nil
		},
	)

	filterDataTool, _ := utils.InferTool(
		"filter_data",
		"Filter data based on conditions",
		func(ctx context.Context, input *struct {
			Column   string `json:"column" jsonschema:"required,description=Column to filter on"`
			Operator string `json:"operator" jsonschema:"required,description=Filter operator: eq, gt, lt, contains"`
			Value    string `json:"value" jsonschema:"required,description=Value to compare"`
		}) (string, error) {
			return fmt.Sprintf("数据筛选完成\n条件: %s %s %s\n结果: 15 条记录", input.Column, input.Operator, input.Value), nil
		},
	)

	aggregateTool, _ := utils.InferTool(
		"aggregate",
		"Aggregate data with statistics",
		func(ctx context.Context, input *struct {
			Column      string `json:"column" jsonschema:"required,description=Column to aggregate"`
			AggFunction string `json:"agg_function" jsonschema:"required,description=Aggregation function: sum, avg, min, max, count"`
			GroupBy     string `json:"group_by" jsonschema:"optional,description=Column to group by"`
		}) (string, error) {
			if input.GroupBy != "" {
				return fmt.Sprintf(`分组聚合结果:
分组列: %s
聚合列: %s
函数: %s

| 分组值 | 结果 |
|--------|------|
| A      | 1500 |
| B      | 2300 |
| C      | 1800 |`, input.GroupBy, input.Column, input.AggFunction), nil
			}
			return fmt.Sprintf("聚合结果: %s(%s) = 5600", input.AggFunction, input.Column), nil
		},
	)

	createChartTool, _ := utils.InferTool(
		"create_chart",
		"Create visualization chart",
		func(ctx context.Context, input *struct {
			ChartType string   `json:"chart_type" jsonschema:"required,description=Chart type: bar, line, pie, scatter"`
			XColumn   string   `json:"x_column" jsonschema:"required,description=Column for X axis"`
			YColumns  []string `json:"y_columns" jsonschema:"required,description=Columns for Y axis"`
			Title     string   `json:"title" jsonschema:"optional,description=Chart title"`
		}) (string, error) {
			return fmt.Sprintf(`图表创建成功
类型: %s
X轴: %s
Y轴: %v
标题: %s

[图表预览区域]`, input.ChartType, input.XColumn, input.YColumns, input.Title), nil
		},
	)

	exportReportTool, _ := utils.InferTool(
		"export_report",
		"Export analysis report",
		func(ctx context.Context, input *struct {
			Format     string `json:"format" jsonschema:"required,description=Export format: pdf, html, markdown"`
			Title      string `json:"title" jsonschema:"required,description=Report title"`
			Content    string `json:"content" jsonschema:"required,description=Report content"`
			OutputPath string `json:"output_path" jsonschema:"required,description=Output file path"`
		}) (string, error) {
			return fmt.Sprintf("报告已导出: %s\n格式: %s\n标题: %s", input.OutputPath, input.Format, input.Title), nil
		},
	)

	planner, _ := planexecute.NewPlanner(ctx, &planexecute.PlannerConfig{
		ChatModelWithFormattedOutput: model,
	})

	executor, _ := planexecute.NewExecutor(ctx, &planexecute.ExecutorConfig{
		Model: model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{readExcelTool, filterDataTool, aggregateTool, createChartTool},
			},
		},
	})

	replanner, _ := planexecute.NewReplanner(ctx, &planexecute.ReplannerConfig{
		ChatModel: model,
	})

	planExecuteAgent, _ := planexecute.New(ctx, &planexecute.Config{
		Planner:   planner,
		Executor:  executor,
		Replanner: replanner,
	})

	reporter, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "reporter",
		Description: "报告者 - 生成最终分析报告",
		Instruction: `你是数据分析报告专家。整合分析结果生成专业报告。

报告结构:
1. 数据概述
2. 分析方法
3. 关键发现
4. 可视化图表
5. 结论建议`,
		Model: model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{exportReportTool},
			},
		},
	})

	excelSupervisor, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "excel_analysis_agent",
		Description: "Excel 分析助手 - 完整的数据分析流程",
		Instruction: `你是 Excel 数据分析助手，集成 Plan-Execute-Replan 模式。

工作流程:
1. planner: 分析需求，制定计划
2. executor: 执行数据分析
3. replanner: 评估结果，决定是否调整
4. reporter: 生成分析报告

自动迭代直到分析完成，然后生成报告。`,
		Model: model,
	})

	agent, _ := supervisor.New(ctx, &supervisor.Config{
		Supervisor: excelSupervisor,
		SubAgents:  []adk.Agent{planExecuteAgent, reporter},
	})

	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})

	fmt.Println("=== Excel Agent (ADK 集成版) 示例 ===")
	fmt.Println("这是完整的 Excel 分析系统，集成多种模式:")
	fmt.Println()
	fmt.Println("架构:")
	fmt.Println("  excel_analysis_agent")
	fmt.Println("  ├── plan_execute_replan (Plan-Execute-Replan 模式)")
	fmt.Println("  │   ├── planner (规划者)")
	fmt.Println("  │   ├── executor (执行者)")
	fmt.Println("  │   └── replanner (重规划者)")
	fmt.Println("  └── reporter (报告者)")
	fmt.Println()

	query := `分析 sales_data.xlsx 文件:
1. 读取销售数据
2. 按产品分类统计总收入
3. 筛选出利润大于 1000 的记录
4. 创建收入趋势图表
5. 生成分析报告`

	fmt.Printf("用户需求:\n%s\n\n", query)

	iter := runner.Query(ctx, query)

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			fmt.Printf("[错误] %v\n", event.Err)
			continue
		}

		if event.Action != nil && event.Action.TransferToAgent != nil {
			dest := event.Action.TransferToAgent.DestAgentName
			phase := ""
			switch dest {
			case "planner":
				phase = "规划阶段"
			case "executor":
				phase = "执行阶段"
			case "replanner":
				phase = "评估阶段"
			case "reporter":
				phase = "报告阶段"
			case "plan_execute_replan":
				phase = "Plan-Execute-Replan"
			}
			if phase != "" {
				fmt.Printf("\n[%s]\n", phase)
			}
		}

		if msg, _, err := adk.GetMessage(event); err == nil && msg.Content != "" {
			content := strings.TrimSpace(msg.Content)
			if content != "" {
				if strings.HasPrefix(content, "##") || strings.HasPrefix(content, "**") {
					fmt.Println()
				}
				fmt.Println(content)
			}
		}
	}
	fmt.Println("\n\n=== 分析完成 ===")
}
