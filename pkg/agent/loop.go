package agent

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/myersguo/cc-mono/pkg/ai"
)

// AgentLoopConfig represents configuration for the agent loop
type AgentLoopConfig struct {
	MaxTurns         int     // Maximum number of turns
	MaxToolCalls     int     // Maximum number of tool calls per turn
	EnableSteering   bool    // Enable steering messages
	EnableCompaction bool    // Enable automatic context compaction
	CompactionRatio  float64 // Trigger compaction at this ratio (default: 0.8)
}

// AgentLoop is the main agent loop that processes messages and tool calls
func AgentLoop(
	ctx context.Context,
	prompts []AgentMessage,
	agentContext *AgentContext,
	config *AgentLoopConfig,
	eventBus *EventBus,
) error {
	// Emit agent start event
	eventBus.Publish(NewAgentStartEvent())
	defer func() {
		eventBus.Publish(NewAgentEndEvent(agentContext.Agent.state.GetMessages()))
	}()

	agent := agentContext.Agent
	state := agent.state

	// Add initial prompts to message history
	for _, prompt := range prompts {
		state.AddMessage(prompt)
		eventBus.Publish(NewPromptAddedEvent(prompt))
	}

	turnCount := 0

	// Outer loop: process follow-up messages
	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Check max turns
		if turnCount >= config.MaxTurns {
			return fmt.Errorf("max turns (%d) exceeded", config.MaxTurns)
		}

		turnCount++

		// Emit turn start event
		eventBus.Publish(NewTurnStartEvent())

		// Build context from current messages
		messages := state.GetMessages()
		aiContext := BuildContext(state, messages)

		// Build stream options
		options := BuildStreamOptions(state)

		// Call LLM
		stream := agent.provider.Stream(ctx, state.GetModel(), aiContext, options)

		// Process the stream
		assistantMessage, toolResults, err := processStream(
			ctx,
			stream,
			state,
			agentContext,
			config,
			eventBus,
		)

		if err != nil {
			state.SetError(err.Error())
			eventBus.Publish(NewErrorEvent(err, "stream processing"))
			return fmt.Errorf("stream processing failed: %w", err)
		}

		// Add assistant message to history
		state.AddMessage(assistantMessage)

		// Add tool results to history
		for _, toolResult := range toolResults {
			toolResultMessage := AgentMessage{
				Message:   toolResult,
				ID:        fmt.Sprintf("tool-%d", time.Now().UnixNano()),
				CreatedAt: time.Now().UnixMilli(),
			}
			state.AddMessage(toolResultMessage)
		}

		// Emit turn end event
		eventBus.Publish(NewTurnEndEvent(assistantMessage, toolResults))

		// Check if we should continue
		shouldContinue := false

		// Check for follow-up messages
		if !agentContext.FollowUpQueue.IsEmpty() {
			// Process follow-up messages
			for !agentContext.FollowUpQueue.IsEmpty() {
				msg, ok := agentContext.FollowUpQueue.Pop()
				if ok {
					state.AddMessage(msg)
					shouldContinue = true
				}
			}
		}

		// Check if there are pending tool calls (need to continue)
		if len(toolResults) > 0 {
			shouldContinue = true
		}

		if !shouldContinue {
			// No more work to do
			break
		}
	}

	return nil
}

