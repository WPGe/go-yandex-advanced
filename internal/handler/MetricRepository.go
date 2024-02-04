package handler

import (
	"github.com/WPGe/go-yandex-advanced/internal/entity"
)

type MetricRepository interface {
	AddMetric(id string, metric entity.Metric) error
	GetMetric(id string) (entity.Metric, bool, error)
	GetAllMetrics() (map[string]entity.Metric, error)
	ClearMetrics() error
}
