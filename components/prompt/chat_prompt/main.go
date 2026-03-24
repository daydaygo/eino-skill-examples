package main

import (
	"fmt"
	"os"

	"github.com/cloudwego/eino/schema"
)

func main() {
	fmt.Println("=== Chat Prompt Demo ===")
	fmt.Println("This demo shows how to use Chat Prompt templates")
	fmt.Println()

	_ = getEnvOrDefault("OPENAI_BASE_URL", "https://api.openai.com/v1")
	_ = os.Getenv("OPENAI_API_KEY")
	_ = getEnvOrDefault("OPENAI_MODEL", "gpt-4o-mini")

	fmt.Println("=== 1. Basic Message Construction ===")

	messages := []*schema.Message{
		schema.SystemMessage("You are a helpful coding assistant."),
		schema.UserMessage("What is Go?"),
	}

	for i, msg := range messages {
		fmt.Printf("Message %d [%s]: %s\n", i+1, msg.Role, msg.Content)
	}

	fmt.Println("\n=== 2. Message with Variables ===")

	type PromptVars struct {
		Name   string
		Topic  string
		Detail string
	}

	vars := PromptVars{
		Name:   "Alice",
		Topic:  "machine learning",
		Detail: "neural networks",
	}

	tplSystem := "Hello {{.Name}}, you are an expert in {{.Topic}}."
	tplUser := "Please explain {{.Detail}} to a beginner."

	fmt.Printf("System template: %s\n", tplSystem)
	fmt.Printf("User template: %s\n", tplUser)
	fmt.Printf("Variables: %+v\n", vars)

	fmt.Println("\n=== 3. Multi-turn Conversation ===")

	conversation := []struct {
		role    string
		content string
	}{
		{"system", "You are a helpful math tutor."},
		{"user", "What is 2 + 2?"},
		{"assistant", "2 + 2 equals 4."},
		{"user", "And what is 4 * 3?"},
	}

	fmt.Println("Conversation history:")
	for i, turn := range conversation {
		fmt.Printf("  %d. [%s]: %s\n", i+1, turn.role, turn.content)
	}

	fmt.Println("\n=== 4. Chat Prompt with Tools ===")

	weatherTool := &schema.ToolInfo{
		Name: "get_weather",
		Desc: "Get current weather for a city",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"city": {Type: schema.String, Desc: "City name", Required: true},
		}),
	}

	toolPrompt := fmt.Sprintf(`You have access to the following tools:

1. %s: %s

Use these tools when appropriate to answer user questions.`, weatherTool.Name, weatherTool.Desc)

	fmt.Printf("System prompt with tools:\n%s\n", toolPrompt)

	fmt.Println("\n=== 5. Message Types ===")

	textMsg := schema.UserMessage("Hello, how are you?")

	fmt.Println("Text message:")
	fmt.Printf("  Role: %s\n", textMsg.Role)
	fmt.Printf("  Content: %s\n", textMsg.Content)

	fmt.Println("\n=== 6. Building Conversation ===")

	chatPrompt := NewChatPrompt().
		SetSystem("You are a helpful assistant specialized in {{.domain}}").
		AddHistory(schema.UserMessage("Hi!")).
		AddHistory(schema.AssistantMessage("Hello! How can I help you?", []schema.ToolCall{})).
		SetUser("What is {{.topic}}?")

	builtMessages := chatPrompt.Build()
	fmt.Printf("Built %d messages:\n", len(builtMessages))
	for i, msg := range builtMessages {
		fmt.Printf("  %d. [%s]: %s\n", i+1, msg.Role, truncate(msg.Content, 50))
	}

	fmt.Println("\n=== Demo Complete ===")
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

type ChatPrompt struct {
	systemPrompt string
	userPrompt   string
	tools        []*schema.ToolInfo
	history      []*schema.Message
}

func NewChatPrompt() *ChatPrompt {
	return &ChatPrompt{
		tools:   make([]*schema.ToolInfo, 0),
		history: make([]*schema.Message, 0),
	}
}

func (p *ChatPrompt) SetSystem(prompt string) *ChatPrompt {
	p.systemPrompt = prompt
	return p
}

func (p *ChatPrompt) SetUser(prompt string) *ChatPrompt {
	p.userPrompt = prompt
	return p
}

func (p *ChatPrompt) AddTool(tool *schema.ToolInfo) *ChatPrompt {
	p.tools = append(p.tools, tool)
	return p
}

func (p *ChatPrompt) AddHistory(messages ...*schema.Message) *ChatPrompt {
	p.history = append(p.history, messages...)
	return p
}

func (p *ChatPrompt) Build() []*schema.Message {
	var messages []*schema.Message

	if p.systemPrompt != "" {
		messages = append(messages, schema.SystemMessage(p.systemPrompt))
	}

	messages = append(messages, p.history...)

	if p.userPrompt != "" {
		messages = append(messages, schema.UserMessage(p.userPrompt))
	}

	return messages
}
