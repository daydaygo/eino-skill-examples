package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
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
		"Read and parse an Excel file",
		func(ctx context.Context, input *struct {
			FilePath string `json:"file_path" jsonschema:"required,description=Path to the Excel file"`
		}) (string, error) {
			return fmt.Sprintf(`Excel 文件分析结果:
文件: %s
工作表: Sheet1, Sheet2
行数: 100
列数: 5

列名: [ID, 姓名, 部门, 薪资, 入职日期]
数据预览:
| ID | 姓名   | 部门   | 薪资   |
|----|--------|--------|--------|
| 1  | 张三   | 技术   | 15000  |
| 2  | 李四   | 销售   | 12000  |
| 3  | 王五   | 财务   | 13000  |`, input.FilePath), nil
		},
	)

	analyzeDataTool, _ := utils.InferTool(
		"analyze_data",
		"Analyze data and generate statistics",
		func(ctx context.Context, input *struct {
			AnalysisType string `json:"analysis_type" jsonschema:"required,description=Type of analysis: sum, avg, count, filter"`
			Column       string `json:"column" jsonschema:"optional,description=Column to analyze"`
			Condition    string `json:"condition" jsonschema:"optional,description=Filter condition"`
		}) (string, error) {
			switch input.AnalysisType {
			case "sum":
				return fmt.Sprintf("薪资总和: 40000"), nil
			case "avg":
				return fmt.Sprintf("平均薪资: 13333.33"), nil
			case "count":
				return fmt.Sprintf("记录数: 3"), nil
			case "filter":
				return fmt.Sprintf("筛选结果 (部门=技术): 1 条记录"), nil
			default:
				return fmt.Sprintf("分析类型: %s", input.AnalysisType), nil
			}
		},
	)

	createChartTool, _ := utils.InferTool(
		"create_chart",
		"Create a chart from data",
		func(ctx context.Context, input *struct {
			ChartType string   `json:"chart_type" jsonschema:"required,description=Type of chart: bar, line, pie"`
			Columns   []string `json:"columns" jsonschema:"required,description=Columns to include in chart"`
			Title     string   `json:"title" jsonschema:"optional,description=Chart title"`
		}) (string, error) {
			return fmt.Sprintf(`图表已创建:
类型: %s
列: %v
标题: %s

[图表渲染成功 - 显示薪资分布柱状图]`, input.ChartType, input.Columns, input.Title), nil
		},
	)

	exportDataTool, _ := utils.InferTool(
		"export_data",
		"Export data to a file",
		func(ctx context.Context, input *struct {
			Format     string `json:"format" jsonschema:"required,description=Export format: csv, json, xlsx"`
			OutputPath string `json:"output_path" jsonschema:"required,description=Output file path"`
			Data       string `json:"data" jsonschema:"optional,description=Data to export"`
		}) (string, error) {
			return fmt.Sprintf("数据已导出: %s\n格式: %s\n状态: 成功", input.OutputPath, input.Format), nil
		},
	)

	understander, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "understander",
		Description: "理解者 - 解析用户意图，理解 Excel 文件结构",
		Instruction: `你是一个 Excel 数据理解专家。你的职责是：
1. 理解用户的分析需求
2. 读取并分析 Excel 文件结构
3. 确定哪些列需要分析

首先使用 read_excel 工具了解文件结构，然后解释数据含义。`,
		Model: model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{readExcelTool},
			},
		},
	})

	analyzer, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "analyzer",
		Description: "分析者 - 执行数据分析",
		Instruction: `你是一个数据分析专家。你的职责是：
1. 根据用户需求执行数据分析
2. 使用 analyze_data 工具进行统计计算
3. 使用 create_chart 工具创建可视化图表

支持的统计类型:
- sum: 求和
- avg: 平均值
- count: 计数
- filter: 筛选`,
		Model: model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{analyzeDataTool, createChartTool},
			},
		},
	})

	reporter, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "reporter",
		Description: "报告者 - 生成分析报告",
		Instruction: `你是一个报告撰写专家。你的职责是：
1. 整理分析结果
2. 生成清晰的分析报告
3. 可以使用 export_data 工具导出数据

报告应包含:
- 数据概述
- 分析结果
- 可视化图表说明
- 结论和建议`,
		Model: model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{exportDataTool},
			},
		},
	})

	excelAgent, _ := deep.New(ctx, &deep.Config{
		Name:        "excel_agent",
		Description: "Excel 智能助手 - 深度分析和处理 Excel 文件",
		Instruction: `你是一个 Excel 智能助手。你有三个专家团队成员：

1. understander (理解者)
   - 读取 Excel 文件
   - 理解数据结构
   - 解析用户意图

2. analyzer (分析者)
   - 执行数据分析
   - 创建可视化图表
   - 统计计算

3. reporter (报告者)
   - 整理分析结果
   - 生成报告
   - 导出数据

根据用户需求，协调团队成员逐步完成任务：
理解 -> 分析 -> 报告`,
		ChatModel: model,
		SubAgents: []adk.Agent{understander, analyzer, reporter},
	})

	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: excelAgent})

	fmt.Println("=== Deep Agent (Excel Agent) 示例 ===")
	fmt.Println("这是一个智能 Excel 助手，采用分步骤处理模式:")
	fmt.Println("  1. understander: 理解数据和用户意图")
	fmt.Println("  2. analyzer: 执行数据分析")
	fmt.Println("  3. reporter: 生成报告")
	fmt.Println()

	query := `请分析 data.xlsx 文件:
1. 统计各部门的薪资总和
2. 计算平均薪资
3. 创建薪资分布图表
4. 生成分析报告`

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
			case "understander":
				phase = "理解阶段"
			case "analyzer":
				phase = "分析阶段"
			case "reporter":
				phase = "报告阶段"
			case "excel_agent":
				phase = "Excel 助手"
			}
			if phase != "" {
				fmt.Printf("\n[%s]\n", phase)
			}
		}

		if msg, _, err := adk.GetMessage(event); err == nil && msg.Content != "" {
			content := strings.TrimSpace(msg.Content)
			if content != "" {
				fmt.Println(content)
			}
		}
	}
	fmt.Println("\n\n=== 分析完成 ===")
}
