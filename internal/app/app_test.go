package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"shortener/internal/auth"
	"shortener/internal/storage"
)

func TestShorten(t *testing.T) {
	application := New(storage.NewMemoryStorage())

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/shorten",
		strings.NewReader(`{"url":"https://example.com/"}`),
	)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	application.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("получили статус %d, ожидали 201", rec.Code)
	}

	var response shortenResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("не удалось прочитать JSON ответа: %v", err)
	}

	if !strings.HasPrefix(response.Result, baseURL+"/") {
		t.Errorf("неправильная короткая ссылка: %q", response.Result)
	}
}

func TestRedirect(t *testing.T) {
	store := storage.NewMemoryStorage()

	link := storage.UserURL{
		ShortURL:    baseURL + "/abc123",
		OriginalURL: "https://example.com/",
	}

	if !store.Save("user1", "abc123", link) {
		t.Fatal("не удалось подготовить ссылку для теста")
	}

	application := New(store)

	req := httptest.NewRequest(http.MethodGet, "/abc123", nil)
	rec := httptest.NewRecorder()

	application.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusTemporaryRedirect {
		t.Fatalf("получили статус %d, ожидали 307", rec.Code)
	}

	location := rec.Header().Get("Location")
	if location != link.OriginalURL {
		t.Errorf("получили Location %q, ожидали %q", location, link.OriginalURL)
	}
}

func TestRedirectNotFound(t *testing.T) {
	application := New(storage.NewMemoryStorage())

	req := httptest.NewRequest(http.MethodGet, "/missing123", nil)
	rec := httptest.NewRecorder()

	application.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf(
			"получили статус %d, ожидали 404; тело ответа: %s",
			rec.Code,
			rec.Body.String(),
		)
	}
}

func TestShortenSetsCookie(t *testing.T) {
	application := New(storage.NewMemoryStorage())

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/shorten",
		strings.NewReader(`{"url":"https://example.com/"}`),
	)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	application.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("получили статус %d, ожидали 201", rec.Code)
	}

	response := rec.Result()
	defer response.Body.Close()

	for _, cookie := range response.Cookies() {
		if cookie.Name == "user_id" {
			if cookie.Value == "" {
				t.Fatal("cookie user_id имеет пустое значение")
			}
			return
		}
	}

	t.Fatal("сервер не выдал cookie user_id")
}

func TestUserURLsEmpty(t *testing.T) {
	application := New(storage.NewMemoryStorage())

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/user/urls",
		nil,
	)
	rec := httptest.NewRecorder()

	application.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf(
			"получили статус %d, ожидали 204; тело ответа: %s",
			rec.Code,
			rec.Body.String(),
		)
	}

	if rec.Body.Len() != 0 {
		t.Errorf(
			"ожидали пустое тело ответа, получили: %q",
			rec.Body.String(),
		)
	}
}

func TestUserURLsMalformedCookie(t *testing.T) {
	application := New(storage.NewMemoryStorage())

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/user/urls",
		nil,
	)
	req.AddCookie(&http.Cookie{
		Name:  "user_id",
		Value: "broken",
	})

	rec := httptest.NewRecorder()
	application.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf(
			"получили статус %d, ожидали 401; тело ответа: %s",
			rec.Code,
			rec.Body.String(),
		)
	}
}

