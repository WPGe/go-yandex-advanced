package storage

import (
	"errors"
	"github.com/WPGe/go-yandex-advanced/internal/entity"
	"sync"
)

type MemStorage struct {
	mu      sync.RWMutex
	metrics map[string]entity.Metric
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		metrics: make(map[string]entity.Metric),
	}
}

func NewMemStorageWithMetrics(initialMetrics map[string]entity.Metric) *MemStorage {
	return &MemStorage{
		metrics: initialMetrics,
	}
}

func (m *MemStorage) AddMetric(id string, metric entity.Metric) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existingMetric, ok := m.metrics[id]
	if !ok {
		m.metrics[id] = metric
		return nil
	}

	if existingMetric.MType == entity.Gauge {
		existingMetric.Value = metric.Value
		m.metrics[id] = existingMetric
		return nil
	}

	if metric.Delta == nil {
		return errors.New("delta cannot be nil for counter metric")
	}

	if existingMetric.Delta == nil {
		existingMetric.Delta = new(int64)
	}

	*existingMetric.Delta += *metric.Delta
	m.metrics[id] = existingMetric

	return nil
}

func (m *MemStorage) GetMetric(id string) (entity.Metric, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	metric, ok := m.metrics[id]
	return metric, ok, nil
}

func (m *MemStorage) GetAllMetrics() (map[string]entity.Metric, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.metrics, nil
}

func (m *MemStorage) ClearMetrics() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metrics = make(map[string]entity.Metric)

	return nil
}
