package storage

// TODO: Think in the future of concurrency; Adding a mutex for multiple go routines calling this
import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// WriteJSON marshals value to JSON and writes it to dir/name, creating dir
// if it does not exist.
func WriteJSON(dir, name string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal value: %w", err)
	}

	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("create dir %s: %w", dir, err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, os.ModePerm); err != nil {
		return fmt.Errorf("write file %s: %w", path, err)
	}
	return nil
}
