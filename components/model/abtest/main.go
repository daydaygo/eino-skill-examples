package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"sync/atomic"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

type ABTestChatModel struct {
	modelA    *openai.ChatModel
	modelB    *openai.ChatModel
	ratioA    float64
	counterA  atomic.Uint64
	counterB  atomic.Uint64
	useModelA bool
	rand      *rand.Rand
}

type ABTestConfig struct {
	ModelAConfig *openai.ChatModelConfig
	ModelBConfig *openai.ChatModelConfig
	RatioA       float64
}

func NewABTestChatModel(ctx context.Context, config *ABTestConfig) (*ABTestChatModel, error) {
	modelA, err := openai.NewChatModel(ctx, config.ModelAConfig)
	if err != nil {
		return nil, fmt.Errorf("create model A failed: %w", err)
	}

	modelB, err := openai.NewChatModel(ctx, config.ModelBConfig)
	if err != nil {
		return nil, fmt.Errorf("create model B failed: %w", err)
	}

	ratio := config.RatioA
	if ratio <= 0 || ratio > 1 {
		ratio = 0.5
	}

	return &ABTestChatModel{
		modelA:    modelA,
		modelB:    modelB,
		ratioA:    ratio,
		rand:      rand.New(rand.NewSource(time.Now().UnixNano())),
		useModelA: true,
	}, nil
}

func (m *ABTestChatModel) Generate(ctx context.Context, messages []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	selectedModel := m.selectModel()
	return selectedModel.Generate(ctx, messages, opts...)
}

func (m *ABTestChatModel) Stream(ctx context.Context, messages []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	selectedModel := m.selectModel()
	return selectedModel.Stream(ctx, messages, opts...)
}

func (m *ABTestChatModel) BindTools(tools []*schema.ToolInfo) error {
	if err := m.modelA.BindTools(tools); err != nil {
		return err
	}
	return m.modelB.BindTools(tools)
}

func (m *ABTestChatModel) selectModel() *openai.ChatModel {
	if m.rand.Float64() < m.ratioA {
		m.counterA.Add(1)
		fmt.Println("[ABTest] Using Model A")
		return m.modelA
	}
	m.counterB.Add(1)
	fmt.Println("[ABTest] Using Model B")
	return m.modelB
}

func (m *ABTestChatModel) SwitchToModelA() {
	m.useModelA = true
	fmt.Println("[ABTest] Switched to Model A (forced)")
}

func (m *ABTestChatModel) SwitchToModelB() {
	m.useModelA = false
	fmt.Println("[ABTest] Switched to Model B (forced)")
}

func (m *ABTestChatModel) GetStats() (countA, countB uint64) {
	return m.counterA.Load(), m.counterB.Load()
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func main() {
	ctx := context.Background()

	baseURL := getEnvOrDefault("OPENAI_BASE_URL", "https://api.openai.com/v1")
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		panic("OPENAI_API_KEY is required")
	}

	modelAName := getEnvOrDefault("OPENAI_MODEL_A", "gpt-4o-mini")
	modelBName := getEnvOrDefault("OPENAI_MODEL_B", "gpt-3.5-turbo")

	config := &ABTestConfig{
		ModelAConfig: &openai.ChatModelConfig{
			BaseURL: baseURL,
			APIKey:  apiKey,
			Model:   modelAName,
		},
		ModelBConfig: &openai.ChatModelConfig{
			BaseURL: baseURL,
			APIKey:  apiKey,
			Model:   modelBName,
		},
		RatioA: 0.7,
	}

	abModel, err := NewABTestChatModel(ctx, config)
	if err != nil {
		panic(err)
	}

	fmt.Println("=== A/B Test ChatModel Demo ===")
	fmt.Printf("Model A: %s (ratio: %.0f%%)\n", modelAName, config.RatioA*100)
	fmt.Printf("Model B: %s (ratio: %.0f%%)\n", modelBName, (1-config.RatioA)*100)
	fmt.Println()

	messages := []*schema.Message{
		schema.SystemMessage("You are a helpful assistant. Keep responses brief."),
		schema.UserMessage("Say hello in one sentence."),
	}

	fmt.Println("=== Test 1: Random routing (70% A / 30% B) ===")
	for i := 0; i < 3; i++ {
		msg, err := abModel.Generate(ctx, messages)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		fmt.Printf("Response: %s\n\n", msg.Content)
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Println("=== Test 2: Force switch to Model A ===")
	abModel.SwitchToModelA()
	for i := 0; i < 2; i++ {
		sr, err := abModel.Stream(ctx, messages)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		defer sr.Close()

		fmt.Print("Stream response: ")
		for {
			msg, err := sr.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				fmt.Printf("Stream error: %v\n", err)
				break
			}
			fmt.Print(msg.Content)
		}
		fmt.Println("\n")
	}

	countA, countB := abModel.GetStats()
	fmt.Printf("=== Statistics ===\n")
	fmt.Printf("Model A calls: %d\n", countA)
	fmt.Printf("Model B calls: %d\n", countB)
	fmt.Printf("Total calls: %d\n", countA+countB)
}
