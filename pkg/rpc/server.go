package rpc

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/myersguo/cc-mono/pkg/agent"
	"github.com/myersguo/cc-mono/pkg/ai"
	"github.com/myersguo/cc-mono/pkg/codingagent"
)

// Server 是 RPC 服务器
type Server struct {
	agent          *agent.Agent
	modelRegistry  *codingagent.ModelRegistry
	providerConfig *codingagent.ProvidersConfig
	sessionManager *codingagent.SessionManager
	agentContext   *agent.AgentContext
	ctx            context.Context
	cancel         context.CancelFunc

	mu       sync.Mutex
	isRunning bool
	stopChan chan struct{}

	reader io.Reader
	writer io.Writer
}

// NewServer 创建新的 RPC 服务器
func NewServer(
	agentInst *agent.Agent,
	registry *codingagent.ModelRegistry,
	providers *codingagent.ProvidersConfig,
	sessionMgr *codingagent.SessionManager,
	reader io.Reader,
	writer io.Writer,
) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		agent:          agentInst,
		modelRegistry:  registry,
		providerConfig: providers,
		sessionManager: sessionMgr,
		agentContext:   agent.NewAgentContext(agentInst), // 使用包级函数 NewAgentContext
		ctx:            ctx,
		cancel:         cancel,
		isRunning:      true,
		stopChan:       make(chan struct{}),
		reader:         reader,
		writer:         writer,
	}
}

// Run 启动 RPC 服务器，监听输入并响应命令
func (s *Server) Run(ctx context.Context) error {
	// 设置事件监听器
	s.setupEventListeners()

	scanner := bufio.NewScanner(s.reader)
	
	// 如果是标准输入，打印就绪消息
	if s.reader == io.Reader(os.Stdin) {
		fmt.Println("RPC server ready. Waiting for commands...")
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// 解析命令
		var cmd RpcCommand
		if err := json.Unmarshal([]byte(line), &cmd); err != nil {
			s.sendError("", "invalid_json", fmt.Sprintf("Invalid JSON: %v", err))
			continue
		}

		// 处理命令
		go s.handleCommand(ctx, cmd)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read error: %w", err)
	}

	return nil
}

// setupEventListeners 设置代理事件监听器
func (s *Server) setupEventListeners() {
	if s.agent == nil {
		return
	}

	// 监听代理事件并转换为 RPC 事件
	eventChan := s.agent.GetEventBus().Subscribe(100)

	go func() {
		for {
			select {
			case event, ok := <-eventChan:
				if !ok {
					return
				}

				rpcEvent := RpcEvent{
					Timestamp: time.Now(),
				}

				switch e := event.(type) {
				case agent.AgentStartEvent:
					rpcEvent.Type = EventTypeAgentStart
					rpcEvent.Data = e
				case agent.AgentEndEvent:
					rpcEvent.Type = EventTypeAgentEnd
					rpcEvent.Data = e
				case agent.TurnStartEvent:
					rpcEvent.Type = "turn_start"
					rpcEvent.Data = e
				case agent.TurnEndEvent:
					rpcEvent.Type = "turn_end"
					rpcEvent.Data = e
				case agent.MessageUpdateEvent:
					rpcEvent.Type = "message_update"
					rpcEvent.Data = e
				case agent.ToolExecutionStartEvent:
					rpcEvent.Type = EventTypeToolCall
					rpcEvent.Data = e
				case agent.ToolExecutionEndEvent:
					rpcEvent.Type = EventTypeToolResult
					rpcEvent.Data = e
				case agent.ErrorEvent:
					rpcEvent.Type = EventTypeError
					rpcEvent.Data = e
				case agent.PermissionRequestEvent:
					rpcEvent.Type = "permission_request"
					rpcEvent.Data = e
				case agent.PromptAddedEvent:
					rpcEvent.Type = "prompt_added"
					rpcEvent.Data = e
				default:
					rpcEvent.Type = "unknown"
					rpcEvent.Data = e
				}

				// 发送事件
				s.sendEvent(rpcEvent)
			case <-s.stopChan:
				return
			}
		}
	}()
}

