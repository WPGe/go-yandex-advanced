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
	"github.com/WPGe/go-yandex-advanced/internal/service"
	"log"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/shirou/gopsutil/v3/mem"
	"go.uber.org/zap"

	"github.com/WPGe/go-yandex-advanced/internal/model"
	"github.com/WPGe/go-yandex-advanced/internal/storage"
)

type Agent struct {
	logger         *zap.Logger
	storage        *storage.MemStorage
	hookPath       string
	hashKey        string
	reportInterval time.Duration
	pollInterval   time.Duration
	rateLimit      int
}

func NewAgent(
	logger *zap.Logger,
	storage *storage.MemStorage,
	hookPath string,
	hashKey string,
	reportInterval time.Duration,
	pollInterval time.Duration,
	rateLimit int) *Agent {
	return &Agent{
		logger:         logger,
		storage:        storage,
		hookPath:       hookPath,
		hashKey:        hashKey,
		reportInterval: reportInterval,
		pollInterval:   pollInterval,
		rateLimit:      rateLimit,
	}
}

func (a *Agent) MetricAgent(ctx context.Context, stopCh <-chan struct{}) {
	go a.collectMetricsRoutine(stopCh)

	metrics := a.prepareMetrics(stopCh)
	go a.sendMetricsRoutine(ctx, metrics)
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

func (a *Agent) collectMetricsRoutine(stop <-chan struct{}) {
	t := time.NewTicker(a.pollInterval * time.Second)

	for {
		select {
		case <-stop:
			return
		case <-t.C:
			a.collectMetrics()
		}
	}
}

func (a *Agent) prepareMetrics(stop <-chan struct{}) <-chan model.MetricsStore {
	ch := make(chan model.MetricsStore)
	wg := &sync.WaitGroup{}
	t := time.NewTicker(a.reportInterval * time.Second)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				metrics, err := a.storage.GetAllMetrics()
				if err != nil {
					a.logger.Error("Get all error:", zap.Error(err))
					return
				}
				ch <- metrics
			}
		}
	}()

	go func() {
		wg.Wait()
		close(ch)
	}()

	return ch
}

func (a *Agent) sendMetricsRoutine(ctx context.Context, metrics <-chan model.MetricsStore) {
	for i := 0; i < a.rateLimit; i++ {
		go a.Retry(ctx, 3, func(ctx context.Context) error {
			return a.sendMetrics(metrics)
		}, 1*time.Second, 3*time.Second, 5*time.Second)
	}
}

func (a *Agent) collectMetrics() {
	a.collectGaugeRuntimeMetrics()
	a.collectCounterRuntimeMetrics()
	a.collectGopsutilRuntimeMetrics()
}

func (a *Agent) collectGaugeRuntimeMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	a.addMetricToStorage(model.NewGaugeMetric("Alloc", float64(m.Alloc)))
	a.addMetricToStorage(model.NewGaugeMetric("BuckHashSys", float64(m.BuckHashSys)))
	a.addMetricToStorage(model.NewGaugeMetric("Frees", float64(m.Frees)))
	a.addMetricToStorage(model.NewGaugeMetric("GCCPUFraction", m.GCCPUFraction))
	a.addMetricToStorage(model.NewGaugeMetric("GCSys", float64(m.GCSys)))
	a.addMetricToStorage(model.NewGaugeMetric("HeapAlloc", float64(m.HeapAlloc)))
	a.addMetricToStorage(model.NewGaugeMetric("HeapIdle", float64(m.HeapIdle)))
	a.addMetricToStorage(model.NewGaugeMetric("HeapInuse", float64(m.HeapInuse)))
	a.addMetricToStorage(model.NewGaugeMetric("HeapObjects", float64(m.HeapObjects)))
	a.addMetricToStorage(model.NewGaugeMetric("HeapReleased", float64(m.HeapReleased)))
	a.addMetricToStorage(model.NewGaugeMetric("HeapSys", float64(m.HeapSys)))
	a.addMetricToStorage(model.NewGaugeMetric("LastGC", float64(m.LastGC)))
	a.addMetricToStorage(model.NewGaugeMetric("Lookups", float64(m.Lookups)))
	a.addMetricToStorage(model.NewGaugeMetric("MCacheInuse", float64(m.MCacheInuse)))
	a.addMetricToStorage(model.NewGaugeMetric("MCacheSys", float64(m.MCacheSys)))
	a.addMetricToStorage(model.NewGaugeMetric("MSpanInuse", float64(m.MSpanInuse)))
	a.addMetricToStorage(model.NewGaugeMetric("MSpanSys", float64(m.MSpanSys)))
	a.addMetricToStorage(model.NewGaugeMetric("Mallocs", float64(m.Mallocs)))
	a.addMetricToStorage(model.NewGaugeMetric("NextGC", float64(m.NextGC)))
	a.addMetricToStorage(model.NewGaugeMetric("NumForcedGC", float64(m.NumForcedGC)))
	a.addMetricToStorage(model.NewGaugeMetric("NumGC", float64(m.NumGC)))
	a.addMetricToStorage(model.NewGaugeMetric("OtherSys", float64(m.OtherSys)))
	a.addMetricToStorage(model.NewGaugeMetric("PauseTotalNs", float64(m.PauseTotalNs)))
	a.addMetricToStorage(model.NewGaugeMetric("StackInuse", float64(m.StackInuse)))
	a.addMetricToStorage(model.NewGaugeMetric("StackSys", float64(m.StackSys)))
	a.addMetricToStorage(model.NewGaugeMetric("Sys", float64(m.Sys)))
	a.addMetricToStorage(model.NewGaugeMetric("TotalAlloc", float64(m.TotalAlloc)))
	a.addMetricToStorage(model.NewGaugeMetric("RandomValue", rand.Float64()))
}

func (a *Agent) collectCounterRuntimeMetrics() {
	a.addMetricToStorage(model.NewCounterMetric("PollCount", 1))
}

func (a *Agent) collectGopsutilRuntimeMetrics() {
	v, err := mem.VirtualMemory()
	if err != nil {
		return
	}

	a.addMetricToStorage(model.NewGaugeMetric("TotalMemory", float64(v.Total)))
	a.addMetricToStorage(model.NewGaugeMetric("FreeMemory", float64(v.Free)))
	a.addMetricToStorage(model.NewGaugeMetric("CPUutilization1", v.UsedPercent))
}

func (a *Agent) addMetricToStorage(metric model.Metric) {
	err := a.storage.AddMetric(metric)
	if err != nil {
		a.logger.Fatal("Add error", zap.Error(err))
	}
}

func (a *Agent) sendMetrics(metrics <-chan model.MetricsStore) error {
	for {
		allMetrics, ok := <-metrics
		if !ok || len(allMetrics) == 0 {
			return nil
		} else {
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
		}
	}
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
