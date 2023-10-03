package storage

import (
	"github.com/WPGe/go-yandex-advanced/internal/entity"
	"sync"
)

type MemStorage struct {
	mu      sync.Mutex
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

func (m *MemStorage) AddMetric(name string, metric entity.Metric) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if existingMetric, ok := m.metrics[name]; ok {
		if existingMetric.Type == entity.Counter {
			existingMetric.Value = existingMetric.Value.(int64) + metric.Value.(int64)
		} else {
			existingMetric.Value = metric.Value
		}
		m.metrics[name] = existingMetric
	} else {
		m.metrics[name] = metric
	}
}

func (m *MemStorage) GetMetric(name string) (entity.Metric, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	metric, ok := m.metrics[name]
	return metric, ok
}

func (m *MemStorage) GetAllMetrics() map[string]entity.Metric {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.metrics
}

func (m *MemStorage) ClearMetrics() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metrics = make(map[string]entity.Metric)
}
