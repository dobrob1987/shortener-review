package main

import (
	"log"

	"shortener/internal/app"
	"shortener/internal/storage"
)

func main() {
	store := storage.NewMemoryStorage()
	application := app.New(store)

	log.Println("Запускаем сервер на http://localhost:8080")

	err := application.Run()
	if err != nil {
		log.Fatal(err)
	}
}
