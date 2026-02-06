package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/myersguo/cc-mono/pkg/agent"
	"github.com/myersguo/cc-mono/pkg/ai"
)

// CreateBashTool creates the bash command execution tool
func CreateBashTool(workingDir string) agent.AgentTool {
	tool := ai.NewTool(
		"bash",
		"Execute a bash command and return the output. Use with caution.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]any{
					"type":        "string",
					"description": "The bash command to execute",
				},
				"timeout": map[string]any{
					"type":        "number",
					"description": "Optional: Timeout in seconds (default: 30)",
				},
			},
			"required": []string{"command"},
		},
	)

	execute := func(
		ctx context.Context,
		toolCallID string,
		params map[string]any,
		onUpdate agent.AgentToolUpdateCallback,
	) (agent.AgentToolResult, error) {
		// Parse parameters
		command, ok := params["command"].(string)
		if !ok {
			return agent.AgentToolResult{
				Content: []ai.Content{ai.NewTextContent("Error: command must be a string")},
				IsError: true,
			}, fmt.Errorf("command must be a string")
		}

		timeout := 30 * time.Second
		if val, ok := params["timeout"].(float64); ok {
			timeout = time.Duration(val) * time.Second
		}

		// Send progress update
		if onUpdate != nil {
			onUpdate(agent.AgentToolUpdate{
				Type:    "progress",
				Message: fmt.Sprintf("Executing: %s", command),
			})
		}

		// Create context with timeout
		cmdCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		// Execute command
		cmd := exec.CommandContext(cmdCtx, "bash", "-c", command)
		cmd.Dir = workingDir

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		startTime := time.Now()
		err := cmd.Run()
		duration := time.Since(startTime)

		// Prepare output
		output := stdout.String()
		errOutput := stderr.String()

		// Truncate output if too large
		maxOutputSize := 50000 // 50KB
		if len(output) > maxOutputSize {
			output = output[:maxOutputSize] + "\n\n... (output truncated)"
		}
		if len(errOutput) > maxOutputSize {
			errOutput = errOutput[:maxOutputSize] + "\n\n... (stderr truncated)"
		}

		// Build result message
		var resultMsg strings.Builder
		resultMsg.WriteString(fmt.Sprintf("Command: %s\n", command))
		resultMsg.WriteString(fmt.Sprintf("Duration: %.2fs\n", duration.Seconds()))

		if err != nil {
			resultMsg.WriteString(fmt.Sprintf("Exit code: %d\n", cmd.ProcessState.ExitCode()))
			resultMsg.WriteString("\nStderr:\n")
			resultMsg.WriteString(errOutput)

			if len(output) > 0 {
				resultMsg.WriteString("\nStdout:\n")
				resultMsg.WriteString(output)
			}

			return agent.AgentToolResult{
				Content: []ai.Content{ai.NewTextContent(resultMsg.String())},
				Details: map[string]any{
					"command":   command,
					"exit_code": cmd.ProcessState.ExitCode(),
					"duration":  duration.Seconds(),
					"stdout":    output,
					"stderr":    errOutput,
				},
				IsError: true,
			}, nil
		}

		resultMsg.WriteString("Exit code: 0\n")
		if len(output) > 0 {
			resultMsg.WriteString("\nOutput:\n")
			resultMsg.WriteString(output)
		}
		if len(errOutput) > 0 {
			resultMsg.WriteString("\nStderr:\n")
			resultMsg.WriteString(errOutput)
		}

		return agent.AgentToolResult{
			Content: []ai.Content{ai.NewTextContent(resultMsg.String())},
			Details: map[string]any{
				"command":   command,
				"exit_code": 0,
				"duration":  duration.Seconds(),
				"stdout":    output,
				"stderr":    errOutput,
			},
			IsError: false,
		}, nil
	}

	return agent.NewAgentTool(tool, "Bash Command", execute)
}
