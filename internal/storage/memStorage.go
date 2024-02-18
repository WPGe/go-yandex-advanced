package storage

import (
	"encoding/json"
	"fmt"
	"github.com/WPGe/go-yandex-advanced/internal/entity"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"log"
	"os"
	"sync"
)

type MemStorage struct {
	mu      sync.RWMutex
	metrics entity.MetricsStore
	logger  *zap.Logger
}

func NewMemStorage(logger *zap.Logger) *MemStorage {
	return &MemStorage{
		metrics: make(entity.MetricsStore),
		logger:  logger,
	}
}

func NewMemStorageWithMetrics(initialMetrics entity.MetricsStore, logger *zap.Logger) *MemStorage {
	return &MemStorage{
		metrics: initialMetrics,
		logger:  logger,
	}
}

func NewMemStorageFromFile(fileStoragePath string, logger *zap.Logger) *MemStorage {
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
		return NewMemStorageWithMetrics(make(entity.MetricsStore), logger)
	}

	decoder := json.NewDecoder(file)
	initialMetrics := entity.MetricsStore{}
	if err := decoder.Decode(&initialMetrics); err != nil {
		log.Fatalf("%+v", errors.Wrap(err, "failed to decode metrics"))
	}

	return NewMemStorageWithMetrics(initialMetrics, logger)
}

func (m *MemStorage) AddMetric(metric entity.Metric) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var existingMetric entity.Metric

	_, ok := m.metrics[metric.MType]
	if !ok {
		m.metrics[metric.MType] = map[string]entity.Metric{
			metric.ID: metric,
		}
		return nil
	}

	if metric.MType == entity.Gauge {
		m.metrics[metric.MType][metric.ID] = metric
		return nil
	}

	if metric.Delta == nil {
		return errors.New("delta cannot be nil for counter metric")
	}

	existingMetric = m.metrics[metric.MType][metric.ID]
	if existingMetric.Delta == nil {
		existingMetric.Delta = new(int64)
	}

	*existingMetric.Delta += *metric.Delta
	m.metrics[metric.MType][metric.ID] = existingMetric

	return nil
}

func (m *MemStorage) AddMetrics(metrics []entity.Metric) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, metric := range metrics {
		err := m.AddMetric(metric)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *MemStorage) GetMetric(id, metricType string) (*entity.Metric, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metrics, ok := m.metrics[metricType]
	if !ok {
		return nil, fmt.Errorf("metric type: %s does not exist", metricType)
	}
	metric, ok := metrics[id]
	if !ok {
		return nil, fmt.Errorf("metric id: %s does not exist", id)
	}
	return &metric, nil
}

func (m *MemStorage) GetAllMetrics() (entity.MetricsStore, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.metrics, nil
}

func (m *MemStorage) ClearMetrics() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metrics = make(entity.MetricsStore)

	return nil
}
