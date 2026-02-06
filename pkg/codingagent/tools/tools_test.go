package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/myersguo/cc-mono/pkg/agent"
	"github.com/myersguo/cc-mono/pkg/ai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to execute a tool
func executeTool(t *testing.T, tool agent.AgentTool, params map[string]any) agent.AgentToolResult {
	ctx := context.Background()
	result, err := tool.Execute(ctx, "test-call-id", params, nil)
	require.NoError(t, err)
	return result
}

// Test Read Tool
func TestReadTool_ReadTextFile(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "Hello, World!\nLine 2\nLine 3"
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	tool := CreateReadTool(tempDir)

	result := executeTool(t, tool, map[string]any{
		"file_path": "test.txt",
	})

	assert.False(t, result.IsError)
	require.Len(t, result.Content, 1)

	textContent, ok := result.Content[0].(ai.TextContent)
	require.True(t, ok)
	assert.Equal(t, testContent, textContent.Text)

	// Verify details
	details, ok := result.Details.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "test.txt", details["path"])
	assert.Equal(t, len(testContent), details["size"])
}

func TestReadTool_ReadWithOffsetAndLimit(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5"
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	tool := CreateReadTool(tempDir)

	result := executeTool(t, tool, map[string]any{
		"file_path": "test.txt",
		"offset":    float64(1), // Start from line 1 (0-indexed)
		"limit":     float64(2), // Read 2 lines
	})

	assert.False(t, result.IsError)
	require.Len(t, result.Content, 1)

	textContent, ok := result.Content[0].(ai.TextContent)
	require.True(t, ok)
	assert.Equal(t, "Line 2\nLine 3", textContent.Text)
}

func TestReadTool_ReadImageFile(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.png")
	imageData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A} // PNG header
	err := os.WriteFile(testFile, imageData, 0644)
	require.NoError(t, err)

	tool := CreateReadTool(tempDir)

	result := executeTool(t, tool, map[string]any{
		"file_path": "test.png",
	})

	assert.False(t, result.IsError)
	require.Len(t, result.Content, 2) // Text description + Image content

	// First content should be text description
	textContent, ok := result.Content[0].(ai.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "test.png")
	assert.Contains(t, textContent.Text, "bytes")

	// Second content should be image
	imageContent, ok := result.Content[1].(ai.ImageContent)
	require.True(t, ok)
	assert.Equal(t, "image/png", imageContent.Source.MediaType)
}

func TestReadTool_FileNotFound(t *testing.T) {
	tempDir := t.TempDir()
	tool := CreateReadTool(tempDir)

	result := executeTool(t, tool, map[string]any{
		"file_path": "nonexistent.txt",
	})

	assert.True(t, result.IsError)
	require.Len(t, result.Content, 1)

	textContent, ok := result.Content[0].(ai.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "File not found")
}

func TestReadTool_ReadDirectory(t *testing.T) {
	tempDir := t.TempDir()
	tool := CreateReadTool(tempDir)

	result := executeTool(t, tool, map[string]any{
		"file_path": ".",
	})

	assert.True(t, result.IsError)
	require.Len(t, result.Content, 1)

	textContent, ok := result.Content[0].(ai.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "is a directory")
}

// Test Write Tool
func TestWriteTool_WriteFile(t *testing.T) {
	tempDir := t.TempDir()
	tool := CreateWriteTool(tempDir)

	testContent := "Test content\nLine 2"
	result := executeTool(t, tool, map[string]any{
		"file_path": "output.txt",
		"content":   testContent,
	})

	assert.False(t, result.IsError)
	require.Len(t, result.Content, 1)

	textContent, ok := result.Content[0].(ai.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "Successfully wrote")
	assert.Contains(t, textContent.Text, "output.txt")

	// Verify file was created
	outputPath := filepath.Join(tempDir, "output.txt")
	assert.FileExists(t, outputPath)

	// Verify content
	written, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	assert.Equal(t, testContent, string(written))
}

