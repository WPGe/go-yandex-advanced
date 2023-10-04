package handler

import (
	"fmt"
	"github.com/WPGe/go-yandex-advanced/internal/entity"
	"github.com/WPGe/go-yandex-advanced/internal/repository"
	"github.com/go-chi/chi/v5"
	"io"
	"net/http"
	"strconv"
)

func MetricUpdateHandler(repo repository.MetricRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metricType := chi.URLParam(r, "type")
		metricName := chi.URLParam(r, "name")
		metricValue := chi.URLParam(r, "value")

		if metricType == "" || metricName == "" || metricValue == "" {
			http.Error(w, "Некорректный формат URL", http.StatusNotFound)
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
			http.Error(w, "Некорректный тип метрики", http.StatusBadRequest)
			return
		}
		if err != nil {
			http.Error(w, "Некорректное значение", http.StatusBadRequest)
			return
		}

		metric := entity.Metric{
			Type:  entity.Type(metricType),
			Name:  metricName,
			Value: typedMetricValue,
		}

		repo.AddMetric(metricName, metric)
		w.WriteHeader(http.StatusOK)
	}
}

func MetricGetHandler(repo repository.MetricRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metricType := chi.URLParam(r, "type")
		metricName := chi.URLParam(r, "name")

		resultMetric, ok := repo.GetMetric(metricName)
		if !ok {
			http.Error(w, "Метрика не найдена", http.StatusNotFound)
			return
		}

		switch metricType {
		case string(entity.Gauge):
			if _, err := io.WriteString(w, fmt.Sprintf("%g", resultMetric.Value)); err != nil {
				http.Error(w, "Ошибка вывода значения", http.StatusBadRequest)
				return
			}
		case string(entity.Counter):
			if _, err := io.WriteString(w, fmt.Sprintf("%d", resultMetric.Value)); err != nil {
				http.Error(w, "Ошибка вывода значения", http.StatusBadRequest)
				return
			}
		}

		w.WriteHeader(http.StatusOK)
	}
}

func MetricGetAllHandler(repo repository.MetricRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resultMetrics := repo.GetAllMetrics()

		for _, metric := range resultMetrics {
			switch metric.Type {
			case entity.Gauge:
				if _, err := io.WriteString(w, fmt.Sprintf("{{%s}}: {{%g}}\n", metric.Name, metric.Value)); err != nil {
					http.Error(w, "Ошибка вывода", http.StatusBadRequest)
					return
				}
			case entity.Counter:
				if _, err := io.WriteString(w, fmt.Sprintf("{{%s}}: {{%d}}\n", metric.Name, metric.Value)); err != nil {
					http.Error(w, "Ошибка вывода", http.StatusBadRequest)
					return
				}
			}
		}
		w.WriteHeader(http.StatusOK)
	}
}
