package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// PermissionRequest represents a request for permission
type PermissionRequest struct {
	ToolName    string         `json:"tool_name"`
	Action      string         `json:"action"`       // e.g., "read", "write", "execute"
	Resource    string         `json:"resource"`     // e.g., file path, command
	Params      map[string]any `json:"params"`       // Tool parameters
	RiskLevel   string         `json:"risk_level"`   // "safe", "medium", "dangerous"
	Description string         `json:"description"`  // Human-readable description
	RequestID   string         `json:"request_id"`   // Unique request ID
	Timestamp   int64          `json:"timestamp"`
}

// PermissionResponse represents a user's response to a permission request
type PermissionResponse struct {
	RequestID string `json:"request_id"`
	Allowed   bool   `json:"allowed"`
	Remember  bool   `json:"remember"` // Whether to remember this choice
	Scope     string `json:"scope"`    // "project" or "global"
	Timestamp int64  `json:"timestamp"`
}

// PermissionSettings represents the permissions section in settings
type PermissionSettings struct {
	Allow []string `json:"allow,omitempty"`
	Deny  []string `json:"deny,omitempty"`
}

// Settings represents the complete settings structure
type Settings struct {
	Permissions *PermissionSettings `json:"permissions,omitempty"`
	// Other settings fields...
}

// PermissionManager manages tool execution permissions
type PermissionManager struct {
	mu              sync.RWMutex
	allowPatterns   []string // Patterns like "Bash(go build:*)"
	denyPatterns    []string
	pendingRequests map[string]*PermissionRequest
	responseChan    map[string]chan PermissionResponse
	globalPath      string // Global settings path
	projectPath     string // Project-local settings path
}

// NewPermissionManager creates a new permission manager
func NewPermissionManager(globalConfigDir, projectDir string) (*PermissionManager, error) {
	globalPath := filepath.Join(globalConfigDir, "settings.json")
	projectPath := filepath.Join(projectDir, ".cc-mono", "settings.local.json")

	pm := &PermissionManager{
		allowPatterns:   make([]string, 0),
		denyPatterns:    make([]string, 0),
		pendingRequests: make(map[string]*PermissionRequest),
		responseChan:    make(map[string]chan PermissionResponse),
		globalPath:      globalPath,
		projectPath:     projectPath,
	}

	// Load existing permissions
	if err := pm.loadPermissions(); err != nil {
		// If files don't exist, that's OK
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load permissions: %w", err)
		}
	}

	return pm, nil
}

// CheckPermission checks if an operation is allowed
func (pm *PermissionManager) CheckPermission(req *PermissionRequest) (bool, bool, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Generate pattern for this request
	pattern := pm.generatePattern(req)

	// Check deny patterns first
	for _, denyPattern := range pm.denyPatterns {
		if pm.matchPattern(pattern, denyPattern) {
			return false, false, nil // Explicitly denied
		}
	}

	// Check allow patterns
	for _, allowPattern := range pm.allowPatterns {
		if pm.matchPattern(pattern, allowPattern) {
			return true, false, nil // Allowed, no need to ask
		}
	}

	// No matching rule, need to ask user
	return false, true, nil
}

// RequestPermission requests permission from the user
func (pm *PermissionManager) RequestPermission(req *PermissionRequest) (*PermissionResponse, error) {
	pm.mu.Lock()

	// Generate request ID if not set
	if req.RequestID == "" {
		req.RequestID = pm.generateRequestID(req)
	}
	req.Timestamp = time.Now().UnixMilli()

	// Store pending request
	pm.pendingRequests[req.RequestID] = req

	// Create response channel
	respChan := make(chan PermissionResponse, 1)
	pm.responseChan[req.RequestID] = respChan

	pm.mu.Unlock()

	// Wait for response (with timeout)
	select {
	case resp := <-respChan:
		// Clean up
		pm.mu.Lock()
		delete(pm.pendingRequests, req.RequestID)
		delete(pm.responseChan, req.RequestID)
		pm.mu.Unlock()

		// Save rule if user chose to remember
		if resp.Remember {
			if err := pm.savePermission(req, resp.Allowed, resp.Scope); err != nil {
				return &resp, fmt.Errorf("failed to save permission: %w", err)
			}
		}

		return &resp, nil

	case <-time.After(5 * time.Minute):
		// Timeout - deny by default
		pm.mu.Lock()
		delete(pm.pendingRequests, req.RequestID)
		delete(pm.responseChan, req.RequestID)
		pm.mu.Unlock()

		return &PermissionResponse{
			RequestID: req.RequestID,
			Allowed:   false,
			Remember:  false,
			Timestamp: time.Now().UnixMilli(),
		}, fmt.Errorf("permission request timed out")
	}
}

