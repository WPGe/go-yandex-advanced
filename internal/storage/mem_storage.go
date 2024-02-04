package storage

import (
	"encoding/json"
	"github.com/WPGe/go-yandex-advanced/internal/entity"
	"github.com/pkg/errors"
	"log"
	"os"
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

func NewMemStorageFromFile(fileStoragePath string) *MemStorage {
	file, err := os.OpenFile(fileStoragePath, os.O_RDONLY|os.O_CREATE, 0755)
	if err != nil {
		log.Fatalf("%+v", errors.Wrap(err, "failed to open file"))
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Printf("Error closing file: %v", err)
		}
	}(file)

	// Получаем размер файла
	fileStat, err := file.Stat()
	if err != nil {
		log.Fatalf("%+v", errors.Wrap(err, "failed to get file stats"))
	}

	if fileStat.Size() == 0 {
		// Файл пустой, возвращаем MemStorage с пустым map
		return NewMemStorageWithMetrics(make(map[string]entity.Metric))
	}

	decoder := json.NewDecoder(file)
	initialMetrics := map[string]entity.Metric{}
	if err := decoder.Decode(&initialMetrics); err != nil {
		log.Fatalf("%+v", errors.Wrap(err, "failed to decode metrics"))
	}

	return NewMemStorageWithMetrics(initialMetrics)
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
