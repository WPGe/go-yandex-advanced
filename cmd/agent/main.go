package main

import (
	"github.com/WPGe/go-yandex-advanced/internal/agent"
	"github.com/WPGe/go-yandex-advanced/internal/repository"
	"github.com/WPGe/go-yandex-advanced/internal/storage"
)

func main() {
	parseFlags()

	memStorage := storage.NewMemStorage()
	repo := repository.MetricRepository(memStorage)

	stopCh := make(chan struct{})
	go agent.MetricAgent(repo, "http://"+flagRunAddr+"/update", flagReportInterval, flagPollInterval, stopCh)
	select {}
}
