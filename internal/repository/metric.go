package repository

import (
	"github.com/WPGe/go-yandex-advanced/internal/entity"
)

type MetricRepository interface {
	AddMetric(name string, metric entity.Metric)
	GetMetric(name string) (entity.Metric, bool)
	GetAllMetrics() map[string]entity.Metric
	ClearMetrics()
}
