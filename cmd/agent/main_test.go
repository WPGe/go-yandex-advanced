package main

import (
	"github.com/WPGe/go-yandex-advanced/internal/agent"
	"github.com/WPGe/go-yandex-advanced/internal/entity"
	"github.com/WPGe/go-yandex-advanced/internal/handler"
	"github.com/WPGe/go-yandex-advanced/internal/storage"
	"github.com/stretchr/testify/assert"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAgent_MetricAgent(t *testing.T) {
	agentStorage := storage.NewMemStorageWithMetrics(make(map[string]entity.Metric))
	serverStorage := storage.NewMemStorageWithMetrics(make(map[string]entity.Metric))

	server := httptest.NewServer(handler.MetricUpdateHandler(serverStorage))
	defer server.Close()

	stopCh := make(chan struct{})
	go agent.MetricAgent(agentStorage, server.URL+"/update", stopCh)

	time.Sleep(1 * time.Second)
	close(stopCh)

	assert.Equal(t, agentStorage, serverStorage)
}
