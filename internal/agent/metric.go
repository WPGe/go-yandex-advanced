package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"github.com/WPGe/go-yandex-advanced/internal/entity"
	"github.com/WPGe/go-yandex-advanced/internal/handler"
	"github.com/WPGe/go-yandex-advanced/internal/storage"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
	"log"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"time"
)

func MetricAgent(storage *storage.MemStorage, hookPath string, reportInterval time.Duration, pollInterval time.Duration, stopCh <-chan struct{}, logger *zap.Logger) {
	ticker := time.NewTicker(pollInterval * time.Second)
	sendTicker := time.NewTicker(reportInterval * time.Second)

	for {
		select {
		case <-ticker.C:
			collectGaugeRuntimeMetrics(storage, logger)
			increasePollIteration(storage)
		case <-sendTicker.C:
			err := sendMetrics(storage, hookPath, logger)
			if err != nil {
				logger.Error("Send error:", zap.Error(err))
			}

			err = storage.ClearMetrics()
			if err != nil {
				logger.Error("Clear error:", zap.Error(err))
			}

		case <-stopCh:
			return
		}
	}
}

func collectGaugeRuntimeMetrics(storage *storage.MemStorage, logger *zap.Logger) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	addGaugeMetricToStorage("Alloc", float64(m.Alloc), storage, logger)
	addGaugeMetricToStorage("BuckHashSys", float64(m.BuckHashSys), storage, logger)
	addGaugeMetricToStorage("Frees", float64(m.Frees), storage, logger)
	addGaugeMetricToStorage("GCCPUFraction", float64(m.GCCPUFraction), storage, logger)
	addGaugeMetricToStorage("GCSys", float64(m.GCSys), storage, logger)
	addGaugeMetricToStorage("HeapAlloc", float64(m.HeapAlloc), storage, logger)
	addGaugeMetricToStorage("HeapIdle", float64(m.HeapIdle), storage, logger)
	addGaugeMetricToStorage("HeapInuse", float64(m.HeapInuse), storage, logger)
	addGaugeMetricToStorage("HeapObjects", float64(m.HeapObjects), storage, logger)
	addGaugeMetricToStorage("HeapReleased", float64(m.HeapReleased), storage, logger)
	addGaugeMetricToStorage("HeapSys", float64(m.HeapSys), storage, logger)
	addGaugeMetricToStorage("LastGC", float64(m.LastGC), storage, logger)
	addGaugeMetricToStorage("Lookups", float64(m.Lookups), storage, logger)
	addGaugeMetricToStorage("MCacheInuse", float64(m.MCacheInuse), storage, logger)
	addGaugeMetricToStorage("MCacheSys", float64(m.MCacheSys), storage, logger)
	addGaugeMetricToStorage("MSpanInuse", float64(m.MSpanInuse), storage, logger)
	addGaugeMetricToStorage("MSpanSys", float64(m.MSpanSys), storage, logger)
	addGaugeMetricToStorage("Mallocs", float64(m.Mallocs), storage, logger)
	addGaugeMetricToStorage("NextGC", float64(m.NextGC), storage, logger)
	addGaugeMetricToStorage("NumForcedGC", float64(m.NumForcedGC), storage, logger)
	addGaugeMetricToStorage("NumGC", float64(m.NumGC), storage, logger)
	addGaugeMetricToStorage("OtherSys", float64(m.OtherSys), storage, logger)
	addGaugeMetricToStorage("PauseTotalNs", float64(m.PauseTotalNs), storage, logger)
	addGaugeMetricToStorage("StackInuse", float64(m.StackInuse), storage, logger)
	addGaugeMetricToStorage("StackSys", float64(m.StackSys), storage, logger)
	addGaugeMetricToStorage("Sys", float64(m.Sys), storage, logger)
	addGaugeMetricToStorage("TotalAlloc", float64(m.TotalAlloc), storage, logger)
	addGaugeMetricToStorage("RandomValue", rand.Float64(), storage, logger)
}

func addGaugeMetricToStorage(name string, value float64, storage *storage.MemStorage, logger *zap.Logger) {
	metric := entity.Metric{
		MType: entity.Gauge,
		ID:    name,
		Value: &value,
	}

	err := storage.AddMetric(metric)
	if err != nil {
		logger.Fatal("Add error", zap.Error(err))
	}
}

func addCounterMetricToStorage(name string, value int64, storage *storage.MemStorage) {
	metric := entity.Metric{
		MType: entity.Counter,
		ID:    name,
		Delta: &value,
	}

	err := storage.AddMetric(metric)
	if err != nil {
		log.Fatal(err)
	}
}

func increasePollIteration(storage *storage.MemStorage) {
	addCounterMetricToStorage("PollCount", 1, storage)
}

func sendMetrics(storage *storage.MemStorage, hookPath string, logger *zap.Logger) error {
	allMetrics, err := storage.GetAllMetrics()
	if err != nil {
		logger.Error("Get all error:", zap.Error(err))
		return err
	}

	var metricsForSend []entity.Metric
	for _, typedMetrics := range allMetrics {
		for _, metric := range typedMetrics {
			metricsForSend = append(metricsForSend, metric)
		}
	}
	jsonMetrics, err := json.Marshal(metricsForSend)
	if err != nil {
		logger.Error("Marshaling error:", zap.Error(err))
		return err
	}

	var gzippedMetric bytes.Buffer
	zb := gzip.NewWriter(&gzippedMetric)

	_, err = zb.Write(jsonMetrics)
	if err != nil {
		logger.Error("Failed to gzip metrics:", zap.Error(err))
		return err
	}
	zb.Close()

	url := fmt.Sprintf("%s/", hookPath)
	req := resty.New().R()
	req.Method = http.MethodPost
	req.URL = url
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.SetBody(gzippedMetric.Bytes())

	res, err := req.Send()
	if err != nil {
		logger.Error("Failed to send metrics", zap.Error(err))
		return err
	}
	if res.StatusCode() != http.StatusOK {
		logger.Error("Failed to send metric: wrong response code: ", zap.Int("status", res.StatusCode()))
	}

	return nil
}

func SaveMetricsInFileAgent(storage handler.Repository, fileStoragePath string, storeInterval time.Duration, ctx context.Context) error {
	ticker := time.NewTicker(storeInterval * time.Second)

	for {
		select {
		case <-ticker.C:
			if err := saveMetricsInFile(storage, fileStoragePath); err != nil {
				return fmt.Errorf("failed to save metrics: %v", err)
			}
		case <-ctx.Done():
			if err := saveMetricsInFile(storage, fileStoragePath); err != nil {
				return fmt.Errorf("failed to save metrics: %v", err)
			}
			return nil
		}
	}
}

func saveMetricsInFile(repo handler.Repository, fileStoragePath string) error {
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
