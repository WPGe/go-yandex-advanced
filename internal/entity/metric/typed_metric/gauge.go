package typed_metric

import "strconv"

type Gauge struct {
	Name  string
	Value float64
}

func NewGauge(name, value string) (*Gauge, error) {
	gaugeValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil, err
	}
	return &Gauge{
		Name:  name,
		Value: gaugeValue,
	}, nil
}
