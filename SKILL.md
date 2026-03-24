---
name: eino-skill
description: |
  Eino 框架 AI 编码助手 - 帮助开发者使用 CloudWeGo Eino 框架构建 LLM 应用。
  
  当用户提到以下任何内容时，使用此 skill：
  - Eino 框架、eino、cloudwego eino
  - Go 语言 LLM 应用开发、大模型应用框架
  - Agent、智能体、ReAct Agent、ChatModel Agent
  - Multi-Agent、多智能体协作、Supervisor、Plan-Execute
  - Tool、工具调用、工具创建
  - Chain、Graph、Workflow 编排
  - ChatModel、Embedding、Retriever 组件
  - 流式处理、Stream、StreamReader
  - Callback、回调、追踪、Trace
  - 人机协作、中断恢复、Interrupt、Resume
  
  即使没有明确提到 "eino"，如果用户在 Go 中构建 LLM 应用或 Agent，也使用此 skill。
---

# Eino 框架 AI 编码助手

Eino['aino] 是 CloudWeGo 开源的 Go 语言大模型应用开发框架。提供组件抽象、编排能力、流处理、ADK 智能体开发套件等核心能力。

## 核心架构

```
eino/                    # 核心库
├── schema/             # 数据类型（Message、ToolInfo 等）
├── components/         # 组件接口（ChatModel、Tool、Retriever 等）
├── compose/            # 编排（Chain、Graph、Workflow）
├── flow/               # Flow 集成组件（ReAct Agent）
├── adk/                # Agent Development Kit
└── callbacks/          # 切面机制

eino-ext/               # 扩展库
├── components/         # 组件实现（OpenAI、Ark、Ollama 等）
└── callbacks/          # 回调实现（CozeLoop、Langfuse 等）
```

## 快速决策树

```
用户需求 →
├─ 简单对话/单次调用 → 使用 ChatModel 组件
├─ 需要 Tool 调用 → ReAct Agent 或 ChatModelAgent
├─ 多步骤线性流程 → Chain 编排
├─ 复杂分支/状态管理 → Graph 编排
├─ 多 Agent 协作 → ADK（Supervisor、Plan-Execute 等）
└─ 需要 RAG → Retriever + ChatModel 组合
```

## 依赖安装

```bash
# 核心库
go get github.com/cloudwego/eino@latest

# 扩展库（按需选择）
go get github.com/cloudwego/eino-ext/components/model/openai@latest
go get github.com/cloudwego/eino-ext/components/model/ark@latest
go get github.com/cloudwego/eino-ext/components/model/ollama@latest
```

## 核心代码模式

### 1. ChatModel 基础使用

```go
import (
    "github.com/cloudwego/eino-ext/components/model/openai"
    "github.com/cloudwego/eino/schema"
)

baseURL := os.Getenv("OPENAI_BASE_URL")
if baseURL == "" {
    baseURL = "https://api.openai.com/v1"
}

model, _ := openai.NewChatModel(ctx, &openai.ChatModelConfig{
    BaseURL: baseURL,
    APIKey:  os.Getenv("OPENAI_API_KEY"),
    Model:   os.Getenv("OPENAI_MODEL"),
})

msg, _ := model.Generate(ctx, []*schema.Message{
    schema.SystemMessage("You are a helpful assistant."),
    schema.UserMessage("Hello!"),
})
```

**环境变量配置：**

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `OPENAI_BASE_URL` | API 地址 | `https://api.openai.com/v1` |
| `OPENAI_API_KEY` | API 密钥 | 无 |
| `OPENAI_MODEL` | 模型名称 | 无 |

### 2. 创建 Tool

```go
import (
    "github.com/cloudwego/eino/components/tool/utils"
    "github.com/cloudwego/eino/schema"
)

type WeatherInput struct {
    City string `json:"city" jsonschema:"required" jsonschema_description:"城市名称"`
}

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
import (
    "github.com/cloudwego/eino/flow/agent/react"
    "github.com/cloudwego/eino/compose"
)

agent, _ := react.NewAgent(ctx, &react.AgentConfig{
    ToolCallingModel: model,
    ToolsConfig: compose.ToolsNodeConfig{
        Tools: []tool.BaseTool{weatherTool},
    },
})

// 流式调用
sr, _ := agent.Stream(ctx, []*schema.Message{
    schema.UserMessage("北京今天天气怎么样？"),
})
defer sr.Close()

for {
    msg, err := sr.Recv()
    if errors.Is(err, io.EOF) {
        break
    }
    fmt.Print(msg.Content)
}
```

### 4. ADK ChatModelAgent

```go
import "github.com/cloudwego/eino/adk"

agent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    Name:        "assistant",
    Description: "A helpful assistant",
    Instruction: "You are a helpful assistant.",
    Model:       model,
    ToolsConfig: adk.ToolsConfig{
        ToolsNodeConfig: compose.ToolsNodeConfig{
            Tools: []tool.BaseTool{weatherTool},
        },
    },
})

runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})
events := runner.Query(ctx, "北京天气如何？")

for {
    event, ok := events.Next()
    if !ok {
        break
    }
    // 处理事件
}
```

### 5. Chain 编排

```go
import "github.com/cloudwego/eino/compose"

chain, _ := compose.NewChain[map[string]any, *schema.Message]().
    AppendChatTemplate(prompt).
    AppendChatModel(model).
    Compile(ctx)

result, _ := chain.Invoke(ctx, map[string]any{"query": "Hello"})
```

## 选择 Agent 类型

| 场景 | 推荐 | 说明 |
|------|------|------|
| 简单 Tool 调用 | ReAct Agent | 基础 ReAct 循环，适合大多数场景 |
| 需要 Runner/事件流 | ChatModelAgent | ADK 提供 Runner、事件流、状态管理 |
| 多 Agent 协作 | Supervisor | 层级协调，子 Agent 执行后回到 Supervisor |
| 复杂任务规划 | Plan-Execute | 计划-执行-重规划循环 |
| 线性工作流 | SequentialAgent | 按顺序执行多个 Agent |
| 并行任务 | ParallelAgent | 并发执行多个 Agent |

## 参考文档

按需读取以下参考文档获取详细信息：

- `references/react-agent.md` - ReAct Agent 完整指南
- `references/adk-agent.md` - ADK Agent 详细说明
- `references/multi-agent.md` - 多 Agent 协作模式
- `references/tools.md` - Tool 创建完整指南
- `references/orchestration.md` - Chain/Graph 编排

## 常见问题

### Q: ReAct Agent 和 ChatModelAgent 的区别？

**ReAct Agent** (`flow/agent/react`):
- 基于 Graph 编排
- 直接调用 Generate/Stream
- 适合简单场景

**ChatModelAgent** (`adk`):
- ADK 框架提供
- 通过 Runner 运行
- 支持事件流、状态管理、中断恢复
- 支持多 Agent 协作

### Q: 如何选择 ChatModel 实现？

- **OpenAI**: `eino-ext/components/model/openai`
- **字节火山引擎 Ark**: `eino-ext/components/model/ark`
- **Ollama 本地**: `eino-ext/components/model/ollama`
- **其他**: 查看 einfo-ext/components/model 目录

### Q: 流式输出如何处理？

```go
sr, _ := model.Stream(ctx, messages)
defer sr.Close()  // 必须关闭

for {
    msg, err := sr.Recv()
    if errors.Is(err, io.EOF) {
        break
    }
    fmt.Print(msg.Content)
}
```

## 最佳实践

1. **总是处理错误和关闭流**：`defer sr.Close()`
2. **Tool 参数使用 jsonschema tag**：便于大模型理解
3. **复杂 Tool 考虑拆分**：单一职责原则
4. **使用 MessageModifier 添加 system prompt**
5. **MaxStep 设置**：希望运行 N 个循环，设置 MaxStep = 2N

## 相关链接

- GitHub: https://github.com/cloudwego/eino
- 扩展库: https://github.com/cloudwego/einfo-ext
- 示例: https://github.com/cloudwego/eino-examples
- 文档: https://www.cloudwego.io/zh/docs/eino/