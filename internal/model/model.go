package model

const (
	Gauge   string = "gauge"
	Counter string = "counter"
)

type Metric struct {
	ID    string   `json:"id" db:"id"`
	MType string   `json:"type" db:"type"`
	Delta *int64   `json:"delta,omitempty" db:"delta"`
	Value *float64 `json:"value,omitempty" db:"value"`
}

func NewGaugeMetric(name string, value float64) Metric {
	return Metric{
		MType: Gauge,
		ID:    name,
		Value: &value,
	}
}

func NewCounterMetric(name string, value int64) Metric {
	return Metric{
		MType: Counter,
		ID:    name,
		Delta: &value,
	}
}

type MetricsStore map[string]map[string]Metric

type Error struct {
	Error string `json:"error"`
}
