package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

func main() {
	fmt.Println("=== Lambda Component Demo ===")
	fmt.Println("Lambda is a function component for custom processing")
	fmt.Println()

	ctx := context.Background()

	baseURL := getEnvOrDefault("OPENAI_BASE_URL", "https://api.openai.com/v1")
	apiKey := os.Getenv("OPENAI_API_KEY")

	fmt.Println("=== 1. Basic Lambda in Chain ===")

	chain := compose.NewChain[string, string]().
		AppendLambda(compose.InvokableLambda(
			func(ctx context.Context, input string) (string, error) {
				return strings.TrimSpace(input), nil
			},
		)).
		AppendLambda(compose.InvokableLambda(
			func(ctx context.Context, input string) (string, error) {
				return strings.ToLower(input), nil
			},
		)).
		AppendLambda(compose.InvokableLambda(
			func(ctx context.Context, input string) (string, error) {
				return "Processed: " + input, nil
			},
		))

	compiled, err := chain.Compile(ctx)
	if err != nil {
		panic(err)
	}

	result, err := compiled.Invoke(ctx, "  HELLO WORLD  ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Chain result: %s\n", result)

	fmt.Println("\n=== 2. Lambda with Different Types ===")

	jsonTransform := compose.InvokableLambda(
		func(ctx context.Context, input map[string]any) (string, error) {
			var parts []string
			for k, v := range input {
				parts = append(parts, fmt.Sprintf("%s=%v", k, v))
			}
			return strings.Join(parts, ", "), nil
		},
	)

	jsonChain, _ := compose.NewChain[map[string]any, string]().
		AppendLambda(jsonTransform).
		Compile(ctx)

	result2, err := jsonChain.Invoke(ctx, map[string]any{
		"name": "Alice",
		"age":  30,
		"city": "Beijing",
	})
	if err != nil {
		panic(err)
	}
	fmt.Printf("JSON transform result: %s\n", result2)

	fmt.Println("\n=== 3. Struct Type Lambda ===")

	type Input struct {
		Text   string
		Repeat int
	}

	type Output struct {
		Result string
		Length int
	}

	structLambda := compose.InvokableLambda(
		func(ctx context.Context, input *Input) (*Output, error) {
			result := strings.Repeat(input.Text, input.Repeat)
			return &Output{
				Result: result,
				Length: len(result),
			}, nil
		},
	)

	structChain, _ := compose.NewChain[*Input, *Output]().
		AppendLambda(structLambda).
		Compile(ctx)

	output, err := structChain.Invoke(ctx, &Input{Text: "Go ", Repeat: 3})
	if err != nil {
		panic(err)
	}
	fmt.Printf("Struct lambda result: %s (length: %d)\n", output.Result, output.Length)

	fmt.Println("\n=== 4. Parallel Lambdas ===")

	fmt.Println("Manual parallel execution:")
	upperResult := strings.ToUpper("hello")
	lowerResult := strings.ToLower("hello")
	bracketResult := "[" + "hello" + "]"
	fmt.Printf("Parallel results: [%s, %s, %s]\n", upperResult, lowerResult, bracketResult)

	fmt.Println("\n=== 5. Using with ChatModel (if API key available) ===")

	if apiKey != "" {
		model, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
			BaseURL: baseURL,
			APIKey:  apiKey,
			Model:   getEnvOrDefault("OPENAI_MODEL", "gpt-4o-mini"),
		})
		if err != nil {
			fmt.Printf("Model creation error: %v\n", err)
		} else {
			processChain := compose.NewChain[[]*schema.Message, *schema.Message]().
				AppendChatModel(model).
				AppendLambda(compose.InvokableLambda(
					func(ctx context.Context, msg *schema.Message) (*schema.Message, error) {
						msg.Content = "Modified: " + msg.Content
						return msg, nil
					},
				))

			_, err := processChain.Compile(ctx)
			if err != nil {
				fmt.Printf("Chain compile error: %v\n", err)
			} else {
				fmt.Println("Chain with ChatModel created successfully")
			}
		}
	} else {
		fmt.Println("Skipping ChatModel integration (no API key)")
	}

	fmt.Println("\n=== 6. Lambda Processing Examples ===")

	processor := NewProcessingLambda("uppercase", func(ctx context.Context, input string) (string, error) {
		return strings.ToUpper(input), nil
	})

	processed, err := processor.Process(ctx, "hello world")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Processed result: %s\n", processed)

	fmt.Println("\n=== Demo Complete ===")
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

type ProcessingLambda struct {
	name      string
	processFn func(context.Context, string) (string, error)
}

func NewProcessingLambda(name string, fn func(context.Context, string) (string, error)) *ProcessingLambda {
	return &ProcessingLambda{
		name:      name,
		processFn: fn,
	}
}

func (l *ProcessingLambda) Process(ctx context.Context, input string) (string, error) {
	fmt.Printf("[%s] Processing: %s\n", l.name, input)
	return l.processFn(ctx, input)
}
