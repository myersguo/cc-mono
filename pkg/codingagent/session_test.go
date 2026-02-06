package codingagent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/myersguo/cc-mono/pkg/agent"
	"github.com/myersguo/cc-mono/pkg/ai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionManager_CreateSession(t *testing.T) {
	tempDir := t.TempDir()
	sm, err := NewSessionManager(filepath.Join(tempDir, "sessions"))
	require.NoError(t, err)

	state := agent.NewAgentState("system", ai.Model{}, []agent.AgentTool{})
	session := sm.NewSession("Test Session", state)
	assert.NotEmpty(t, session.Metadata.ID)
	assert.Equal(t, "Test Session", session.Metadata.Title)
	assert.NotZero(t, session.Metadata.CreatedAt)
	assert.NotZero(t, session.Metadata.UpdatedAt)
	assert.Empty(t, session.Metadata.ParentID)
	assert.Equal(t, 0, session.Metadata.BranchPoint)
}

func TestSessionManager_SaveAndLoad(t *testing.T) {
	tempDir := t.TempDir()
	sm, err := NewSessionManager(filepath.Join(tempDir, "sessions"))
	require.NoError(t, err)

	// Create a session with some state
	state := agent.NewAgentState("system", ai.Model{}, []agent.AgentTool{})
	session := sm.NewSession("Test Session", state)

	// Add some messages to state
	userMsg := ai.UserMessage{
		Type:      ai.MessageTypeUser,
		Content:   []ai.Content{ai.NewTextContent("Hello")},
		Timestamp: time.Now().UnixMilli(),
	}
	session.State.AddMessage(agent.NewAgentMessage(userMsg, "msg-1", time.Now().UnixMilli()))

	assistantMsg := ai.AssistantMessage{
		Type:      ai.MessageTypeAssistant,
		Content:   []ai.Content{ai.NewTextContent("Hi there!")},
		Timestamp: time.Now().UnixMilli(),
	}
	session.State.AddMessage(agent.NewAgentMessage(assistantMsg, "msg-2", time.Now().UnixMilli()))

	// Save the session
	err = sm.Save(session)
	require.NoError(t, err)

	// Verify file was created
	sessionPath := filepath.Join(tempDir, "sessions", session.Metadata.ID+".json")
	assert.FileExists(t, sessionPath)

	// Load the session
	loaded, err := sm.Load(session.Metadata.ID)
	require.NoError(t, err)

	// Verify metadata
	assert.Equal(t, session.Metadata.ID, loaded.Metadata.ID)
	assert.Equal(t, session.Metadata.Title, loaded.Metadata.Title)

	// Verify messages
	messages := loaded.State.GetMessages()
	assert.Len(t, messages, 2)

	// Check first message
	assert.Equal(t, ai.MessageTypeUser, messages[0].Message.GetType())

	// Check second message
	assert.Equal(t, ai.MessageTypeAssistant, messages[1].Message.GetType())
}

func TestSessionManager_LoadNonExistent(t *testing.T) {
	tempDir := t.TempDir()
	sm, err := NewSessionManager(filepath.Join(tempDir, "sessions"))
	require.NoError(t, err)

	_, err = sm.Load("nonexistent-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read session file")
}

func TestSessionManager_Fork(t *testing.T) {
	tempDir := t.TempDir()
	sm, err := NewSessionManager(filepath.Join(tempDir, "sessions"))
	require.NoError(t, err)

	// Create original session with messages
	state := agent.NewAgentState("system", ai.Model{}, []agent.AgentTool{})
	original := sm.NewSession("Original Session", state)

	msg1 := ai.UserMessage{
		Type:      ai.MessageTypeUser,
		Content:   []ai.Content{ai.NewTextContent("Message 1")},
		Timestamp: time.Now().UnixMilli(),
	}
	original.State.AddMessage(agent.NewAgentMessage(msg1, "msg-1", time.Now().UnixMilli()))

	msg2 := ai.AssistantMessage{
		Type:      ai.MessageTypeAssistant,
		Content:   []ai.Content{ai.NewTextContent("Response 1")},
		Timestamp: time.Now().UnixMilli(),
	}
	original.State.AddMessage(agent.NewAgentMessage(msg2, "msg-2", time.Now().UnixMilli()))

	msg3 := ai.UserMessage{
		Type:      ai.MessageTypeUser,
		Content:   []ai.Content{ai.NewTextContent("Message 2")},
		Timestamp: time.Now().UnixMilli(),
	}
	original.State.AddMessage(agent.NewAgentMessage(msg3, "msg-3", time.Now().UnixMilli()))

	err = sm.Save(original)
	require.NoError(t, err)

	// Fork at message index 1 (after first assistant response)
	forked, err := sm.Fork(original.Metadata.ID, 1, "Forked Session")
	require.NoError(t, err)

	// Verify forked session metadata
	assert.NotEqual(t, original.Metadata.ID, forked.Metadata.ID)
	assert.Equal(t, "Forked Session", forked.Metadata.Title)
	assert.Equal(t, original.Metadata.ID, forked.Metadata.ParentID)
	assert.Equal(t, 1, forked.Metadata.BranchPoint)

	// Verify forked session has only messages up to branch point
	forkedMessages := forked.State.GetMessages()
	assert.Len(t, forkedMessages, 1) // Only Message 1

	// Verify original session is unchanged
	originalMessages := original.State.GetMessages()
	assert.Len(t, originalMessages, 3)
}

