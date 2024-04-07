package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"io"
	"net/http"
	"strconv"

	"github.com/WPGe/go-yandex-advanced/internal/entity"
	"github.com/go-chi/chi/v5"
)

func MetricUpdateHandler(srv Service, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Update: start")

		var metric entity.Metric
		if r.Header.Get("Content-Type") == "application/json" {
			decoder := json.NewDecoder(r.Body)
			if err := decoder.Decode(&metric); err != nil {
				http.Error(w, "Failed to decode JSON request", http.StatusBadRequest)
				return
			}
		} else {
			metricType := chi.URLParam(r, "type")
			metricName := chi.URLParam(r, "name")
			metricValue := chi.URLParam(r, "value")

			if metricType == "" || metricName == "" || metricValue == "" {
				logger.Fatal("Update: Incorrect URL format")
				http.Error(w, "Incorrect URL format", http.StatusNotFound)
				return
			}

			var err error
			switch metricType {
			case entity.Gauge:
				var value float64
				if value, err = strconv.ParseFloat(metricValue, 64); err != nil {
					logger.Info("Update: Incorrect value")
					http.Error(w, "Incorrect value", http.StatusBadRequest)
					return
				}
				metric.Value = &value
			case entity.Counter:
				var delta int64
				if delta, err = strconv.ParseInt(metricValue, 10, 64); err != nil {
					http.Error(w, "Incorrect value", http.StatusBadRequest)
					return
				}
				metric.Delta = &delta
			default:
				logger.Info("Update: Incorrect metric type")
				http.Error(w, "Incorrect metric type", http.StatusBadRequest)
				return
			}

			metric.MType = metricType
			metric.ID = metricName
		}

		if err := srv.AddMetric(metric); err != nil {
			logger.Fatal("Update: add error", zap.Error(err))
		}

		w.WriteHeader(http.StatusOK)
	}
}

func MetricUpdatesHandler(srv Service, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Update: start")

		var metrics []entity.Metric

		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&metrics); err != nil {
			logger.Error("Update: Failed to decode JSON request", zap.Error(err))
			http.Error(w, "Failed to decode JSON request", http.StatusBadRequest)
			return
		}

		if err := srv.AddMetrics(metrics); err != nil {
			logger.Fatal("Update: add error", zap.Error(err))
		}

		w.WriteHeader(http.StatusOK)
	}
}

func MetricGetHandler(srv Service, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")

		metricType := chi.URLParam(r, "type")
		metricName := chi.URLParam(r, "name")

		resultMetric, err := srv.GetMetric(metricName, metricType)
		if err != nil {
			logger.Error("Get: Metric not found", zap.Error(err))
			http.Error(w, "Metric not found", http.StatusNotFound)
			return
		}

		switch metricType {
		case entity.Gauge:
			if _, err := io.WriteString(w, fmt.Sprintf("%g", *resultMetric.Value)); err != nil {
				logger.Error("Get: Output error", zap.Error(err))
				http.Error(w, "Output error", http.StatusBadRequest)
				return
			}
		case entity.Counter:
			if _, err := io.WriteString(w, fmt.Sprintf("%d", *resultMetric.Delta)); err != nil {
				logger.Error("Get: Output error", zap.Error(err))
				http.Error(w, "Output error", http.StatusBadRequest)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
	}
}

func MetricPostHandler(srv Service, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var incomingMetric entity.Metric

		if err := json.NewDecoder(r.Body).Decode(&incomingMetric); err != nil {
			logger.Error("Get: Invalid JSON format", zap.Error(err))
			http.Error(w, "Invalid JSON format", http.StatusBadRequest)
			return
		}

		resultMetric, err := srv.GetMetric(incomingMetric.ID, incomingMetric.MType)
		if err != nil {
			logger.Error("Get: Metric not found", zap.Error(err))
			http.Error(w, "Metric not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(resultMetric); err != nil {
			logger.Error("Get: Error encoding JSON", zap.Error(err))
			http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
			return
		}
	}
}

func MetricGetAllHandler(srv Service, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Get all: start")

		w.Header().Set("Content-Type", "text/html")

		resultMetrics, err := srv.GetAllMetrics()
		if err != nil {
			logger.Fatal("Get all: error", zap.Error(err))
		}

		if resultMetrics[entity.Gauge] != nil {
			for _, metric := range resultMetrics[entity.Gauge] {
				logger.Info("metric", zap.Any("metric", metric))
				if _, err := io.WriteString(w, fmt.Sprintf("{{%s}}: {{%s}}: {{%g}}\n", entity.Gauge, metric.ID, *metric.Value)); err != nil {
					logger.Error("Get all: print error", zap.Error(err))
					http.Error(w, "Output error", http.StatusBadRequest)
					return
				}
			}
		}
		if resultMetrics[entity.Counter] != nil {
			for _, metric := range resultMetrics[entity.Counter] {
				if _, err := io.WriteString(w, fmt.Sprintf("{{%s}}: {{%s}}: {{%d}}\n", entity.Counter, metric.ID, *metric.Delta)); err != nil {
					logger.Error("Get all: print error", zap.Error(err))
					http.Error(w, "Output error", http.StatusBadRequest)
					return
				}
			}
		}

		logger.Info("Get all: end")

		w.WriteHeader(http.StatusOK)
	}
}

func PingDB(db *sql.DB, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if db == nil {
			return
		}
		if err := db.Ping(); err != nil {
			logger.Error("Pinging DB error", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
