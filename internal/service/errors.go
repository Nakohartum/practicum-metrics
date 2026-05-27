package service

import "errors"

var (
	// ErrMetricNameRequired reports that a metric name was not provided.
	ErrMetricNameRequired = errors.New("no metric's name")
	// ErrMetricTypeNotSupported reports an unknown metric type.
	ErrMetricTypeNotSupported = errors.New("no such metric type")
	// ErrCounterDeltaRequired reports that a counter metric has no delta value.
	ErrCounterDeltaRequired = errors.New("delta value is required for counter type")
	// ErrGaugeValueRequired reports that a gauge metric has no value.
	ErrGaugeValueRequired = errors.New("value is required for gauge type")
	// ErrMetricNotFound reports that a metric was not found.
	ErrMetricNotFound = errors.New("metric not found error")
)
