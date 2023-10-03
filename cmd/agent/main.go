package main

import (
	"github.com/WPGe/go-yandex-advanced/internal/agent"
	"github.com/WPGe/go-yandex-advanced/internal/repository"
	"github.com/WPGe/go-yandex-advanced/internal/storage"
)

func main() {
	memStorage := storage.NewMemStorage()
	repo := repository.MetricRepository(memStorage)

	stopCh := make(chan struct{})
	go agent.MetricAgent(repo, "http://localhost:8080/update", stopCh)
	select {}
}
