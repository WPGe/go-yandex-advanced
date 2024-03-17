package application

import (
	"context"
	"database/sql"
	"github.com/WPGe/go-yandex-advanced/internal/utils"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/WPGe/go-yandex-advanced/internal/agent"
	"github.com/WPGe/go-yandex-advanced/internal/config"
	"github.com/WPGe/go-yandex-advanced/internal/handler"
	"github.com/WPGe/go-yandex-advanced/internal/service"
	"github.com/WPGe/go-yandex-advanced/internal/storage"
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

func (s *Server) InitHandlers(srv handler.Service, db *sql.DB) {
	r := chi.NewRouter()
	r.Post("/update/", utils.WithGzip(utils.WithLogging(handler.MetricUpdateHandler(srv, s.logger), sugar)))
	r.Post("/updates/", utils.WithGzip(utils.WithLogging(handler.MetricUpdatesHandler(srv, s.logger), sugar)))
	r.Post("/update/{type}/{name}/{value}", utils.WithGzip(utils.WithLogging(handler.MetricUpdateHandler(srv, s.logger), sugar)))
	r.Get("/value/{type}/{name}", utils.WithGzip(utils.WithLogging(handler.MetricGetHandler(srv, s.logger), sugar)))
	r.Post("/value/", utils.WithGzip(utils.WithLogging(handler.MetricPostHandler(srv, s.logger), sugar)))
	r.Get("/", utils.WithGzip(utils.WithLogging(handler.MetricGetAllHandler(srv, s.logger), sugar)))
	r.Get("/ping", handler.PingDB(db, s.logger))

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

	var repo service.Repository
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
		repo = storage.NewDbStorage(logger, db)
	} else {
		if cfg.Restore {
			repo = storage.NewMemStorageFromFile(filepath.Join(cfg.RootDir, cfg.FileStoragePath), logger)
		} else {
			repo = storage.NewMemStorage(logger)
		}
	}

	srv := service.New(repo)
	server := NewServer(logger, cfg.Address)
	server.InitHandlers(srv, db)

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
		return agent.SaveMetricsInFileAgent(repo, filepath.Join(cfg.RootDir, cfg.FileStoragePath), time.Duration(cfg.StoreInterval), gCtx)
	})

	if err := g.Wait(); err != nil {
		logger.Fatal("Exit reason:", zap.Error(err))
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
