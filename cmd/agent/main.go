package main

import (
	"log"
	"time"

	"go.uber.org/zap"

	"github.com/WPGe/go-yandex-advanced/internal/agent"
	"github.com/WPGe/go-yandex-advanced/internal/config"
	"github.com/WPGe/go-yandex-advanced/internal/storage"
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

	agentStruct := agent.NewAgent(logger, memStorage, "http://"+cfg.Address+"/updates", cfg.HashKey)
	agentStruct.MetricAgent(time.Duration(cfg.ReportInterval), time.Duration(cfg.PollInterval), stopCh)
}