func TestUserURLs(t *testing.T) {
	application := New(storage.NewMemoryStorage())
	handler := application.Handler()

	// Создаём ссылку.
	createReq := httptest.NewRequest(
		http.MethodPost,
		"/api/shorten",
		strings.NewReader(`{"url":"https://example.com/"}`),
	)
	createReq.Header.Set("Content-Type", "application/json")

	createRec := httptest.NewRecorder()
	handler.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("создание ссылки: получили %d, ожидали 201", createRec.Code)
	}

	var created shortenResponse
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("не удалось прочитать ответ создания: %v", err)
	}

	createResponse := createRec.Result()
	defer createResponse.Body.Close()

	var userCookie *http.Cookie
	for _, cookie := range createResponse.Cookies() {
		if cookie.Name == "user_id" {
			userCookie = cookie
			break
		}
	}
	if userCookie == nil {
		t.Fatal("сервер не выдал cookie user_id")
	}

	// Запрашиваем список от имени того же пользователя.
	listReq := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	listReq.AddCookie(userCookie)

	listRec := httptest.NewRecorder()
	handler.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf(
			"получение списка: получили %d, ожидали 200; тело: %s",
			listRec.Code,
			listRec.Body.String(),
		)
	}

	var links []storage.UserURL
	if err := json.Unmarshal(listRec.Body.Bytes(), &links); err != nil {
		t.Fatalf("не удалось прочитать список ссылок: %v", err)
	}

	if len(links) != 1 {
		t.Fatalf("получили %d ссылок, ожидали 1", len(links))
	}

	if links[0].OriginalURL != "https://example.com/" {
		t.Errorf("неожиданный исходный адрес: %q", links[0].OriginalURL)
	}

	if links[0].ShortURL != created.Result {
		t.Errorf(
			"получили короткую ссылку %q, ожидали %q",
			links[0].ShortURL,
			created.Result,
		)
	}
}

func TestShortenBadRequest(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "invalid_json",
			body: `{"url":`,
		},
		{
			name: "empty_url",
			body: `{"url":""}`,
		},
		{
			name: "unsupported_scheme",
			body: `{"url":"ftp://example.com/file"}`,
		},
		{
			name: "missing_host",
			body: `{"url":"https://"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			application := New(storage.NewMemoryStorage())

			req := httptest.NewRequest(
				http.MethodPost,
				"/api/shorten",
				strings.NewReader(tt.body),
			)
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			application.Handler().ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf(
					"получили статус %d, ожидали 400; тело: %s",
					rec.Code,
					rec.Body.String(),
				)
			}
		})
	}
}

func TestUserURLsDoesNotExposeOtherUsersLinks(t *testing.T) {
	store := storage.NewMemoryStorage()

	ownerCookie := auth.NewCookie()
	ownerID, err := auth.ParseCookie(ownerCookie.Value)
	if err != nil {
		t.Fatalf("не удалось прочитать cookie владельца: %v", err)
	}

	link := storage.UserURL{
		ShortURL:    baseURL + "/private123",
		OriginalURL: "https://example.com/",
	}
	if !store.Save(ownerID, "private123", link) {
		t.Fatal("не удалось сохранить ссылку владельца")
	}

	application := New(store)

	otherCookie := auth.NewCookie()
	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	req.AddCookie(otherCookie)

	rec := httptest.NewRecorder()
	application.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf(
			"получили статус %d, ожидали 204; тело: %s",
			rec.Code,
			rec.Body.String(),
		)
	}

	if rec.Body.Len() != 0 {
		t.Errorf("ожидали пустое тело, получили: %q", rec.Body.String())
	}
}

func TestUserURLsInvalidSignature(t *testing.T) {
	application := New(storage.NewMemoryStorage())

	cookie := auth.NewCookie()
	cookie.Value += "x"
	oldValue := cookie.Value

	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	req.AddCookie(cookie)

	rec := httptest.NewRecorder()
	application.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf(
			"получили статус %d, ожидали 204; тело: %s",
			rec.Code,
			rec.Body.String(),
		)
	}

	response := rec.Result()
	defer response.Body.Close()

	for _, newCookie := range response.Cookies() {
		if newCookie.Name != "user_id" {
			continue
		}

		if newCookie.Value == oldValue {
			t.Fatal("сервер вернул прежнюю подделанную cookie")
		}

		if _, err := auth.ParseCookie(newCookie.Value); err != nil {
			t.Fatalf("новая cookie не прошла проверку: %v", err)
		}

		return
	}

	t.Fatal("сервер не выдал новую cookie user_id")
}
