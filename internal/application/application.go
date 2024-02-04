package application

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/WPGe/go-yandex-advanced/internal/agent"
	"github.com/WPGe/go-yandex-advanced/internal/config"
	"github.com/WPGe/go-yandex-advanced/internal/handler"
	"github.com/WPGe/go-yandex-advanced/internal/storage"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

var sugar zap.SugaredLogger

type Server struct {
	srv    *http.Server
	logger *zap.Logger
}

func NewServer(log *zap.Logger, addr string) *Server {
	return &Server{
		srv:    &http.Server{Addr: addr},
		logger: log,
	}
}

func (s *Server) InitHandlers(rep handler.Repository, db *sql.DB) {
	r := chi.NewRouter()
	r.Post("/update/", handler.WithGzip(handler.WithLogging(handler.MetricUpdateHandler(rep), sugar)))
	r.Post("/update/{type}/{name}/{value}", handler.WithGzip(handler.WithLogging(handler.MetricUpdateHandler(rep), sugar)))
	r.Get("/value/{type}/{name}", handler.WithGzip(handler.WithLogging(handler.MetricGetHandler(rep), sugar)))
	r.Post("/value/", handler.WithGzip(handler.WithLogging(handler.MetricPostHandler(rep), sugar)))
	r.Get("/", handler.WithGzip(handler.WithLogging(handler.MetricGetAllHandler(rep), sugar)))
	r.Get("/ping", handler.PingDb(db, s.logger))

	s.srv.Handler = r
}

func Run() {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)

		<-c
		cancel()
	}()

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
	sugar = *logger.Sugar()

	cfg, err := config.NewServer()
	if err != nil {
		logger.Error("Init config error", zap.Error(err))
	}

	var db *sql.DB
	if cfg.DatabaseDSN != "" {
		db, err = ConnectDB(&cfg)
		if err != nil {
			logger.Error("DB init error", zap.Error(err))
			return
		}
		defer func(db *sql.DB) {
			err := db.Close()
			if err != nil {
				logger.Error("Close db error", zap.Error(err))
			}
		}(db)
	}

	var memStorage *storage.MemStorage
	if cfg.Restore {
		memStorage = storage.NewMemStorageFromFile(filepath.Join(cfg.RootDir, cfg.FileStoragePath))
	} else {
		memStorage = storage.NewMemStorage()
	}

	server := NewServer(logger, cfg.Address)
	server.InitHandlers(memStorage, db)

	logger.Info("Starting server", zap.String("addr", cfg.Address))

	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return server.srv.ListenAndServe()
	})
	g.Go(func() error {
		<-gCtx.Done()
		return server.srv.Shutdown(context.Background())
	})
	g.Go(func() error {
		// Запускаем агент с использованием контекста
		return agent.SaveMetricsInFileAgent(memStorage, filepath.Join(cfg.RootDir, cfg.FileStoragePath), time.Duration(cfg.StoreInterval), gCtx)
	})

	if err := g.Wait(); err != nil {
		logger.Fatal("Start server", zap.Error(err))
		fmt.Printf("exit reason: %s \n", err)
	}
}

func ConnectDB(cfg *config.Config) (*sql.DB, error) {
	db, err := sql.Open("pgx", cfg.DatabaseDSN)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}
