package storage

import (
	"errors"
	"os"
	"path/filepath"
)

type Store struct {
	DataDir string
}

func New(dataDir string) (*Store, error) {
	if dataDir == "" {
		return nil, errors.New("dataDir must not be empty")
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, err
	}
	return &Store{DataDir: dataDir}, nil
}

func (s *Store) Path(elem ...string) string {
	parts := append([]string{s.DataDir}, elem...)
	return filepath.Join(parts...)
}
