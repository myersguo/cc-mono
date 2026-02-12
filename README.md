# CC - Coding Agent

<div align="center">
<img width="2418" height="860" alt="image" src="https://github.com/user-attachments/assets/1cade535-50f2-4a17-83b7-c003ef311263" />

**An AI-powered coding assistant in your terminal**

Interactive TUI • Multi-LLM Support • Tool System • Session Management

[Quick Start](#quick-start) · [Features](#features) · [Configuration](#configuration) · [Documentation](#documentation)

</div>

---

## What is CC?

CC-Mono is a terminal-based AI coding assistant that helps you with software development tasks. It provides an interactive chat interface with powerful tools for reading, writing, and editing code, running commands, and managing your workflow.

Built with Go for speed and reliability, CC-Mono supports multiple LLM providers and features a clean architecture that's easy to extend.

## Features

✨ **Interactive TUI** - Beautiful terminal interface built with Bubbletea  
🤖 **Multi-LLM Support** - OpenAI, Google Gemini, Anthropic Claude, and any OpenAI-compatible API  
🛠️ **Built-in Tools** - Read, write, edit files, and execute bash commands  
💾 **Session Management** - Save, load, and fork conversation sessions  
🔐 **Permission System** - Granular control over tool execution  
📜 **Command History** - Persistent input history across sessions  
🎨 **Customizable** - Themes, extensions, and plugin system  
⚡ **Fast & Lightweight** - Single binary, minimal dependencies  

## Quick Start

### Installation

```bash
# Clone the repository
git clone https://github.com/myersguo/cc-mono.git
cd cc-mono

# Build the binary
go build -o cc ./cmd/cc

# Move to PATH (optional)
sudo mv cc /usr/local/bin/
```

### Setup

1. **Set up your API key** (choose one):

```bash
# OpenAI
export OPENAI_API_KEY="sk-..."

# Google Gemini
export GOOGLE_API_KEY="..."

# Anthropic Claude
export ANTHROPIC_API_KEY="sk-ant-..."
```

2. **Run CC-Mono**:

```bash
./cc
```

That's it! You're ready to start chatting with your AI assistant.

## Usage

### Interactive Chat

Start a conversation:

```bash
./cc chat
```

**Keyboard shortcuts:**

- `Enter` - Send message
- `Ctrl+J` - New line in message
- `↑/↓` - Browse command history
- `Ctrl+C` - Quit
- `Ctrl+R` - Regenerate last response
- `Ctrl+K/J` - Scroll messages
- `Esc` - Clear input

### Example Conversation

```
> Read the README.md file

I'll read the README.md file for you.

[Tool: Read(README.md)]
...

> Create a new function to validate email addresses

I'll create an email validation function in utils.go:

[Tool: Write(utils/validate.go)]
...

> Run the tests

[Tool: Bash(go test ./...)]
PASS
ok      github.com/myersguo/cc-mono/pkg/ai      0.123s
...
```

### Available Models

List all configured models:

```bash
./cc model list
```

Use a specific model:

```bash
./cc --model gpt-4o
./cc --model claude-sonnet-4-5
./cc --model gemini-2.0-flash-exp
```

### Session Management

```bash
# List sessions
./cc session list

# Delete a session
./cc session delete <session-id>
```

## Configuration

CC-Mono uses JSON configuration files stored in `~/.cc-mono/`.

### Provider Configuration

Create `~/.cc-mono/providers.json`:

```json
{
  "default_provider": "google",
  "providers": {
    "openai": {
      "api_key": "${OPENAI_API_KEY}",
      "base_url": "https://api.openai.com/v1",
      "default_model": "gpt-4o"
    },
    "google": {
      "api_key": "${GOOGLE_API_KEY}",
      "default_model": "gemini-2.0-flash-exp"
    },
    "anthropic": {
      "api_key": "${ANTHROPIC_API_KEY}",
      "default_model": "claude-sonnet-4-5"
    }
  }
}
```

**Pro tip:** Use `${VAR_NAME}` to reference environment variables instead of hardcoding API keys.

### Model Configuration

Create `~/.cc-mono/models.json`:

```json
{
  "models": [
    {
      "id": "gpt-4o",
      "provider": "openai",
      "name": "GPT-4o",
      "context_window": 128000,
      "max_output": 16384,
      "input_cost_per_million": 2.5,
      "output_cost_per_million": 10.0,
      "supports_vision": true,
      "supports_tools": true
    }
  ]
}
```

Or use the provided defaults:

```bash
cp configs/models.json ~/.cc-mono/
cp configs/providers.json ~/.cc-mono/
```

## Advanced Features

### Custom Providers (DeepSeek, Qwen, Local LLMs)

CC-Mono supports any OpenAI-compatible API:

```json
{
  "providers": {
    "deepseek": {
      "api_key": "${DEEPSEEK_API_KEY}",
      "base_url": "https://api.deepseek.com/v1",
      "default_model": "deepseek-chat"
    },
    "ollama": {
      "api_key": "ollama",
      "base_url": "http://localhost:11434/v1",
      "default_model": "llama2"
    }
  }
}
```

### Permission Management

Control which tools can execute without asking:

```json
{
  "permissions": {
    "allow": [
      "Read(*)",
      "Bash(git:*)",
      "Bash(go test:*)"
    ],
    "deny": [
      "Bash(rm -rf:*)"
    ]
  }
}
```

Save to `~/.cc-mono/settings.json` for global permissions, or `./.cc-mono/settings.local.json` for project-specific rules.

### Command History

All your inputs are saved to `~/.cc-mono/history` and shared across sessions:

- Use `↑/↓` to browse history
- Supports multi-line commands
- Automatically persisted on exit
- Configurable size limit (default: 1000 entries)

### Using as a Library

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/myersguo/cc-mono/pkg/ai"
    "github.com/myersguo/cc-mono/pkg/ai/providers/openai"
)

