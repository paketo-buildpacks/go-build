package gobuild

import (
	"fmt"
	"os"
	"path/filepath"
)

type SourceDeleter struct{}

func NewSourceDeleter() SourceDeleter {
	return SourceDeleter{}
}

func (d SourceDeleter) Clear(path string) error {
	matches, err := filepath.Glob(filepath.Join(path, "*"))
	if err != nil {
		return fmt.Errorf("failed to remove source: %w", err)
	}

	for _, match := range matches {
		err = os.RemoveAll(match)
		if err != nil {
			return fmt.Errorf("failed to remove source: %w", err)
		}
	}

	return nil
}