// RequestPermissionWithContext requests permission with context support
func (pm *PermissionManager) RequestPermissionWithContext(ctx context.Context, req *PermissionRequest) (*PermissionResponse, error) {
	pm.mu.Lock()

	// Generate request ID if not set
	if req.RequestID == "" {
		req.RequestID = pm.generateRequestID(req)
	}
	req.Timestamp = time.Now().UnixMilli()

	// Store pending request
	pm.pendingRequests[req.RequestID] = req

	// Create response channel
	respChan := make(chan PermissionResponse, 1)
	pm.responseChan[req.RequestID] = respChan

	pm.mu.Unlock()

	// Wait for response with context
	select {
	case resp := <-respChan:
		// Clean up
		pm.mu.Lock()
		delete(pm.pendingRequests, req.RequestID)
		delete(pm.responseChan, req.RequestID)
		pm.mu.Unlock()

		// Save rule if user chose to remember
		if resp.Remember {
			if err := pm.savePermission(req, resp.Allowed, resp.Scope); err != nil {
				return &resp, fmt.Errorf("failed to save permission: %w", err)
			}
		}

		return &resp, nil

	case <-ctx.Done():
		// Context cancelled
		pm.mu.Lock()
		delete(pm.pendingRequests, req.RequestID)
		delete(pm.responseChan, req.RequestID)
		pm.mu.Unlock()

		return &PermissionResponse{
			RequestID: req.RequestID,
			Allowed:   false,
			Remember:  false,
			Timestamp: time.Now().UnixMilli(),
		}, ctx.Err()
	}
}

// RespondToRequest sends a response to a pending permission request
func (pm *PermissionManager) RespondToRequest(requestID string, allowed bool, remember bool, scope string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	respChan, exists := pm.responseChan[requestID]
	if !exists {
		return fmt.Errorf("no pending request with ID: %s", requestID)
	}

	resp := PermissionResponse{
		RequestID: requestID,
		Allowed:   allowed,
		Remember:  remember,
		Scope:     scope,
		Timestamp: time.Now().UnixMilli(),
	}

	respChan <- resp
	return nil
}

// GetPendingRequest retrieves a pending permission request
func (pm *PermissionManager) GetPendingRequest(requestID string) (*PermissionRequest, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	req, exists := pm.pendingRequests[requestID]
	return req, exists
}

// savePermission saves a permission pattern to settings
func (pm *PermissionManager) savePermission(req *PermissionRequest, allowed bool, scope string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pattern := pm.generatePattern(req)

	// Choose which file to save to
	settingsPath := pm.projectPath
	if scope == "global" {
		settingsPath = pm.globalPath
	}

	// Load existing settings
	settings := &Settings{}
	data, err := os.ReadFile(settingsPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, settings); err != nil {
			return err
		}
	}

	// Initialize permissions if needed
	if settings.Permissions == nil {
		settings.Permissions = &PermissionSettings{}
	}

	// Add pattern to allow or deny list
	if allowed {
		// Check if already exists
		found := false
		for _, p := range settings.Permissions.Allow {
			if p == pattern {
				found = true
				break
			}
		}
		if !found {
			settings.Permissions.Allow = append(settings.Permissions.Allow, pattern)
			pm.allowPatterns = append(pm.allowPatterns, pattern)
		}
	} else {
		found := false
		for _, p := range settings.Permissions.Deny {
			if p == pattern {
				found = true
				break
			}
		}
		if !found {
			settings.Permissions.Deny = append(settings.Permissions.Deny, pattern)
			pm.denyPatterns = append(pm.denyPatterns, pattern)
		}
	}

	return pm.persistSettings(settingsPath, settings)
}

