package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	models "github.com/Nakohartum/practicum-metrics/internal/model"
)

func TestNewFileWriter(t *testing.T) {
	tmp := t.TempDir()

	tests := []struct {
		name     string
		makeFile bool
		wantErr  bool
	}{
		{name: "file exists -> ok", makeFile: true, wantErr: false},
		{name: "file missing -> error", makeFile: false, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(tmp, sanitize(tt.name)+".json")

			if tt.makeFile {
				if err := os.WriteFile(path, []byte{}, 0666); err != nil {
					t.Fatalf("prepare file: %v", err)
				}
			}

			fw, err := NewFileWriter(path)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer fw.file.Close()
		})
	}
}

func TestFileWriter_WriteData(t *testing.T) {
	tmp := t.TempDir()

	tests := []struct {
		name    string
		data    []models.Metrics
		wantErr bool
	}{
		{
			name: "write gauge metric",
			data: []models.Metrics{
				{ID: "g1", MType: "gauge", Value: ptrFloat64(1.23)},
			},
			wantErr: false,
		},
		{
			name: "write counter metric",
			data: []models.Metrics{
				{ID: "c1", MType: "counter", Delta: ptrInt64(5)},
			},
			wantErr: false,
		},
		{
			name:    "write empty slice",
			data:    []models.Metrics{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(tmp, sanitize(tt.name)+".json")

			
			if err := os.WriteFile(path, []byte{}, 0666); err != nil {
				t.Fatalf("prepare file: %v", err)
			}

			fw, err := NewFileWriter(path)
			if err != nil {
				t.Fatalf("NewFileWriter: %v", err)
			}
			defer fw.file.Close()

			err = fw.WriteData(tt.data)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			
			b, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile: %v", err)
			}
			if len(b) == 0 {
				t.Fatal("expected non-empty file content")
			}

			
			var decoded []models.Metrics
			if err := json.Unmarshal(b, &decoded); err != nil {
				t.Fatalf("expected valid json, got error: %v\ncontent: %s", err, string(b))
			}
			if len(decoded) != len(tt.data) {
				t.Fatalf("decoded len=%d want %d", len(decoded), len(tt.data))
			}
		})
	}
}

func TestNewFileReader(t *testing.T) {
	tmp := t.TempDir()

	tests := []struct {
		name     string
		preExist bool
		content  []byte
		wantErr  bool
	}{
		{name: "file exists -> ok", preExist: true, content: []byte("[]"), wantErr: false},
		{name: "file missing but O_CREATE -> ok", preExist: false, content: nil, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(tmp, sanitize(tt.name)+".json")

			if tt.preExist {
				if err := os.WriteFile(path, tt.content, 0666); err != nil {
					t.Fatalf("prepare file: %v", err)
				}
			}

			fr, err := NewFileReader(path)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer fr.file.Close()

			
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("file should exist: %v", err)
			}
		})
	}
}

func TestFileReader_ReadData(t *testing.T) {
	tmp := t.TempDir()

	tests := []struct {
		name    string
		content []byte
		wantLen int
		wantErr bool
	}{
		{
			name:    "valid json with one counter",
			content: []byte(`[{"id":"c1","type":"counter","delta":5}]`),
			wantLen: 1,
			wantErr: false,
		},
		{
			name:    "valid empty json array",
			content: []byte(`[]`),
			wantLen: 0,
			wantErr: false,
		},
		{
			name:    "invalid json",
			content: []byte(`{oops`),
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
			if err := os.WriteFile(path, tt.content, 0666); err != nil {
				t.Fatalf("prepare file: %v", err)
			}

			fr, err := NewFileReader(path)
			if err != nil {
				t.Fatalf("NewFileReader: %v", err)
			}
			defer fr.file.Close()

			got, err := fr.ReadData()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(got) != tt.wantLen {
				t.Fatalf("len(got)=%d want %d; got=%+v", len(got), tt.wantLen, got)
			}
		})
	}
}

func TestNewFileManager(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "data.json")

	
	if err := os.WriteFile(path, []byte("[]"), 0666); err != nil {
		t.Fatalf("prepare file: %v", err)
	}

	fr, err := NewFileReader(path)
	if err != nil {
		t.Fatalf("NewFileReader: %v", err)
	}
	defer fr.file.Close()

	fw, err := NewFileWriter(path)
	if err != nil {
		t.Fatalf("NewFileWriter: %v", err)
	}
	defer fw.file.Close()

	fm := NewFileManager(fr, fw)
	if fm == nil {
		t.Fatal("expected non-nil FileManager")
	}
}

func TestFileManager_WriteData(t *testing.T) {
	tmp := t.TempDir()

	tests := []struct {
		name    string
		input   []models.Metrics
		wantErr bool
	}{
		{
			name: "write counter",
			input: []models.Metrics{
				{ID: "c1", MType: "counter", Delta: ptrInt64(10)},
			},
			wantErr: false,
		},
		{
			name: "write empty slice",
			input: []models.Metrics{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(tmp, sanitize(tt.name)+".json")

			// writer требует существующий файл
			if err := os.WriteFile(path, []byte{}, 0666); err != nil {
				t.Fatalf("prepare file: %v", err)
			}

			fw, err := NewFileWriter(path)
			if err != nil {
				t.Fatalf("NewFileWriter: %v", err)
			}
			defer fw.file.Close()

			fr, err := NewFileReader(path)
			if err != nil {
				t.Fatalf("NewFileReader: %v", err)
			}
			defer fr.file.Close()

			fm := NewFileManager(fr, fw)

			err = fm.WriteData(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Проверим, что в файле валидный JSON
			b, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile: %v", err)
			}
			var decoded []models.Metrics
			if err := json.Unmarshal(b, &decoded); err != nil {
				t.Fatalf("invalid json: %v\ncontent: %s", err, string(b))
			}
			if len(decoded) != len(tt.input) {
				t.Fatalf("decoded len=%d want %d", len(decoded), len(tt.input))
			}
		})
	}
}



func sanitize(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			out = append(out, r)
		case r >= 'A' && r <= 'Z':
			out = append(out, r)
		case r >= '0' && r <= '9':
			out = append(out, r)
		default:
			out = append(out, '_')
		}
	}
	return string(out)
}