// handleCommand 处理单个 RPC 命令
func (s *Server) handleCommand(ctx context.Context, cmd RpcCommand) {
	switch cmd.Type {
	case CommandPrompt:
		s.handlePrompt(cmd)
	case CommandSteer:
		s.handleSteer(cmd)
	case CommandFollowUp:
		s.handleFollowUp(cmd)
	case CommandAbort:
		s.handleAbort(cmd)
	case CommandGetState:
		s.handleGetState(cmd)
	case CommandNewSession:
		s.handleNewSession(cmd)
	case CommandSetModel:
		s.handleSetModel(cmd)
	case CommandGetAvailableModels:
		s.handleGetAvailableModels(cmd)
	case CommandBash:
		s.handleBash(cmd)
	case CommandGetMessages:
		s.handleGetMessages(cmd)
	case CommandGetSessionStats:
		s.handleGetSessionStats(cmd)
	default:
		s.sendError(cmd.ID, cmd.Type, fmt.Sprintf("Unknown command: %s", cmd.Type))
	}
}

func (s *Server) handlePrompt(cmd RpcCommand) {
	if s.agent == nil {
		s.sendError(cmd.ID, cmd.Type, "Agent not initialized")
		return
	}

	// 创建用户消息
	userMsg := agent.AgentMessage{
		Message:   ai.NewUserTextMessage(cmd.Message),
		ID:        fmt.Sprintf("msg-%d", time.Now().UnixNano()),
		CreatedAt: time.Now().UnixMilli(),
	}

	// 运行代理
	go func() {
		if err := s.agent.Run(s.ctx, []agent.AgentMessage{userMsg}); err != nil {
			s.sendError(cmd.ID, cmd.Type, fmt.Sprintf("Prompt failed: %v", err))
		} else {
			s.sendSuccess(cmd.ID, cmd.Type, nil)
		}
	}()
}

func (s *Server) handleSteer(cmd RpcCommand) {
	if s.agent == nil {
		s.sendError(cmd.ID, cmd.Type, "Agent not initialized")
		return
	}

	// 添加转向消息
	steerMsg := agent.AgentMessage{
		Message:   ai.NewUserTextMessage(cmd.Message),
		ID:        fmt.Sprintf("steer-%d", time.Now().UnixNano()),
		CreatedAt: time.Now().UnixMilli(),
	}
	s.agentContext.AddSteeringMessage(steerMsg)

	s.sendSuccess(cmd.ID, cmd.Type, nil)
}

func (s *Server) handleFollowUp(cmd RpcCommand) {
	if s.agent == nil {
		s.sendError(cmd.ID, cmd.Type, "Agent not initialized")
		return
	}

	// 添加跟进消息
	followUpMsg := agent.AgentMessage{
		Message:   ai.NewUserTextMessage(cmd.Message),
		ID:        fmt.Sprintf("followup-%d", time.Now().UnixNano()),
		CreatedAt: time.Now().UnixMilli(),
	}
	s.agentContext.AddFollowUpMessage(followUpMsg)

	s.sendSuccess(cmd.ID, cmd.Type, nil)
}

func (s *Server) handleAbort(cmd RpcCommand) {
	if s.cancel == nil {
		s.sendError(cmd.ID, cmd.Type, "Agent not initialized")
		return
	}

	// 取消上下文以停止代理
	s.cancel()
	
	// 重新创建上下文以允许后续运行
	ctx, cancel := context.WithCancel(context.Background())
	s.ctx = ctx
	s.cancel = cancel
	
	s.sendSuccess(cmd.ID, cmd.Type, nil)
}

func (s *Server) handleGetState(cmd RpcCommand) {
	if s.agent == nil {
		s.sendError(cmd.ID, cmd.Type, "Agent not initialized")
		return
	}

	// 获取当前状态
	state := s.agent.GetState()
	rpcState := RpcSessionState{
		SystemPrompt: state.GetSystemPrompt(),
		Model:        state.GetModel(),
		ThinkingLevel: "medium", // 默认思考级别
		Messages:     state.GetMessages(),
	}

	s.sendSuccess(cmd.ID, cmd.Type, rpcState)
}

func (s *Server) handleNewSession(cmd RpcCommand) {
	s.sendSuccess(cmd.ID, cmd.Type, map[string]interface{}{
		"cancelled": false,
	})
}

