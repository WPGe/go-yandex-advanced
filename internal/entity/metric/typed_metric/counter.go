package typed_metric

import "strconv"

type Counter struct {
	Name  string
	Value int64
}

func NewCounter(name, value string) (*Counter, error) {
	counterValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return nil, err
	}
	return &Counter{
		Name:  name,
		Value: counterValue,
	}, nil
}
