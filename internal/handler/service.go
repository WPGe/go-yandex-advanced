package handler

import "github.com/WPGe/go-yandex-advanced/internal/model"

type Service interface {
	AddMetric(metric model.Metric) error
	AddMetrics(metric []model.Metric) error
	GetMetric(id, metricType string) (*model.Metric, error)
	GetAllMetrics() (model.MetricsStore, error)
}