// processStream processes the LLM response stream
func processStream(
	ctx context.Context,
	stream *ai.AssistantMessageEventStream,
	state *AgentState,
	agentContext *AgentContext,
	config *AgentLoopConfig,
	eventBus *EventBus,
) (AgentMessage, []ai.ToolResultMessage, error) {
	// Accumulate content
	var textContent string
	var thinkingContent string
	var toolCalls []ai.ToolCall
	var usage ai.Usage

	state.SetIsStreaming(true)
	defer state.SetIsStreaming(false)

	// Process streaming events
	for event := range stream.Events() {
		// Check for steering messages
		if config.EnableSteering && !agentContext.SteeringQueue.IsEmpty() {
			// Cancel the stream
			stream.Close()
			break
		}

		// Process event
		switch event.Type {
		case ai.EventTypeStart:
			// Stream started
		case ai.EventTypeContentDelta:
			if event.ContentType == ai.ContentTypeText {
				textContent += event.TextDelta
			} else if event.ContentType == ai.ContentTypeThinking {
				thinkingContent += event.ThinkingDelta
			}

			// Build current message for streaming
			currentContent := make([]ai.Content, 0)
			if textContent != "" {
				currentContent = append(currentContent, ai.NewTextContent(textContent))
			}
			if thinkingContent != "" {
				currentContent = append(currentContent, ai.NewThinkingContent(thinkingContent))
			}

			streamMsg := AgentMessage{
				Message: ai.NewAssistantMessage(
					currentContent,
					"streaming",
					state.GetModel().Provider,
					state.GetModel().ID,
					usage,
					ai.StopReasonEndTurn,
				),
				ID:        fmt.Sprintf("stream-%d", time.Now().UnixNano()),
				CreatedAt: time.Now().UnixMilli(),
			}

			state.SetStreamMessage(streamMsg)

			// Emit message update event
			eventBus.Publish(NewMessageUpdateEvent(streamMsg, event))

		case ai.EventTypeToolCall:
			if event.ToolCall != nil {
				toolCalls = append(toolCalls, *event.ToolCall)
				state.AddPendingToolCall(event.ToolCall.ID)
			}

		case ai.EventTypeUsage:
			if event.Usage != nil {
				usage = *event.Usage
			}

		case ai.EventTypeEnd:
			// Stream ended

		case ai.EventTypeError:
			return AgentMessage{}, nil, fmt.Errorf("stream error: %s", event.Error)
		}
	}

	// Check for stream errors
	if err := stream.Error(); err != nil {
		return AgentMessage{}, nil, fmt.Errorf("stream error: %w", err)
	}

	// Get final result
	result := <-stream.Result()

	// Build final assistant message
	assistantMessage := AgentMessage{
		Message:   result,
		ID:        fmt.Sprintf("msg-%d", time.Now().UnixNano()),
		CreatedAt: time.Now().UnixMilli(),
	}

	// Extract tool calls from final message content
	// (In streaming, tool calls are accumulated in the final message, not in events)
	finalToolCalls := make([]ai.ToolCall, 0)
	for _, content := range result.Content {
		if tc, ok := content.(ai.ToolCall); ok {
			finalToolCalls = append(finalToolCalls, tc)
			state.AddPendingToolCall(tc.ID)
		}
	}

	// Use final tool calls if available, otherwise use accumulated ones
	if len(finalToolCalls) > 0 {
		toolCalls = finalToolCalls
	}

	// Execute tool calls if any
	var toolResults []ai.ToolResultMessage
	if len(toolCalls) > 0 {
		results, err := executeToolCalls(ctx, toolCalls, state, agentContext, config, eventBus)
		if err != nil {
			return assistantMessage, nil, fmt.Errorf("tool execution failed: %w", err)
		}
		toolResults = results
	}

	return assistantMessage, toolResults, nil
}

// executeToolCalls executes multiple tool calls concurrently
func executeToolCalls(
	ctx context.Context,
	toolCalls []ai.ToolCall,
	state *AgentState,
	agentContext *AgentContext,
	config *AgentLoopConfig,
	eventBus *EventBus,
) ([]ai.ToolResultMessage, error) {
	// Check max tool calls
	if len(toolCalls) > config.MaxToolCalls {
		return nil, fmt.Errorf("too many tool calls: %d (max: %d)", len(toolCalls), config.MaxToolCalls)
	}

	// Use errgroup for concurrent execution
	g, ctx := errgroup.WithContext(ctx)

	// Result channel
	results := make([]ai.ToolResultMessage, len(toolCalls))

	// Execute each tool call
	for i, toolCall := range toolCalls {
		i := i
		toolCall := toolCall

		g.Go(func() error {
			result, err := executeToolCall(ctx, toolCall, state, eventBus)
			if err != nil {
				// Create error result
				results[i] = ai.NewToolResultMessage(
					toolCall.ID,
					toolCall.Name,
					[]ai.Content{ai.NewTextContent(fmt.Sprintf("Error: %v", err))},
					true,
				)
			} else {
				results[i] = result
			}

			state.RemovePendingToolCall(toolCall.ID)
			return nil // Don't propagate error, we capture it in the result
		})
	}

	// Wait for all tool calls to complete
	if err := g.Wait(); err != nil {
		return nil, err
	}

	return results, nil
}

