package config

import "errors"

var (
	// ErrNotExists reports that the requested metric does not exist.
	ErrNotExists = errors.New("item does not exist")
	// ErrMetricNotFound reports that the requested metric was not found.
	ErrMetricNotFound = errors.New("no metric found")
)
