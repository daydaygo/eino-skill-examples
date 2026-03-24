package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
)

type MockRetriever struct {
	documents []*schema.Document
}

func NewMockRetriever() *MockRetriever {
	return &MockRetriever{
		documents: []*schema.Document{
			{ID: "1", Content: "Go is a programming language designed at Google. It is statically typed and has garbage collection.", MetaData: map[string]any{"source": "wiki"}},
			{ID: "2", Content: "Python is a high-level programming language known for its clear syntax and readability.", MetaData: map[string]any{"source": "wiki"}},
			{ID: "3", Content: "Rust is a systems programming language focused on safety and performance.", MetaData: map[string]any{"source": "wiki"}},
			{ID: "4", Content: "JavaScript is the programming language of the web, used for both frontend and backend development.", MetaData: map[string]any{"source": "wiki"}},
			{ID: "5", Content: "TypeScript is a typed superset of JavaScript that compiles to plain JavaScript.", MetaData: map[string]any{"source": "wiki"}},
		},
	}
}

func (r *MockRetriever) Retrieve(ctx context.Context, query string, opts ...retriever.Option) ([]*schema.Document, error) {
	var results []*schema.Document
	queryLower := strings.ToLower(query)

	for _, doc := range r.documents {
		if strings.Contains(strings.ToLower(doc.Content), queryLower) {
			results = append(results, doc)
		}
	}

	if len(results) == 0 {
		for _, doc := range r.documents {
			words := strings.Fields(queryLower)
			for _, word := range words {
				if len(word) > 3 && strings.Contains(strings.ToLower(doc.Content), word) {
					results = append(results, doc)
					break
				}
			}
		}
	}

	return results, nil
}

type MultiQueryRetriever struct {
	baseRetriever retriever.Retriever
	chatModel     *openai.ChatModel
	numQueries    int
}

type MultiQueryRetrieverConfig struct {
	BaseRetriever retriever.Retriever
	ChatModel     *openai.ChatModel
	NumQueries    int
}

func NewMultiQueryRetriever(config *MultiQueryRetrieverConfig) *MultiQueryRetriever {
	numQueries := config.NumQueries
	if numQueries <= 0 {
		numQueries = 3
	}
	return &MultiQueryRetriever{
		baseRetriever: config.BaseRetriever,
		chatModel:     config.ChatModel,
		numQueries:    numQueries,
	}
}

func (r *MultiQueryRetriever) Retrieve(ctx context.Context, query string, opts ...retriever.Option) ([]*schema.Document, error) {
	fmt.Printf("\n[MultiQuery] Original query: %s\n", query)

	queries, err := r.generateQueries(ctx, query)
	if err != nil {
		fmt.Printf("[MultiQuery] Failed to generate queries: %v, using original query only\n", err)
		queries = []string{query}
	}

	fmt.Printf("[MultiQuery] Generated %d query variants:\n", len(queries))
	for i, q := range queries {
		fmt.Printf("  %d. %s\n", i+1, q)
	}

	allDocs := make([]*schema.Document, 0)
	seenIDs := make(map[string]bool)

	for i, q := range queries {
		fmt.Printf("\n[MultiQuery] Retrieving with query %d...\n", i+1)
		docs, err := r.baseRetriever.Retrieve(ctx, q, opts...)
		if err != nil {
			fmt.Printf("[MultiQuery] Retrieve error for query %d: %v\n", i+1, err)
			continue
		}

		for _, doc := range docs {
			if !seenIDs[doc.ID] {
				seenIDs[doc.ID] = true
				allDocs = append(allDocs, doc)
			}
		}
	}

	fmt.Printf("\n[MultiQuery] Total unique documents retrieved: %d\n", len(allDocs))
	return allDocs, nil
}

func (r *MultiQueryRetriever) generateQueries(ctx context.Context, query string) ([]string, error) {
	prompt := fmt.Sprintf(`You are an AI assistant helping to generate multiple search queries for better document retrieval.

Original query: %s

Generate %d different versions of this query that:
1. Rephrase the original question using different words
2. Add relevant context or synonyms
3. Break down complex queries into simpler ones

Output ONLY the queries, one per line, no numbering or explanation.`, query, r.numQueries)

	messages := []*schema.Message{
		schema.SystemMessage("You are a helpful assistant that generates search query variations."),
		schema.UserMessage(prompt),
	}

	sr, err := r.chatModel.Stream(ctx, messages)
	if err != nil {
		return nil, err
	}
	defer sr.Close()

	var fullResponse strings.Builder
	for {
		chunk, err := sr.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		fullResponse.WriteString(chunk.Content)
	}

	lines := strings.Split(strings.TrimSpace(fullResponse.String()), "\n")
	var queries []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			queries = append(queries, line)
		}
	}

	if len(queries) > r.numQueries {
		queries = queries[:r.numQueries]
	}

	return queries, nil
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

	model, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   getEnvOrDefault("OPENAI_MODEL", "gpt-4o-mini"),
	})
	if err != nil {
		panic(err)
	}

	baseRetriever := NewMockRetriever()

	multiQueryRetriever := NewMultiQueryRetriever(&MultiQueryRetrieverConfig{
		BaseRetriever: baseRetriever,
		ChatModel:     model,
		NumQueries:    3,
	})

	fmt.Println("=== MultiQuery Retriever Demo ===")
	fmt.Println("This retriever generates multiple query variants to improve recall")
	fmt.Println()

	testQueries := []string{
		"What is Go programming language?",
		"Tell me about type-safe languages",
	}

	for _, query := range testQueries {
		fmt.Printf("\n%s\n", strings.Repeat("=", 60))
		fmt.Printf("Testing query: %s\n", query)
		fmt.Println(strings.Repeat("=", 60))

		docs, err := multiQueryRetriever.Retrieve(ctx, query)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Println("\nRetrieved documents:")
		for i, doc := range docs {
			fmt.Printf("\n--- Document %d (ID: %s) ---\n", i+1, doc.ID)
			fmt.Printf("Content: %s\n", doc.Content)
			fmt.Printf("Metadata: %v\n", doc.MetaData)
		}
	}
}
