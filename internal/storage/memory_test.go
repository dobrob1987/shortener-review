package storage

import "testing"

func TestSaveAndGet(t *testing.T) {
	store := NewMemoryStorage()

	link := UserURL{
		ShortURL:    "http://localhost:8080/abc123",
		OriginalURL: "https://example.com/long/path",
	}

	saved := store.Save("user1", "abc123", link)
	if !saved {
		t.Fatal("не удалось сохранить ссылку")
	}

	got, found := store.Get("abc123")
	if !found {
		t.Fatal("сохранённая ссылка не найдена")
	}

	if got != link.OriginalURL {
		t.Errorf("получили %q, ожидали %q", got, link.OriginalURL)
	}
}

func TestGetUserURLs(t *testing.T) {
	store := NewMemoryStorage()

	link := UserURL{
		ShortURL:    "http://localhost:8080/abc123",
		OriginalURL: "https://example.com/long/path",
	}

	if !store.Save("user1", "abc123", link) {
		t.Fatal("не удалось сохранить ссылку")
	}

	urls := store.GetUserURLs("user1")

	if len(urls) != 1 {
		t.Fatalf("получили %d ссылок, ожидали 1", len(urls))
	}

	if urls[0] != link {
		t.Errorf("получили %+v, ожидали %+v", urls[0], link)
	}

	otherURLs := store.GetUserURLs("user2")

	if len(otherURLs) != 0 {
		t.Error("другой пользователь получил чужие ссылки")
	}
}