func main() {
    // Create provider
    provider, _ := openai.NewProvider(openai.Config{
        APIKey: os.Getenv("OPENAI_API_KEY"),
    })

    // Create request
    ctx := context.Background()
    model := ai.Model{ID: "gpt-4o", Provider: "openai"}
    aiContext := ai.NewContext("You are a helpful assistant", []ai.Message{
        ai.NewUserTextMessage("Hello!"),
    })

    // Stream response
    stream := provider.Stream(ctx, model, aiContext, nil)
    for event := range stream.Events() {
        if event.Type == ai.EventTypeContentDelta {
            fmt.Print(event.TextDelta)
        }
    }
}
```

## Architecture

CC-Mono follows a clean 3-layer architecture:

```
┌─────────────────────────────────────┐
│  Coding Agent (Application Layer)  │  Session, Tools, Extensions
│  pkg/codingagent/                   │
└─────────────────────────────────────┘
              ↓
┌─────────────────────────────────────┐
│  Agent Runtime (Runtime Layer)      │  State, Events, Tool Execution
│  pkg/agent/                         │
└─────────────────────────────────────┘
              ↓
┌─────────────────────────────────────┐
│  AI Layer (LLM Layer)               │  Provider Interface, Streaming
│  pkg/ai/                            │
└─────────────────────────────────────┘
```

**Technology Stack:**
- **Go Workspace** - Monorepo structure
- **TUI** - [Bubbletea](https://github.com/charmbracelet/bubbletea) + [Lipgloss](https://github.com/charmbracelet/lipgloss)
- **CLI** - [Cobra](https://github.com/spf13/cobra)
- **Streaming** - Server-Sent Events (SSE)
- **Plugins** - [HashiCorp go-plugin](https://github.com/hashicorp/go-plugin)

## Command-Line Reference

```bash
# Global flags
--config <dir>         Configuration directory (default: ~/.cc-mono)
--models <path>        Path to models.json
--providers <path>     Path to providers.json
--model <id>           Model ID to use
--provider <name>      Provider to use
--theme <name>         TUI theme: dark/light
--dir <path>           Working directory
--extensions <list>    Extensions to load (comma-separated)
-v, --verbose          Verbose output

# Commands
cc                     Start interactive chat (default)
cc chat                Start interactive chat
cc model list          List available models
cc session list        List chat sessions
cc session delete <id> Delete a session
cc extension list      List available extensions
cc version             Show version
cc help                Show help
```

## Documentation

- [Phase Summaries](docs/)

## Project Structure

```
cc-mono/
├── cmd/cc/              # CLI executable
├── pkg/                 # Core libraries
│   ├── ai/              # LLM providers and streaming
│   ├── agent/           # Agent runtime and event system
│   ├── codingagent/     # Application layer
│   └── shared/          # Shared utilities
├── internal/tui/        # Terminal UI components
├── configs/             # Default configuration files
└── docs/                # Documentation
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see [LICENSE](LICENSE) file for details.

---

<div align="center">

**Made with ❤️ using Go**

[Report Bug](https://github.com/myersguo/cc-mono/issues) · [Request Feature](https://github.com/myersguo/cc-mono/issues)

</div>
