package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// LoadDotenv loads the first .env file it finds, searching the working
// directory and executable directory and their parents. Missing files are not
// an error. Shared by all entrypoints (worker, bot).
func LoadDotenv() error {
	paths := []string{".env"}
	if wd, err := os.Getwd(); err == nil {
		for current := wd; ; current = filepath.Dir(current) {
			paths = append(paths, filepath.Join(current, ".env"))
			parent := filepath.Dir(current)
			if parent == current {
				break
			}
		}
	}
	if exe, err := os.Executable(); err == nil {
		for current := filepath.Dir(exe); ; current = filepath.Dir(current) {
			paths = append(paths, filepath.Join(current, ".env"))
			parent := filepath.Dir(current)
			if parent == current {
				break
			}
		}
	}

	seen := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		if err := godotenv.Load(path); err == nil {
			return nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}

	return nil
}
