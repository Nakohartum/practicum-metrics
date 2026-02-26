package config

import "context"

type DatabaseAdapter interface {
	Open(ctx context.Context) error
	Close(ctx context.Context) error
	CheckConnection(ctx context.Context) error
}