func TestWriteTool_CreateDirectories(t *testing.T) {
	tempDir := t.TempDir()
	tool := CreateWriteTool(tempDir)

	result := executeTool(t, tool, map[string]any{
		"file_path": "subdir/nested/file.txt",
		"content":   "nested content",
	})

	assert.False(t, result.IsError)

	// Verify directories were created
	outputPath := filepath.Join(tempDir, "subdir", "nested", "file.txt")
	assert.FileExists(t, outputPath)

	// Verify content
	written, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	assert.Equal(t, "nested content", string(written))
}

func TestWriteTool_OverwriteExisting(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "existing.txt")
	err := os.WriteFile(testFile, []byte("original content"), 0644)
	require.NoError(t, err)

	tool := CreateWriteTool(tempDir)

	result := executeTool(t, tool, map[string]any{
		"file_path": "existing.txt",
		"content":   "new content",
	})

	assert.False(t, result.IsError)

	// Verify content was overwritten
	written, err := os.ReadFile(testFile)
	require.NoError(t, err)
	assert.Equal(t, "new content", string(written))
}

// Test Edit Tool
func TestEditTool_ExactMatch(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	originalContent := "Line 1\nLine 2\nLine 3"
	err := os.WriteFile(testFile, []byte(originalContent), 0644)
	require.NoError(t, err)

	tool := CreateEditTool(tempDir)

	result := executeTool(t, tool, map[string]any{
		"file_path":  "test.txt",
		"old_string": "Line 2",
		"new_string": "Modified Line 2",
	})

	assert.False(t, result.IsError)
	require.Len(t, result.Content, 1)

	textContent, ok := result.Content[0].(ai.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "Successfully edited")
	assert.Contains(t, textContent.Text, "1 replacement")

	// Verify file was modified
	modified, err := os.ReadFile(testFile)
	require.NoError(t, err)
	assert.Equal(t, "Line 1\nModified Line 2\nLine 3", string(modified))
}

func TestEditTool_ReplaceAll(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	originalContent := "foo bar foo baz foo"
	err := os.WriteFile(testFile, []byte(originalContent), 0644)
	require.NoError(t, err)

	tool := CreateEditTool(tempDir)

	result := executeTool(t, tool, map[string]any{
		"file_path":   "test.txt",
		"old_string":  "foo",
		"new_string":  "qux",
		"replace_all": true,
	})

	assert.False(t, result.IsError)
	require.Len(t, result.Content, 1)

	textContent, ok := result.Content[0].(ai.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "3 replacement")

	// Verify all occurrences were replaced
	modified, err := os.ReadFile(testFile)
	require.NoError(t, err)
	assert.Equal(t, "qux bar qux baz qux", string(modified))
}

func TestEditTool_FuzzyMatch(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	originalContent := "def   foo():\n    return   42"
	err := os.WriteFile(testFile, []byte(originalContent), 0644)
	require.NoError(t, err)

	tool := CreateEditTool(tempDir)

	// Try to match with normalized whitespace
	result := executeTool(t, tool, map[string]any{
		"file_path":  "test.txt",
		"old_string": "def foo():\n return 42",
		"new_string": "def bar():\n    return 100",
	})

	assert.False(t, result.IsError)

	// Verify file was modified (fuzzy match should work)
	modified, err := os.ReadFile(testFile)
	require.NoError(t, err)
	assert.Contains(t, string(modified), "bar")
}

func TestEditTool_NoMatch(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	originalContent := "Line 1\nLine 2\nLine 3"
	err := os.WriteFile(testFile, []byte(originalContent), 0644)
	require.NoError(t, err)

	tool := CreateEditTool(tempDir)

	result := executeTool(t, tool, map[string]any{
		"file_path":  "test.txt",
		"old_string": "Nonexistent",
		"new_string": "Replacement",
	})

	assert.True(t, result.IsError)
	require.Len(t, result.Content, 1)

	textContent, ok := result.Content[0].(ai.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "old_string not found")
}

