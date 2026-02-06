package openai

// OpenAI API types for chat completion

// ChatCompletionRequest represents a chat completion request
type ChatCompletionRequest struct {
	Model            string                 `json:"model"`
	Messages         []ChatMessage          `json:"messages"`
	Tools            []ChatTool             `json:"tools,omitempty"`
	Temperature      *float64               `json:"temperature,omitempty"`
	MaxTokens        *int                   `json:"max_tokens,omitempty"`
	Stream           bool                   `json:"stream"`
	StreamOptions    *StreamOptions         `json:"stream_options,omitempty"`
	ResponseFormat   *ResponseFormat        `json:"response_format,omitempty"`
}

// StreamOptions represents stream options
type StreamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}

// ResponseFormat represents response format
type ResponseFormat struct {
	Type string `json:"type"` // "text" or "json_object"
}

// ChatMessage represents a message in OpenAI format
type ChatMessage struct {
	Role             string         `json:"role"` // "system", "user", "assistant", "tool"
	Content          any            `json:"content,omitempty"` // string or []ContentPart
	ToolCalls        []ToolCall     `json:"tool_calls,omitempty"`
	ToolCallID       string         `json:"tool_call_id,omitempty"`
	Name             string         `json:"name,omitempty"`
	ReasoningContent string         `json:"reasoning_content,omitempty"` // For o1 models
}

// ContentPart represents a part of message content (for multimodal)
type ContentPart struct {
	Type     string      `json:"type"` // "text" or "image_url"
	Text     string      `json:"text,omitempty"`
	ImageURL *ImageURL   `json:"image_url,omitempty"`
}

// ImageURL represents an image URL
type ImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"` // "auto", "low", "high"
}

// ToolCall represents a tool call in OpenAI format
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // "function"
	Function FunctionCall `json:"function"`
}

// FunctionCall represents a function call
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// ChatTool represents a tool definition in OpenAI format
type ChatTool struct {
	Type     string       `json:"type"` // "function"
	Function ToolFunction `json:"function"`
}

// ToolFunction represents a function definition
type ToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"` // JSON Schema
}

// ChatCompletionResponse represents a chat completion response
type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   *Usage   `json:"usage,omitempty"`
}

// Choice represents a completion choice
type Choice struct {
	Index        int          `json:"index"`
	Message      ChatMessage  `json:"message"`
	Delta        *ChatMessage `json:"delta,omitempty"`
	FinishReason string       `json:"finish_reason,omitempty"`
}

// Usage represents token usage
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatCompletionChunk represents a streaming chunk
type ChatCompletionChunk struct {
	ID      string        `json:"id"`
	Object  string        `json:"object"`
	Created int64         `json:"created"`
	Model   string        `json:"model"`
	Choices []ChunkChoice `json:"choices"`
	Usage   *Usage        `json:"usage,omitempty"`
}

// ChunkChoice represents a choice in a streaming chunk
type ChunkChoice struct {
	Index        int          `json:"index"`
	Delta        ChatMessage  `json:"delta"`
	FinishReason *string      `json:"finish_reason,omitempty"`
}

// ErrorResponse represents an error response from OpenAI API
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail represents error details
type ErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param,omitempty"`
	Code    string `json:"code,omitempty"`
}
