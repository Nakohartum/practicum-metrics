package service

import "errors"

var (
	ErrMetricNameRequired     = errors.New("no metric's name")
	ErrMetricTypeNotSupported = errors.New("no such metric type")
	ErrCounterDeltaRequired   = errors.New("delta value is required for counter type")
	ErrGaugeValueRequired     = errors.New("value is required for gauge type")
	ErrMetricNotFound         = errors.New("metric not found error")
)
