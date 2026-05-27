package config

import "errors"

var (
	// ErrFileDoesNotExist reports that the configured file storage is missing.
	ErrFileDoesNotExist = errors.New("file does not exist")
)
