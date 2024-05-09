package storage

import (
	"database/sql"
	"sync"

	"go.uber.org/zap"

	"github.com/WPGe/go-yandex-advanced/internal/model"
)

type DBStorage struct {
	mu      sync.RWMutex
	metrics map[string]model.Metric
	db      *sql.DB
	logger  *zap.Logger
}

func NewDBStorage(logger *zap.Logger, db *sql.DB) *DBStorage {
	return &DBStorage{
		metrics: make(map[string]model.Metric),
		db:      db,
		logger:  logger,
	}
}

func add(tx *sql.Tx, logger *zap.Logger, metric model.Metric) error {
	var err error

	_, err = tx.Exec(
		"INSERT INTO metrics (id, type, delta, value) VALUES ($1, $2, $3, $4)"+
			"ON CONFLICT (id, type)"+
			"DO UPDATE "+
			"SET delta = COALESCE(metrics.delta, NULL) + COALESCE(excluded.delta, NULL),"+
			"value = excluded.value",
		metric.ID, metric.MType, metric.Delta, metric.Value,
	)
	if err != nil {
		logger.Error("add: error to add", zap.Error(err))
		return err
	}

	return nil
}

func (storage *DBStorage) AddMetric(metric model.Metric) error {
	tx, err := storage.db.Begin()
	if err != nil {
		storage.logger.Error("Add: begin transaction error", zap.Error(err))
		return err
	}
	if err := add(tx, storage.logger, metric); err != nil {
		storage.logger.Error("Add: data error", zap.Error(err))
		defer tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		storage.logger.Error("Add: commit transaction error", zap.Error(err))
		return err
	}
	return nil
}

func (storage *DBStorage) AddMetrics(metrics []model.Metric) error {
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

func (storage *DBStorage) GetMetric(id, metricType string) (*model.Metric, error) {
	var mID, mType string
	var mDelta sql.NullInt64
	var mValue sql.NullFloat64

	row := storage.db.QueryRow("SELECT id, type, delta, value FROM metrics WHERE id = $1 AND type = $2", id, metricType)
	err := row.Scan(&mID, &mType, &mDelta, &mValue)
	if err != nil {
		storage.logger.Error("Get: scan row error", zap.Error(err))
		return nil, err
	}

	return &model.Metric{
		ID:    mID,
		MType: mType,
		Delta: parseDelta(mDelta),
		Value: parseValue(mValue),
	}, nil
}

func (storage *DBStorage) GetAllMetrics() (model.MetricsStore, error) {
	rows, err := storage.db.Query("SELECT id, type, delta, value FROM metrics")
	if err != nil {
		storage.logger.Error("GetAll: select error", zap.Error(err))
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			storage.logger.Error("GetAll: select error", zap.Error(err))
		}
	}(rows)

	metrics := make(model.MetricsStore)
	for rows.Next() {
		var m model.Metric

		err := rows.Scan(&m.ID, &m.MType, &m.Delta, &m.Value)
		if err != nil {
			storage.logger.Error("GetAll: scan row error", zap.Error(err))
			return nil, err
		}

		_, ok := metrics[m.MType]
		if !ok {
			metrics[m.MType] = map[string]model.Metric{
				m.ID: m,
			}
		} else {
			metrics[m.MType][m.ID] = m
		}
	}

	if err := rows.Err(); err != nil {
		storage.logger.Error("GetAll: rows error", zap.Error(err))
		return nil, err
	}

	return metrics, err
}

func (storage *DBStorage) ClearMetrics() error {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	storage.metrics = make(map[string]model.Metric)

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
