package entity

const (
	Gauge   string = "gauge"
	Counter string = "counter"
)

type Metric struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
}

type MetricsStore map[string]map[string]Metric
