package tui

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// HistoryManager manages persistent input history across sessions
type HistoryManager struct {
	mu          sync.RWMutex
	historyPath string
	history     []string
	maxSize     int
}

// NewHistoryManager creates a new history manager
func NewHistoryManager(configDir string, maxSize int) (*HistoryManager, error) {
	if maxSize <= 0 {
		maxSize = 1000
	}

	historyPath := filepath.Join(configDir, "history")

	hm := &HistoryManager{
		historyPath: historyPath,
		history:     make([]string, 0),
		maxSize:     maxSize,
	}

	// Load existing history
	if err := hm.load(); err != nil {
		// If file doesn't exist, that's OK
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	return hm, nil
}

// load reads history from disk
func (hm *HistoryManager) load() error {
	file, err := os.Open(hm.historyPath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			// Unescape newlines in multi-line entries
			unescaped := strings.ReplaceAll(line, "\\n", "\n")
			hm.history = append(hm.history, unescaped)
		}
	}

	return scanner.Err()
}

// save writes history to disk
func (hm *HistoryManager) save() error {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	// Ensure directory exists
	dir := filepath.Dir(hm.historyPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Write to temporary file first
	tmpPath := hm.historyPath + ".tmp"
	file, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	writer := bufio.NewWriter(file)
	for _, entry := range hm.history {
		// Escape newlines in multi-line entries
		escaped := strings.ReplaceAll(entry, "\n", "\\n")
		if _, err := writer.WriteString(escaped + "\n"); err != nil {
			file.Close()
			os.Remove(tmpPath)
			return err
		}
	}

	if err := writer.Flush(); err != nil {
		file.Close()
		os.Remove(tmpPath)
		return err
	}

	if err := file.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}

	// Atomic rename
	return os.Rename(tmpPath, hm.historyPath)
}

// Add adds a new entry to history (in-memory only, call Flush to persist)
func (hm *HistoryManager) Add(entry string) {
	entry = strings.TrimSpace(entry)
	if entry == "" {
		return
	}

	hm.mu.Lock()
	defer hm.mu.Unlock()

	// Don't add duplicate consecutive entries
	if len(hm.history) > 0 && hm.history[len(hm.history)-1] == entry {
		return
	}

	hm.history = append(hm.history, entry)

	// Limit history size
	if len(hm.history) > hm.maxSize {
		hm.history = hm.history[len(hm.history)-hm.maxSize:]
	}
}

// GetAll returns all history entries
func (hm *HistoryManager) GetAll() []string {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	// Return a copy
	result := make([]string, len(hm.history))
	copy(result, hm.history)
	return result
}

// Flush writes the current history to disk
func (hm *HistoryManager) Flush() error {
	return hm.save()
}

// Clear clears all history (in-memory only, call Flush to persist)
func (hm *HistoryManager) Clear() {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	hm.history = make([]string, 0)
}

// Size returns the number of history entries
func (hm *HistoryManager) Size() int {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	return len(hm.history)
}
