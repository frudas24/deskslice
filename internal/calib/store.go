package calib

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// Load reads calibration data from disk. Missing files return empty data.
func Load(path string) (Calib, error) {
	var c Calib
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return c, nil
		}
		return c, err
	}
	if err := json.Unmarshal(data, &c); err != nil {
		return c, err
	}
	return c, nil
}

// Save writes calibration data to disk, creating parent directories as needed.
func Save(path string, c Calib) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
