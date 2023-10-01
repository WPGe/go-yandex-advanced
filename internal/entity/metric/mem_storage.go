package metric

type MemStorage struct {
	Metrics map[string]map[string]interface{}
}

func NewMemStorage() *MemStorage {
	return &MemStorage{Metrics: make(map[string]map[string]interface{})}
}
