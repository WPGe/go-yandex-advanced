package main

import (
	"github.com/WPGe/go-yandex-advanced/internal/handler"
	"github.com/WPGe/go-yandex-advanced/internal/repository"
	"github.com/WPGe/go-yandex-advanced/internal/storage"
	"net/http"
)

func main() {
	memStorage := storage.NewMemStorage()
	repo := repository.MetricRepository(memStorage)

	http.Handle("/update/", handler.MetricUpdateHandler(repo))
	http.ListenAndServe(":8080", nil)
}
