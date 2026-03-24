package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
)

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

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "hello_agent",
		Description: "A simple hello world agent",
		Instruction: "You are a friendly assistant. Greet the user and answer questions briefly.",
		Model:       model,
	})
	if err != nil {
		panic(err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	fmt.Println("=== Hello World Agent ===")
	fmt.Println("Type 'quit' to exit.")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("You: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "quit" || input == "" {
			if input == "quit" {
				break
			}
			continue
		}

		fmt.Print("Agent: ")
		iter := runner.Query(ctx, input)

		for {
			event, ok := iter.Next()
			if !ok {
				break
			}

			if event.Err != nil {
				fmt.Printf("Error: %v\n", event.Err)
				continue
			}

			if msg, _, err := adk.GetMessage(event); err == nil {
				fmt.Print(msg.Content)
			}
		}
		fmt.Println()
	}

	fmt.Println("=== Goodbye ===")
}
