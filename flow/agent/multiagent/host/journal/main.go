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

type JournalEntry struct {
	Date    string
	Content string
	Tags    []string
}

type JournalStore struct {
	mu      sync.RWMutex
	entries map[string][]JournalEntry
}

func NewJournalStore() *JournalStore {
	return &JournalStore{
		entries: make(map[string][]JournalEntry),
	}
}

func (s *JournalStore) AddEntry(userID, date, content string, tags []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[userID] = append(s.entries[userID], JournalEntry{
		Date:    date,
		Content: content,
		Tags:    tags,
	})
}

func (s *JournalStore) GetEntries(userID string) []JournalEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.entries[userID]
}

func (s *JournalStore) SearchByTag(userID, tag string) []JournalEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var results []JournalEntry
	for _, entry := range s.entries[userID] {
		for _, t := range entry.Tags {
			if t == tag {
				results = append(results, entry)
			}
		}
	}
	return results
}

var journalStore = NewJournalStore()

func createWriteJournalTool() (tool.BaseTool, error) {
	return utils.InferTool(
		"write_journal",
		"写一条新的日记",
		func(ctx context.Context, input *struct {
			Date    string   `json:"date" jsonschema:"required" jsonschema_description:"日期，格式：YYYY-MM-DD"`
			Content string   `json:"content" jsonschema:"required" jsonschema_description:"日记内容"`
			Tags    []string `json:"tags" jsonschema_description:"标签列表"`
		}) (string, error) {
			userID := "default_user"
			journalStore.AddEntry(userID, input.Date, input.Content, input.Tags)
			return fmt.Sprintf("日记已保存：\n日期：%s\n内容：%s\n标签：%v", input.Date, input.Content, input.Tags), nil
		},
	)
}

func createReadJournalTool() (tool.BaseTool, error) {
	return utils.InferTool(
		"read_journal",
		"读取日记",
		func(ctx context.Context, input *struct {
			Date string `json:"date" jsonschema_description:"指定日期，格式：YYYY-MM-DD，为空则读取所有"`
		}) (string, error) {
			userID := "default_user"
			entries := journalStore.GetEntries(userID)
			if len(entries) == 0 {
				return "暂无日记记录", nil
			}

			if input.Date != "" {
				for _, e := range entries {
					if e.Date == input.Date {
						return fmt.Sprintf("日期：%s\n内容：%s\n标签：%v", e.Date, e.Content, e.Tags), nil
					}
				}
				return fmt.Sprintf("未找到 %s 的日记", input.Date), nil
			}

			result := "所有日记：\n"
			for _, e := range entries {
				result += fmt.Sprintf("\n【%s】%s\n标签：%v\n", e.Date, e.Content, e.Tags)
			}
			return result, nil
		},
	)
}

func createSearchJournalTool() (tool.BaseTool, error) {
	return utils.InferTool(
		"search_journal",
		"按标签搜索日记",
		func(ctx context.Context, input *struct {
			Tag string `json:"tag" jsonschema:"required" jsonschema_description:"要搜索的标签"`
		}) (string, error) {
			userID := "default_user"
			entries := journalStore.SearchByTag(userID, input.Tag)
			if len(entries) == 0 {
				return fmt.Sprintf("没有找到标签为 '%s' 的日记", input.Tag), nil
			}

			result := fmt.Sprintf("标签 '%s' 相关日记：\n", input.Tag)
			for _, e := range entries {
				result += fmt.Sprintf("\n【%s】%s\n", e.Date, e.Content)
			}
			return result, nil
		},
	)
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

	writeTool, _ := createWriteJournalTool()
	readTool, _ := createReadJournalTool()
	searchTool, _ := createSearchJournalTool()

	writeAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "journal_writer",
		Description: "负责写日记，记录用户的日常生活和想法",
		Instruction: `你是一个日记写作助手。
帮助用户组织和记录他们的日记。
使用 write_journal 工具保存日记内容。
确保日期格式正确（YYYY-MM-DD），并为日记添加合适的标签。`,
		Model: model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{writeTool},
			},
		},
	})
	if err != nil {
		panic(err)
	}

	readAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "journal_reader",
		Description: "负责读取日记，帮助用户回顾过去的记录",
		Instruction: `你是一个日记阅读助手。
帮助用户查找和阅读他们的日记。
使用 read_journal 工具读取日记。
可以读取特定日期的日记，也可以浏览所有日记。`,
		Model: model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{readTool, searchTool},
			},
		},
	})
	if err != nil {
		panic(err)
	}

	qaAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "journal_qa",
		Description: "根据日记内容回答用户问题",
		Instruction: `你是一个日记问答助手。
根据用户的日记内容回答问题。
首先使用 read_journal 或 search_journal 获取相关日记，然后基于内容回答。`,
		Model: model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{readTool, searchTool},
			},
		},
	})
	if err != nil {
		panic(err)
	}

	hostAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "journal_host",
		Description: "日记助手主机，协调各个专业助手",
		Instruction: `你是日记助手的主机，负责协调各个专业助手：
- journal_writer: 负责写日记
- journal_reader: 负责读取日记
- journal_qa: 负责根据日记内容回答问题

根据用户的需求，选择合适的助手来完成任务。
如果用户要写日记，转交给 journal_writer。
如果用户要读日记，转交给 journal_reader。
如果用户要问关于日记的问题，转交给 journal_qa。`,
		Model: model,
	})
	if err != nil {
		panic(err)
	}

	hostAgentWithSubAgents, err := adk.SetSubAgents(ctx, hostAgent, []adk.Agent{writeAgent, readAgent, qaAgent})
	if err != nil {
		panic(err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           hostAgentWithSubAgents,
		EnableStreaming: true,
	})

	fmt.Println("=== 日记助手 Multi-Agent 示例 ===")
	fmt.Println("支持功能：写日记、读日记、根据日记回答问题")
	fmt.Println()

	interactions := []string{
		"帮我写一篇今天的日记，内容是今天学习了 Go 语言，标签是 学习",
		"再写一篇昨天的日记，内容是去公园跑步，标签是 运动",
		"帮我读一下今天的日记",
		"查找标签是 学习 的日记",
		"我这周做了哪些运动？",
	}

	for i, query := range interactions {
		fmt.Printf("--- 交互 %d ---\n", i+1)
		fmt.Printf("用户: %s\n", query)

		iter := runner.Query(ctx, query)
		fmt.Print("助手: ")
		for {
			event, ok := iter.Next()
			if !ok {
				break
			}
			if event.Err != nil {
				fmt.Printf("错误: %v\n", event.Err)
				continue
			}
			if msg, _, err := adk.GetMessage(event); err == nil && msg != nil {
				fmt.Print(msg.Content)
			}
		}
		fmt.Println("\n")
	}
}
