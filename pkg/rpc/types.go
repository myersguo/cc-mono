package rpc

import (
	"time"

	"github.com/myersguo/cc-mono/pkg/agent"
	"github.com/myersguo/cc-mono/pkg/ai"
)

// RPC 命令类型
const (
	CommandPrompt           = "prompt"
	CommandSteer            = "steer"
	CommandFollowUp         = "follow_up"
	CommandAbort            = "abort"
	CommandNewSession       = "new_session"
	CommandGetState         = "get_state"
	CommandSetModel         = "set_model"
	CommandCycleModel       = "cycle_model"
	CommandSetThinkingLevel = "set_thinking_level"
	CommandCycleThinking    = "cycle_thinking_level"
	CommandGetAvailableModels = "get_available_models"
	CommandBash             = "bash"
	CommandAbortBash        = "abort_bash"
	CommandGetSessionStats  = "get_session_stats"
	CommandGetMessages      = "get_messages"
)

// RpcCommand 表示 RPC 命令
type RpcCommand struct {
	ID      string      `json:"id"`      // 请求 ID，用于关联响应
	Type    string      `json:"type"`    // 命令类型
	Message string      `json:"message,omitempty"`  // prompt、steer、follow_up 命令的消息
	Images  []ImageContent `json:"images,omitempty"` // 可选的图片内容
	Provider string      `json:"provider,omitempty"` // set_model 命令的提供商
	ModelID string      `json:"model_id,omitempty"` // set_model 命令的模型 ID
	Level   string      `json:"level,omitempty"`    // set_thinking_level 命令的思考级别
	Command string      `json:"command,omitempty"`  // bash 命令
}

// ImageContent 表示图片内容
type ImageContent struct {
	Data string `json:"data"` // Base64 编码的图片数据
	MimeType string `json:"mime_type"` // 图片类型，如 image/png
}

// RpcResponse 表示 RPC 响应
type RpcResponse struct {
	ID      string      `json:"id"`
	Type    string      `json:"type"`    // "response"
	Command string      `json:"command"` // 请求的命令类型
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// RpcSessionState 表示会话状态
type RpcSessionState struct {
	SystemPrompt string          `json:"system_prompt"`
	Model        ai.Model        `json:"model"`
	ThinkingLevel string         `json:"thinking_level"`
	Messages     []agent.AgentMessage `json:"messages"`
}

// ModelInfo 表示模型信息
type ModelInfo struct {
	Provider      string `json:"provider"`
	ID            string `json:"id"`
	Name          string `json:"name"`
	ContextWindow int    `json:"context_window"`
	MaxOutput     int    `json:"max_output"`
}

// SessionStats 表示会话统计信息
type SessionStats struct {
	MessageCount int       `json:"message_count"`
	TotalTokens  int       `json:"total_tokens"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// BashResult 表示 Bash 命令结果
type BashResult struct {
	Output   string `json:"output"`
	ExitCode int    `json:"exit_code"`
	Error    string `json:"error,omitempty"`
}

// 事件类型
const (
	EventTypeAgentStart     = "agent_start"
	EventTypeAgentEnd       = "agent_end"
	EventTypeMessage        = "message"
	EventTypeThinking       = "thinking"
	EventTypeToolCall       = "tool_call"
	EventTypeToolResult     = "tool_result"
	EventTypeError          = "error"
)

// RpcEvent 表示 RPC 事件
type RpcEvent struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data,omitempty"`
}
