package memory

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type Store struct {
	mu   sync.Mutex
	path string
}

func NewStore(path string) (*Store, error) {
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	return &Store{path: path}, nil
}

func (s *Store) Append(event map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	b, _ := json.Marshal(event)
	_, err = f.Write(append(b, '\n'))
	return err
}

func (s *Store) Hash() ([32]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := os.ReadFile(s.path)
	if err != nil {
		return [32]byte{}, err
	}
	return sha256.Sum256(b), nil
}
