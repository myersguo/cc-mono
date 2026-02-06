package codingagent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/myersguo/cc-mono/pkg/agent"
)

// SessionMetadata represents metadata about a session
type SessionMetadata struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	ParentID    string    `json:"parent_id,omitempty"`
	BranchPoint int       `json:"branch_point,omitempty"` // Message index where branch occurred
	Tags        []string  `json:"tags,omitempty"`
}

// Session represents a complete agent session
type Session struct {
	Metadata SessionMetadata `json:"metadata"`
	State    *agent.AgentState `json:"state"`
}

// SessionManager manages agent sessions
type SessionManager struct {
	mu             sync.RWMutex
	sessionsDir    string
	currentSession *Session
	cache          map[string]*Session
}

// NewSessionManager creates a new session manager
func NewSessionManager(sessionsDir string) (*SessionManager, error) {
	// Create sessions directory if it doesn't exist
	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create sessions directory: %w", err)
	}

	return &SessionManager{
		sessionsDir: sessionsDir,
		cache:       make(map[string]*Session),
	}, nil
}

// NewSession creates a new session
func (sm *SessionManager) NewSession(title string, state *agent.AgentState) *Session {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	session := &Session{
		Metadata: SessionMetadata{
			ID:        generateSessionID(),
			Title:     title,
			CreatedAt: now,
			UpdatedAt: now,
		},
		State: state,
	}

	sm.currentSession = session
	sm.cache[session.Metadata.ID] = session

	return session
}

// Save saves a session to disk
func (sm *SessionManager) Save(session *Session) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session.Metadata.UpdatedAt = time.Now()

	// Marshal session to JSON
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	// Write to file
	path := sm.getSessionPath(session.Metadata.ID)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	// Update cache
	sm.cache[session.Metadata.ID] = session

	return nil
}

// Load loads a session from disk
func (sm *SessionManager) Load(id string) (*Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Check cache first
	if session, ok := sm.cache[id]; ok {
		return session, nil
	}

	// Read from disk
	path := sm.getSessionPath(id)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	// Unmarshal session
	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	// Update cache
	sm.cache[id] = &session

	return &session, nil
}

// List lists all sessions
func (sm *SessionManager) List() ([]SessionMetadata, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Read all session files
	entries, err := os.ReadDir(sm.sessionsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read sessions directory: %w", err)
	}

	metadataList := make([]SessionMetadata, 0)

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		// Extract session ID from filename
		id := entry.Name()[:len(entry.Name())-5] // Remove .json

		// Try to get from cache first
		if session, ok := sm.cache[id]; ok {
			metadataList = append(metadataList, session.Metadata)
			continue
		}

		// Read metadata from file
		path := sm.getSessionPath(id)
		data, err := os.ReadFile(path)
		if err != nil {
			continue // Skip files we can't read
		}

		var session Session
		if err := json.Unmarshal(data, &session); err != nil {
			continue // Skip files we can't parse
		}

		metadataList = append(metadataList, session.Metadata)
	}

	// Sort by updated time (most recent first)
	sort.Slice(metadataList, func(i, j int) bool {
		return metadataList[i].UpdatedAt.After(metadataList[j].UpdatedAt)
	})

	return metadataList, nil
}

// Delete deletes a session
func (sm *SessionManager) Delete(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Remove from cache
	delete(sm.cache, id)

	// Delete file
	path := sm.getSessionPath(id)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete session file: %w", err)
	}

	return nil
}

// Fork creates a new session branching from an existing one
func (sm *SessionManager) Fork(sessionID string, branchPoint int, title string) (*Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Load the parent session
	parentSession, err := sm.loadLocked(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load parent session: %w", err)
	}

	// Create new state with messages up to branch point
	messages := parentSession.State.GetMessages()
	if branchPoint < 0 || branchPoint > len(messages) {
		return nil, fmt.Errorf("invalid branch point: %d (total messages: %d)", branchPoint, len(messages))
	}

	// Create new state
	newState := agent.NewAgentState(
		parentSession.State.GetSystemPrompt(),
		parentSession.State.GetModel(),
		parentSession.State.GetTools(),
	)

	// Copy messages up to branch point
	for i := 0; i < branchPoint && i < len(messages); i++ {
		newState.AddMessage(messages[i])
	}

	// Create new session
	now := time.Now()
	newSession := &Session{
		Metadata: SessionMetadata{
			ID:          generateSessionID(),
			Title:       title,
			CreatedAt:   now,
			UpdatedAt:   now,
			ParentID:    sessionID,
			BranchPoint: branchPoint,
		},
		State: newState,
	}

	// Update cache
	sm.cache[newSession.Metadata.ID] = newSession

	return newSession, nil
}