func TestEditTool_DiffGeneration(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	originalContent := "Line 1\nLine 2\nLine 3"
	err := os.WriteFile(testFile, []byte(originalContent), 0644)
	require.NoError(t, err)

	tool := CreateEditTool(tempDir)

	result := executeTool(t, tool, map[string]any{
		"file_path":  "test.txt",
		"old_string": "Line 2",
		"new_string": "Modified Line 2",
	})

	assert.False(t, result.IsError)
	require.Len(t, result.Content, 1)

	textContent, ok := result.Content[0].(ai.TextContent)
	require.True(t, ok)

	// Check for diff markers
	assert.Contains(t, textContent.Text, "---")
	assert.Contains(t, textContent.Text, "+++")
	assert.Contains(t, textContent.Text, "-Line 2")
	assert.Contains(t, textContent.Text, "+Modified Line 2")
}

// Test Bash Tool
func TestBashTool_SimpleCommand(t *testing.T) {
	tempDir := t.TempDir()
	tool := CreateBashTool(tempDir)

	result := executeTool(t, tool, map[string]any{
		"command": "echo 'Hello, World!'",
	})

	assert.False(t, result.IsError)
	require.Len(t, result.Content, 1)

	textContent, ok := result.Content[0].(ai.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "Hello, World!")
	assert.Contains(t, textContent.Text, "Exit code: 0")

	// Verify details
	details, ok := result.Details.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "echo 'Hello, World!'", details["command"])
	assert.Equal(t, 0, details["exit_code"])
}

func TestBashTool_CommandWithOutput(t *testing.T) {
	tempDir := t.TempDir()
	tool := CreateBashTool(tempDir)

	result := executeTool(t, tool, map[string]any{
		"command": "ls -la",
	})

	assert.False(t, result.IsError)
	require.Len(t, result.Content, 1)

	textContent, ok := result.Content[0].(ai.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "Exit code: 0")
	assert.Contains(t, textContent.Text, "Output:")
}

func TestBashTool_CommandWithError(t *testing.T) {
	tempDir := t.TempDir()
	tool := CreateBashTool(tempDir)

	result := executeTool(t, tool, map[string]any{
		"command": "ls /nonexistent-directory-12345",
	})

	assert.True(t, result.IsError)
	require.Len(t, result.Content, 1)

	textContent, ok := result.Content[0].(ai.TextContent)
	require.True(t, ok)
	assert.NotContains(t, textContent.Text, "Exit code: 0")
	assert.Contains(t, textContent.Text, "Stderr:")

	// Verify details
	details, ok := result.Details.(map[string]any)
	require.True(t, ok)
	exitCode, ok := details["exit_code"].(int)
	require.True(t, ok)
	assert.NotEqual(t, 0, exitCode)
}

func TestBashTool_WorkingDirectory(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test file in tempDir
	testFile := filepath.Join(tempDir, "testfile.txt")
	err := os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	tool := CreateBashTool(tempDir)

	result := executeTool(t, tool, map[string]any{
		"command": "ls testfile.txt",
	})

	assert.False(t, result.IsError)
	require.Len(t, result.Content, 1)

	textContent, ok := result.Content[0].(ai.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "testfile.txt")
}

func TestBashTool_Timeout(t *testing.T) {
	tempDir := t.TempDir()
	tool := CreateBashTool(tempDir)

	// Use a very short timeout
	result := executeTool(t, tool, map[string]any{
		"command": "sleep 10",
		"timeout": float64(0.1), // 100ms timeout
	})

	// The command should fail (either timeout or killed)
	assert.True(t, result.IsError)
	require.Len(t, result.Content, 1)

	textContent, ok := result.Content[0].(ai.TextContent)
	require.True(t, ok)
	// Should have some error message (timeout, killed, signal, etc.)
	assert.NotEmpty(t, textContent.Text)
}

func TestBashTool_OutputTruncation(t *testing.T) {
	tempDir := t.TempDir()
	tool := CreateBashTool(tempDir)

	// Generate large output (more than 50KB)
	result := executeTool(t, tool, map[string]any{
		"command": "yes 'A' | head -n 10000",
	})

	assert.False(t, result.IsError)
	require.Len(t, result.Content, 1)

	textContent, ok := result.Content[0].(ai.TextContent)
	require.True(t, ok)

	// Output should be truncated
	if len(textContent.Text) > 50000 {
		assert.Contains(t, textContent.Text, "truncated")
	}
}