func (s *Server) handleSetModel(cmd RpcCommand) {
	if s.modelRegistry == nil {
		s.sendError(cmd.ID, cmd.Type, "Model registry not available")
		return
	}

	if cmd.Provider == "" || cmd.ModelID == "" {
		s.sendError(cmd.ID, cmd.Type, "Provider and model ID are required")
		return
	}

	// 查找模型
	model, err := s.modelRegistry.ToAIModel(cmd.ModelID)
	if err != nil {
		s.sendError(cmd.ID, cmd.Type, fmt.Sprintf("Model not found: %v", err))
		return
	}

	// 更新代理的模型
	if s.agent != nil {
		s.agent.SetModel(model)
	}

	s.sendSuccess(cmd.ID, cmd.Type, map[string]string{
		"provider": cmd.Provider,
		"id":       cmd.ModelID,
	})
}

func (s *Server) handleGetAvailableModels(cmd RpcCommand) {
	if s.modelRegistry == nil {
		s.sendError(cmd.ID, cmd.Type, "Model registry not available")
		return
	}

	models := s.modelRegistry.List()
	infos := make([]ModelInfo, len(models))
	for i, m := range models {
		infos[i] = ModelInfo{
			Provider:      m.Provider,
			ID:            m.ID,
			Name:          m.Name,
			ContextWindow: m.ContextWindow,
			MaxOutput:     m.MaxOutput,
		}
	}

	s.sendSuccess(cmd.ID, cmd.Type, map[string]interface{}{
		"models": infos,
	})
}

func (s *Server) handleBash(cmd RpcCommand) {
	if cmd.Command == "" {
		s.sendError(cmd.ID, cmd.Type, "Bash command is required")
		return
	}

	// 直接执行 bash 命令
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "bash", "-c", cmd.Command).CombinedOutput()
	bashResult := BashResult{
		Output: string(out),
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			bashResult.ExitCode = exitErr.ExitCode()
		} else {
			bashResult.ExitCode = 1
		}
		bashResult.Error = err.Error()
	} else {
		bashResult.ExitCode = 0
	}

	s.sendSuccess(cmd.ID, cmd.Type, bashResult)
}

func (s *Server) handleGetMessages(cmd RpcCommand) {
	if s.agent == nil {
		s.sendError(cmd.ID, cmd.Type, "Agent not initialized")
		return
	}

	s.sendSuccess(cmd.ID, cmd.Type, map[string]interface{}{
		"messages": s.agent.GetState().GetMessages(),
	})
}

func (s *Server) handleGetSessionStats(cmd RpcCommand) {
	if s.sessionManager == nil {
		s.sendError(cmd.ID, cmd.Type, "Session manager not available")
		return
	}

	stats := SessionStats{
		MessageCount: 0,
		TotalTokens:  0,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// 从会话获取统计信息
	if current := s.sessionManager.GetCurrent(); current != nil {
		stats.MessageCount = len(current.State.GetMessages())
		stats.CreatedAt = current.Metadata.CreatedAt
		stats.UpdatedAt = current.Metadata.UpdatedAt
	}

	s.sendSuccess(cmd.ID, cmd.Type, stats)
}

func (s *Server) sendSuccess(id, command string, data interface{}) {
	res := RpcResponse{
		ID:      id,
		Type:    "response",
		Command: command,
		Success: true,
		Data:    data,
	}

	s.sendJSON(res)
}

func (s *Server) sendError(id, command string, errMsg string) {
	res := RpcResponse{
		ID:      id,
		Type:    "response",
		Command: command,
		Success: false,
		Error:   errMsg,
	}

	s.sendJSON(res)
}

func (s *Server) sendEvent(event RpcEvent) {
	s.sendJSON(event)
}

func (s *Server) sendJSON(data interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	jsonData, err := json.Marshal(data)
	if err != nil {
		msg := fmt.Sprintf(`{"type":"error","timestamp":"%s","data":"%s"}\n`,
			time.Now().Format(time.RFC3339),
			"Failed to marshal response")
		fmt.Fprint(s.writer, msg)
		return
	}

	fmt.Fprintln(s.writer, string(jsonData))
}