// GetCurrent returns the current session
func (sm *SessionManager) GetCurrent() *Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.currentSession
}

// SetCurrent sets the current session
func (sm *SessionManager) SetCurrent(session *Session) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.currentSession = session
}

// Export exports a session to HTML
func (sm *SessionManager) Export(sessionID string, outputPath string) error {
	session, err := sm.Load(sessionID)
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}

	html := generateHTML(session)

	if err := os.WriteFile(outputPath, []byte(html), 0644); err != nil {
		return fmt.Errorf("failed to write HTML file: %w", err)
	}

	return nil
}

// GetSessionTree returns the session tree starting from a session
func (sm *SessionManager) GetSessionTree(sessionID string) ([]SessionMetadata, error) {
	allSessions, err := sm.List()
	if err != nil {
		return nil, err
	}

	// Build tree
	tree := make([]SessionMetadata, 0)
	visited := make(map[string]bool)

	var buildTree func(string)
	buildTree = func(id string) {
		if visited[id] {
			return
		}
		visited[id] = true

		// Find this session
		for _, session := range allSessions {
			if session.ID == id {
				tree = append(tree, session)

				// Find children
				for _, child := range allSessions {
					if child.ParentID == id {
						buildTree(child.ID)
					}
				}
				break
			}
		}
	}

	buildTree(sessionID)

	return tree, nil
}

// Private helper methods

func (sm *SessionManager) getSessionPath(id string) string {
	return filepath.Join(sm.sessionsDir, id+".json")
}

func (sm *SessionManager) loadLocked(id string) (*Session, error) {
	// Check cache first
	if session, ok := sm.cache[id]; ok {
		return session, nil
	}

	// Read from disk
	path := sm.getSessionPath(id)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}

	sm.cache[id] = &session
	return &session, nil
}

func generateSessionID() string {
	return fmt.Sprintf("session-%d", time.Now().UnixNano())
}

func generateHTML(session *Session) string {
	html := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>` + session.Metadata.Title + `</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            max-width: 900px;
            margin: 0 auto;
            padding: 20px;
            line-height: 1.6;
        }
        .header {
            border-bottom: 2px solid #333;
            padding-bottom: 20px;
            margin-bottom: 30px;
        }
        .message {
            margin-bottom: 20px;
            padding: 15px;
            border-radius: 8px;
        }
        .user {
            background-color: #e3f2fd;
        }
        .assistant {
            background-color: #f5f5f5;
        }
        .tool-result {
            background-color: #fff3e0;
        }
        .role {
            font-weight: bold;
            margin-bottom: 8px;
            color: #555;
        }
        .content {
            white-space: pre-wrap;
        }
        .metadata {
            font-size: 0.9em;
            color: #666;
            margin-top: 10px;
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>` + session.Metadata.Title + `</h1>
        <div class="metadata">
            <div>Session ID: ` + session.Metadata.ID + `</div>
            <div>Created: ` + session.Metadata.CreatedAt.Format(time.RFC3339) + `</div>
            <div>Updated: ` + session.Metadata.UpdatedAt.Format(time.RFC3339) + `</div>
        </div>
    </div>
    <div class="messages">`

	// Add messages
	messages := session.State.GetMessages()
	for _, msg := range messages {
		msgType := msg.Message.GetType()
		html += `
        <div class="message ` + string(msgType) + `">
            <div class="role">` + string(msgType) + `</div>
            <div class="content">`

		// Simple content extraction (would need proper formatting for production)
		html += fmt.Sprintf("%v", msg.Message)

		html += `</div>
        </div>`
	}

	html += `
    </div>
</body>
</html>`

	return html
}
