package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/myersguo/cc-mono/pkg/agent"
	"github.com/myersguo/cc-mono/pkg/ai"
)

// ReadToolParams represents parameters for the read tool
type ReadToolParams struct {
	FilePath string `json:"file_path"`
	Offset   int    `json:"offset,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

// CreateReadTool creates the read file tool
func CreateReadTool(workingDir string) agent.AgentTool {
	tool := ai.NewTool(
		"read",
		"Read a file from the filesystem. Returns file contents. Supports images.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"file_path": map[string]any{
					"type":        "string",
					"description": "Path to the file to read (relative or absolute)",
				},
				"offset": map[string]any{
					"type":        "number",
					"description": "Optional: Line number to start reading from (0-indexed)",
				},
				"limit": map[string]any{
					"type":        "number",
					"description": "Optional: Number of lines to read",
				},
			},
			"required": []string{"file_path"},
		},
	)

	execute := func(
		ctx context.Context,
		toolCallID string,
		params map[string]any,
		onUpdate agent.AgentToolUpdateCallback,
	) (agent.AgentToolResult, error) {
		// Parse parameters
		filePath, ok := params["file_path"].(string)
		if !ok {
			return agent.AgentToolResult{
				Content: []ai.Content{ai.NewTextContent("Error: file_path must be a string")},
				IsError: true,
			}, fmt.Errorf("file_path must be a string")
		}

		offset := 0
		if val, ok := params["offset"].(float64); ok {
			offset = int(val)
		}

		limit := 0 // 0 means no limit
		if val, ok := params["limit"].(float64); ok {
			limit = int(val)
		}

		// Resolve path
		absPath := resolvePath(workingDir, filePath)

		// Send progress update
		if onUpdate != nil {
			onUpdate(agent.AgentToolUpdate{
				Type:    "progress",
				Message: fmt.Sprintf("Reading %s...", filePath),
			})
		}

		// Check if file exists
		info, err := os.Stat(absPath)
		if err != nil {
			if os.IsNotExist(err) {
				return agent.AgentToolResult{
					Content: []ai.Content{ai.NewTextContent(fmt.Sprintf("Error: File not found: %s", filePath))},
					IsError: true,
				}, nil
			}
			return agent.AgentToolResult{
				Content: []ai.Content{ai.NewTextContent(fmt.Sprintf("Error: %v", err))},
				IsError: true,
			}, nil
		}

		// Check if it's a directory
		if info.IsDir() {
			return agent.AgentToolResult{
				Content: []ai.Content{ai.NewTextContent(fmt.Sprintf("Error: %s is a directory", filePath))},
				IsError: true,
			}, nil
		}

		// Check if it's an image
		if isImageFile(absPath) {
			return readImageFile(absPath, filePath)
		}

		// Read text file
		return readTextFile(absPath, filePath, offset, limit)
	}

	return agent.NewAgentTool(tool, "Read File", execute)
}

// resolvePath resolves a relative or absolute path
func resolvePath(workingDir, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(workingDir, path)
}

// isImageFile checks if a file is an image based on extension
func isImageFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	imageExts := []string{".png", ".jpg", ".jpeg", ".gif", ".bmp", ".webp", ".svg"}

	for _, imgExt := range imageExts {
		if ext == imgExt {
			return true
		}
	}
	return false
}

// readImageFile reads an image file and returns it as base64
func readImageFile(absPath, displayPath string) (agent.AgentToolResult, error) {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return agent.AgentToolResult{
			Content: []ai.Content{ai.NewTextContent(fmt.Sprintf("Error reading image: %v", err))},
			IsError: true,
		}, nil
	}

	// Determine media type
	ext := strings.ToLower(filepath.Ext(absPath))
	mediaType := "image/png"
	switch ext {
	case ".jpg", ".jpeg":
		mediaType = "image/jpeg"
	case ".gif":
		mediaType = "image/gif"
	case ".webp":
		mediaType = "image/webp"
	case ".svg":
		mediaType = "image/svg+xml"
	}

	// Create image content
	content := []ai.Content{
		ai.NewTextContent(fmt.Sprintf("Image: %s (%d bytes)", displayPath, len(data))),
		ai.NewImageContentFromBase64(string(data), mediaType),
	}

	return agent.AgentToolResult{
		Content: content,
		Details: map[string]any{
			"path":      displayPath,
			"size":      len(data),
			"mediaType": mediaType,
		},
		IsError: false,
	}, nil
}

// readTextFile reads a text file with optional offset and limit
func readTextFile(absPath, displayPath string, offset, limit int) (agent.AgentToolResult, error) {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return agent.AgentToolResult{
			Content: []ai.Content{ai.NewTextContent(fmt.Sprintf("Error: %v", err))},
			IsError: true,
		}, nil
	}

	content := string(data)
	lines := strings.Split(content, "\n")

	// Apply offset and limit
	if offset > 0 || limit > 0 {
		start := offset
		if start >= len(lines) {
			return agent.AgentToolResult{
				Content: []ai.Content{ai.NewTextContent(fmt.Sprintf("Error: Offset %d exceeds file length %d", offset, len(lines)))},
				IsError: true,
			}, nil
		}

		end := len(lines)
		if limit > 0 && start+limit < end {
			end = start + limit
		}

		lines = lines[start:end]
		content = strings.Join(lines, "\n")
	}

	// Check file size
	maxSize := 1024 * 1024 // 1MB
	if len(content) > maxSize {
		content = content[:maxSize] + "\n\n... (content truncated, file too large)"
	}

	return agent.AgentToolResult{
		Content: []ai.Content{ai.NewTextContent(content)},
		Details: map[string]any{
			"path":      displayPath,
			"size":      len(data),
			"lines":     len(lines),
			"truncated": len(content) > maxSize,
		},
		IsError: false,
	}, nil
}
