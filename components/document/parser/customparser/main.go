package main

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino/schema"
)

type DocumentParser interface {
	Parse(ctx context.Context, reader io.Reader) ([]*schema.Document, error)
}

type CustomParser struct {
	chunkSize    int
	chunkOverlap int
	metadata     map[string]any
}

type CustomParserConfig struct {
	ChunkSize    int
	ChunkOverlap int
	Metadata     map[string]any
}

func NewCustomParser(config *CustomParserConfig) *CustomParser {
	chunkSize := config.ChunkSize
	if chunkSize <= 0 {
		chunkSize = 500
	}
	return &CustomParser{
		chunkSize:    chunkSize,
		chunkOverlap: config.ChunkOverlap,
		metadata:     config.Metadata,
	}
}

func (p *CustomParser) Parse(ctx context.Context, reader io.Reader) ([]*schema.Document, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read content: %w", err)
	}

	text := string(content)

	chunks := p.splitText(text)

	docs := make([]*schema.Document, 0, len(chunks))
	for i, chunk := range chunks {
		doc := &schema.Document{
			ID:      fmt.Sprintf("chunk-%d", i),
			Content: chunk,
			MetaData: map[string]any{
				"chunk_index": i,
				"chunk_size":  len(chunk),
			},
		}

		for k, v := range p.metadata {
			doc.MetaData[k] = v
		}

		docs = append(docs, doc)
	}

	return docs, nil
}

func (p *CustomParser) splitText(text string) []string {
	paragraphs := strings.Split(text, "\n\n")

	var chunks []string
	var currentChunk strings.Builder

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		if currentChunk.Len()+len(para) > p.chunkSize && currentChunk.Len() > 0 {
			chunks = append(chunks, currentChunk.String())
			currentChunk.Reset()
		}

		if currentChunk.Len() > 0 {
			currentChunk.WriteString("\n\n")
		}
		currentChunk.WriteString(para)
	}

	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	return chunks
}

type MarkdownParser struct {
	CustomParser
}

func NewMarkdownParser(config *CustomParserConfig) *MarkdownParser {
	return &MarkdownParser{
		CustomParser: *NewCustomParser(config),
	}
}

func (p *MarkdownParser) Parse(ctx context.Context, reader io.Reader) ([]*schema.Document, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read content: %w", err)
	}

	text := string(content)
	sections := p.splitMarkdown(text)

	docs := make([]*schema.Document, 0, len(sections))
	for i, section := range sections {
		doc := &schema.Document{
			ID:      fmt.Sprintf("section-%d", i),
			Content: section.content,
			MetaData: map[string]any{
				"section_index": i,
				"heading":       section.heading,
				"level":         section.level,
			},
		}

		for k, v := range p.metadata {
			doc.MetaData[k] = v
		}

		docs = append(docs, doc)
	}

	return docs, nil
}

type markdownSection struct {
	heading string
	level   int
	content string
}

func (p *MarkdownParser) splitMarkdown(text string) []markdownSection {
	lines := strings.Split(text, "\n")

	var sections []markdownSection
	var currentSection markdownSection
	var contentLines []string

	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			if len(contentLines) > 0 || currentSection.content != "" {
				if len(contentLines) > 0 {
					currentSection.content = strings.Join(contentLines, "\n")
				}
				if currentSection.content != "" {
					sections = append(sections, currentSection)
				}
			}

			level := 0
			for _, c := range line {
				if c == '#' {
					level++
				} else {
					break
				}
			}

			currentSection = markdownSection{
				heading: strings.TrimSpace(line[level:]),
				level:   level,
			}
			contentLines = nil
		} else {
			contentLines = append(contentLines, line)
		}
	}

	if len(contentLines) > 0 {
		currentSection.content = strings.Join(contentLines, "\n")
	}
	if currentSection.content != "" {
		sections = append(sections, currentSection)
	}

	return sections
}

func main() {
	fmt.Println("=== Custom Document Parser Demo ===")
	fmt.Println()

	ctx := context.Background()

	sampleText := `This is the first paragraph of our sample document. It contains some text that will be processed by our custom parser.

This is the second paragraph. It provides additional content for demonstration purposes. The parser will split this into appropriate chunks.

The third paragraph continues our document. Each paragraph is separated by double newlines for proper chunking.

Finally, this is the fourth paragraph. It concludes our sample text.`

	parser := NewCustomParser(&CustomParserConfig{
		ChunkSize:    100,
		ChunkOverlap: 20,
		Metadata: map[string]any{
			"source": "sample.txt",
			"author": "demo",
		},
	})

	fmt.Println("=== Parsing Plain Text ===")
	docs, err := parser.Parse(ctx, strings.NewReader(sampleText))
	if err != nil {
		panic(err)
	}

	for _, doc := range docs {
		fmt.Printf("\n--- Document: %s ---\n", doc.ID)
		fmt.Printf("Content: %s\n", truncate(doc.Content, 80))
		fmt.Printf("Metadata: %v\n", doc.MetaData)
	}

	fmt.Println("\n\n=== Parsing Markdown ===")

	sampleMarkdown := `# Introduction
This is the introduction section of our document.

## Background
Some background information about the topic.

## Objectives
The main objectives of this project.

# Main Content
This section contains the main content.

## Details
Detailed information goes here.

# Conclusion
Final thoughts and summary.`

	mdParser := NewMarkdownParser(&CustomParserConfig{
		ChunkSize: 200,
		Metadata: map[string]any{
			"source": "sample.md",
			"type":   "markdown",
		},
	})

	mdDocs, err := mdParser.Parse(ctx, strings.NewReader(sampleMarkdown))
	if err != nil {
		panic(err)
	}

	for _, doc := range mdDocs {
		fmt.Printf("\n--- Section: %s ---\n", doc.ID)
		fmt.Printf("Heading: %s (Level %d)\n", doc.MetaData["heading"], doc.MetaData["level"])
		fmt.Printf("Content: %s\n", truncate(doc.Content, 60))
	}

	fmt.Println("\n\n=== Demo Complete ===")
}

func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
