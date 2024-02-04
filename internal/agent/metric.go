package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"github.com/WPGe/go-yandex-advanced/internal/entity"
	"github.com/WPGe/go-yandex-advanced/internal/handler"
	"github.com/go-resty/resty/v2"
	"log"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"time"
)

func MetricAgent(repo handler.Repository, hookPath string, reportInterval time.Duration, pollInterval time.Duration, stopCh <-chan struct{}) {
	ticker := time.NewTicker(pollInterval * time.Second)
	sendTicker := time.NewTicker(reportInterval * time.Second)

	for {
		select {
		case <-ticker.C:
			collectGaugeRuntimeMetrics(repo)
			increasePollIteration(repo)
		case <-sendTicker.C:
			sendMetrics(repo, hookPath)
			err := repo.ClearMetrics()
			if err != nil {
				log.Fatal(err)
			}

		case <-stopCh:
			return
		}
	}
}

func collectGaugeRuntimeMetrics(repo handler.Repository) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	addGaugeMetricToStorage("Alloc", float64(m.Alloc), repo)
	addGaugeMetricToStorage("BuckHashSys", float64(m.BuckHashSys), repo)
	addGaugeMetricToStorage("Frees", float64(m.Frees), repo)
	addGaugeMetricToStorage("GCCPUFraction", float64(m.GCCPUFraction), repo)
	addGaugeMetricToStorage("GCSys", float64(m.GCSys), repo)
	addGaugeMetricToStorage("HeapAlloc", float64(m.HeapAlloc), repo)
	addGaugeMetricToStorage("HeapIdle", float64(m.HeapIdle), repo)
	addGaugeMetricToStorage("HeapInuse", float64(m.HeapInuse), repo)
	addGaugeMetricToStorage("HeapObjects", float64(m.HeapObjects), repo)
	addGaugeMetricToStorage("HeapReleased", float64(m.HeapReleased), repo)
	addGaugeMetricToStorage("HeapSys", float64(m.HeapSys), repo)
	addGaugeMetricToStorage("LastGC", float64(m.LastGC), repo)
	addGaugeMetricToStorage("Lookups", float64(m.Lookups), repo)
	addGaugeMetricToStorage("MCacheInuse", float64(m.MCacheInuse), repo)
	addGaugeMetricToStorage("MCacheSys", float64(m.MCacheSys), repo)
	addGaugeMetricToStorage("MSpanInuse", float64(m.MSpanInuse), repo)
	addGaugeMetricToStorage("MSpanSys", float64(m.MSpanSys), repo)
	addGaugeMetricToStorage("Mallocs", float64(m.Mallocs), repo)
	addGaugeMetricToStorage("NextGC", float64(m.NextGC), repo)
	addGaugeMetricToStorage("NumForcedGC", float64(m.NumForcedGC), repo)
	addGaugeMetricToStorage("NumGC", float64(m.NumGC), repo)
	addGaugeMetricToStorage("OtherSys", float64(m.OtherSys), repo)
	addGaugeMetricToStorage("PauseTotalNs", float64(m.PauseTotalNs), repo)
	addGaugeMetricToStorage("StackInuse", float64(m.StackInuse), repo)
	addGaugeMetricToStorage("StackSys", float64(m.StackSys), repo)
	addGaugeMetricToStorage("Sys", float64(m.Sys), repo)
	addGaugeMetricToStorage("TotalAlloc", float64(m.TotalAlloc), repo)
	addGaugeMetricToStorage("RandomValue", rand.Float64(), repo)
}

func addGaugeMetricToStorage(name string, value float64, repo handler.Repository) {
	metric := entity.Metric{
		MType: entity.Gauge,
		ID:    name,
		Value: &value,
	}

	err := repo.AddMetric(name, metric)
	if err != nil {
		log.Fatal(err)
	}
}

func addCounterMetricToStorage(name string, value int64, repo handler.Repository) {
	metric := entity.Metric{
		MType: entity.Counter,
		ID:    name,
		Delta: &value,
	}

	err := repo.AddMetric(name, metric)
	if err != nil {
		log.Fatal(err)
	}
}

func increasePollIteration(repo handler.Repository) {
	addCounterMetricToStorage("PollCount", 1, repo)
}

func sendMetrics(repo handler.Repository, hookPath string) {
	allMetrics, err := repo.GetAllMetrics()
	if err != nil {
		log.Fatal(err)
	}

	for _, metric := range allMetrics {
		jsonMetric, err := json.Marshal(metric)
		if err != nil {
			log.Printf("Failed to encode metric: %+v, Error: %v", metric, err)
			continue
		}

		var gzippedMetric bytes.Buffer
		zb := gzip.NewWriter(&gzippedMetric)

		_, err = zb.Write(jsonMetric)
		if err != nil {
			log.Printf("Failed to gzip metric: %+v, Error: %v", metric, err)
			err := zb.Close()
			if err != nil {
				return
			}
			continue
		}

		err = zb.Close()
		if err != nil {
			return
		}

		url := fmt.Sprintf("%s/", hookPath)
		req := resty.New().R()
		req.Method = http.MethodPost
		req.URL = url
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Encoding", "gzip")
		req.SetBody(gzippedMetric.Bytes())

		res, err := req.Send()
		if err != nil {
			fmt.Println("Failed to send metric: problem triggered", metric, "Error:", err)
		}
		if res.StatusCode() != http.StatusOK {
			fmt.Println("Failed to send metric: ", metric, "Wrong response code: ", res.StatusCode())
		}
	}
}

func SaveMetricsInFileAgent(repo handler.Repository, fileStoragePath string, storeInterval time.Duration, ctx context.Context) error {
	ticker := time.NewTicker(storeInterval * time.Second)

	for {
		select {
		case <-ticker.C:
			if err := saveMetricsInFile(repo, fileStoragePath); err != nil {
				return fmt.Errorf("failed to save metrics: %v", err)
			}
		case <-ctx.Done():
			if err := saveMetricsInFile(repo, fileStoragePath); err != nil {
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
