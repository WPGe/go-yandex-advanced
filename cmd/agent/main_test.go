package main

import (
	"context"
	"github.com/WPGe/go-yandex-advanced/internal/model"
	"github.com/WPGe/go-yandex-advanced/internal/service"
	"github.com/WPGe/go-yandex-advanced/internal/utils"
	"log"
	"net/http/httptest"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/WPGe/go-yandex-advanced/internal/agent"
	"github.com/WPGe/go-yandex-advanced/internal/handler"
	"github.com/WPGe/go-yandex-advanced/internal/storage"
)

func TestAgent_MetricAgent(t *testing.T) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}
	logger.Sync()

	agentStorage := storage.NewMemStorageWithMetrics(make(map[string]map[string]model.Metric), logger)
	serverStorage := storage.NewMemStorageWithMetrics(make(map[string]map[string]model.Metric), logger)

	srv := service.New(serverStorage)

	gzipMiddleware := utils.WithGzip()
	server := httptest.NewServer(gzipMiddleware(handler.MetricUpdatesHandler(srv, logger)))
	defer server.Close()

	stopCh := make(chan struct{})

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGKILL, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	agentStruct := agent.NewAgent(
		logger, agentStorage,
		server.URL+"/updates",
		"",
		time.Duration(3),
		time.Duration(2),
		3)
	agentStruct.MetricAgent(ctx, stopCh)

	time.Sleep(3*time.Second + 500*time.Millisecond)
	close(stopCh)

	time.Sleep(2 * time.Second)

	assert.Equal(t, agentStorage, serverStorage)
}
