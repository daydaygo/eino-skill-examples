package main

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/cloudwego/eino/schema"
)

type ExtParser struct {
	parserType string
	options    ExtParserOptions
}

type ExtParserOptions struct {
	RemoveScripts   bool
	RemoveStyles    bool
	ExtractLinks    bool
	ExtractImages   bool
	PreserveHeaders bool
}

func NewExtParser(parserType string, options ExtParserOptions) *ExtParser {
	return &ExtParser{
		parserType: parserType,
		options:    options,
	}
}

func (p *ExtParser) Parse(ctx context.Context, reader io.Reader) ([]*schema.Document, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read content: %w", err)
	}

	switch p.parserType {
	case "html":
		return p.parseHTML(string(content))
	case "xml":
		return p.parseXML(string(content))
	case "json":
		return p.parseJSON(string(content))
	default:
		return p.parseText(string(content))
	}
}

func (p *ExtParser) parseHTML(html string) ([]*schema.Document, error) {
	fmt.Println("[HTML Parser] Processing HTML content...")

	text := html

	if p.options.RemoveScripts {
		scriptRegex := regexp.MustCompile(`<script[^>]*>[\s\S]*?</script>`)
		text = scriptRegex.ReplaceAllString(text, "")
	}

	if p.options.RemoveStyles {
		styleRegex := regexp.MustCompile(`<style[^>]*>[\s\S]*?</style>`)
		text = styleRegex.ReplaceAllString(text, "")
	}

	tagRegex := regexp.MustCompile(`<[^>]+>`)
	text = tagRegex.ReplaceAllString(text, " ")

	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	var links []string
	var images []string

	if p.options.ExtractLinks {
		linkRegex := regexp.MustCompile(`href="([^"]+)"`)
		matches := linkRegex.FindAllStringSubmatch(html, -1)
		for _, match := range matches {
			if len(match) > 1 {
				links = append(links, match[1])
			}
		}
	}

	if p.options.ExtractImages {
		imgRegex := regexp.MustCompile(`src="([^"]+\.(jpg|jpeg|png|gif|webp)[^"]*)"`)
		matches := imgRegex.FindAllStringSubmatch(html, -1)
		for _, match := range matches {
			if len(match) > 1 {
				images = append(images, match[1])
			}
		}
	}

	title := ""
	titleRegex := regexp.MustCompile(`<title[^>]*>([^<]+)</title>`)
	if matches := titleRegex.FindStringSubmatch(html); len(matches) > 1 {
		title = strings.TrimSpace(matches[1])
	}

	doc := &schema.Document{
		ID:      "html-doc",
		Content: text,
		MetaData: map[string]any{
			"parser":    "html",
			"title":     title,
			"links":     links,
			"images":    images,
			"linkCount": len(links),
			"imgCount":  len(images),
		},
	}

	return []*schema.Document{doc}, nil
}

func (p *ExtParser) parseXML(xml string) ([]*schema.Document, error) {
	fmt.Println("[XML Parser] Processing XML content...")

	tagRegex := regexp.MustCompile(`<([^/>]+)>([^<]*)</\1>`)
	matches := tagRegex.FindAllStringSubmatch(xml, -1)

	elements := make(map[string][]string)
	for _, match := range matches {
		if len(match) >= 3 {
			tag := match[1]
			value := strings.TrimSpace(match[2])
			if value != "" {
				elements[tag] = append(elements[tag], value)
			}
		}
	}

	var contentParts []string
	for tag, values := range elements {
		for _, v := range values {
			contentParts = append(contentParts, fmt.Sprintf("%s: %s", tag, v))
		}
	}

	doc := &schema.Document{
		ID:      "xml-doc",
		Content: strings.Join(contentParts, "\n"),
		MetaData: map[string]any{
			"parser":   "xml",
			"elements": elements,
		},
	}

	return []*schema.Document{doc}, nil
}

