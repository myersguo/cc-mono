package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/myersguo/cc-mono/pkg/agent"
	"github.com/myersguo/cc-mono/pkg/ai"
)

// CreateWriteTool creates the write file tool
func CreateWriteTool(workingDir string) agent.AgentTool {
	tool := ai.NewTool(
		"write",
		"Write content to a file. Creates parent directories if needed. Overwrites existing files.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"file_path": map[string]any{
					"type":        "string",
					"description": "Path to the file to write (relative or absolute)",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "Content to write to the file",
				},
			},
			"required": []string{"file_path", "content"},
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

		content, ok := params["content"].(string)
		if !ok {
			return agent.AgentToolResult{
				Content: []ai.Content{ai.NewTextContent("Error: content must be a string")},
				IsError: true,
			}, fmt.Errorf("content must be a string")
		}

		// Resolve path
		absPath := resolvePath(workingDir, filePath)

		// Send progress update
		if onUpdate != nil {
			onUpdate(agent.AgentToolUpdate{
				Type:    "progress",
				Message: fmt.Sprintf("Writing to %s...", filePath),
			})
		}

		// Create parent directories
		dir := filepath.Dir(absPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return agent.AgentToolResult{
				Content: []ai.Content{ai.NewTextContent(fmt.Sprintf("Error creating directory: %v", err))},
				IsError: true,
			}, nil
		}

		// Write file
		if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
			return agent.AgentToolResult{
				Content: []ai.Content{ai.NewTextContent(fmt.Sprintf("Error writing file: %v", err))},
				IsError: true,
			}, nil
		}

		return agent.AgentToolResult{
			Content: []ai.Content{ai.NewTextContent(fmt.Sprintf("Successfully wrote to %s (%d bytes)", filePath, len(content)))},
			Details: map[string]any{
				"path": filePath,
				"size": len(content),
			},
			IsError: false,
		}, nil
	}

	return agent.NewAgentTool(tool, "Write File", execute)
}