// loadPermissions loads permissions from both global and project settings
func (pm *PermissionManager) loadPermissions() error {
	// Load global settings
	if err := pm.loadSettingsFile(pm.globalPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	// Load project settings (overwrites/extends global)
	if err := pm.loadSettingsFile(pm.projectPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

// loadSettingsFile loads permissions from a single settings file
func (pm *PermissionManager) loadSettingsFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	settings := &Settings{}
	if err := json.Unmarshal(data, settings); err != nil {
		return err
	}

	if settings.Permissions != nil {
		// Add allow patterns (avoid duplicates)
		for _, pattern := range settings.Permissions.Allow {
			found := false
			for _, existing := range pm.allowPatterns {
				if existing == pattern {
					found = true
					break
				}
			}
			if !found {
				pm.allowPatterns = append(pm.allowPatterns, pattern)
			}
		}

		// Add deny patterns (avoid duplicates)
		for _, pattern := range settings.Permissions.Deny {
			found := false
			for _, existing := range pm.denyPatterns {
				if existing == pattern {
					found = true
					break
				}
			}
			if !found {
				pm.denyPatterns = append(pm.denyPatterns, pattern)
			}
		}
	}

	return nil
}

// persistSettings saves settings to disk
func (pm *PermissionManager) persistSettings(path string, settings *Settings) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// generatePattern generates a Claude Code style pattern for a permission request
// Format: "Bash(command:*)" or "Read(*)" or "Write(path/*)"
func (pm *PermissionManager) generatePattern(req *PermissionRequest) string {
	// Normalize tool name to lowercase for comparison, but use capitalized form in pattern
	toolName := strings.ToLower(req.ToolName)
	// Capitalize first letter manually
	var capitalizedName string
	if len(toolName) > 0 {
		capitalizedName = strings.ToUpper(toolName[:1]) + toolName[1:]
	} else {
		capitalizedName = toolName
	}

	switch toolName {
	case "bash":
		// For bash, pattern is: Bash(command:*)
		if cmd, ok := req.Params["command"].(string); ok {
			// Extract command name (first word)
			parts := strings.Fields(cmd)
			if len(parts) > 0 {
				cmdName := parts[0]
				return fmt.Sprintf("Bash(%s:*)", cmdName)
			}
		}
		return "Bash(*)"

	case "read":
		// For read, pattern is: Read(*)
		return "Read(*)"

	case "write":
		// For write, can be more specific with directory
		if filepath.IsAbs(req.Resource) {
			dir := filepath.Dir(req.Resource)
			return fmt.Sprintf("Write(%s/*)", dir)
		}
		return "Write(*)"

	case "edit":
		// For edit, can be more specific with directory
		if filepath.IsAbs(req.Resource) {
			dir := filepath.Dir(req.Resource)
			return fmt.Sprintf("Edit(%s/*)", dir)
		}
		return "Edit(*)"

	default:
		return fmt.Sprintf("%s(*)", capitalizedName)
	}
}

// matchPattern checks if a request pattern matches a permission pattern
func (pm *PermissionManager) matchPattern(reqPattern, permPattern string) bool {
	// Exact match
	if reqPattern == permPattern {
		return true
	}

	// Wildcard matching
	// permPattern format: "Bash(git:*)" or "Write(/path/*)"
	if strings.Contains(permPattern, "*") {
		prefix := strings.Split(permPattern, "*")[0]
		return strings.HasPrefix(reqPattern, prefix)
	}

	return false
}

// generateRequestID generates a unique request ID
func (pm *PermissionManager) generateRequestID(req *PermissionRequest) string {
	data := fmt.Sprintf("%s:%s:%s:%d",
		req.ToolName,
		req.Action,
		req.Resource,
		time.Now().UnixNano(),
	)

	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:8]) // Use first 8 bytes
}

// AnalyzeRiskLevel analyzes the risk level of a permission request
func AnalyzeRiskLevel(req *PermissionRequest) string {
	// Normalize tool name to lowercase for comparison
	toolName := strings.ToLower(req.ToolName)

	// Safe operations
	if toolName == "read" {
		return "safe"
	}

	// Dangerous operations - check paths
	dangerousPaths := []string{
		"/etc",
		"/System",
		"/usr/bin",
		"/usr/sbin",
		"/bin",
		"/sbin",
		"/.ssh",
		"/.gnupg",
	}

	for _, path := range dangerousPaths {
		if strings.HasPrefix(req.Resource, path) || strings.Contains(req.Resource, path) {
			return "dangerous"
		}
	}

	// Bash commands
	if toolName == "bash" {
		if cmd, ok := req.Params["command"].(string); ok {
			// Check for dangerous commands
			dangerousCommands := []string{
				"rm -rf",
				"sudo",
				"chmod",
				"chown",
				"dd",
				"mkfs",
				"fdisk",
				"> /dev/",
			}

			cmdLower := strings.ToLower(cmd)
			for _, dangerous := range dangerousCommands {
				if strings.Contains(cmdLower, dangerous) {
					return "dangerous"
				}
			}

			// Check for safe read-only commands
			safeCommands := []string{
				"ls", "pwd", "echo", "cat", "head", "tail", "grep",
				"find", "which", "whoami", "date", "uname",
			}

			// Extract first word (command name)
			parts := strings.Fields(cmd)
			if len(parts) > 0 {
				firstCmd := parts[0]
				for _, safe := range safeCommands {
					if firstCmd == safe {
						return "safe"
					}
				}
			}

			// All other bash commands are medium risk by default
			return "medium"
		}
		// If can't get command string, consider medium risk
		return "medium"
	}

	// Write/Edit operations
	if toolName == "write" || toolName == "edit" {
		return "medium"
	}

	return "safe"
}