func (p *ExtParser) parseJSON(jsonStr string) ([]*schema.Document, error) {
	fmt.Println("[JSON Parser] Processing JSON content...")

	text := jsonStr
	text = strings.ReplaceAll(text, "{", "")
	text = strings.ReplaceAll(text, "}", "")
	text = strings.ReplaceAll(text, "[", "")
	text = strings.ReplaceAll(text, "]", "")
	text = strings.ReplaceAll(text, "\"", "")
	text = strings.ReplaceAll(text, ",", "\n")
	text = strings.ReplaceAll(text, ":", ": ")
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	doc := &schema.Document{
		ID:      "json-doc",
		Content: text,
		MetaData: map[string]any{
			"parser": "json",
			"raw":    jsonStr,
		},
	}

	return []*schema.Document{doc}, nil
}

func (p *ExtParser) parseText(text string) ([]*schema.Document, error) {
	fmt.Println("[Text Parser] Processing plain text...")

	doc := &schema.Document{
		ID:      "text-doc",
		Content: text,
		MetaData: map[string]any{
			"parser": "text",
			"length": len(text),
		},
	}

	return []*schema.Document{doc}, nil
}

func main() {
	fmt.Println("=== Extended Document Parser Demo ===")
	fmt.Println("This demo shows how to use extended parsers for various formats")
	fmt.Println()

	ctx := context.Background()

	fmt.Println("=== HTML Parsing ===")
	htmlParser := NewExtParser("html", ExtParserOptions{
		RemoveScripts: true,
		RemoveStyles:  true,
		ExtractLinks:  true,
		ExtractImages: true,
	})

	sampleHTML := `<!DOCTYPE html>
<html>
<head>
	<title>Sample Page</title>
	<style>body { color: red; }</style>
</head>
<body>
	<h1>Welcome to the Sample Page</h1>
	<script>console.log("test");</script>
	<p>This is a paragraph with some text content.</p>
	<a href="https://example.com/page1">Link 1</a>
	<a href="https://example.com/page2">Link 2</a>
	<img src="image1.jpg" alt="Image 1">
	<img src="image2.png" alt="Image 2">
</body>
</html>`

	htmlDocs, err := htmlParser.Parse(ctx, strings.NewReader(sampleHTML))
	if err != nil {
		panic(err)
	}

	for _, doc := range htmlDocs {
		fmt.Printf("Title: %s\n", doc.MetaData["title"])
		fmt.Printf("Content: %s\n", truncate(doc.Content, 100))
		fmt.Printf("Links (%d): %v\n", doc.MetaData["linkCount"], doc.MetaData["links"])
		fmt.Printf("Images (%d): %v\n", doc.MetaData["imgCount"], doc.MetaData["images"])
	}

	fmt.Println("\n=== XML Parsing ===")
	xmlParser := NewExtParser("xml", ExtParserOptions{})

	sampleXML := `<?xml version="1.0"?>
<article>
	<title>Introduction to Go</title>
	<author>John Doe</author>
	<content>Go is a programming language...</content>
	<tags>golang, programming, tutorial</tags>
</article>`

	xmlDocs, err := xmlParser.Parse(ctx, strings.NewReader(sampleXML))
	if err != nil {
		panic(err)
	}

	for _, doc := range xmlDocs {
		fmt.Printf("Content:\n%s\n", doc.Content)
		fmt.Printf("Elements: %v\n", doc.MetaData["elements"])
	}

	fmt.Println("\n=== JSON Parsing ===")
	jsonParser := NewExtParser("json", ExtParserOptions{})

	sampleJSON := `{
	"name": "Test Project",
	"version": "1.0.0",
	"description": "A sample project"
}`

	jsonDocs, err := jsonParser.Parse(ctx, strings.NewReader(sampleJSON))
	if err != nil {
		panic(err)
	}

	for _, doc := range jsonDocs {
		fmt.Printf("Content: %s\n", truncate(doc.Content, 80))
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
