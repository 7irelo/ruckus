package store

import (
	"os"
	"path/filepath"
)

const (
	DefaultDirName = ".ruckus"
	DefaultDBName  = "ruckus.db"
)

func DefaultDBPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, DefaultDirName, DefaultDBName), nil
}
