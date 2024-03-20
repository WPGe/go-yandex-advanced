package main

import (
	"github.com/WPGe/go-yandex-advanced/internal/service"
	"github.com/WPGe/go-yandex-advanced/internal/utils"
	"log"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/WPGe/go-yandex-advanced/internal/agent"
	"github.com/WPGe/go-yandex-advanced/internal/entity"
	"github.com/WPGe/go-yandex-advanced/internal/handler"
	"github.com/WPGe/go-yandex-advanced/internal/storage"
)

func TestAgent_MetricAgent(t *testing.T) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}
	logger.Sync()

	agentStorage := storage.NewMemStorageWithMetrics(make(map[string]map[string]entity.Metric), logger)
	serverStorage := storage.NewMemStorageWithMetrics(make(map[string]map[string]entity.Metric), logger)

	srv := service.New(serverStorage)
	server := httptest.NewServer(utils.WithGzip(handler.MetricUpdatesHandler(srv, logger)))
	defer server.Close()

	stopCh := make(chan struct{})
	agentStruct := agent.NewAgent(logger, agentStorage, server.URL+"/updates")
	go agentStruct.MetricAgent(1, 1, stopCh)

	time.Sleep(1 * time.Second)
	close(stopCh)

	assert.Equal(t, agentStorage, serverStorage)
}
