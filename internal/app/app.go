package app

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"

	"shortener/internal/auth"
	"shortener/internal/storage"
)

const baseURL = "http://localhost:8080"

type shortenRequest struct {
	URL string `json:"url"`
}

type shortenResponse struct {
	Result string `json:"result"`
}

type App struct {
	store *storage.MemoryStorage
}

func New(store *storage.MemoryStorage) *App {
	return &App{
		store: store,
	}
}

func (a *App) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/shorten", a.shorten)
	mux.HandleFunc("GET /{id}", a.redirect)
	mux.HandleFunc("GET /api/user/urls", a.userURLs)

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	return mux
}

func (a *App) Run() error {
	return http.ListenAndServe("localhost:8080", a.Handler())
}

func (a *App) shorten(w http.ResponseWriter, r *http.Request) {
	var req shortenRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "некорректный JSON", http.StatusBadRequest)
		return
	}
	if req.URL == "" {
		http.Error(w, "поле url обязательно", http.StatusBadRequest)
		return
	}

	parsedURL, err := url.Parse(req.URL)
	if err != nil {
		http.Error(w, "некорректный URL", http.StatusBadRequest)
		return
	}

	if (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") ||
		parsedURL.Hostname() == "" {
		http.Error(w, "нужна ссылка http или https с именем хоста", http.StatusBadRequest)
		return
	}

	userID := auth.GetOrCreateUser(w, r)

	var id string
	saved := false

	for i := 0; i < 5; i++ {
		id = a.newID()

		link := storage.UserURL{
			ShortURL:    baseURL + "/" + id,
			OriginalURL: req.URL,
		}

		if a.store.Save(userID, id, link) {
			saved = true
			break
		}
	}

	if !saved {
		http.Error(w, "не удалось создать короткую ссылку", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	err = json.NewEncoder(w).Encode(shortenResponse{
		Result: baseURL + "/" + id,
	})
	if err != nil {
		log.Printf("не удалось отправить ответ создания ссылки: %v", err)
		return
	}
}

func (a *App) newID() string {
	return rand.Text()[:8]
}

func (a *App) redirect(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	originalURL, found := a.store.Get(id)
	if !found {
		http.NotFound(w, r)
		return
	}

	http.Redirect(w, r, originalURL, http.StatusTemporaryRedirect)
}

func (a *App) userURLs(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("user_id")
	if err == nil {
		_, parseErr := auth.ParseCookie(cookie.Value)
		if errors.Is(parseErr, auth.ErrMalformedCookie) {
			http.Error(w, "некорректный формат cookie", http.StatusUnauthorized)
			return
		}
	}

	userID := auth.GetOrCreateUser(w, r)
	urls := a.store.GetUserURLs(userID)

	if len(urls) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(urls); err != nil {
		log.Printf("не удалось отправить список ссылок: %v", err)
		return
	}
}
