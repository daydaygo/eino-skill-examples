# Eino 框架示例集合

[![Eino](https://img.shields.io/badge/Eino-CloudWeGo-blue)](https://github.com/cloudwego/eino)
[![Go](https://img.shields.io/badge/Go-1.18+-00ADD8)](https://golang.org/)
[![License](https://img.shields.io/badge/License-Apache%202.0-green)](LICENSE)

**Eino['aino]** 是 CloudWeGo 开源的 Go 语言大模型应用开发框架。本仓库包含完整的 Eino 框架使用示例，涵盖组件、编排、Agent、多智能体协作等核心功能。

> **AI 编码助手技能**: [eino-skill](https://github.com/daydaygo/eino-skill) - 帮助开发者使用 Eino 框架构建 LLM 应用

## 快速开始

### 环境准备

```bash
# 设置环境变量
export OPENAI_BASE_URL=https://api.openai.com/v1
export OPENAI_API_KEY=your-api-key
export OPENAI_MODEL=gpt-4o-mini

# 或使用火山引擎 Ark
export OPENAI_BASE_URL=https://ark.cn-beijing.volces.com/api/v3
export OPENAI_API_KEY=your-ark-api-key
export OPENAI_MODEL=doubao-seed-2-0-lite-260215
```

### 安装依赖

```bash
go mod download
```

### 运行示例

```bash
# ChatModel 基础示例
go run ./quickstart/chat

# ReAct Agent 示例
go run ./flow/agent/react

# Supervisor 多 Agent 协作
go run ./adk/multiagent/supervisor

# Chain 编排
go run ./compose/chain

# Graph 编排
go run ./compose/graph/simple
```

## 示例目录结构

```
├── adk/                          # ADK (Agent Development Kit)
│   ├── helloworld/               # Hello World Agent
│   ├── intro/                    # 入门示例
│   │   ├── chatmodel/            # ChatModelAgent 基础
│   │   ├── custom/               # 自定义 Agent
│   │   ├── session/              # Session 管理
│   │   ├── transfer/             # Agent 转移
│   │   ├── workflow/             # 工作流 Agent
│   │   │   ├── loop/             # LoopAgent
│   │   │   ├── parallel/         # ParallelAgent
│   │   │   └── sequential/       # SequentialAgent
│   │   └── http-sse-service/     # HTTP SSE 服务
│   ├── human-in-the-loop/        # 人机协作
│   │   ├── 1_approval/           # 审批模式
│   │   ├── 2_review-and-edit/    # 审核编辑
│   │   ├── 3_feedback-loop/      # 反馈循环
│   │   └── ...                   # 更多模式
│   └── multiagent/               # 多 Agent 协作
│       ├── supervisor/           # Supervisor 模式
│       ├── layered-supervisor/   # 分层 Supervisor
│       ├── plan-execute-replan/  # 计划执行重规划
│       └── integration-*/        # 集成示例
│
├── compose/                      # 编排
│   ├── chain/                    # Chain 链式编排
│   ├── graph/                    # Graph 图编排
│   │   ├── simple/               # 简单 Graph
│   │   ├── state/                # State Graph
│   │   ├── tool_call_agent/      # Tool Call Agent
│   │   └── ...                   # 更多示例
│   ├── workflow/                 # Workflow 工作流
│   └── batch/                    # BatchNode 批处理
│
├── flow/                         # Flow 流程模块
│   └── agent/
│       ├── react/                # ReAct Agent
│       ├── multiagent/           # Multi-Agent
│       ├── manus/                # Manus Agent
│       └── deer-go/              # Deer-Go 研究团队
│
├── components/                   # 组件
│   ├── model/                    # ChatModel
│   ├── tool/                     # Tool 工具
│   ├── retriever/                # Retriever 检索器
│   ├── document/                 # Document 文档
│   └── prompt/                   # Prompt 提示词
│
├── quickstart/                   # 快速开始
│   ├── chat/                     # Chat 基础示例
│   ├── eino_assistant/           # Eino 助手 (RAG)
│   └── todoagent/                # Todo Agent
│
└── devops/                       # 开发运维
    ├── debug/                    # 调试工具
    └── visualize/                # 可视化工具
```

## 核心概念

### 1. ChatModel 基础

```go
model, _ := openai.NewChatModel(ctx, &openai.ChatModelConfig{
    BaseURL: os.Getenv("OPENAI_BASE_URL"),
    APIKey:  os.Getenv("OPENAI_API_KEY"),
    Model:   os.Getenv("OPENAI_MODEL"),
})

msg, _ := model.Generate(ctx, []*schema.Message{
    schema.SystemMessage("You are a helpful assistant."),
    schema.UserMessage("Hello!"),
})
```

### 2. Tool 创建

```go
weatherTool, _ := utils.InferTool(
    "get_weather",
    "获取指定城市的天气信息",
    func(ctx context.Context, input *WeatherInput) (string, error) {
        return fmt.Sprintf("%s 的天气晴朗，温度 25°C", input.City), nil
    },
)
```

### 3. ReAct Agent

```go
agent, _ := react.NewAgent(ctx, &react.AgentConfig{
    ToolCallingModel: model,
    ToolsConfig: compose.ToolsNodeConfig{
        Tools: []tool.BaseTool{weatherTool},
    },
})
```

### 4. Supervisor 多 Agent

```go
supervisor, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    Name:        "supervisor",
    Description: "协调多个专家 Agent",
    Model:       model,
})
supervisor.SetSubAgents(mathAgent, researchAgent)
```

## 示例分类

| 分类 | 示例数 | 说明 |
|------|--------|------|
| ADK 入门 | 10 | ChatModelAgent、Workflow、Session 等 |
| 人机协作 | 8 | 审批、审核编辑、反馈循环等 |
| 多 Agent | 6 | Supervisor、Plan-Execute 等 |
| 编排 | 15 | Chain、Graph、Workflow、Batch |
| Flow | 8 | ReAct Agent、Multi-Agent |
| 组件 | 13 | Model、Tool、Retriever 等 |
| 快速开始 | 3 | Chat、RAG、Todo Agent |

## 相关链接

- **Eino 官方文档**: https://www.cloudwego.io/zh/docs/eino/
- **Eino 核心**: https://github.com/cloudwego/eino
- **Eino 扩展**: https://github.com/cloudwego/eino-ext
- **Eino 示例**: https://github.com/cloudwego/eino-examples
- **eino-skill**: https://github.com/daydaygo/eino-skill

## License

Apache License 2.0