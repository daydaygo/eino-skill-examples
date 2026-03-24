package main

import "context"

func (s *MemoryStore) IndexDocuments(ctx context.Context, documents []string) {
	for i, doc := range documents {
		s.Index(ctx, doc, map[string]any{"index": i})
	}
}

func (s *MemoryStore) IndexWithMetadata(ctx context.Context, content string, metadata map[string]any) {
	s.Index(ctx, content, metadata)
}

func (s *MemoryStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.documents = make(map[string]*Document)
}

func (s *MemoryStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.documents)
}
