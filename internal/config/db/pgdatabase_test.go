package config

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPgDatabaseAdapter(t *testing.T) {
	tests := []struct {
		name             string
		connectionString string
	}{
		{
			name:             "stores connection string",
			connectionString: "postgres://user:pass@localhost:5432/db",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewPgDatabaseAdapter(tt.connectionString)
			require.NotNil(t, adapter)
			assert.Equal(t, tt.connectionString, adapter.connectionString)
		})
	}
}

func TestClose(t *testing.T) {
	tests := []struct {
		name    string
		adapter *PgDatabaseAdapter
		wantErr error
	}{
		{
			name:    "returns explicit error when no connection",
			adapter: &PgDatabaseAdapter{},
			wantErr: errNoConnectionToClose,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.adapter.Close(context.Background())
			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestRunMigrations(t *testing.T) {
	tests := []struct {
		name    string
		dir     string
		wantErr bool
	}{
		{
			name:    "returns error for missing directory",
			dir:     filepath.Join(t.TempDir(), "missing"),
			wantErr: true,
		},
		{
			name:    "empty directory succeeds",
			dir:     t.TempDir(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := &PgDatabaseAdapter{}
			err := adapter.runMigrations(context.Background(), tt.dir)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
