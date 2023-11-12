package handler

import (
	"fmt"
	"github.com/WPGe/go-yandex-advanced/internal/entity"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

func MetricUpdateHandler(repo MetricRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metricType := chi.URLParam(r, "type")
		metricName := chi.URLParam(r, "name")
		metricValue := chi.URLParam(r, "value")

		if metricType == "" || metricName == "" || metricValue == "" {
			http.Error(w, "Incorrect URL format", http.StatusNotFound)
			return
		}

		var typedMetricValue interface{}
		var err error
		switch metricType {
		case string(entity.Gauge):
			typedMetricValue, err = strconv.ParseFloat(metricValue, 64)
		case string(entity.Counter):
			typedMetricValue, err = strconv.ParseInt(metricValue, 10, 64)
		default:
			http.Error(w, "Incorrect metric type", http.StatusBadRequest)
			return
		}
		if err != nil {
			http.Error(w, "Incorrect value", http.StatusBadRequest)
			return
		}

		metric := entity.Metric{
			Type:  entity.Type(metricType),
			Name:  metricName,
			Value: typedMetricValue,
		}

		err = repo.AddMetric(metricName, metric)
		if err != nil {
			log.Fatal(err)
		}
		w.WriteHeader(http.StatusOK)
	}
}

func MetricGetHandler(repo MetricRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metricType := chi.URLParam(r, "type")
		metricName := chi.URLParam(r, "name")

		resultMetric, ok, err := repo.GetMetric(metricName)
		if !ok {
			http.Error(w, "Metric not found", http.StatusNotFound)
			return
		}
		if err != nil {
			log.Fatal(err)
		}

		switch metricType {
		case string(entity.Gauge):
			if _, err := io.WriteString(w, fmt.Sprintf("%g", resultMetric.Value)); err != nil {
				http.Error(w, "Output error", http.StatusBadRequest)
				return
			}
		case string(entity.Counter):
			if _, err := io.WriteString(w, fmt.Sprintf("%d", resultMetric.Value)); err != nil {
				http.Error(w, "Output error", http.StatusBadRequest)
				return
			}
		}

		w.WriteHeader(http.StatusOK)
	}
}

func MetricGetAllHandler(repo MetricRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resultMetrics, err := repo.GetAllMetrics()
		if err != nil {
			log.Fatal(err)
		}

		for _, metric := range resultMetrics {
			switch metric.Type {
			case entity.Gauge:
				if _, err := io.WriteString(w, fmt.Sprintf("{{%s}}: {{%g}}\n", metric.Name, metric.Value)); err != nil {
					http.Error(w, "Output error", http.StatusBadRequest)
					return
				}
			case entity.Counter:
				if _, err := io.WriteString(w, fmt.Sprintf("{{%s}}: {{%d}}\n", metric.Name, metric.Value)); err != nil {
					http.Error(w, "Output error", http.StatusBadRequest)
					return
				}
			}
		}
		w.WriteHeader(http.StatusOK)
	}
}

func WithLogging(h http.HandlerFunc, sugar zap.SugaredLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		responseData := &ResponseData{
			status: 0,
			size:   0,
		}
		lw := LoggingResponseWriter{
			ResponseWriter: w,
			ResponseData:   responseData,
		}

		h.ServeHTTP(&lw, r)

		duration := time.Since(start)

		sugar.Infoln(
			"uri", r.RequestURI,
			"method", r.Method,
			"duration", duration,
			"status", responseData.status,
			"size", responseData.size,
		)
	}
}
