package storage_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"recoTask/internal/storage"
)

func TestWriteJSON_WritesMarshaledValue(t *testing.T) {
	dir := t.TempDir()

	type record struct {
		GID  string `json:"gid"`
		Name string `json:"name"`
	}
	value := record{GID: "123", Name: "test"}

	if err := storage.WriteJSON(dir, "record.json", value); err != nil {
		t.Fatalf("WriteJSON returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "record.json"))
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}

	var got record
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal written file: %v", err)
	}
	if got != value {
		t.Errorf("got %+v, want %+v", got, value)
	}
}

func TestWriteJSON_CreatesNestedDirs(t *testing.T) {
	base := t.TempDir()
	nested := filepath.Join(base, "a", "b", "c")

	if err := storage.WriteJSON(nested, "file.json", map[string]int{"x": 1}); err != nil {
		t.Fatalf("WriteJSON returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(nested, "file.json")); err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}
}

func TestWriteJSON_MarshalError(t *testing.T) {
	dir := t.TempDir()

	// channels are not JSON-marshalable.
	err := storage.WriteJSON(dir, "bad.json", make(chan int))
	if err == nil {
		t.Fatal("expected an error for an unmarshalable value, got nil")
	}

	if _, statErr := os.Stat(filepath.Join(dir, "bad.json")); !os.IsNotExist(statErr) {
		t.Errorf("expected no file to be written on marshal error, stat err = %v", statErr)
	}
}
