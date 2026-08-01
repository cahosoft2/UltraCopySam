//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const stateFileName = ".ucsam-state"

type sessionState struct {
	Version       int             `json:"version"`
	Source        string          `json:"source"`
	Destination   string          `json:"destination"`
	StartedAt     time.Time       `json:"started_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
	CompletedDirs map[string]bool `json:"completed_dirs"` // rutas relativas en minúsculas

	mu      sync.RWMutex `json:"-"`
	dstPath string       `json:"-"`
}

func newSessionState(src, dst string) *sessionState {
	return &sessionState{
		Version:       1,
		Source:        strings.ToLower(filepath.Clean(displayPath(src))),
		Destination:   strings.ToLower(filepath.Clean(displayPath(dst))),
		StartedAt:     time.Now(),
		CompletedDirs: make(map[string]bool),
		dstPath:       dst,
	}
}

func loadSessionState(src, dst string) (*sessionState, error) {
	stateFile := join(dst, stateFileName)
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return nil, err
	}

	var st sessionState
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, fmt.Errorf("archivo de estado corrupto: %w", err)
	}
	st.dstPath = dst
	if st.CompletedDirs == nil {
		st.CompletedDirs = make(map[string]bool)
	}

	cleanSrc := strings.ToLower(filepath.Clean(displayPath(src)))
	cleanDst := strings.ToLower(filepath.Clean(displayPath(dst)))

	if st.Source != "" && st.Source != cleanSrc {
		return nil, fmt.Errorf("el origen previo (%s) no coincide con el actual (%s)", st.Source, cleanSrc)
	}
	if st.Destination != "" && st.Destination != cleanDst {
		return nil, fmt.Errorf("el destino previo (%s) no coincide con el actual (%s)", st.Destination, cleanDst)
	}

	return &st, nil
}

func (s *sessionState) isDirCompleted(rel string) bool {
	if s == nil {
		return false
	}
	key := normRelPath(rel)
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.CompletedDirs[key]
}

func (s *sessionState) markDirCompleted(rel string) {
	if s == nil {
		return
	}
	key := normRelPath(rel)
	s.mu.Lock()
	s.CompletedDirs[key] = true
	s.mu.Unlock()
}

func (s *sessionState) saveAtomic() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	s.UpdatedAt = time.Now()
	data, err := json.MarshalIndent(s, "", "  ")
	s.mu.Unlock()
	if err != nil {
		return err
	}

	targetFile := join(s.dstPath, stateFileName)
	tmpFile := targetFile + ".tmp"

	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return err
	}
	_ = os.Remove(targetFile)
	return os.Rename(tmpFile, targetFile)
}

func (s *sessionState) remove() {
	if s == nil {
		return
	}
	targetFile := join(s.dstPath, stateFileName)
	_ = os.Remove(targetFile)
	_ = os.Remove(targetFile + ".tmp")
}

func normRelPath(rel string) string {
	r := strings.ToLower(filepath.Clean(rel))
	if r == "." || r == `\` || r == "/" {
		return ""
	}
	return strings.TrimPrefix(r, `\`)
}
