package camera

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type TestCamera struct {
	// MockStorage is the local path to your test images folder
	MockStorage string
}

// ListPhotos returns the names of all files in the mock directory
func (t *TestCamera) ListPhotos() ([]string, error) {
	entries, err := os.ReadDir(t.MockStorage)
	if err != nil {
		return nil, fmt.Errorf("failed to list mock photos: %w", err)
	}

	var photos []string
	for _, entry := range entries {
		if !entry.IsDir() {
			photos = append(photos, entry.Name())
		}
	}
	return photos, nil
}

// GetPhotoReader opens a local file and returns it as an io.ReadCloser
func (t *TestCamera) GetPhotoReader(path string) (io.ReadCloser, error) {
	// filepath.Join handles the OS-specific separators
	fullPath := filepath.Join(t.MockStorage, path)

	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("could not open mock file %s: %w", path, err)
	}
	return file, nil
}

// DeletePhoto removes the file from your local mock directory
func (t *TestCamera) DeletePhoto(path string) error {
	fullPath := filepath.Join(t.MockStorage, path)

	err := os.Remove(fullPath)
	if err != nil {
		return fmt.Errorf("failed to delete mock file %s: %w", path, err)
	}
	return nil
}
