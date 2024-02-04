package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/WPGe/go-yandex-advanced/internal/entity"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func MetricUpdateHandler(repo Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
				http.Error(w, "Incorrect URL format", http.StatusNotFound)
				return
			}

			var err error
			switch metricType {
			case entity.Gauge:
				var value float64
				if value, err = strconv.ParseFloat(metricValue, 64); err != nil {
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
				http.Error(w, "Incorrect metric type", http.StatusBadRequest)
				return
			}

			metric.MType = metricType
			metric.ID = metricName
		}

		if err := repo.AddMetric(metric.ID, metric); err != nil {
			log.Fatal(err)
		}

		w.WriteHeader(http.StatusOK)
	}
}

func MetricGetHandler(repo Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")

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
		case entity.Gauge:
			if _, err := io.WriteString(w, fmt.Sprintf("%g", *resultMetric.Value)); err != nil {
				http.Error(w, "Output error", http.StatusBadRequest)
				return
			}
		case entity.Counter:
			if _, err := io.WriteString(w, fmt.Sprintf("%d", *resultMetric.Delta)); err != nil {
				http.Error(w, "Output error", http.StatusBadRequest)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
	}
}

func MetricPostHandler(repo Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var incomingMetric entity.Metric

		if err := json.NewDecoder(r.Body).Decode(&incomingMetric); err != nil {
			http.Error(w, "Invalid JSON format", http.StatusBadRequest)
			return
		}

		resultMetric, ok, err := repo.GetMetric(incomingMetric.ID)
		if !ok {
			http.Error(w, "Metric not found", http.StatusNotFound)
			return
		}
		if err != nil {
			log.Fatal(err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(resultMetric); err != nil {
			http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
			return
		}
	}
}

func MetricGetAllHandler(repo Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")

		resultMetrics, err := repo.GetAllMetrics()
		if err != nil {
			log.Fatal(err)
		}

		for _, metric := range resultMetrics {
			switch metric.MType {
			case entity.Gauge:
				if _, err := io.WriteString(w, fmt.Sprintf("{{%s}}: {{%g}}\n", metric.ID, *metric.Value)); err != nil {
					http.Error(w, "Output error", http.StatusBadRequest)
					return
				}
			case entity.Counter:
				if _, err := io.WriteString(w, fmt.Sprintf("{{%s}}: {{%d}}\n", metric.ID, *metric.Delta)); err != nil {
					http.Error(w, "Output error", http.StatusBadRequest)
					return
				}
			}
		}
		w.WriteHeader(http.StatusOK)
	}
}

func PingDb(db *sql.DB, logger *zap.Logger) http.HandlerFunc {
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

func WithLogging(h http.HandlerFunc, sugar zap.SugaredLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		responseData := &ResponseData{
			status: 0,
			size:   0,
			body:   "",
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
			"body", responseData.body,
		)
	}
}

func WithGzip(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ow := w

		acceptEncoding := r.Header.Get("Accept-Encoding")
		supportsGzip := strings.Contains(acceptEncoding, "gzip")
		acceptType := r.Header.Get("Accept")
		supportType := strings.Contains(acceptType, "application/json") || strings.Contains(acceptType, "html/text") || strings.Contains(acceptType, "text/html")
		if supportsGzip && supportType {
			w.Header().Set("Content-Encoding", "gzip")
			cw := newCompressWriter(w)
			ow = cw
			defer func(cw *compressWriter) {
				err := cw.Close()
				if err != nil {
					log.Fatal(err)
				}
			}(cw)
		}

		contentEncoding := r.Header.Get("Content-Encoding")
		sendsGzip := strings.Contains(contentEncoding, "gzip")
		if sendsGzip {
			cr, err := newCompressReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			r.Body = cr
			defer func(cr *compressReader) {
				err := cr.Close()
				if err != nil {
					log.Fatal(err)
				}
			}(cr)
		}

		h.ServeHTTP(ow, r)
	}
}
