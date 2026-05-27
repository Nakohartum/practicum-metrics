package repository

import "errors"

var (
	errNoConnectionToClose    = errors.New("no connection to close")
	errMetricTypeNotSupported = errors.New("no such metric type")
)
