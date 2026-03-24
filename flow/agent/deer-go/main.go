package main

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
)

type ResearchState struct {
	mu            sync.RWMutex
	Topic         string
	Subtopics     []string
	ResearchNotes map[string]string
	Synthesis     string
	FinalReport   string
	CurrentPhase  string
}

func NewResearchState() *ResearchState {
	return &ResearchState{
		ResearchNotes: make(map[string]string),
		CurrentPhase:  "initialized",
	}
}

type WebSearchInput struct {
	Query string `json:"query" jsonschema:"required" jsonschema_description:"搜索关键词"`
}

type SaveNoteInput struct {
	Topic   string `json:"topic" jsonschema:"required" jsonschema_description:"笔记主题"`
	Content string `json:"content" jsonschema:"required" jsonschema_description:"笔记内容"`
}

type GetNoteInput struct {
	Topic string `json:"topic" jsonschema_description:"要获取的笔记主题，为空则获取所有"`
}

var researchState = NewResearchState()

func createResearchTools() ([]tool.BaseTool, error) {
	searchTool, err := utils.InferTool(
		"web_search",
		"搜索互联网获取研究资料",
		func(ctx context.Context, input *WebSearchInput) (string, error) {
			return fmt.Sprintf(`搜索结果：'%s'

1. 官方文档
   - 详细介绍核心概念
   - 包含最佳实践指南

2. 技术博客
   - 实战案例分析
   - 性能优化技巧

3. 学术论文
   - 理论基础
   - 研究前沿`, input.Query), nil
		},
	)
	if err != nil {
		return nil, err
	}

	saveNoteTool, err := utils.InferTool(
		"save_note",
		"保存研究笔记",
		func(ctx context.Context, input *SaveNoteInput) (string, error) {
			researchState.mu.Lock()
			defer researchState.mu.Unlock()
			researchState.ResearchNotes[input.Topic] = input.Content
			return fmt.Sprintf("笔记已保存：%s", input.Topic), nil
		},
	)
	if err != nil {
		return nil, err
	}

	getNoteTool, err := utils.InferTool(
		"get_note",
		"获取研究笔记",
		func(ctx context.Context, input *GetNoteInput) (string, error) {
			researchState.mu.RLock()
			defer researchState.mu.RUnlock()

			if input.Topic != "" {
				if content, ok := researchState.ResearchNotes[input.Topic]; ok {
					return fmt.Sprintf("【%s】\n%s", input.Topic, content), nil
				}
				return fmt.Sprintf("未找到主题为 '%s' 的笔记", input.Topic), nil
			}

			if len(researchState.ResearchNotes) == 0 {
				return "暂无笔记", nil
			}

			result := "所有研究笔记：\n"
			for topic, content := range researchState.ResearchNotes {
				result += fmt.Sprintf("\n【%s】\n%s\n", topic, content)
			}
			return result, nil
		},
	)
	if err != nil {
		return nil, err
	}

	return []tool.BaseTool{searchTool, saveNoteTool, getNoteTool}, nil
}

func createResearchTeam(ctx context.Context, model *openai.ChatModel, tools []tool.BaseTool) ([]adk.Agent, error) {
	plannerAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "planner",
		Description: "研究规划专家，负责分解研究主题和制定研究计划",
		Instruction: `你是一个研究规划专家。

职责：
1. 分析研究主题
2. 分解为多个子主题
3. 为每个子主题分配研究员

输出格式：
## 研究主题分析
[主题分析]

## 子主题划分
1. 子主题1：[名称] - [简介]
2. 子主题2：[名称] - [简介]
3. 子主题3：[名称] - [简介]

## 研究计划
- 第一阶段：[内容]
- 第二阶段：[内容]
- 第三阶段：[内容]`,
		Model: model,
	})
	if err != nil {
		return nil, err
	}

	researcherAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "researcher",
		Description: "研究员，负责深入研究特定子主题",
		Instruction: `你是一个研究员。

职责：
1. 使用 web_search 搜索相关资料
2. 阅读和分析搜索结果
3. 使用 save_note 保存研究笔记

研究方法：
- 从多个来源收集信息
- 对比不同观点
- 提取关键信息

输出要求：
- 详实的研究笔记
- 注明信息来源
- 突出重点内容`,
		Model: model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: tools,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	analystAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "analyst",
		Description: "分析师，负责分析和综合研究结果",
		Instruction: `你是一个分析师。

职责：
1. 使用 get_note 获取所有研究笔记
2. 分析和综合研究结果
3. 发现规律和洞察

分析方法：
- 对比不同子主题的发现
- 找出共同点和差异
- 提炼核心观点

输出要求：
- 结构化的分析报告
- 数据支持的结论
- 有价值的洞察`,
		Model: model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: tools,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	writerAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "writer",
		Description: "报告撰写专家，负责撰写最终研究报告",
		Instruction: `你是一个报告撰写专家。

职责：
1. 整合所有研究笔记和分析结果
2. 撰写结构清晰的研究报告
3. 确保报告内容完整、逻辑清晰

报告结构：
# 研究报告：[主题]

## 摘要
[简要概述研究内容和主要发现]

## 背景
[研究背景和目的]

## 主要发现
### 子主题1
[内容]

### 子主题2
[内容]

## 结论与建议
[总结和建议]

## 参考资料
[列出主要参考来源]`,
		Model: model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: tools,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	reviewerAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "reviewer",
		Description: "审核专家，负责审核研究报告质量",
		Instruction: `你是一个审核专家。

职责：
1. 审核报告的完整性和准确性
2. 检查逻辑和结构
3. 提出改进建议

审核要点：
- 内容是否完整
- 逻辑是否清晰
- 结论是否有据
- 格式是否规范

输出：
- 审核意见
- 改进建议
- 是否通过审核`,
		Model: model,
	})
	if err != nil {
		return nil, err
	}

	return []adk.Agent{plannerAgent, researcherAgent, analystAgent, writerAgent, reviewerAgent}, nil
}

