package main

import (
	"github.com/WPGe/go-yandex-advanced/internal/agent"
	"github.com/WPGe/go-yandex-advanced/internal/entity"
	"github.com/WPGe/go-yandex-advanced/internal/handler"
	"github.com/WPGe/go-yandex-advanced/internal/storage"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"log"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAgent_MetricAgent(t *testing.T) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}
	defer func(logger *zap.Logger) {
		err := logger.Sync()
		if err != nil {
			panic(err)
		}
	}(logger)

	agentStorage := storage.NewMemStorageWithMetrics(make(map[string]map[string]entity.Metric), logger)
	serverStorage := storage.NewMemStorageWithMetrics(make(map[string]map[string]entity.Metric), logger)

	server := httptest.NewServer(handler.MetricUpdateHandler(serverStorage, logger))
	defer server.Close()

	stopCh := make(chan struct{})
	go agent.MetricAgent(agentStorage, server.URL+"/updates", 2, 10, stopCh, logger)

	time.Sleep(1 * time.Second)
	close(stopCh)

	assert.Equal(t, agentStorage, serverStorage)
}
