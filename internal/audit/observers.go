package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
)

// FileObserver writes audit events to a file.
type FileObserver struct {
	path string
}

// NewFileObserver creates a FileObserver for the provided path.
func NewFileObserver(path string) *FileObserver {
	return &FileObserver{path: path}
}

// Notify appends an audit event to the configured file.
func (fO *FileObserver) Notify(ctx context.Context, event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(fO.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(append(data, '\n'))
	return err
}

// HTTPObserver sends audit events to an HTTP endpoint.
type HTTPObserver struct {
	url    string
	client *http.Client
}

// NewHTTPObserver creates an HTTPObserver for the provided URL and client.
func NewHTTPObserver(url string, client *http.Client) *HTTPObserver {
	return &HTTPObserver{
		url:    url,
		client: client,
	}
}

// Notify posts an audit event to the configured HTTP endpoint.
func (hO *HTTPObserver) Notify(ctx context.Context, event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, hO.url, bytes.NewBuffer(data))

	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := hO.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return errUnexpectedStatusCode
	}
	return nil
}
