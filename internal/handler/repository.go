package handler

import (
	"github.com/WPGe/go-yandex-advanced/internal/entity"
)

type Repository interface {
	AddMetric(metric entity.Metric) error
	AddMetrics(metric []entity.Metric) error
	GetMetric(id, metricType string) (*entity.Metric, error)
	GetAllMetrics() (entity.MetricsStore, error)
}
