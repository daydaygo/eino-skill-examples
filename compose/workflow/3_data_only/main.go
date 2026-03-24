package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cloudwego/eino/compose"
)

func main() {
	ctx := context.Background()

	_ = os.Getenv("OPENAI_API_KEY")

	type Input struct {
		RawData string
	}

	type ProcessedData struct {
		Upper  string
		Lower  string
		Length int
	}

	type Output struct {
		Result ProcessedData
	}

	workflow := compose.NewWorkflow[Input, Output]()

	workflow.AddLambdaNode("process", compose.InvokableLambda(
		func(ctx context.Context, input Input) (ProcessedData, error) {
			return ProcessedData{
				Upper:  strings.ToUpper(input.RawData),
				Lower:  strings.ToLower(input.RawData),
				Length: len(input.RawData),
			}, nil
		},
	), compose.WithOutputKey("processed"))

	workflow.AddLambdaNode("validate", compose.InvokableLambda(
		func(ctx context.Context, data ProcessedData) (Output, error) {
			if data.Length == 0 {
				return Output{Result: ProcessedData{Upper: "EMPTY", Lower: "empty", Length: 0}}, nil
			}
			return Output{Result: data}, nil
		},
	))

	workflow.End().AddDependency("validate")

	compiled, err := workflow.Compile(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Println("=== 纯数据流 Workflow 示例 ===")
	fmt.Println("展示无 LLM 的纯数据处理流程")
	fmt.Println()

	result, err := compiled.Invoke(ctx, Input{
		RawData: "Hello World",
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("原始数据: Hello World\n")
	fmt.Printf("处理后:\n")
	fmt.Printf("  大写: %s\n", result.Result.Upper)
	fmt.Printf("  小写: %s\n", result.Result.Lower)
	fmt.Printf("  长度: %d\n", result.Result.Length)
	fmt.Println("=== 完成 ===")
}
