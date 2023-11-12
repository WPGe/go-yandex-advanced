package main

import (
	"context"
	"fmt"
	"github.com/WPGe/go-yandex-advanced/internal/handler"
	"github.com/WPGe/go-yandex-advanced/internal/storage"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

var sugar zap.SugaredLogger

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)

		<-c
		cancel()
	}()

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	sugar = *logger.Sugar()

	parseFlags()

	memStorage := storage.NewMemStorage()

	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", handler.WithLogging(handler.MetricUpdateHandler(memStorage), sugar))
	r.Get("/value/{type}/{name}", handler.WithLogging(handler.MetricGetHandler(memStorage), sugar))
	r.Get("/", handler.WithLogging(handler.MetricGetAllHandler(memStorage), sugar))

	sugar.Infow(
		"Starting server",
		"addr", flagRunAddr,
	)

	httpServer := &http.Server{
		Addr:    flagRunAddr,
		Handler: r,
	}

	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return httpServer.ListenAndServe()
	})
	g.Go(func() error {
		<-gCtx.Done()
		return httpServer.Shutdown(context.Background())
	})

	if err := g.Wait(); err != nil {
		sugar.Fatalw(err.Error(), "event", "start server")
		fmt.Printf("exit reason: %s \n", err)
	}
}