// executeToolCall executes a single tool call
func executeToolCall(
	ctx context.Context,
	toolCall ai.ToolCall,
	state *AgentState,
	eventBus *EventBus,
) (ai.ToolResultMessage, error) {
	// Find the tool
	agentTool, found := findToolByName(state, toolCall.Name)
	if !found {
		return ai.ToolResultMessage{}, fmt.Errorf("tool not found: %s", toolCall.Name)
	}

	// Check if permission manager is available in context
	permManager := ctx.Value("permission_manager")

	if permManager != nil {
		pm := permManager.(*PermissionManager)

		// Create permission request
		req := &PermissionRequest{
			ToolName: toolCall.Name,
			Action:   "execute",
			Resource: extractResource(toolCall),
			Params:   toolCall.Params,
		}
		req.RiskLevel = AnalyzeRiskLevel(req)
		req.Description = describeToolCall(toolCall)

		// Check permission
		allowed, needAsk, err := pm.CheckPermission(req)
		if err != nil {
			return ai.ToolResultMessage{}, fmt.Errorf("permission check failed: %w", err)
		}

		if needAsk {
			// Emit permission request event
			eventBus.Publish(NewPermissionRequestEvent(req))

			// Request permission from user with context
			resp, err := pm.RequestPermissionWithContext(ctx, req)
			if err != nil {
				return ai.ToolResultMessage{}, fmt.Errorf("permission request failed: %w", err)
			}

			if !resp.Allowed {
				return ai.ToolResultMessage{}, fmt.Errorf("permission denied by user")
			}
		} else if !allowed {
			return ai.ToolResultMessage{}, fmt.Errorf("permission denied by policy")
		}
	}

	// Emit tool execution start event
	eventBus.Publish(NewToolExecutionStartEvent(toolCall.ID, toolCall.Name, toolCall.Params))

	// Create update callback
	onUpdate := func(update AgentToolUpdate) {
		// Can emit progress events here if needed
	}

	// Execute the tool
	result, err := agentTool.Execute(ctx, toolCall.ID, toolCall.Params, onUpdate)

	// Emit tool execution end event
	eventBus.Publish(NewToolExecutionEndEvent(toolCall.ID, toolCall.Name, result, err != nil))

	if err != nil {
		return ai.ToolResultMessage{}, err
	}

	// Convert to tool result message
	toolResultMsg := ai.NewToolResultMessage(
		toolCall.ID,
		toolCall.Name,
		result.Content,
		result.IsError,
	)
	toolResultMsg.Details = result.Details

	return toolResultMsg, nil
}

// findToolByName finds a tool by name in the agent state
func findToolByName(state *AgentState, name string) (AgentTool, bool) {
	tools := state.GetTools()
	for _, tool := range tools {
		if tool.Tool.Name == name {
			return tool, true
		}
	}
	return AgentTool{}, false
}

// extractResource extracts the resource identifier from a tool call
func extractResource(toolCall ai.ToolCall) string {
	switch toolCall.Name {
	case "Read", "Write", "Edit":
		if path, ok := toolCall.Params["file_path"].(string); ok {
			return path
		}
	case "Bash":
		if cmd, ok := toolCall.Params["command"].(string); ok {
			return cmd
		}
	}
	return ""
}

// describeToolCall creates a human-readable description of a tool call
func describeToolCall(toolCall ai.ToolCall) string {
	switch toolCall.Name {
	case "Read":
		if path, ok := toolCall.Params["file_path"].(string); ok {
			return fmt.Sprintf("Read file: %s", path)
		}
	case "Write":
		if path, ok := toolCall.Params["file_path"].(string); ok {
			return fmt.Sprintf("Write file: %s", path)
		}
	case "Edit":
		if path, ok := toolCall.Params["file_path"].(string); ok {
			return fmt.Sprintf("Edit file: %s", path)
		}
	case "Bash":
		if cmd, ok := toolCall.Params["command"].(string); ok {
			return fmt.Sprintf("Execute command: %s", cmd)
		}
	}
	return fmt.Sprintf("Execute tool: %s", toolCall.Name)
}
