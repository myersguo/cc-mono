package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHistoryManager_AddAndLoad(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Create history manager
	hm, err := NewHistoryManager(tempDir, 100)
	require.NoError(t, err)

	// Add entries
	hm.Add("first entry")
	hm.Add("second entry")
	hm.Add("third entry")

	// Flush to disk
	err = hm.Flush()
	require.NoError(t, err)

	// Verify entries
	history := hm.GetAll()
	assert.Len(t, history, 3)
	assert.Equal(t, "first entry", history[0])
	assert.Equal(t, "second entry", history[1])
	assert.Equal(t, "third entry", history[2])

	// Create new manager (simulates new session)
	hm2, err := NewHistoryManager(tempDir, 100)
	require.NoError(t, err)

	// Verify history is loaded
	history2 := hm2.GetAll()
	assert.Len(t, history2, 3)
	assert.Equal(t, "first entry", history2[0])
	assert.Equal(t, "second entry", history2[1])
	assert.Equal(t, "third entry", history2[2])
}

func TestHistoryManager_DuplicateConsecutive(t *testing.T) {
	tempDir := t.TempDir()
	hm, err := NewHistoryManager(tempDir, 100)
	require.NoError(t, err)

	// Add duplicate consecutive entries
	hm.Add("command 1")
	hm.Add("command 1") // Duplicate
	hm.Add("command 2")

	// Verify only unique consecutive entries
	history := hm.GetAll()
	assert.Len(t, history, 2)
	assert.Equal(t, "command 1", history[0])
	assert.Equal(t, "command 2", history[1])
}

func TestHistoryManager_MaxSize(t *testing.T) {
	tempDir := t.TempDir()
	hm, err := NewHistoryManager(tempDir, 10)
	require.NoError(t, err)

	// Add more than max size
	for i := 0; i < 15; i++ {
		hm.Add(fmt.Sprintf("command %d", i))
	}

	// Verify size is limited
	history := hm.GetAll()
	assert.Len(t, history, 10)
	// Should keep the last 10 entries (5-14)
	assert.Equal(t, "command 5", history[0])
	assert.Equal(t, "command 14", history[9])
}

func TestHistoryManager_MultilineEntry(t *testing.T) {
	tempDir := t.TempDir()
	hm, err := NewHistoryManager(tempDir, 100)
	require.NoError(t, err)

	// Add multi-line entry
	multiline := "line 1\nline 2\nline 3"
	hm.Add(multiline)

	// Flush to disk
	err = hm.Flush()
	require.NoError(t, err)

	// Create new manager to test persistence
	hm2, err := NewHistoryManager(tempDir, 100)
	require.NoError(t, err)

	history := hm2.GetAll()
	assert.Len(t, history, 1)
	assert.Equal(t, multiline, history[0])
}

func TestHistoryManager_Clear(t *testing.T) {
	tempDir := t.TempDir()
	hm, err := NewHistoryManager(tempDir, 100)
	require.NoError(t, err)

	// Add entries
	hm.Add("entry 1")
	hm.Add("entry 2")
	assert.Equal(t, 2, hm.Size())

	// Clear
	hm.Clear()

	// Verify cleared
	assert.Equal(t, 0, hm.Size())

	// Flush to persist the clear
	err = hm.Flush()
	require.NoError(t, err)

	// Verify file is updated
	hm2, err := NewHistoryManager(tempDir, 100)
	require.NoError(t, err)
	assert.Equal(t, 0, hm2.Size())
}

func TestHistoryManager_EmptyEntry(t *testing.T) {
	tempDir := t.TempDir()
	hm, err := NewHistoryManager(tempDir, 100)
	require.NoError(t, err)

	// Try to add empty entry
	hm.Add("")

	// Verify not added
	assert.Equal(t, 0, hm.Size())

	// Try to add whitespace-only entry
	hm.Add("   ")

	// Verify not added
	assert.Equal(t, 0, hm.Size())
}

func TestHistoryManager_FilePermissions(t *testing.T) {
	tempDir := t.TempDir()
	hm, err := NewHistoryManager(tempDir, 100)
	require.NoError(t, err)

	// Add entry
	hm.Add("test entry")

	// Flush to create file
	err = hm.Flush()
	require.NoError(t, err)

	// Check file exists and has correct permissions
	historyPath := filepath.Join(tempDir, "history")
	info, err := os.Stat(historyPath)
	require.NoError(t, err)
	assert.False(t, info.IsDir())
	// File should be readable/writable by owner
	mode := info.Mode()
	assert.Equal(t, os.FileMode(0644), mode.Perm())
}