func TestSessionManager_ListSessions(t *testing.T) {
	tempDir := t.TempDir()
	sm, err := NewSessionManager(filepath.Join(tempDir, "sessions"))
	require.NoError(t, err)

	// Create multiple sessions
	state1 := agent.NewAgentState("system", ai.Model{}, []agent.AgentTool{})
	session1 := sm.NewSession("Session 1", state1)
	err = sm.Save(session1)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond) // Ensure different timestamps

	state2 := agent.NewAgentState("system", ai.Model{}, []agent.AgentTool{})
	session2 := sm.NewSession("Session 2", state2)
	err = sm.Save(session2)
	require.NoError(t, err)

	// List sessions
	sessions, err := sm.List()
	require.NoError(t, err)
	assert.Len(t, sessions, 2)

	// Should be sorted by UpdatedAt descending (newest first)
	assert.Equal(t, "Session 2", sessions[0].Title)
	assert.Equal(t, "Session 1", sessions[1].Title)
}

func TestSessionManager_DeleteSession(t *testing.T) {
	tempDir := t.TempDir()
	sm, err := NewSessionManager(filepath.Join(tempDir, "sessions"))
	require.NoError(t, err)

	// Create and save a session
	state := agent.NewAgentState("system", ai.Model{}, []agent.AgentTool{})
	session := sm.NewSession("Test Session", state)
	err = sm.Save(session)
	require.NoError(t, err)

	// Verify it exists
	sessionPath := filepath.Join(tempDir, "sessions", session.Metadata.ID+".json")
	assert.FileExists(t, sessionPath)

	// Delete the session
	err = sm.Delete(session.Metadata.ID)
	require.NoError(t, err)

	// Verify it's gone
	_, err = os.Stat(sessionPath)
	assert.True(t, os.IsNotExist(err))

	// Verify it's not in the list
	sessions, err := sm.List()
	require.NoError(t, err)
	assert.Len(t, sessions, 0)
}

func TestSessionManager_GetSessionTree(t *testing.T) {
	tempDir := t.TempDir()
	sm, err := NewSessionManager(filepath.Join(tempDir, "sessions"))
	require.NoError(t, err)

	// Create parent session
	parentState := agent.NewAgentState("system", ai.Model{}, []agent.AgentTool{})
	parent := sm.NewSession("Parent", parentState)
	msg := ai.UserMessage{
		Type:      ai.MessageTypeUser,
		Content:   []ai.Content{ai.NewTextContent("Message 1")},
		Timestamp: time.Now().UnixMilli(),
	}
	parent.State.AddMessage(agent.NewAgentMessage(msg, "msg-1", time.Now().UnixMilli()))
	err = sm.Save(parent)
	require.NoError(t, err)

	// Create first fork
	fork1, err := sm.Fork(parent.Metadata.ID, 0, "Fork 1")
	require.NoError(t, err)
	err = sm.Save(fork1)
	require.NoError(t, err)

	// Create second fork
	fork2, err := sm.Fork(parent.Metadata.ID, 0, "Fork 2")
	require.NoError(t, err)
	err = sm.Save(fork2)
	require.NoError(t, err)

	// Create fork of fork1
	fork1_1, err := sm.Fork(fork1.Metadata.ID, 0, "Fork 1.1")
	require.NoError(t, err)
	err = sm.Save(fork1_1)
	require.NoError(t, err)

	// Get session tree
	tree, err := sm.GetSessionTree(parent.Metadata.ID)
	require.NoError(t, err)

	// Verify tree contains parent
	assert.GreaterOrEqual(t, len(tree), 1)
	assert.Equal(t, parent.Metadata.ID, tree[0].ID)
}

