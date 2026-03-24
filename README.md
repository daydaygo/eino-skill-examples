# Eino Skill Examples

[![Eino](https://img.shields.io/badge/Eino-CloudWeGo-blue)](https://github.com/cloudwego/eino)
[![eino-skill](https://img.shields.io/badge/eino--skill-AI%20Coding-green)](https://github.com/daydaygo/eino-skill)

本项目所有示例代码均使用 **[eino-skill](https://github.com/daydaygo/eino-skill)** 通过 AI 编码生成。

## 关于 eino-skill

[eino-skill](https://github.com/daydaygo/eino-skill) 是 Eino 框架的 AI 编码助手技能，帮助开发者使用 CloudWeGo Eino 框架构建 LLM 应用。

### 安装 eino-skill

**Claude Code:**
```bash
git clone https://github.com/daydaygo/eino-skill.git ~/.claude/skills/eino-skill
```

**OpenCode:**
```bash
git clone https://github.com/daydaygo/eino-skill.git ~/.agents/skills/eino-skill
```

### 使用 eino-skill

安装后，AI 编码助手将自动加载 Eino 框架知识，帮助你：

- 编写 ChatModel、Tool、Agent 等组件代码
- 构建 ReAct Agent、Supervisor 等多 Agent 系统
- 实现 Chain、Graph、Workflow 编排
- 处理流式输出、回调、中断恢复等场景

## 环境准备

### 安装 Eino

```bash
# 核心库
go get github.com/cloudwego/eino@latest

# OpenAI 模型支持
go get github.com/cloudwego/eino-ext/components/model/openai@latest

# 其他模型支持
go get github.com/cloudwego/eino-ext/components/model/ark@latest      # 字节火山引擎
go get github.com/cloudwego/eino-ext/components/model/ollama@latest   # Ollama
```

### 配置环境变量

```bash
# 复制配置模板
cp .env.example .env

# 编辑配置
vim .env
```

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `OPENAI_BASE_URL` | API 地址 | `https://api.openai.com/v1` |
| `OPENAI_API_KEY` | API 密钥 | - |
| `OPENAI_MODEL` | 模型名称 | - |

## 运行示例

```bash
# 安装依赖
go mod download

# 运行示例
go run ./adk/helloworld
go run ./flow/agent/react
go run ./adk/multiagent/supervisor
go run ./compose/chain
```

## 示例索引

| 分类 | 示例数 | 目录 |
|------|--------|------|
| ADK 入门 | 10 | `adk/intro/` |
| 人机协作 | 8 | `adk/human-in-the-loop/` |
| 多 Agent | 6 | `adk/multiagent/` |
| 编排 | 15 | `compose/` |
| Flow | 8 | `flow/` |
| 组件 | 13 | `components/` |
| 快速开始 | 3 | `quickstart/` |

## 相关链接

- **eino-skill**: https://github.com/daydaygo/eino-skill
- **Eino 官方文档**: https://www.cloudwego.io/zh/docs/eino/
- **Eino GitHub**: https://github.com/cloudwego/eino
- **Eino 扩展**: https://github.com/cloudwego/eino-ext
- **官方示例**: https://github.com/cloudwego/eino-examples

## License

Apache License 2.0