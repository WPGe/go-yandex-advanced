package repository

import (
	"github.com/WPGe/go-yandex-advanced/internal/entity"
)

type MetricRepository interface {
	AddMetric(name string, metric entity.Metric) error
	GetMetric(name string) (entity.Metric, bool, error)
	GetAllMetrics() (map[string]entity.Metric, error)
	ClearMetrics() error
}
