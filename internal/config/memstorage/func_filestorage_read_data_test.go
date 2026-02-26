package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFileManager_ReadData(t *testing.T) {
	tmp := t.TempDir()

	tests := []struct {
		name    string
		content []byte
		wantLen int
		wantErr bool
	}{
		{
			name:    "read one gauge from file",
			content: []byte(`[{"id":"g1","type":"gauge","value":7.7}]`),
			wantLen: 1,
			wantErr: false,
		},
		{
			name:    "read invalid json -> error",
			content: []byte(`{bad`),
			wantLen: 0,
			wantErr: true,
		},
		{
			name:    "empty file -> EOF error",
			content: []byte(``),
			wantLen: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(tmp, sanitize(tt.name)+".json")

			err := os.WriteFile(path, tt.content, 0666)
			require.NoError(t, err)
			

			fr, err := NewFileReader(path)
			require.NoError(t, err)
			defer fr.file.Close()

			fw, err := NewFileWriter(path)
			require.NoError(t, err)
			defer fw.file.Close()

			fm := NewFileManager(fr, fw)

			got, err := fm.ReadData()
			if tt.wantErr {
				require.Error(t, err)
			} else{
				require.NoError(t, err)
				require.Equal(t, tt.wantLen, len(got))
			}
			
		})
	}
}