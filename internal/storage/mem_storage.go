package storage

import (
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

func (m *MemStorage) AddMetric(name string, metric entity.Metric) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existingMetric, ok := m.metrics[name]
	if !ok {
		m.metrics[name] = metric
		return nil
	}

	if existingMetric.Type == entity.Counter {
		existingMetric.Value = existingMetric.Value.(int64) + metric.Value.(int64)
	} else {
		existingMetric.Value = metric.Value
	}
	m.metrics[name] = existingMetric

	return nil
}

func (m *MemStorage) GetMetric(name string) (entity.Metric, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	metric, ok := m.metrics[name]
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
