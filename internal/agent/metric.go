package agent

import (
	"fmt"
	"github.com/WPGe/go-yandex-advanced/internal/entity"
	"github.com/WPGe/go-yandex-advanced/internal/repository"
	"github.com/go-resty/resty/v2"
	"math/rand"
	"net/http"
	"runtime"
	"time"
)

func MetricAgent(repo repository.MetricRepository, hookPath string, reportInterval time.Duration, pollInterval time.Duration, stopCh <-chan struct{}) {
	ticker := time.NewTicker(pollInterval * time.Second)
	sendTicker := time.NewTicker(reportInterval * time.Second)

	gaugeRuntimeMetrics := map[string]float64{}
	counterRuntimeMetrics := map[string]int64{}

	for {
		select {
		case <-ticker.C:
			collectGaugeRuntimeMetrics(&gaugeRuntimeMetrics)
			for name, value := range gaugeRuntimeMetrics {
				metric := entity.Metric{
					Type:  entity.Gauge,
					Name:  name,
					Value: value,
				}
				repo.AddMetric(name, metric)
			}

			collectCounterRuntimeMetrics(&counterRuntimeMetrics)
		case <-sendTicker.C:
			for name, value := range counterRuntimeMetrics {
				metric := entity.Metric{
					Type:  entity.Counter,
					Name:  name,
					Value: value,
				}
				repo.AddMetric(name, metric)
			}
			clearCounterRuntimeMetrics(&counterRuntimeMetrics)

			sendMetrics(repo, hookPath)
			repo.ClearMetrics()

		case <-stopCh:
			return
		}
	}
}

func collectGaugeRuntimeMetrics(myMap *map[string]float64) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	(*myMap)["Alloc"] = float64(m.Alloc)
	(*myMap)["BuckHashSys"] = float64(m.BuckHashSys)
	(*myMap)["Frees"] = float64(m.Frees)
	(*myMap)["GCCPUFraction"] = m.GCCPUFraction
	(*myMap)["GCSys"] = float64(m.GCSys)
	(*myMap)["HeapAlloc"] = float64(m.HeapAlloc)
	(*myMap)["HeapIdle"] = float64(m.HeapIdle)
	(*myMap)["HeapInuse"] = float64(m.HeapInuse)
	(*myMap)["HeapObjects"] = float64(m.HeapObjects)
	(*myMap)["HeapReleased"] = float64(m.HeapReleased)
	(*myMap)["HeapSys"] = float64(m.HeapSys)
	(*myMap)["LastGC"] = float64(m.LastGC)
	(*myMap)["Lookups"] = float64(m.Lookups)
	(*myMap)["MCacheInuse"] = float64(m.MCacheInuse)
	(*myMap)["MCacheSys"] = float64(m.MCacheSys)
	(*myMap)["MSpanInuse"] = float64(m.MSpanInuse)
	(*myMap)["MSpanSys"] = float64(m.MSpanSys)
	(*myMap)["Mallocs"] = float64(m.Mallocs)
	(*myMap)["NextGC"] = float64(m.NextGC)
	(*myMap)["NumForcedGC"] = float64(m.NumForcedGC)
	(*myMap)["NumGC"] = float64(m.NumGC)
	(*myMap)["OtherSys"] = float64(m.OtherSys)
	(*myMap)["PauseTotalNs"] = float64(m.PauseTotalNs)
	(*myMap)["StackInuse"] = float64(m.StackInuse)
	(*myMap)["StackSys"] = float64(m.StackSys)
	(*myMap)["Sys"] = float64(m.Sys)
	(*myMap)["TotalAlloc"] = float64(m.TotalAlloc)
	(*myMap)["RandomValue"] = rand.Float64()
}

func collectCounterRuntimeMetrics(myMap *map[string]int64) {
	(*myMap)["PollCount"]++
}

func clearCounterRuntimeMetrics(myMap *map[string]int64) {
	for key := range *myMap {
		delete(*myMap, key)
	}
}

func sendMetrics(repo repository.MetricRepository, hookPath string) {
	allMetrics := repo.GetAllMetrics()

	for _, metric := range allMetrics {
		url := fmt.Sprintf("%s/%s/%s/%v", hookPath, metric.Type, metric.Name, metric.Value)
		req := resty.New().R()
		req.Method = http.MethodPost
		req.URL = url
		res, err := req.Send()
		if err != nil {
			fmt.Println("Failed to send metric:", metric, "Error:", err)
		}
		if res.StatusCode() != http.StatusOK {
			fmt.Println("Failed to send metric: ", metric, "Wrong response code: ", res.StatusCode())
		}
	}
}
