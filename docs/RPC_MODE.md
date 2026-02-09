# CC-Mono RPC 模式使用说明

## 概述

CC-Mono 现在支持 RPC（远程过程调用）模式，允许外部程序（如 Web 应用、桌面应用或其他系统）通过标准化的 JSON 协议与 CC-Mono 的 AI 代理进行通信。这为 CC-Mono 的功能扩展提供了强大的接口。

## 什么是 RPC 模式

在 RPC 模式下，CC-Mono 会以服务器的形式运行，通过标准输入/输出（stdin/stdout）使用 JSON 格式的 RPC 命令进行通信。这提供了轻量级且高效的通信方式。

## 功能特性

- 完整的 API，支持所有 CC-Mono 核心功能
- 类型安全的通信协议
- 使用标准输入输出，无需额外网络端口
- 支持流式事件通知
- 响应式和双向通信
- 包含完整的错误处理

## 快速开始

### 1. 启动 RPC 服务器

通过命令行启动 CC-Mono 的 RPC 模式：

```bash
# 基本启动
./cc chat --mode rpc

# 使用自定义配置
./cc chat --mode rpc --config ~/.cc-mono --model gpt-4o --dir /path/to/workspace

# 后台运行示例
./cc chat --mode rpc 2>/dev/null
```

### 2. 使用 Go 客户端（推荐）

我们提供了官方的 Go 客户端库：

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"
    "time"

    "github.com/myersguo/cc-mono/pkg/rpc"
)

func main() {
    // 创建 RPC 客户端
    client, err := rpc.NewClient("./bin/cc")
    if err != nil {
        log.Fatalf("Error creating client: %v", err)
    }
    defer client.Close()

    fmt.Println("Connected to CC-Mono RPC server")

    // 发送命令并接收响应
    commands := []rpc.RpcCommand{
        {
            ID:   "1",
            Type: "get_state",
        },
        {
            ID:      "2",
            Type:    "bash",
            Command: "pwd; ls -la",
        },
    }

    for _, cmd := range commands {
        if err := client.SendCommand(cmd); err != nil {
            fmt.Printf("Error sending command '%s': %v\n", cmd.Type, err)
            continue
        }
        
        // 接收响应
        resp, err := client.ReadResponse()
        if err != nil {
            fmt.Printf("Error reading response for '%s': %v\n", cmd.Type, err)
            continue
        }
        
        fmt.Printf("\n=== Response to command '%s' ===\n", cmd.Type)
        fmt.Println(string(resp))
    }
}
```

### 3. 使用简单的 Python 客户端

您可以使用任何支持 JSON 和标准输入/输出的语言与 CC-Mono 通信。以下是 Python 示例：

```python
import json
import subprocess

# 启动 CC-Mono RPC 服务器
proc = subprocess.Popen(
    ["./bin/cc", "chat", "--mode", "rpc"],
    stdin=subprocess.PIPE,
    stdout=subprocess.PIPE,
    stderr=subprocess.PIPE,
    text=True
)

# 定义命令
commands = [
    {
        "id": "test-state",
        "type": "get_state"
    },
    {
        "id": "test-pwd",
        "type": "bash",
        "command": "pwd"
    },
    {
        "id": "test-available-models",
        "type": "get_available_models"
    }
]

# 发送所有命令
for cmd in commands:
    print(f"Sending command: {cmd['id']} ({cmd['type']})")
    line = json.dumps(cmd, ensure_ascii=False) + "\n"
    proc.stdin.write(line)
    proc.stdin.flush()
    
# 读取响应
print("\n=== Server responses ===")
while True:
    if proc.poll() is not None:
        print(f"Process exited with code {proc.poll()}")
        break
    
    line = proc.stdout.readline()
    if line:
        try:
            data = json.loads(line.strip())
            print(json.dumps(data, indent=4, ensure_ascii=False))
        except:
            print(line.strip())