func TestSession_ExportToHTML(t *testing.T) {
	tempDir := t.TempDir()
	sm, err := NewSessionManager(filepath.Join(tempDir, "sessions"))
	require.NoError(t, err)

	state := agent.NewAgentState("system", ai.Model{}, []agent.AgentTool{})
	session := sm.NewSession("Test Session", state)

	// Add messages
	userMsg := ai.UserMessage{
		Type:      ai.MessageTypeUser,
		Content:   []ai.Content{ai.NewTextContent("Hello, how are you?")},
		Timestamp: time.Now().UnixMilli(),
	}
	session.State.AddMessage(agent.NewAgentMessage(userMsg, "msg-1", time.Now().UnixMilli()))

	assistantMsg := ai.AssistantMessage{
		Type: ai.MessageTypeAssistant,
		Content: []ai.Content{
			ai.NewTextContent("I'm doing well, thank you!"),
		},
		Timestamp: time.Now().UnixMilli(),
	}
	session.State.AddMessage(agent.NewAgentMessage(assistantMsg, "msg-2", time.Now().UnixMilli()))

	// Save first
	err = sm.Save(session)
	require.NoError(t, err)

	// Export to HTML
	outputPath := filepath.Join(tempDir, "export.html")
	err = sm.Export(session.Metadata.ID, outputPath)
	require.NoError(t, err)

	// Verify file was created
	assert.FileExists(t, outputPath)

	// Read and verify contents
	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	html := string(content)
	assert.Contains(t, html, "<title>Test Session</title>")
	assert.Contains(t, html, "Test Session")
}

func TestSession_JSONSerialization(t *testing.T) {
	state := agent.NewAgentState("system prompt", ai.Model{}, []agent.AgentTool{})

	session := &Session{
		Metadata: SessionMetadata{
			ID:          "test-id",
			Title:       "Test Session",
			CreatedAt:   time.Now().Round(time.Second),
			UpdatedAt:   time.Now().Round(time.Second),
			ParentID:    "parent-id",
			BranchPoint: 5,
		},
		State: state,
	}

	// Add a message
	userMsg := ai.UserMessage{
		Type:      ai.MessageTypeUser,
		Content:   []ai.Content{ai.NewTextContent("Test message")},
		Timestamp: time.Now().UnixMilli(),
	}
	session.State.AddMessage(agent.NewAgentMessage(userMsg, "msg-1", time.Now().UnixMilli()))

	// Serialize to JSON
	data, err := json.Marshal(session)
	require.NoError(t, err)

	// Verify we can serialize
	assert.NotEmpty(t, data)

	// Verify metadata is in the JSON
	var jsonMap map[string]any
	err = json.Unmarshal(data, &jsonMap)
	require.NoError(t, err)

	metadata, ok := jsonMap["metadata"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "test-id", metadata["id"])
	assert.Equal(t, "Test Session", metadata["title"])
	assert.Equal(t, "parent-id", metadata["parent_id"])
	assert.Equal(t, float64(5), metadata["branch_point"]) // JSON numbers are float64
}

func TestSessionManager_UpdateSession(t *testing.T) {
	tempDir := t.TempDir()
	sm, err := NewSessionManager(filepath.Join(tempDir, "sessions"))
	require.NoError(t, err)

	// Create session
	state := agent.NewAgentState("system", ai.Model{}, []agent.AgentTool{})
	session := sm.NewSession("Original Title", state)
	err = sm.Save(session)
	require.NoError(t, err)

	originalUpdatedAt := session.Metadata.UpdatedAt
	time.Sleep(10 * time.Millisecond)

	// Update session
	session.Metadata.Title = "Updated Title"
	userMsg := ai.UserMessage{
		Type:      ai.MessageTypeUser,
		Content:   []ai.Content{ai.NewTextContent("New message")},
		Timestamp: time.Now().UnixMilli(),
	}
	session.State.AddMessage(agent.NewAgentMessage(userMsg, "msg-1", time.Now().UnixMilli()))

	err = sm.Save(session)
	require.NoError(t, err)

	// Load and verify
	loaded, err := sm.Load(session.Metadata.ID)
	require.NoError(t, err)

	assert.Equal(t, "Updated Title", loaded.Metadata.Title)
	assert.True(t, loaded.Metadata.UpdatedAt.After(originalUpdatedAt))
	assert.Len(t, loaded.State.GetMessages(), 1)
}