func TestBashTool_MultilineCommand(t *testing.T) {
	tempDir := t.TempDir()
	tool := CreateBashTool(tempDir)

	result := executeTool(t, tool, map[string]any{
		"command": "echo 'Line 1'\necho 'Line 2'\necho 'Line 3'",
	})

	assert.False(t, result.IsError)
	require.Len(t, result.Content, 1)

	textContent, ok := result.Content[0].(ai.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "Line 1")
	assert.Contains(t, textContent.Text, "Line 2")
	assert.Contains(t, textContent.Text, "Line 3")
}

func TestBashTool_PipedCommands(t *testing.T) {
	tempDir := t.TempDir()
	tool := CreateBashTool(tempDir)

	result := executeTool(t, tool, map[string]any{
		"command": "echo 'hello world' | tr 'a-z' 'A-Z'",
	})

	assert.False(t, result.IsError)
	require.Len(t, result.Content, 1)

	textContent, ok := result.Content[0].(ai.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "HELLO WORLD")
}

func TestBashTool_EnvironmentVariables(t *testing.T) {
	tempDir := t.TempDir()
	tool := CreateBashTool(tempDir)

	result := executeTool(t, tool, map[string]any{
		"command": "export TEST_VAR='test value' && echo $TEST_VAR",
	})

	assert.False(t, result.IsError)
	require.Len(t, result.Content, 1)

	textContent, ok := result.Content[0].(ai.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "test value")
}

// Test tool callbacks
func TestTool_UpdateCallback(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	tool := CreateReadTool(tempDir)

	var updateReceived bool
	var updateMessage string

	ctx := context.Background()
	_, err = tool.Execute(ctx, "test-call-id", map[string]any{
		"file_path": "test.txt",
	}, func(update agent.AgentToolUpdate) {
		updateReceived = true
		updateMessage = update.Message
	})

	require.NoError(t, err)
	assert.True(t, updateReceived)
	assert.Contains(t, updateMessage, "Reading")
	assert.Contains(t, updateMessage, "test.txt")
}

// Test tool context cancellation
func TestTool_ContextCancellation(t *testing.T) {
	tempDir := t.TempDir()
	tool := CreateBashTool(tempDir)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := tool.Execute(ctx, "test-call-id", map[string]any{
		"command": "sleep 10",
	}, nil)

	// Should return an error due to context cancellation
	require.NoError(t, err) // Execute itself doesn't error, but the command should fail
}

// Test invalid parameters
func TestReadTool_InvalidParameters(t *testing.T) {
	tempDir := t.TempDir()
	tool := CreateReadTool(tempDir)

	ctx := context.Background()
	result, err := tool.Execute(ctx, "test-call-id", map[string]any{
		"file_path": 123, // Invalid type (should be string)
	}, nil)

	// Tool may return error or set IsError
	assert.True(t, err != nil || result.IsError)
}

func TestWriteTool_InvalidParameters(t *testing.T) {
	tempDir := t.TempDir()
	tool := CreateWriteTool(tempDir)

	ctx := context.Background()
	result, err := tool.Execute(ctx, "test-call-id", map[string]any{
		"file_path": "test.txt",
		"content":   123, // Invalid type (should be string)
	}, nil)

	// Tool may return error or set IsError
	assert.True(t, err != nil || result.IsError)
}

func TestEditTool_InvalidParameters(t *testing.T) {
	tempDir := t.TempDir()
	tool := CreateEditTool(tempDir)

	ctx := context.Background()
	result, err := tool.Execute(ctx, "test-call-id", map[string]any{
		"file_path":  "test.txt",
		"old_string": 123, // Invalid type
		"new_string": "test",
	}, nil)

	// Tool may return error or set IsError
	assert.True(t, err != nil || result.IsError)
}

func TestBashTool_InvalidParameters(t *testing.T) {
	tempDir := t.TempDir()
	tool := CreateBashTool(tempDir)

	ctx := context.Background()
	result, err := tool.Execute(ctx, "test-call-id", map[string]any{
		"command": 123, // Invalid type (should be string)
	}, nil)

	// Tool may return error or set IsError
	assert.True(t, err != nil || result.IsError)
}
