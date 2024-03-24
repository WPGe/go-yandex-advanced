package service

import (
	"fmt"
	"time"

	"github.com/WPGe/go-yandex-advanced/internal/entity"
)

type Repository interface {
	AddMetric(metric entity.Metric) error
	AddMetrics(metric []entity.Metric) error
	GetMetric(id, metricType string) (*entity.Metric, error)
	GetAllMetrics() (entity.MetricsStore, error)
}

type Service struct {
	repo Repository
}

func New(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetMetric(id, mType string) (*entity.Metric, error) {
	var m *entity.Metric
	var err error
	err = s.Retry(3, func() error {
		m, err = s.repo.GetMetric(id, mType)
		if err != nil {
			return err
		}
		return nil
	}, 1*time.Second, 3*time.Second, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to get metric %s: %w", id, err)
	}
	return m, nil
}

func (s *Service) GetAllMetrics() (entity.MetricsStore, error) {
	var m entity.MetricsStore
	var err error
	err = s.Retry(3, func() error {
		m, err = s.repo.GetAllMetrics()
		if err != nil {
			return err
		}
		return nil
	}, 1*time.Second, 3*time.Second, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}
	return m, nil
}

func (s *Service) AddMetric(m entity.Metric) error {
	err := s.Retry(3, func() error {
		if err := s.repo.AddMetric(m); err != nil {
			return err
		}
		return nil
	}, 1*time.Second, 3*time.Second, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to add metric: %w", err)
	}
	return nil
}

func (s *Service) AddMetrics(m []entity.Metric) error {
	err := s.Retry(3, func() error {
		if err := s.repo.AddMetrics(m); err != nil {
			return err
		}
		return nil
	}, 1*time.Second, 3*time.Second, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to add metrics: %w", err)
	}
	return nil
}

func (s *Service) Retry(maxRetries int, fn func() error, intervals ...time.Duration) error {
	var err error
	err = fn()
	if err == nil {
		return nil
	}
	for i := 0; i < maxRetries; i++ {
		time.Sleep(intervals[i])
		if err = fn(); err == nil {
			return nil
		}
	}
	return err
}
