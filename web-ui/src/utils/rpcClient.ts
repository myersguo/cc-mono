/**
 * CC-Mono Web UI - RPC Client
 * Real communication with Go server over WebSocket
 */

// RPC 命令类型
export type RpcCommandType = 
  | "prompt"
  | "steer"
  | "follow_up"
  | "abort"
  | "get_state"
  | "new_session"
  | "get_available_models"
  | "set_model"
  | "cycle_model"
  | "bash"
  | "abort_bash"
  | "get_session_stats"
  | "get_messages";

// 会话信息类型
export interface SessionInfo {
  id: string;
  title: string;
  date: string;
  unread?: number;
  favorite?: boolean;
}

// 消息类型
export interface ChatMessage {
  id: string;
  type: 'user' | 'assistant' | 'tool' | 'system';
  content: string;
  timestamp: string;
  isStreaming?: boolean;
}

// 代理消息原始结构
export interface AgentMessage {
  id: string;
  message: {
    type: string;
    content: any[];
    role?: string;
    timestamp?: number;
  };
  created_at: number;
}

// 基础命令接口
interface RpcCommand {
  id: string;
  type: RpcCommandType;
  message?: string;
  command?: string;
}

// 响应类型
interface RpcResponse {
  id: string;
  type: "response";
  command: string;
  success: boolean;
  data: any;
  error?: string;
}

// 事件类型（已注释，未使用时可忽略）
// interface RpcEvent {
//   type: string;
//   timestamp: string;
//   data?: any;
// }

export class RpcClient {
  private ws: WebSocket | null = null;
  private isConnected: boolean = false;
  private messageHandlers: Array<(data: any) => void> = [];
  private errorHandlers: Array<(error: string) => void> = [];
  private connectionHandlers: Array<(connected: boolean) => void> = [];
  
  constructor(private url: string = "ws://localhost:8080/ws/rpc") {
    this.attemptConnection();
  }

