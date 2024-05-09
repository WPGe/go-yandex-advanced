package service

import (
	"fmt"
	"time"

	"github.com/WPGe/go-yandex-advanced/internal/model"
)

type Repository interface {
	AddMetric(metric model.Metric) error
	AddMetrics(metric []model.Metric) error
	GetMetric(id, metricType string) (*model.Metric, error)
	GetAllMetrics() (model.MetricsStore, error)
}

type Service struct {
	repo Repository
}

func New(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetMetric(id, mType string) (*model.Metric, error) {
	var m *model.Metric
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

func (s *Service) GetAllMetrics() (model.MetricsStore, error) {
	var m model.MetricsStore
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

func (s *Service) AddMetric(m model.Metric) error {
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

func (s *Service) AddMetrics(m []model.Metric) error {
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
