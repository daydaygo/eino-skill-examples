package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
)

type WeatherInput struct {
	City string `json:"city" jsonschema:"required" jsonschema_description:"城市名称"`
}

type SSEMessage struct {
	Type    string `json:"type"`
	Content string `json:"content"`
	Agent   string `json:"agent,omitempty"`
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
		log.Fatalf("Failed to create model: %v", err)
	}

	weatherTool, err := utils.InferTool(
		"get_weather",
		"获取指定城市的天气",
		func(ctx context.Context, input *WeatherInput) (string, error) {
			return fmt.Sprintf("%s 天气：晴朗，温度 25°C", input.City), nil
		},
	)
	if err != nil {
		log.Fatalf("Failed to create tool: %v", err)
	}

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "weather_assistant",
		Description: "天气助手",
		Instruction: "你是天气助手。使用 get_weather 工具回答天气问题。",
		Model:       model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{weatherTool},
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	http.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Message string `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		iter := runner.Query(r.Context(), req.Message)

		for {
			event, ok := iter.Next()
			if !ok {
				break
			}

			if event.Err != nil {
				sendSSE(w, flusher, SSEMessage{
					Type:    "error",
					Content: event.Err.Error(),
				})
				continue
			}

			if event.Action != nil {
				if event.Action.TransferToAgent != nil {
					sendSSE(w, flusher, SSEMessage{
						Type:    "transfer",
						Content: event.Action.TransferToAgent.DestAgentName,
					})
				}
			}

			if msg, _, err := adk.GetMessage(event); err == nil {
				if msg.Content != "" {
					sendSSE(w, flusher, SSEMessage{
						Type:    "content",
						Content: msg.Content,
					})
				}
			}
		}

		sendSSE(w, flusher, SSEMessage{
			Type:    "done",
			Content: "",
		})
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Println("=== HTTP SSE Service 示例 ===")
	fmt.Println()
	fmt.Println("服务已启动")
	fmt.Printf("地址: http://localhost:%s\n", port)
	fmt.Println()
	fmt.Println("端点:")
	fmt.Printf("  GET  /health - 健康检查\n")
	fmt.Printf("  POST /chat   - SSE 流式对话\n")
	fmt.Println()
	fmt.Println("示例请求:")
	fmt.Printf("  curl -X POST http://localhost:%s/chat \\\n", port)
	fmt.Printf("    -H 'Content-Type: application/json' \\\n")
	fmt.Printf("    -d '{\"message\": \"北京天气如何？\"}'\n")
	fmt.Println()
	fmt.Println("SSE 消息格式:")
	fmt.Println("  {\"type\": \"content\", \"content\": \"...\"}   - 内容片段")
	fmt.Println("  {\"type\": \"transfer\", \"content\": \"...\"}  - Agent 转移")
	fmt.Println("  {\"type\": \"done\", \"content\": \"\"}        - 完成")
	fmt.Println()

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func sendSSE(w http.ResponseWriter, flusher http.Flusher, msg SSEMessage) {
	data, _ := json.Marshal(msg)
	fmt.Fprintf(w, "data: %s\n\n", string(data))
	flusher.Flush()
}
