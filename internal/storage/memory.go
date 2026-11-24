package storage

import "sync"

type UserURL struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type MemoryStorage struct {
	mu       sync.RWMutex
	urls     map[string]string
	userURLs map[string][]UserURL
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		urls:     make(map[string]string),
		userURLs: make(map[string][]UserURL),
	}
}

func (s *MemoryStorage) Get(id string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	originalURL, ok := s.urls[id]
	return originalURL, ok
}

func (s *MemoryStorage) Save(userID, id string, userURL UserURL) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.urls[id]; exists {
		return false
	}

	s.urls[id] = userURL.OriginalURL
	s.userURLs[userID] = append(s.userURLs[userID], userURL)

	return true
}

func (s *MemoryStorage) GetUserURLs(userID string) []UserURL {
	s.mu.RLock()
	defer s.mu.RUnlock()

	urls := s.userURLs[userID]

	result := make([]UserURL, len(urls))
	copy(result, urls)

	return result
}
