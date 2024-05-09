package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"

	"github.com/WPGe/go-yandex-advanced/internal/model"
	"github.com/WPGe/go-yandex-advanced/internal/service"
	"github.com/WPGe/go-yandex-advanced/internal/storage"
)

type Agent struct {
	logger   *zap.Logger
	storage  *storage.MemStorage
	hookPath string
	hashKey  string
}

func NewAgent(logger *zap.Logger, storage *storage.MemStorage, hookPath string, hashKey string) *Agent {
	return &Agent{
		logger:   logger,
		storage:  storage,
		hookPath: hookPath,
		hashKey:  hashKey,
	}
}

func (a *Agent) MetricAgent(reportInterval time.Duration, pollInterval time.Duration, stopCh <-chan struct{}) {
	pollTicker := time.NewTicker(pollInterval * time.Second)
	sendTicker := time.NewTicker(reportInterval * time.Second)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGKILL, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	for {
		select {
		case <-pollTicker.C:
			a.collectGaugeRuntimeMetrics()
			a.increasePollIteration()
		case <-sendTicker.C:
			err := a.Retry(ctx, 3, func(ctx context.Context) error {
				err := a.sendMetrics()
				if err != nil {
					a.logger.Error("Send error:", zap.Error(err))
				}

				err = a.storage.ClearMetrics()
				if err != nil {
					a.logger.Error("Clear error:", zap.Error(err))
				}
				return err
			})
			if err != nil {
				return
			}
		case <-ctx.Done():
			a.logger.Error("Send stop:", zap.Error(ctx.Err()))
			pollTicker.Stop()
			sendTicker.Stop()
			return
		case <-stopCh:
			err := a.sendMetrics()
			if err != nil {
				a.logger.Error("Send error:", zap.Error(err))
			}
			pollTicker.Stop()
			sendTicker.Stop()
			return
		}
	}
}

