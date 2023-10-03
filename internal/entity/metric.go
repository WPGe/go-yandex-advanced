package entity

type Type string

const (
	Gauge   Type = "gauge"
	Counter Type = "counter"
)

type Metric struct {
	Type  Type
	Name  string
	Value interface{}
}