func main() {
	ctx := context.Background()

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

	tools, err := createResearchTools()
	if err != nil {
		panic(err)
	}

	team, err := createResearchTeam(ctx, model, tools)
	if err != nil {
		panic(err)
	}

	coordinatorAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "coordinator",
		Description: "研究协调员，负责协调整个研究流程",
		Instruction: `你是研究团队的协调员。

团队成员：
- planner: 研究规划专家
- researcher: 研究员
- analyst: 分析师
- writer: 报告撰写专家
- reviewer: 审核专家

工作流程：
1. 调用 planner 制定研究计划
2. 调用 researcher 进行研究（可并行）
3. 调用 analyst 进行分析
4. 调用 writer 撰写报告
5. 调用 reviewer 审核报告

根据研究进展，合理调度各成员工作。`,
		Model: model,
	})
	if err != nil {
		panic(err)
	}

	coordinatorAgentWithSubAgents, err := adk.SetSubAgents(ctx, coordinatorAgent, team)
	if err != nil {
		panic(err)
	}

	sequentialAgent, err := adk.NewSequentialAgent(ctx, &adk.SequentialAgentConfig{
		Name:        "research_workflow",
		Description: "研究工作流，按顺序执行规划-研究-分析-撰写-审核",
		SubAgents:   team,
	})
	if err != nil {
		panic(err)
	}

	parallelResearcher1, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "researcher_1",
		Description: "研究员1，研究主题的第一个方面",
		Instruction: "你负责研究主题的第一个方面。使用 web_search 搜索资料，使用 save_note 保存笔记。",
		Model:       model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: tools,
			},
		},
	})

	parallelResearcher2, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "researcher_2",
		Description: "研究员2，研究主题的第二个方面",
		Instruction: "你负责研究主题的第二个方面。使用 web_search 搜索资料，使用 save_note 保存笔记。",
		Model:       model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: tools,
			},
		},
	})

	parallelResearchAgent, err := adk.NewParallelAgent(ctx, &adk.ParallelAgentConfig{
		Name:        "parallel_research",
		Description: "并行研究，多个研究员同时工作",
		SubAgents:   []adk.Agent{parallelResearcher1, parallelResearcher2},
	})
	if err != nil {
		panic(err)
	}

	_ = parallelResearchAgent
	_ = sequentialAgent

	fmt.Println("=== Deer-Go 研究团队示例 ===")
	fmt.Println("参考 deer-flow 的 Go 语言实现")
	fmt.Println("支持研究团队协作的状态图流转")
	fmt.Println()

	topics := []string{
		"Go 语言的并发编程模式研究",
	}

	for i, topic := range topics {
		researchState = NewResearchState()
		researchState.Topic = topic

		fmt.Printf("--- 研究任务 %d ---\n", i+1)
		fmt.Printf("主题: %s\n\n", topic)

		runner := adk.NewRunner(ctx, adk.RunnerConfig{
			Agent:           coordinatorAgentWithSubAgents,
			EnableStreaming: true,
		})

		iter := runner.Query(ctx, fmt.Sprintf("请协调团队完成关于 '%s' 的研究任务", topic))

		fmt.Println("研究过程：")
		for {
			event, ok := iter.Next()
			if !ok {
				break
			}
			if event.Err != nil {
				fmt.Printf("错误: %v\n", event.Err)
				continue
			}

			if event.Action != nil {
				if event.Action.TransferToAgent != nil {
					fmt.Printf("\n[协调] -> %s\n", event.Action.TransferToAgent.DestAgentName)
				}
			}

			if msg, _, err := adk.GetMessage(event); err == nil && msg != nil && msg.Content != "" {
				fmt.Printf("%s", msg.Content)
			}
		}
		fmt.Println("\n\n--- 研究完成 ---\n")

		fmt.Println("保存的研究笔记：")
		for t, content := range researchState.ResearchNotes {
			fmt.Printf("\n【%s】\n%s\n", t, content)
		}
	}
}
