package main

import (
	"github.com/WPGe/go-yandex-advanced/internal/agent"
	"github.com/WPGe/go-yandex-advanced/internal/storage"
	"time"
)

func main() {
	parseFlags()

	memStorage := storage.NewMemStorage()

	stopCh := make(chan struct{})
	agent.MetricAgent(memStorage, "http://"+flagRunAddr+"/update", time.Duration(flagReportInterval), time.Duration(flagPollInterval), stopCh)
}