func (a *Agent) Retry(ctx context.Context, maxRetries int, fn func(ctx context.Context) error, intervals ...time.Duration) error {
	var err error
	err = fn(ctx)
	if err == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	for i := 0; i < maxRetries; i++ {
		a.logger.Info("Retrying", zap.Int("Attempt", i+1))

		t := time.NewTimer(intervals[i])
		select {
		case <-t.C:
			if err = fn(ctx); err == nil {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	a.logger.Error("Retrying failed", zap.Error(err))
	return err
}

func (a *Agent) collectGaugeRuntimeMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	a.addGaugeMetricToStorage("Alloc", float64(m.Alloc))
	a.addGaugeMetricToStorage("BuckHashSys", float64(m.BuckHashSys))
	a.addGaugeMetricToStorage("Frees", float64(m.Frees))
	a.addGaugeMetricToStorage("GCCPUFraction", float64(m.GCCPUFraction))
	a.addGaugeMetricToStorage("GCSys", float64(m.GCSys))
	a.addGaugeMetricToStorage("HeapAlloc", float64(m.HeapAlloc))
	a.addGaugeMetricToStorage("HeapIdle", float64(m.HeapIdle))
	a.addGaugeMetricToStorage("HeapInuse", float64(m.HeapInuse))
	a.addGaugeMetricToStorage("HeapObjects", float64(m.HeapObjects))
	a.addGaugeMetricToStorage("HeapReleased", float64(m.HeapReleased))
	a.addGaugeMetricToStorage("HeapSys", float64(m.HeapSys))
	a.addGaugeMetricToStorage("LastGC", float64(m.LastGC))
	a.addGaugeMetricToStorage("Lookups", float64(m.Lookups))
	a.addGaugeMetricToStorage("MCacheInuse", float64(m.MCacheInuse))
	a.addGaugeMetricToStorage("MCacheSys", float64(m.MCacheSys))
	a.addGaugeMetricToStorage("MSpanInuse", float64(m.MSpanInuse))
	a.addGaugeMetricToStorage("MSpanSys", float64(m.MSpanSys))
	a.addGaugeMetricToStorage("Mallocs", float64(m.Mallocs))
	a.addGaugeMetricToStorage("NextGC", float64(m.NextGC))
	a.addGaugeMetricToStorage("NumForcedGC", float64(m.NumForcedGC))
	a.addGaugeMetricToStorage("NumGC", float64(m.NumGC))
	a.addGaugeMetricToStorage("OtherSys", float64(m.OtherSys))
	a.addGaugeMetricToStorage("PauseTotalNs", float64(m.PauseTotalNs))
	a.addGaugeMetricToStorage("StackInuse", float64(m.StackInuse))
	a.addGaugeMetricToStorage("StackSys", float64(m.StackSys))
	a.addGaugeMetricToStorage("Sys", float64(m.Sys))
	a.addGaugeMetricToStorage("TotalAlloc", float64(m.TotalAlloc))
	a.addGaugeMetricToStorage("RandomValue", rand.Float64())
}

func (a *Agent) addGaugeMetricToStorage(name string, value float64) {
	metric := model.Metric{
		MType: model.Gauge,
		ID:    name,
		Value: &value,
	}

	err := a.storage.AddMetric(metric)
	if err != nil {
		a.logger.Fatal("Add gauge error", zap.Error(err))
	}
}

func (a *Agent) addCounterMetricToStorage(name string, value int64) {
	metric := model.Metric{
		MType: model.Counter,
		ID:    name,
		Delta: &value,
	}

	err := a.storage.AddMetric(metric)
	if err != nil {
		a.logger.Fatal("Add counter error", zap.Error(err))
	}
}

func (a *Agent) increasePollIteration() {
	a.addCounterMetricToStorage("PollCount", 1)
}

func (a *Agent) sendMetrics() error {
	allMetrics, err := a.storage.GetAllMetrics()
	if err != nil {
		a.logger.Error("Get all error:", zap.Error(err))
		return err
	}

	var metricsForSend []model.Metric
	for _, typedMetrics := range allMetrics {
		for _, metric := range typedMetrics {
			metricsForSend = append(metricsForSend, metric)
		}
	}
	jsonMetrics, err := json.Marshal(metricsForSend)
	if err != nil {
		a.logger.Error("Marshaling error:", zap.Error(err))
		return err
	}

	var gzippedMetric bytes.Buffer
	zb := gzip.NewWriter(&gzippedMetric)

	_, err = zb.Write(jsonMetrics)
	if err != nil {
		a.logger.Error("Failed to gzip metrics:", zap.Error(err))
		return err
	}
	err = zb.Close()
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/", a.hookPath)
	req := resty.New().R()
	req.Method = http.MethodPost
	req.URL = url
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	if a.hashKey != "" {
		h := hmac.New(sha256.New, []byte(a.hashKey))
		h.Write(jsonMetrics)
		hash := h.Sum(nil)
		req.Header.Set("HashSHA256", hex.EncodeToString(hash))
	}

	req.SetBody(gzippedMetric.Bytes())

	res, err := req.Send()
	if err != nil {
		a.logger.Error("Failed to send metrics", zap.Error(err))
		return err
	}
	if res.StatusCode() != http.StatusOK {
		a.logger.Error("Failed to send metric: wrong response code: ", zap.Int("status", res.StatusCode()))
	}

	return nil
}

func SaveMetricsInFileAgent(storage service.Repository, fileStoragePath string, storeInterval time.Duration, ctx context.Context) error {
	ticker := time.NewTicker(storeInterval * time.Second)

	for {
		select {
		case <-ticker.C:
			if err := saveMetricsInFile(storage, fileStoragePath); err != nil {
				return fmt.Errorf("failed to save metrics: %v", err)
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func saveMetricsInFile(repo service.Repository, fileStoragePath string) error {
	metrics, err := repo.GetAllMetrics()
	if err != nil {
		return fmt.Errorf("failed to get metrics: %v", err)
	}

	file, err := os.OpenFile(fileStoragePath, os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Error closing file: %v", err)
		}
	}()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(metrics); err != nil {
		return fmt.Errorf("failed to encode metrics: %v", err)
	}

	return nil
}
