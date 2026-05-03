package models

const (
	// Counter is the metric type for monotonically accumulated integer values.
	Counter = "counter"

	// Gauge is the metric type for floating-point point-in-time values.
	Gauge = "gauge"
)

// NOTE: Не усложняем пример, вводя иерархическую вложенность структур.
// Органичиваясь плоской моделью.
// Delta и Value объявлены через указатели,
// что бы отличать значение "0", от не заданного значения
// и соответственно не кодировать в структуру.

// Metrics describes one counter or gauge value in API and storage payloads.
type Metrics struct {
	ID    string   `json:"id"`              // имя метрики
	MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
	Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
}

// StorageModel stores metrics split by their concrete value type.
type StorageModel struct {
	Counters map[string]int64
	Gauges   map[string]float64
}

// NewStorageModel creates an empty in-memory storage model.
func NewStorageModel() *StorageModel {
	return &StorageModel{
		Counters: make(map[string]int64),
		Gauges:   make(map[string]float64),
	}
}
