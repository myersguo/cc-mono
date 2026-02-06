package tools

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/myersguo/cc-mono/pkg/agent"
	"github.com/myersguo/cc-mono/pkg/ai"
)

// CreateEditTool creates the edit file tool
func CreateEditTool(workingDir string) agent.AgentTool {
	tool := ai.NewTool(
		"edit",
		"Edit a file by replacing old text with new text. Supports fuzzy matching.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"file_path": map[string]any{
					"type":        "string",
					"description": "Path to the file to edit",
				},
				"old_string": map[string]any{
					"type":        "string",
					"description": "The text to replace (must match exactly or with fuzzy matching)",
				},
				"new_string": map[string]any{
					"type":        "string",
					"description": "The new text to insert",
				},
				"replace_all": map[string]any{
					"type":        "boolean",
					"description": "If true, replace all occurrences. Default: false (replace first only)",
				},
			},
			"required": []string{"file_path", "old_string", "new_string"},
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

		oldString, ok := params["old_string"].(string)
		if !ok {
			return agent.AgentToolResult{
				Content: []ai.Content{ai.NewTextContent("Error: old_string must be a string")},
				IsError: true,
			}, fmt.Errorf("old_string must be a string")
		}

		newString, ok := params["new_string"].(string)
		if !ok {
			return agent.AgentToolResult{
				Content: []ai.Content{ai.NewTextContent("Error: new_string must be a string")},
				IsError: true,
			}, fmt.Errorf("new_string must be a string")
		}

		replaceAll := false
		if val, ok := params["replace_all"].(bool); ok {
			replaceAll = val
		}

		// Resolve path
		absPath := resolvePath(workingDir, filePath)

		// Send progress update
		if onUpdate != nil {
			onUpdate(agent.AgentToolUpdate{
				Type:    "progress",
				Message: fmt.Sprintf("Editing %s...", filePath),
			})
		}

		// Read file
		data, err := os.ReadFile(absPath)
		if err != nil {
			return agent.AgentToolResult{
				Content: []ai.Content{ai.NewTextContent(fmt.Sprintf("Error reading file: %v", err))},
				IsError: true,
			}, nil
		}

		content := string(data)

		// Perform replacement
		newContent, numReplacements, err := performEdit(content, oldString, newString, replaceAll)
		if err != nil {
			return agent.AgentToolResult{
				Content: []ai.Content{ai.NewTextContent(fmt.Sprintf("Error: %v", err))},
				IsError: true,
			}, nil
		}

		// Write back
		if err := os.WriteFile(absPath, []byte(newContent), 0644); err != nil {
			return agent.AgentToolResult{
				Content: []ai.Content{ai.NewTextContent(fmt.Sprintf("Error writing file: %v", err))},
				IsError: true,
			}, nil
		}

		// Generate diff
		diff := generateDiff(content, newContent, filePath)

		return agent.AgentToolResult{
			Content: []ai.Content{ai.NewTextContent(fmt.Sprintf("Successfully edited %s (%d replacement(s))\n\n%s", filePath, numReplacements, diff))},
			Details: map[string]any{
				"path":          filePath,
				"replacements":  numReplacements,
				"old_size":      len(content),
				"new_size":      len(newContent),
			},
			IsError: false,
		}, nil
	}

	return agent.NewAgentTool(tool, "Edit File", execute)
}

// performEdit performs the text replacement with fuzzy matching
func performEdit(content, oldString, newString string, replaceAll bool) (string, int, error) {
	// First try exact match
	if strings.Contains(content, oldString) {
		if replaceAll {
			newContent := strings.ReplaceAll(content, oldString, newString)
			count := strings.Count(content, oldString)
			return newContent, count, nil
		}
		newContent := strings.Replace(content, oldString, newString, 1)
		return newContent, 1, nil
	}

	// Try fuzzy match (normalize whitespace)
	normalizedOld := normalizeWhitespace(oldString)
	normalizedContent := normalizeWhitespace(content)

	if strings.Contains(normalizedContent, normalizedOld) {
		// Find the actual match in original content
		idx := findFuzzyMatch(content, oldString)
		if idx >= 0 {
			// Calculate the length of the matched text
			matchLen := len(oldString)
			newContent := content[:idx] + newString + content[idx+matchLen:]
			return newContent, 1, nil
		}
	}

	// No match found
	return "", 0, fmt.Errorf("old_string not found in file (tried exact and fuzzy matching)")
}

// normalizeWhitespace normalizes whitespace for fuzzy matching
func normalizeWhitespace(s string) string {
	// Replace multiple spaces with single space
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.Join(strings.Fields(line), " ")
	}
	return strings.Join(lines, "\n")
}

// findFuzzyMatch finds the start index of a fuzzy match
func findFuzzyMatch(content, pattern string) int {
	normalizedContent := normalizeWhitespace(content)
	normalizedPattern := normalizeWhitespace(pattern)

	idx := strings.Index(normalizedContent, normalizedPattern)
	if idx < 0 {
		return -1
	}

	// Map back to original content index (simplified)
	// This is approximate and works for most cases
	return idx
}

// generateDiff generates a simple unified diff
func generateDiff(oldContent, newContent, filename string) string {
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	var diff strings.Builder
	diff.WriteString(fmt.Sprintf("--- %s (old)\n", filename))
	diff.WriteString(fmt.Sprintf("+++ %s (new)\n", filename))

	// Find differences
	maxLen := len(oldLines)
	if len(newLines) > maxLen {
		maxLen = len(newLines)
	}

	contextLines := 3
	inDiff := false
	diffStart := -1

	for i := 0; i < maxLen; i++ {
		oldLine := ""
		newLine := ""

		if i < len(oldLines) {
			oldLine = oldLines[i]
		}
		if i < len(newLines) {
			newLine = newLines[i]
		}

		if oldLine != newLine {
			if !inDiff {
				// Start of diff hunk
				inDiff = true
				diffStart = i - contextLines
				if diffStart < 0 {
					diffStart = 0
				}

				diff.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", diffStart+1, len(oldLines), diffStart+1, len(newLines)))

				// Write context before diff
				for j := diffStart; j < i; j++ {
					if j < len(oldLines) {
						diff.WriteString(" " + oldLines[j] + "\n")
					}
				}
			}

			// Write diff lines
			if i < len(oldLines) && oldLine != "" {
				diff.WriteString("-" + oldLine + "\n")
			}
			if i < len(newLines) && newLine != "" {
				diff.WriteString("+" + newLine + "\n")
			}
		} else if inDiff {
			// Context after diff
			diff.WriteString(" " + oldLine + "\n")
		}
	}

	if diff.Len() == 0 {
		return "No changes"
	}

	return diff.String()
}
