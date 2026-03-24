package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
)

type DocumentCategory string

const (
	CategoryTech    DocumentCategory = "tech"
	CategoryScience DocumentCategory = "science"
	CategoryGeneral DocumentCategory = "general"
)

type CategoryRetriever struct {
	category  DocumentCategory
	documents []*schema.Document
}

func NewCategoryRetriever(category DocumentCategory, documents []*schema.Document) *CategoryRetriever {
	return &CategoryRetriever{
		category:  category,
		documents: documents,
	}
}

func (r *CategoryRetriever) Retrieve(ctx context.Context, query string, opts ...retriever.Option) ([]*schema.Document, error) {
	fmt.Printf("  [%s Retriever] Searching for: %s\n", strings.ToUpper(string(r.category)), query)

	var results []*schema.Document
	queryLower := strings.ToLower(query)

	for _, doc := range r.documents {
		contentLower := strings.ToLower(doc.Content)
		if strings.Contains(contentLower, queryLower) {
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

type RouterRetriever struct {
	retrievers map[DocumentCategory]retriever.Retriever
	keywords   map[DocumentCategory][]string
}

func NewRouterRetriever() *RouterRetriever {
	techDocs := []*schema.Document{
		{ID: "tech-1", Content: "Go is a statically typed programming language developed at Google.", MetaData: map[string]any{"category": "tech"}},
		{ID: "tech-2", Content: "Python is widely used for machine learning and data science.", MetaData: map[string]any{"category": "tech"}},
		{ID: "tech-3", Content: "Kubernetes is a container orchestration platform.", MetaData: map[string]any{"category": "tech"}},
	}

	scienceDocs := []*schema.Document{
		{ID: "science-1", Content: "Quantum mechanics describes the behavior of matter at atomic scales.", MetaData: map[string]any{"category": "science"}},
		{ID: "science-2", Content: "The theory of relativity was developed by Albert Einstein.", MetaData: map[string]any{"category": "science"}},
		{ID: "science-3", Content: "DNA contains the genetic instructions for all living organisms.", MetaData: map[string]any{"category": "science"}},
	}

	generalDocs := []*schema.Document{
		{ID: "general-1", Content: "Coffee is one of the most popular beverages worldwide.", MetaData: map[string]any{"category": "general"}},
		{ID: "general-2", Content: "The Eiffel Tower is located in Paris, France.", MetaData: map[string]any{"category": "general"}},
	}

	return &RouterRetriever{
		retrievers: map[DocumentCategory]retriever.Retriever{
			CategoryTech:    NewCategoryRetriever(CategoryTech, techDocs),
			CategoryScience: NewCategoryRetriever(CategoryScience, scienceDocs),
			CategoryGeneral: NewCategoryRetriever(CategoryGeneral, generalDocs),
		},
		keywords: map[DocumentCategory][]string{
			CategoryTech:    {"programming", "code", "software", "api", "language", "docker", "kubernetes", "go", "python", "javascript", "database"},
			CategoryScience: {"physics", "quantum", "chemistry", "biology", "dna", "atom", "molecule", "einstein", "relativity", "science"},
			CategoryGeneral: {"coffee", "paris", "tower", "city", "country", "food", "travel"},
		},
	}
}

func (r *RouterRetriever) Retrieve(ctx context.Context, query string, opts ...retriever.Option) ([]*schema.Document, error) {
	fmt.Printf("[Router] Analyzing query: %s\n", query)

	category := r.classifyQuery(query)
	fmt.Printf("[Router] Routed to category: %s\n", strings.ToUpper(string(category)))

	ret, ok := r.retrievers[category]
	if !ok {
		ret = r.retrievers[CategoryGeneral]
	}

	return ret.Retrieve(ctx, query, opts...)
}

func (r *RouterRetriever) classifyQuery(query string) DocumentCategory {
	queryLower := strings.ToLower(query)

	scores := make(map[DocumentCategory]int)
	for cat, keywords := range r.keywords {
		for _, keyword := range keywords {
			if strings.Contains(queryLower, keyword) {
				scores[cat]++
			}
		}
	}

	var maxCategory DocumentCategory = CategoryGeneral
	maxScore := 0

	for cat, score := range scores {
		if score > maxScore {
			maxScore = score
			maxCategory = cat
		}
	}

	if maxScore == 0 {
		return CategoryGeneral
	}

	return maxCategory
}

func (r *RouterRetriever) AddRetriever(category DocumentCategory, ret retriever.Retriever) {
	r.retrievers[category] = ret
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func main() {
	ctx := context.Background()

	_ = getEnvOrDefault("OPENAI_BASE_URL", "https://api.openai.com/v1")
	_ = os.Getenv("OPENAI_API_KEY")
	_ = getEnvOrDefault("OPENAI_MODEL", "gpt-4o-mini")

	routerRetriever := NewRouterRetriever()

	fmt.Println("=== Router Retriever Demo ===")
	fmt.Println("This retriever routes queries to different retrievers based on content")
	fmt.Println()

	testQueries := []string{
		"What programming languages are popular?",
		"Tell me about quantum physics",
		"What is the Eiffel Tower?",
		"How does DNA work?",
		"What is Kubernetes?",
	}

	for _, query := range testQueries {
		fmt.Printf("\n%s\n", strings.Repeat("=", 60))
		fmt.Printf("Query: %s\n", query)
		fmt.Println(strings.Repeat("-", 60))

		docs, err := routerRetriever.Retrieve(ctx, query)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Printf("\nRetrieved %d documents:\n", len(docs))
		for i, doc := range docs {
			fmt.Printf("  %d. [%s] %s\n", i+1, doc.ID, truncate(doc.Content, 60))
		}
	}

	fmt.Printf("\n%s\n", strings.Repeat("=", 60))
	fmt.Println("Demo complete!")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
