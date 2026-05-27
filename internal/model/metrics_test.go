package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStorageModel(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "creates empty maps"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := NewStorageModel()
			require.NotNil(t, model)
			require.NotNil(t, model.Counters)
			require.NotNil(t, model.Gauges)
			assert.Empty(t, model.Counters)
			assert.Empty(t, model.Gauges)
		})
	}
}
