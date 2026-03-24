package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

type TodoItem struct {
	ID    int
	Title string
	Done  bool
}

type TodoStore struct {
	mu     sync.RWMutex
	todos  map[int]*TodoItem
	nextID int
}

func NewTodoStore() *TodoStore {
	return &TodoStore{
		todos:  make(map[int]*TodoItem),
		nextID: 1,
	}
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

	store := NewTodoStore()

	addTool, err := newAddTodoTool(store)
	if err != nil {
		panic(err)
	}
	listTool, err := newListTodosTool(store)
	if err != nil {
		panic(err)
	}
	deleteTool, err := newDeleteTodoTool(store)
	if err != nil {
		panic(err)
	}
	completeTool, err := newCompleteTodoTool(store)
	if err != nil {
		panic(err)
	}

	tools := []tool.BaseTool{addTool, listTool, deleteTool, completeTool}

	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: model,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: tools,
		},
		MessageModifier: func(ctx context.Context, input []*schema.Message) []*schema.Message {
			res := make([]*schema.Message, 0, len(input)+1)
			res = append(res, schema.SystemMessage(`你是 Todo 管理助手。你可以：
- 使用 add_todo 添加新任务
- 使用 list_todos 查看所有任务
- 使用 delete_todo 删除任务
- 使用 complete_todo 标记任务完成

简洁回复用户，确认操作结果。`))
			res = append(res, input...)
			return res
		},
		MaxStep: 10,
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("=== Todo Agent 已就绪 ===")
	fmt.Println("输入指令管理 Todo，输入 'quit' 退出")
	fmt.Println("示例指令：")
	fmt.Println("  - 添加任务：买牛奶")
	fmt.Println("  - 列出所有任务")
	fmt.Println("  - 完成任务 1")
	fmt.Println("  - 删除任务 2")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		query := strings.TrimSpace(line)
		if query == "" {
			continue
		}
		if query == "quit" || query == "exit" {
			fmt.Println("再见！")
			break
		}

		sr, err := agent.Stream(ctx, []*schema.Message{schema.UserMessage(query)})
		if err != nil {
			fmt.Printf("错误: %v\n", err)
			continue
		}

		for {
			msg, err := sr.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				fmt.Printf("错误: %v\n", err)
				break
			}
			fmt.Print(msg.Content)
		}
		fmt.Println("\n")
	}
}

type AddTodoInput struct {
	Title string `json:"title" jsonschema:"required" jsonschema_description:"任务标题"`
}

func newAddTodoTool(store *TodoStore) (tool.BaseTool, error) {
	return utils.InferTool(
		"add_todo",
		"添加新的 Todo 任务",
		func(ctx context.Context, input *AddTodoInput) (string, error) {
			store.mu.Lock()
			defer store.mu.Unlock()

			id := store.nextID
			store.todos[id] = &TodoItem{
				ID:    id,
				Title: input.Title,
				Done:  false,
			}
			store.nextID++

			return fmt.Sprintf("已添加任务 #%d: %s", id, input.Title), nil
		},
	)
}

type ListTodosInput struct{}

func newListTodosTool(store *TodoStore) (tool.BaseTool, error) {
	return utils.InferTool(
		"list_todos",
		"列出所有 Todo 任务",
		func(ctx context.Context, input *ListTodosInput) (string, error) {
			store.mu.RLock()
			defer store.mu.RUnlock()

			if len(store.todos) == 0 {
				return "当前没有任务。", nil
			}

			var sb strings.Builder
			sb.WriteString("当前任务列表：\n")
			for _, todo := range store.todos {
				status := "[ ]"
				if todo.Done {
					status = "[✓]"
				}
				sb.WriteString(fmt.Sprintf("  #%d %s %s\n", todo.ID, status, todo.Title))
			}
			return sb.String(), nil
		},
	)
}

type DeleteTodoInput struct {
	ID int `json:"id" jsonschema:"required" jsonschema_description:"要删除的任务 ID"`
}

func newDeleteTodoTool(store *TodoStore) (tool.BaseTool, error) {
	return utils.InferTool(
		"delete_todo",
		"删除指定的 Todo 任务",
		func(ctx context.Context, input *DeleteTodoInput) (string, error) {
			store.mu.Lock()
			defer store.mu.Unlock()

			todo, exists := store.todos[input.ID]
			if !exists {
				return fmt.Sprintf("任务 #%d 不存在", input.ID), nil
			}

			delete(store.todos, input.ID)
			return fmt.Sprintf("已删除任务 #%d: %s", input.ID, todo.Title), nil
		},
	)
}

type CompleteTodoInput struct {
	ID int `json:"id" jsonschema:"required" jsonschema_description:"要标记完成的任务 ID"`
}

func newCompleteTodoTool(store *TodoStore) (tool.BaseTool, error) {
	return utils.InferTool(
		"complete_todo",
		"标记 Todo 任务为已完成",
		func(ctx context.Context, input *CompleteTodoInput) (string, error) {
			store.mu.Lock()
			defer store.mu.Unlock()

			todo, exists := store.todos[input.ID]
			if !exists {
				return fmt.Sprintf("任务 #%d 不存在", input.ID), nil
			}

			if todo.Done {
				return fmt.Sprintf("任务 #%d 已经完成了", input.ID), nil
			}

			todo.Done = true
			return fmt.Sprintf("已完成任务 #%d: %s", input.ID, todo.Title), nil
		},
	)
}
