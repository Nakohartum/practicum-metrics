package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	models "github.com/Nakohartum/practicum-metrics/internal/model"
)

func closeFileHandles(t *testing.T, reader *FileReader, writer *FileWriter) {
	t.Helper()
	if reader != nil && reader.file != nil {
		require.NoError(t, reader.file.Close())
	}
	if writer != nil && writer.file != nil {
		require.NoError(t, writer.file.Close())
	}
}

func TestReadData(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantLen int
		wantErr bool
	}{
		{name: "empty file returns empty slice", content: "", wantLen: 0},
		{name: "invalid json returns error", content: "{invalid", wantErr: true},
		{
			name:    "valid json returns metrics",
			content: `[{"id":"g1","type":"gauge","value":1.5}]`,
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "metrics.json")
			require.NoError(t, os.WriteFile(path, []byte(tt.content), 0o666))

			reader, err := NewFileReader(path)
			require.NoError(t, err)
			defer closeFileHandles(t, reader, nil)

			got, err := reader.ReadData()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, got, tt.wantLen)
		})
	}
}

func TestWriteData(t *testing.T) {
	tests := []struct {
		name  string
		input []models.Metrics
	}{
		{
			name: "writes metrics array",
			input: []models.Metrics{{
				ID:    "c1",
				MType: models.Counter,
				Delta: func() *int64 { v := int64(10); return &v }(),
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "metrics.json")

			writer, err := NewFileWriter(path)
			require.NoError(t, err)
			reader, err := NewFileReader(path)
			require.NoError(t, err)
			defer closeFileHandles(t, reader, writer)

			require.NoError(t, writer.WriteData(tt.input))

			got, err := reader.ReadData()
			require.NoError(t, err)
			require.Len(t, got, len(tt.input))
			assert.Equal(t, tt.input[0].ID, got[0].ID)
			assert.Equal(t, tt.input[0].MType, got[0].MType)
		})
	}
}

func TestWriteOneData(t *testing.T) {
	tests := []struct {
		name         string
		initial      []models.Metrics
		insert       models.Metrics
		wantContains string
	}{
		{
			name:    "adds first metric",
			initial: []models.Metrics{},
			insert: models.Metrics{
				ID:    "g1",
				MType: models.Gauge,
				Value: func() *float64 { v := 12.3; return &v }(),
			},
			wantContains: "g1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "metrics.json")

			writer, err := NewFileWriter(path)
			require.NoError(t, err)
			reader, err := NewFileReader(path)
			require.NoError(t, err)
			defer closeFileHandles(t, reader, writer)

			manager := NewFileManager(reader, writer)
			require.NoError(t, manager.WriteData(tt.initial))
			require.NoError(t, manager.WriteOneData(tt.insert))

			got, err := manager.ReadData()
			require.NoError(t, err)
			require.NotEmpty(t, got)

			found := false
			for _, m := range got {
				if m.ID == tt.wantContains {
					found = true
					break
				}
			}
			assert.True(t, found)
		})
	}
}

func TestFileExists(t *testing.T) {
	tests := []struct {
		name    string
		build   func(*testing.T) (*FileManager, *FileReader, *FileWriter)
		wantErr bool
	}{
		{
			name: "returns error for zero manager",
			build: func(t *testing.T) (*FileManager, *FileReader, *FileWriter) {
				return &FileManager{}, nil, nil
			},
			wantErr: true,
		},
		{
			name: "returns nil when files initialized",
			build: func(t *testing.T) (*FileManager, *FileReader, *FileWriter) {
				dir := t.TempDir()
				path := filepath.Join(dir, "metrics.json")
				writer, err := NewFileWriter(path)
				require.NoError(t, err)
				reader, err := NewFileReader(path)
				require.NoError(t, err)
				return NewFileManager(reader, writer), reader, writer
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, reader, writer := tt.build(t)
			defer closeFileHandles(t, reader, writer)

			err := manager.FileExists()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestWriteDataProducesJSON(t *testing.T) {
	tests := []struct {
		name string
		data []models.Metrics
	}{
		{
			name: "json can be decoded",
			data: []models.Metrics{{ID: "c", MType: models.Counter, Delta: func() *int64 { v := int64(1); return &v }()}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "metrics.json")
			writer, err := NewFileWriter(path)
			require.NoError(t, err)
			defer closeFileHandles(t, nil, writer)

			require.NoError(t, writer.WriteData(tt.data))

			raw, err := os.ReadFile(path)
			require.NoError(t, err)
			var decoded []models.Metrics
			require.NoError(t, json.Unmarshal(raw, &decoded))
			assert.NotEmpty(t, decoded)
		})
	}
}
