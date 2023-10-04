package main

import (
	"github.com/WPGe/go-yandex-advanced/internal/handler"
	"github.com/WPGe/go-yandex-advanced/internal/repository"
	"github.com/WPGe/go-yandex-advanced/internal/storage"
	"github.com/go-chi/chi/v5"
	"log"
	"net/http"
)

func main() {
	memStorage := storage.NewMemStorage()
	repo := repository.MetricRepository(memStorage)

	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", handler.MetricUpdateHandler(repo))
	r.Get("/value/{type}/{name}", handler.MetricGetHandler(repo))
	r.Get("/", handler.MetricGetAllHandler(repo))

	log.Fatal(http.ListenAndServe(":8080", r))
}