```

## 通信协议

### 命令格式

RPC 命令使用 JSON 格式，发送到 CC-Mono 的标准输入。每个命令必须包含 `id` 和 `type` 字段。

#### 基本命令示例：

```json
{
  "id": "1",
  "type": "prompt",
  "message": "Hello, how are you?"
}
```

```json
{
  "id": "2",
  "type": "bash",
  "command": "ls -la"
}
```

### 支持的命令类型

#### 核心功能

- **prompt**: 向 AI 发送用户提示（需要 message 字段）
- **steer**: 指导 AI 在多轮对话中的行为（需要 message 字段）
- **follow_up**: 发送后续问题（需要 message 字段）
- **abort**: 停止正在处理的请求

#### 会话管理

- **get_state**: 获取当前会话状态
- **new_session**: 创建新会话
- **get_messages**: 获取消息历史

#### 配置

- **set_model**: 设置当前使用的模型
- **cycle_model**: 切换模型（无参数）
- **get_available_models**: 获取可用模型列表（无参数）

#### 工具调用

- **bash**: 执行 bash 命令（需要 command 字段）
- **abort_bash**: 停止当前的 bash 命令

#### 信息获取

- **get_session_stats**: 获取会话统计信息

### 响应格式

服务器响应包含请求的 ID，以方便关联。

#### 成功响应：

```json
{
  "id": "1",
  "type": "response",
  "command": "get_state",
  "success": true,
  "data": {
    "system_prompt": "",
    "model": "gpt-4o",
    "thinking_level": "medium",
    "messages": []
  }
}
```

#### 错误响应：

```json
{
  "id": "3",
  "type": "response",
  "command": "bash",
  "success": false,
  "error": "Error executing bash command"
}
```

## 事件流

服务器会通过标准输出发送实时事件，包括：
- `agent_start`: 代理启动
- `agent_end`: 代理停止
- `turn_start` / `turn_end`: 对话回合开始/结束
- `message_update`: 消息更新
- `tool_call` / `tool_result`: 工具调用/结果
- `error`: 错误事件

## Web UI 集成

CC-Mono 的 Web UI 已与 RPC 服务器配合使用。您可以：

1. 启动 RPC 服务器：
   ```bash
   ./bin/cc chat --mode rpc
   ```

2. 在 web-ui 目录中，启动开发服务器：
   ```bash
   cd web-ui && npm install && npm run dev
   ```

3. Web UI 会通过 RPC 与 CC-Mono 通信，提供完整的用户界面。

## 构建和运行

### 构建项目

```bash
cd /path/to/cc-mono

# 构建主程序
go build -o bin/cc ./cmd/cc

# 确保所有依赖已同步
go work sync

# 运行测试客户端以验证 RPC 功能
go run test_rpc_client.go
```

### 使用工作区

项目使用 Go 工作区管理依赖。所有子模块已正确配置。

## 架构

### RPC 服务器

RPC 服务器代码位于 `pkg/rpc/` 目录：

- **server.go**: 主 RPC 服务器实现
- **types.go**: RPC 类型定义
- **rpc_client.go**: 官方 Go 客户端库

### 架构层次

CC-Mono 遵循清晰的架构分层：

1. **应用层 (Coding Agent)**: 会话、工具、扩展管理
2. **运行时 (Agent Runtime)**: 状态、事件系统、工具执行
3. **LLM 层**: 提供者接口、流式处理
4. **通信层**: RPC 协议、序列化

## 注意事项和限制

1. RPC 模式使用标准输入输出，因此需要确保没有其他进程干扰。
2. 在后台运行时，需要确保有适当的资源限制。
3. 所有命令都需要包含有效的 ID，以确保响应关联。
4. 通信是单行协议 - 每个 JSON 必须单独成行。
5. 某些复杂的工具调用可能需要额外权限确认。

## 高级用法

### 自定义数据类型

对于特定场景，您可以自定义输入输出：

```go
// 发送自定义命令
customCmd := rpc.RpcCommand{
    ID: "custom",
    Type: "custom_command",
}

// 使用扩展字段
customCmdWithData := map[string]interface{}{
    "id": "with-data",
    "type": "prompt",
    "message": "Analyze this file",
    "options": map[string]interface{}{
        "temperature": 0.7,
        "system_prompt": "You are a code analyzer"
    }
}
```

### 事件处理

通过事件流获取实时更新：

```go
client, err := rpc.NewClient("./bin/cc")
if err != nil {
    log.Fatal(err)
}
defer client.Close()

for {
    resp, err := client.ReadResponse()
    if err != nil {
        log.Println(err)
        break
    }
    
    // 解析事件类型
    var event map[string]interface{}
    if err := json.Unmarshal(resp, &event); err == nil {
        if eventType, ok := event["type"].(string); ok {
            switch eventType {
            case "message_update":
                // Handle message updates
            case "agent_start":
                // Handle agent startup
            }
        }
    }
}
```

## 开发和调试

### 启用详细日志

使用 `--verbose` 标志来获得调试输出：

```bash
./cc chat --mode rpc --verbose
```

### 开发模式

使用 `go run` 直接调试：

```bash
cd /path/to/cc-mono && go run ./cmd/cc chat --mode rpc
```

### 日志记录

所有通信都可以通过标准错误流（stderr）获得：

```bash
./cc chat --mode rpc 2> cc_rpc.log
```

---

**RPC 模式已成功添加到 CC-Mono！**

现在您可以使用任何支持标准输入/输出和 JSON 的语言来与 CC-Mono 集成，创建强大的自动化和开发工具。
