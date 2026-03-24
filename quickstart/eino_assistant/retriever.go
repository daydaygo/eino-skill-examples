package main

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
)

type Document struct {
	ID       string
	Content  string
	Metadata map[string]any
	Vector   []float64
}

type MemoryStore struct {
	mu        sync.RWMutex
	documents map[string]*Document
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		documents: make(map[string]*Document),
	}
}

func (s *MemoryStore) Index(ctx context.Context, content string, metadata map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := fmt.Sprintf("%d", len(s.documents)+1)
	vector := simpleEmbed(content)

	s.documents[id] = &Document{
		ID:       id,
		Content:  content,
		Metadata: metadata,
		Vector:   vector,
	}
}

func (s *MemoryStore) Search(ctx context.Context, query string, topK int) []*Document {
	s.mu.RLock()
	defer s.mu.RUnlock()

	queryVector := simpleEmbed(query)

	type scoredDoc struct {
		doc   *Document
		score float64
	}

	var results []scoredDoc
	for _, doc := range s.documents {
		score := cosineSimilarity(queryVector, doc.Vector)
		results = append(results, scoredDoc{doc: doc, score: score})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	if topK > len(results) {
		topK = len(results)
	}

	docs := make([]*Document, topK)
	for i := 0; i < topK; i++ {
		docs[i] = results[i].doc
	}
	return docs
}

func simpleEmbed(text string) []float64 {
	text = strings.ToLower(text)
	words := strings.Fields(text)

	vector := make([]float64, 64)
	for _, word := range words {
		for i, ch := range word {
			idx := (i + int(ch)) % 64
			vector[idx] += 1.0
		}
	}

	norm := 0.0
	for _, v := range vector {
		norm += v * v
	}
	norm = math.Sqrt(norm)
	if norm > 0 {
		for i := range vector {
			vector[i] /= norm
		}
	}

	return vector
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