  private attemptConnection() {
    console.log(`Connecting to CC-Mono server at ${this.url}...`);

    try {
      this.ws = new WebSocket(this.url);
      
      this.ws.onopen = () => {
        console.log('Connected to CC-Mono server');
        this.isConnected = true;
        this.connectionHandlers.forEach(handler => handler(true));
      };
      
      this.ws.onclose = () => {
        console.log('Disconnected from CC-Mono server');
        this.isConnected = false;
        this.connectionHandlers.forEach(handler => handler(false));
        
        // 自动重连
        setTimeout(() => this.attemptConnection(), 5000);
      };
      
      this.ws.onerror = (event) => {
        console.error('WebSocket error:', event);
      };
      
      this.ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          
          if (data.type === "response") {
            // Find specific promise resolver
            this.handleResponse(data);
          } else {
            // Emit as generic event
            this.messageHandlers.forEach(handler => handler(data));
          }
        } catch (error) {
          console.error('Failed to parse JSON response:', error, event.data);
        }
      };
    } catch (err) {
      console.error('Failed to create WebSocket:', err);
      setTimeout(() => this.attemptConnection(), 5000);
    }
  }

  private responseResolvers = new Map<string, (resp: RpcResponse) => void>();

  private handleResponse(resp: RpcResponse) {
    const resolver = this.responseResolvers.get(resp.id);
    if (resolver) {
      resolver(resp);
      this.responseResolvers.delete(resp.id);
    }
  }

  isOnline() {
    return this.isConnected && this.ws?.readyState === WebSocket.OPEN;
  }

  onMessage(callback: (data: any) => void) {
    this.messageHandlers.push(callback);
  }

  offMessage(callback: (data: any) => void) {
    this.messageHandlers = this.messageHandlers.filter(handler => handler !== callback);
  }

  onError(callback: (error: string) => void) {
    this.errorHandlers.push(callback);
  }

  onConnectionChange(callback: (connected: boolean) => void) {
    this.connectionHandlers.push(callback);
  }

  sendCommand(command: RpcCommand): Promise<RpcResponse> {
    return new Promise((resolve, reject) => {
      if (!this.isOnline()) {
        reject(new Error('Not connected'));
        return;
      }

      const timeout = setTimeout(() => {
        this.responseResolvers.delete(command.id);
        reject(new Error('Request timed out'));
      }, 30000);

      this.responseResolvers.set(command.id, (data: RpcResponse) => {
        clearTimeout(timeout);
        if (data.success) {
          resolve(data);
        } else {
          reject(new Error(data.error));
        }
      });

      try {
        this.ws!.send(JSON.stringify(command));
      } catch (error) {
        clearTimeout(timeout);
        this.responseResolvers.delete(command.id);
        reject(error);
      }
    });
  }

  async sendMessage(text: string): Promise<RpcResponse> {
    const id = "msg-" + Date.now();
    
    return this.sendCommand({
      id,
      type: "prompt",
      message: text
    });
  }

  async getState(): Promise<RpcResponse> {
    return this.sendCommand({
      id: "state-" + Date.now(),
      type: "get_state"
    });
  }

  async getAvailableModels(): Promise<RpcResponse> {
    return this.sendCommand({
      id: "models-" + Date.now(),
      type: "get_available_models"
    });
  }

  async runBashCommand(cmd: string): Promise<RpcResponse> {
    return this.sendCommand({
      id: "bash-" + Date.now(),
      type: "bash",
      command: cmd
    });
  }

  async newSession(): Promise<RpcResponse> {
    return this.sendCommand({
      id: "new-" + Date.now(),
      type: "new_session"
    });
  }
  
  // 获取会话列表
  async getSessions(): Promise<SessionInfo[]> {
    try {
      const resp = await this.getState();
      // 在这个版本中，我们只返回一个主会话
      return [{
        id: "main",
        title: "Main Session",
        date: new Date().toLocaleString(),
        favorite: true
      }];
    } catch (error) {
      console.error('Failed to get sessions:', error);
      return [];
    }
  }
  
  // 获取当前会话消息
  async getCurrentSessionMessages(): Promise<ChatMessage[]> {
    try {
      const resp = await this.sendCommand({
        id: "msgs-" + Date.now(),
        type: "get_messages"
      });
      
      if (resp.success && resp.data.messages) {
        return resp.data.messages.map((m: AgentMessage) => this.mapAgentMessageToChatMessage(m));
      }
      return [];
    } catch (error) {
      console.error('Failed to get messages:', error);
      return [];
    }
  }

  public mapAgentMessageToChatMessage(m: AgentMessage): ChatMessage {
    let content = "";
    if (m.message && m.message.content) {
      content = m.message.content
        .filter((c: any) => c.type === "text")
        .map((c: any) => c.text)
        .join("\n");
        
      // Also handle thinking blocks if any
      const thinking = m.message.content
        .filter((c: any) => c.type === "thinking")
        .map((c: any) => c.thinking)
        .join("\n");
      
      if (thinking) {
        content = `<thinking>\n${thinking}\n</thinking>\n\n${content}`;
      }
    }

    let type = (m.message.role || m.message.type) as any;
    if (type === 'tool_result') type = 'tool';

    return {
      id: m.id,
      type: type,
      content: content,
      timestamp: m.message.timestamp ? new Date(m.message.timestamp).toLocaleTimeString() : new Date(m.created_at).toLocaleTimeString()
    };
  }

  public mapToolCallToChatMessage(event: any): ChatMessage {
    const { tool_name, args, tool_call_id } = event.data;
    return {
      id: `tool-call-${tool_call_id}`,
      type: 'tool',
      content: `🛠️ **Running tool:** \`${tool_name}\`\n\`\`\`json\n${JSON.stringify(args, null, 2)}\n\`\`\``,
      timestamp: new Date().toLocaleTimeString(),
      isStreaming: true // Mark as "in progress"
    };
  }

  public mapToolResultToChatMessage(event: any): ChatMessage {
    const { tool_name, result, is_error, tool_call_id } = event.data;
    let resultStr = "";
    if (result && result.content) {
      resultStr = result.content.map((c: any) => c.text).join("\n");
    } else {
      resultStr = JSON.stringify(result, null, 2);
    }

    return {
      id: `tool-call-${tool_call_id}`, // Use same ID to replace/update the "Running" message
      type: 'tool',
      content: `✅ **Tool completed:** \`${tool_name}\`\n${is_error ? '❌ **Error:**' : '📄 **Output:**'}\n\`\`\`\n${resultStr}\n\`\`\``,
      timestamp: new Date().toLocaleTimeString(),
      isStreaming: false
    };
  }
  
  disconnect() {
    if (this.ws) {
      this.ws.close();
    }
  }
}

let instance: RpcClient | null = null;
export function getRpcClient(): RpcClient {
  if (!instance) {
    instance = new RpcClient();
  }
  return instance;
}

// 简单的 HTTP 客户端（备选）
export class HttpClient {
  private baseURL: string;
  
  constructor(baseURL: string = "http://localhost:8080") {
    this.baseURL = baseURL;
  }
  
  async healthCheck(): Promise<boolean> {
    try {
      const response = await fetch(`${this.baseURL}/health`);
      return response.ok;
    } catch (error) {
      return false;
    }
  }
  
  async sendCommand(command: RpcCommand): Promise<RpcResponse> {
    const response = await fetch(`${this.baseURL}/api/rpc`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json"
      },
      body: JSON.stringify(command)
    });
    
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  }
  
  async sendMessage(text: string): Promise<RpcResponse> {
    return this.sendCommand({
      id: "http-" + Date.now(),
      type: "prompt",
      message: text
    });
  }
}
