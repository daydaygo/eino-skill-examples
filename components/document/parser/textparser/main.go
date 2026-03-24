package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cloudwego/eino/schema"
)

type TextParser struct {
	chunkSize    int
	chunkOverlap int
	separator    string
}

type TextParserConfig struct {
	ChunkSize    int
	ChunkOverlap int
	Separator    string
}

func NewTextParser(config *TextParserConfig) *TextParser {
	chunkSize := config.ChunkSize
	if chunkSize <= 0 {
		chunkSize = 1000
	}

	separator := config.Separator
	if separator == "" {
		separator = "\n\n"
	}

	return &TextParser{
		chunkSize:    chunkSize,
		chunkOverlap: config.ChunkOverlap,
		separator:    separator,
	}
}

func (p *TextParser) Parse(ctx context.Context, reader io.Reader) ([]*schema.Document, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read content: %w", err)
	}

	text := string(content)

	chunks := p.splitIntoChunks(text)

	docs := make([]*schema.Document, 0, len(chunks))
	for i, chunk := range chunks {
		doc := &schema.Document{
			ID:      fmt.Sprintf("text-chunk-%d", i),
			Content: chunk,
			MetaData: map[string]any{
				"chunk_index":  i,
				"chunk_size":   len(chunk),
				"total_chunks": len(chunks),
			},
		}
		docs = append(docs, doc)
	}

	return docs, nil
}

func (p *TextParser) splitIntoChunks(text string) []string {
	if len(text) <= p.chunkSize {
		return []string{text}
	}

	var chunks []string
	var current strings.Builder
	currentLen := 0

	paragraphs := strings.Split(text, p.separator)

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		if currentLen+len(para)+len(p.separator) > p.chunkSize && currentLen > 0 {
			chunks = append(chunks, strings.TrimSpace(current.String()))
			current.Reset()
			currentLen = 0

			if p.chunkOverlap > 0 && len(chunks) > 0 {
				lastChunk := chunks[len(chunks)-1]
				if len(lastChunk) > p.chunkOverlap {
					overlap := lastChunk[len(lastChunk)-p.chunkOverlap:]
					current.WriteString(overlap)
					currentLen = len(overlap)
				}
			}
		}

		if currentLen > 0 {
			current.WriteString(p.separator)
			currentLen += len(p.separator)
		}
		current.WriteString(para)
		currentLen += len(para)
	}

	if currentLen > 0 {
		chunks = append(chunks, strings.TrimSpace(current.String()))
	}

	return chunks
}

func (p *TextParser) ParseFile(ctx context.Context, filePath string) ([]*schema.Document, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	return p.Parse(ctx, file)
}

func main() {
	fmt.Println("=== Text Parser Demo ===")
	fmt.Println("A simple text document parser that splits content into chunks")
	fmt.Println()

	ctx := context.Background()

	parser := NewTextParser(&TextParserConfig{
		ChunkSize:    200,
		ChunkOverlap: 50,
		Separator:    "\n\n",
	})

	sampleText := `Introduction to Programming

Programming is the process of creating a set of instructions that tell a computer how to perform a task. Programming can be done using a variety of computer programming languages, such as JavaScript, Python, and C++.

Getting Started

To start programming, you need to understand some basic concepts. These include variables, data types, control structures, and functions. Variables are used to store data, data types define what kind of data a variable can hold, control structures determine the flow of execution, and functions are reusable blocks of code.

Best Practices

When writing code, it's important to follow best practices. This includes writing clean and readable code, using meaningful variable names, adding comments to explain complex logic, and following a consistent coding style. These practices help make your code maintainable and easier for others to understand.

Conclusion

Learning to program takes time and practice. Start with simple projects and gradually work your way up to more complex ones. Remember that making mistakes is part of the learning process. Don't be afraid to experiment and try new things.`

	fmt.Println("=== Parsing Text ===")
	fmt.Printf("Original text length: %d characters\n", len(sampleText))
	fmt.Println()

	docs, err := parser.Parse(ctx, strings.NewReader(sampleText))
	if err != nil {
		panic(err)
	}

	fmt.Printf("Split into %d chunks:\n", len(docs))
	fmt.Println()

	for _, doc := range docs {
		fmt.Printf("--- Chunk %d (size: %d) ---\n", doc.MetaData["chunk_index"], doc.MetaData["chunk_size"])
		fmt.Printf("%s\n\n", truncate(doc.Content, 150))
	}

	fmt.Println("\n=== Different Chunk Sizes ===")

	sizes := []int{100, 200, 500}
	for _, size := range sizes {
		p := NewTextParser(&TextParserConfig{
			ChunkSize: size,
		})
		docs, _ := p.Parse(ctx, strings.NewReader(sampleText))
		fmt.Printf("Chunk size %d: %d chunks\n", size, len(docs))
	}

	fmt.Println("\n=== Demo Complete ===")
}

func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", "\\n")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
