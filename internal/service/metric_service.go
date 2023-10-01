package service

import (
	"fmt"
	"github.com/WPGe/go-yandex-advanced/internal/entity/metric"
	"github.com/WPGe/go-yandex-advanced/internal/entity/metric/typed_metric"
)

type MetricService struct{}

func (s *MetricService) AddGaugeMetric(storage *metric.MemStorage, metricName string, value string) error {
	gauge, err := typed_metric.NewGauge(metricName, value)
	if err != nil {
		return fmt.Errorf("ошибка при создании метрики: %w", err)
	}

	_, ok := storage.Metrics["gauge"]
	if !ok {
		storage.Metrics["gauge"] = make(map[string]interface{})
	}

	storage.Metrics["gauge"][gauge.Name] = gauge.Value

	return nil
}

func (s *MetricService) AddCounterMetric(storage *metric.MemStorage, metricName string, value string) error {
	counter, err := typed_metric.NewCounter(metricName, value)
	if err != nil {
		return fmt.Errorf("ошибка при создании метрики: %w", err)
	}

	_, ok := storage.Metrics["counter"]
	if !ok {
		storage.Metrics["counter"] = make(map[string]interface{})
	}

	if existingValue, exists := storage.Metrics["counter"][counter.Name]; exists {
		existingCounterValue, ok := existingValue.(int64)
		if !ok {
			// Обработка ошибки, если тип значения не int64
			return fmt.Errorf("ошибка: существующее значение не является int64")
		}
		storage.Metrics["counter"][counter.Name] = existingCounterValue + counter.Value
	} else {
		storage.Metrics["counter"][counter.Name] = counter.Value
	}

	return nil
}
