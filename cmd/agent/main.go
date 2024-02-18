package main

import (
	"github.com/WPGe/go-yandex-advanced/internal/agent"
	"github.com/WPGe/go-yandex-advanced/internal/config"
	"github.com/WPGe/go-yandex-advanced/internal/storage"
	"go.uber.org/zap"
	"log"
	"time"
)

func main() {
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

	cfg, err := config.NewAgent()
	if err != nil {
		logger.Error("Init config error", zap.Error(err))
	}

	logger.Info("Starting agent", zap.String("addr", cfg.Address))

	memStorage := storage.NewMemStorage(logger)

	stopCh := make(chan struct{})
	agent.MetricAgent(memStorage, "http://"+cfg.Address+"/update", time.Duration(cfg.ReportInterval), time.Duration(cfg.PollInterval), stopCh, logger)
}
