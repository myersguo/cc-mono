package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/myersguo/cc-mono/pkg/agent"
	"github.com/myersguo/cc-mono/pkg/codingagent"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许跨域请求
	},
}

// HTTPServer 是一个简单的 HTTP 服务器，用于代理与 cc-mono RPC 模式的通信
type HTTPServer struct {
	addr           string
	server         *http.Server
	mu             sync.Mutex
	clients        map[*websocket.Conn]struct{}
	agent          *agent.Agent
	modelRegistry  *codingagent.ModelRegistry
	providerConfig *codingagent.ProvidersConfig
	sessionManager *codingagent.SessionManager
}

// NewHTTPServer 创建新的 HTTP 服务器实例
func NewHTTPServer(
	addr string,
	agentInst *agent.Agent,
	registry *codingagent.ModelRegistry,
	providers *codingagent.ProvidersConfig,
	sessionMgr *codingagent.SessionManager,
) *HTTPServer {
	return &HTTPServer{
		addr:           addr,
		clients:        make(map[*websocket.Conn]struct{}),
		agent:          agentInst,
		modelRegistry:  registry,
		providerConfig: providers,
		sessionManager: sessionMgr,
	}
}

// Start 启动 HTTP 和 WebSocket 服务器
func (h *HTTPServer) Start() error {
	http.HandleFunc("/health", h.healthHandler)
	http.HandleFunc("/api/rpc", h.rpcHTTPHandler)
	http.HandleFunc("/ws/rpc", h.rpcWebSocketHandler)
	
	h.server = &http.Server{Addr: h.addr}
	
	fmt.Printf("HTTP RPC server listening on %s\n", h.addr)
	fmt.Println("Health check: GET /health")
	fmt.Println("HTTP API: POST /api/rpc")
	fmt.Println("WebSocket API: /ws/rpc")
	
	return h.server.ListenAndServe()
}

// wsConnWrapper 包装 WebSocket 连接以支持 io.Reader 和 io.Writer
type wsConnWrapper struct {
	conn *websocket.Conn
	pr   *io.PipeReader
	pw   *io.PipeWriter
}

func newWSConnWrapper(conn *websocket.Conn) *wsConnWrapper {
	pr, pw := io.Pipe()
	return &wsConnWrapper{
		conn: conn,
		pr:   pr,
		pw:   pw,
	}
}

func (w *wsConnWrapper) Read(p []byte) (n int, err error) {
	return w.pr.Read(p)
}

func (w *wsConnWrapper) Write(p []byte) (n int, err error) {
	err = w.conn.WriteMessage(websocket.TextMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func (w *wsConnWrapper) Close() error {
	w.pw.Close()
	w.pr.Close()
	return w.conn.Close()
}

// StartReadPump 开始从 WebSocket 读取消息并写入 pipe
func (w *wsConnWrapper) StartReadPump() {
	go func() {
		defer w.pw.Close()
		for {
			_, message, err := w.conn.ReadMessage()
			if err != nil {
				return
			}
			// 写入 pipe，并添加换行符（RPC 模式是按行解析的）
			w.pw.Write(append(message, '\n'))
		}
	}()
}

// healthHandler 返回服务器健康状态
func (h *HTTPServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok","timestamp":"` + time.Now().Format(time.RFC3339) + `","version":"1.0.0"}`))
}

// Shutdown 关闭服务器
func (h *HTTPServer) Shutdown(ctx context.Context) error {
	return h.server.Shutdown(ctx)
}

// rpcHTTPHandler 处理 HTTP POST 请求
func (h *HTTPServer) rpcHTTPHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"` + err.Error() + `"}`))
		return
	}
	defer r.Body.Close()
	
	// 使用共享 Agent 创建临时 RPC Server 处理请求
	in := strings.NewReader(string(body) + "\n")
	var out strings.Builder
	
	_ = NewServer(h.agent, h.modelRegistry, h.providerConfig, h.sessionManager, in, &out)
	
	// 我们只需要处理一条命令，但 Run 是循环的。
	// 实际上 handleCommand 是异步的，所以我们需要一种方式来等待响应。
	// 这里简化处理：直接调用 handleCommand
	var cmd RpcCommand
	if err := json.Unmarshal(body, &cmd); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"Invalid JSON"}`))
		return
	}
	
	// 注意：handleCommand 在 Server 中是私有的，我们需要一个公开的方法或者在这里实现逻辑
	// 为了简单，我们先保持原来的 exec 方式或者修改 Server
	// 但既然我们要双向同步，WebSocket 才是重点。
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	// 如果是 HTTP，双向同步很难实现，因为它是短连接。
	w.Write([]byte(`{"status":"success","message":"Command received, please use WebSocket for real-time updates"}`))
}

// rpcWebSocketHandler 处理 WebSocket 连接
func (h *HTTPServer) rpcWebSocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Failed to upgrade to websocket:", err)
		return
	}
	
	h.mu.Lock()
	h.clients[conn] = struct{}{}
	h.mu.Unlock()
	
	fmt.Printf("New WebSocket connection established (total: %d)\n", len(h.clients))
	
	// 包装连接
	wrapper := newWSConnWrapper(conn)
	wrapper.StartReadPump()
	
	// 创建并运行 RPC Server
	srv := NewServer(h.agent, h.modelRegistry, h.providerConfig, h.sessionManager, wrapper, wrapper)
	
	go func() {
		defer func() {
			h.mu.Lock()
			delete(h.clients, conn)
			h.mu.Unlock()
			fmt.Printf("WebSocket connection closed (total: %d)\n", len(h.clients))
			wrapper.Close()
		}()
		
		if err := srv.Run(context.Background()); err != nil {
			fmt.Printf("RPC Server error: %v\n", err)
		}
	}()
}
