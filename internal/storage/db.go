package storage

import (
	"database/sql"
	"sync"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/WPGe/go-yandex-advanced/internal/entity"
)

type DbStorage struct {
	mu      sync.RWMutex
	metrics map[string]entity.Metric
	db      *sql.DB
	logger  *zap.Logger
}

func NewDbStorage(logger *zap.Logger, db *sql.DB) *DbStorage {
	return &DbStorage{
		metrics: make(map[string]entity.Metric),
		db:      db,
		logger:  logger,
	}
}

func add(tx *sql.Tx, logger *zap.Logger, metric entity.Metric) error {

	var mID, mType string
	var mDelta sql.NullInt64

	row := tx.QueryRow("SELECT id, type, delta FROM metrics WHERE id = $1 AND type = $2", metric.ID, metric.MType)
	err := row.Scan(&mID, &mType, &mDelta)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		logger.Error("Add: scan row error", zap.Error(err))
		return err
	}

	if metric.MType == entity.Counter {
		switch {
		case mID != "":
			_, err = tx.Exec(
				"UPDATE metrics SET delta = $1 WHERE id = $2 AND type = $3",
				mDelta.Int64+*metric.Delta, metric.ID, metric.MType,
			)
		default:
			_, err = tx.Exec(
				"INSERT INTO metrics (id, type, delta) VALUES ($1, $2, $3)",
				metric.ID, metric.MType, *metric.Delta,
			)
		}
		if err != nil {
			logger.Error("add: error to add counter", zap.Error(err))
			return err
		}
	}

	if metric.MType == entity.Gauge {
		switch {
		case mID != "":
			_, err = tx.Exec(
				"UPDATE metrics SET value = $1 WHERE id = $2 AND type = $3",
				metric.Value, metric.ID, metric.MType,
			)
		default:
			_, err = tx.Exec(
				"INSERT INTO metrics (id, type, value) VALUES ($1, $2, $3)",
				metric.ID, metric.MType, *metric.Value,
			)
		}
		if err != nil {
			logger.Error("add: error to add gauge", zap.Error(err))
			return err
		}
	}

	return nil
}

func (storage *DbStorage) AddMetric(metric entity.Metric) error {
	tx, err := storage.db.Begin()
	if err != nil {
		storage.logger.Error("Add: begin transaction error", zap.Error(err))
		return err
	}
	if err := add(tx, storage.logger, metric); err != nil {
		storage.logger.Error("Add: data error", zap.Error(err))
		tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		storage.logger.Error("Add: commit transaction error", zap.Error(err))
		return err
	}
	return nil
}

func (storage *DbStorage) AddMetrics(metrics []entity.Metric) error {
	tx, err := storage.db.Begin()
	if err != nil {
		storage.logger.Error("Add metrics: begin transaction error", zap.Error(err))
		return err
	}

	for _, metric := range metrics {
		if err := add(tx, storage.logger, metric); err != nil {
			storage.logger.Error("Add metrics: data error", zap.Error(err))
			tx.Rollback()
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		storage.logger.Error("Add metrics: commit transaction error", zap.Error(err))
		return err
	}
	return nil
}

func (storage *DbStorage) GetMetric(id, metricType string) (*entity.Metric, error) {
	var mID, mType string
	var mDelta sql.NullInt64
	var mValue sql.NullFloat64

	row := storage.db.QueryRow("SELECT id, type, delta, value FROM metrics WHERE id = $1 AND type = $2", id, metricType)
	err := row.Scan(&mID, &mType, &mDelta, &mValue)
	if err != nil {
		storage.logger.Error("Get: scan row error", zap.Error(err))
		return nil, err
	}

	return &entity.Metric{
		ID:    mID,
		MType: mType,
		Delta: parseDelta(mDelta),
		Value: parseValue(mValue),
	}, nil
}

func (storage *DbStorage) GetAllMetrics() (entity.MetricsStore, error) {
	rows, err := storage.db.Query("SELECT id, type, delta, value FROM metrics")
	if err != nil {
		storage.logger.Error("GetAll: select error", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	metrics := make(entity.MetricsStore)
	for rows.Next() {
		var mID, mType string
		var mDelta sql.NullInt64
		var mValue sql.NullFloat64

		err := rows.Scan(&mID, &mType, &mDelta, &mValue)
		if err != nil {
			storage.logger.Error("GetAll: scan row error", zap.Error(err))
			return nil, err
		}
		_, ok := metrics[mType]
		if !ok {
			metrics[mType] = map[string]entity.Metric{
				mID: {
					ID:    mID,
					MType: mType,
					Delta: parseDelta(mDelta),
					Value: parseValue(mValue),
				},
			}
		} else {
			metrics[mType][mID] = entity.Metric{
				ID:    mID,
				MType: mType,
				Delta: parseDelta(mDelta),
				Value: parseValue(mValue),
			}
		}
	}

	return metrics, err
}

func (storage *DbStorage) ClearMetrics() error {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	storage.metrics = make(map[string]entity.Metric)

	return nil
}

func parseDelta(v sql.NullInt64) *int64 {
	if v.Valid {
		return &v.Int64
	}
	return nil
}

func parseValue(v sql.NullFloat64) *float64 {
	if v.Valid {
		return &v.Float64
	}
	return nil
}
