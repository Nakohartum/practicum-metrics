package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	models "github.com/Nakohartum/practicum-metrics/internal/model"
)

func readFileText(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	require.NoError(t, err)
	return string(b)
}

func TestNewFileWriter_Table(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		path    func(t *testing.T) string
		wantErr bool
	}{
		{
			name: "ok_creates_file",
			path: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "metrics.json")
			},
			wantErr: false,
		},
		{
			name: "error_when_parent_dir_missing",
			path: func(t *testing.T) string {
				// родительской папки нет -> OpenFile должен упасть
				return filepath.Join(t.TempDir(), "no_such_dir", "metrics.json")
			},
			wantErr: true,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			p := tt.path(t)

			fw, err := NewFileWriter(p)
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, fw)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, fw)
			require.NotNil(t, fw.file)
			defer fw.file.Close()

			_, statErr := os.Stat(p)
			require.NoError(t, statErr)
		})
	}
}

func TestFileWriter_WriteData_Table(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		first  []models.Metrics
		second []models.Metrics
	}{
		{
			name:   "overwrite_with_empty_slice",
			first:  []models.Metrics{{ID: "a", MType: "counter"}},
			second: []models.Metrics{},
		},
		{
			name:   "overwrite_non_empty_to_other_non_empty",
			first:  []models.Metrics{{ID: "a", MType: "counter"}, {ID: "b", MType: "gauge"}},
			second: []models.Metrics{{ID: "c", MType: "counter"}},
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), tt.name+".json")

			fw, err := NewFileWriter(path)
			require.NoError(t, err)
			defer fw.file.Close()

			// если у тебя всё ещё O_APPEND в NewFileWriter — тут и будет падать на Windows.
			require.NoError(t, fw.WriteData(tt.first))
			a := readFileText(t, path)
			require.NotEmpty(t, a)

			require.NoError(t, fw.WriteData(tt.second))
			b := readFileText(t, path)
			require.NotEmpty(t, b)

			if len(tt.first) != len(tt.second) {
				assert.NotEqual(t, a, b)
			}
		})
	}
}

func TestFileWriter_WriteOneData_Appends(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "append.json")

	fw, err := NewFileWriter(path)
	require.NoError(t, err)
	defer fw.file.Close()
	metrics := make([]models.Metrics, 0)
	require.NoError(t, fw.WriteOneData(metrics,models.Metrics{ID: "a", MType: "counter"}))
	require.NoError(t, fw.WriteOneData(metrics, models.Metrics{ID: "b", MType: "gauge"}))

	txt := readFileText(t, path)

	// у тебя JSON теги lowercase: "id" / "type"
	assert.Contains(t, txt, `"id":"a"`)
	assert.Contains(t, txt, `"id":"b"`)
}

func TestNewFileReader_Table(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		path    func(t *testing.T) string
		wantErr bool
	}{
		{
			name: "ok_creates_if_missing",
			path: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "missing.json")
			},
			wantErr: false,
		},
		{
			name: "error_when_parent_dir_missing",
			path: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "no_such_dir", "missing.json")
			},
			wantErr: true,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			p := tt.path(t)

			fr, err := NewFileReader(p)
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, fr)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, fr)
			require.NotNil(t, fr.file)
			defer fr.file.Close()

			_, statErr := os.Stat(p)
			require.NoError(t, statErr)
		})
	}
}

func TestFileReader_ReadData_Table(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		fileText string
		wantErr  bool
		wantLen  int
	}{
		{
			name:     "valid_json_array",
			fileText: `[{"id":"a","type":"counter"},{"id":"b","type":"gauge"}]`,
			wantErr:  false,
			wantLen:  2,
		},
		{
			name:     "empty_file_returns_error",
			fileText: ``,
			wantErr:  true,
			wantLen:  0,
		},
		{
			name:     "invalid_json_returns_error",
			fileText: `{not-json`,
			wantErr:  true,
			wantLen:  0,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), tt.name+".json")
			require.NoError(t, os.WriteFile(path, []byte(tt.fileText), 0666))

			fr, err := NewFileReader(path)
			require.NoError(t, err)
			defer fr.file.Close()

			data, err := fr.ReadData()
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Len(t, data, tt.wantLen)
		})
	}
}

func TestFileManager_Table(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		run  func(t *testing.T, fm *FileManager, path string)
	}{
		{
			name: "write_and_read_roundtrip",
			run: func(t *testing.T, fm *FileManager, path string) {
				in := []models.Metrics{{ID: "x", MType: "counter"}}
				require.NoError(t, fm.WriteData(in))

				// decoder читает с текущей позиции, поэтому для roundtrip переоткроем reader
				_ = fm.fileReader.file.Close()
				r2, err := NewFileReader(path)
				require.NoError(t, err)
				defer r2.file.Close()
				fm.fileReader = *r2

				out, err := fm.ReadData()
				require.NoError(t, err)
				require.Len(t, out, 1)
				assert.Equal(t, "x", out[0].ID)
			},
		},
		{
			name: "write_one_appends",
			run: func(t *testing.T, fm *FileManager, _ string) {
				require.NoError(t, fm.WriteOneData(models.Metrics{ID: "a", MType: "counter"}))
				require.NoError(t, fm.WriteOneData(models.Metrics{ID: "b", MType: "gauge"}))

				txt := readFileText(t, fm.fileWriter.file.Name())
				assert.Contains(t, txt, `"id":"a"`)
				assert.Contains(t, txt, `"id":"b"`)
			},
		},
		{
			name: "file_exists_ok",
			run: func(t *testing.T, fm *FileManager, _ string) {
				require.NoError(t, fm.FileExists())
			},
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), tt.name+".json")

			fw, err := NewFileWriter(path)
			require.NoError(t, err)
			defer fw.file.Close()

			fr, err := NewFileReader(path)
			require.NoError(t, err)
			defer fr.file.Close()

			fm := NewFileManager(fr, fw)
			tt.run(t, fm, path)
		})
	}
}

func TestFileManager_FileExists_NilFiles(t *testing.T) {
	t.Parallel()

	fm := &FileManager{
		fileReader: FileReader{},
		fileWriter: FileWriter{},
	}
	require.Error(t, fm.FileExists())
}