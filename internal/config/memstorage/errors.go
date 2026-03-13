package config

import "errors"

var (
	ErrNotExists      = errors.New("item does not exist")
	ErrMetricNotFound = errors.New("no metric found")
)
