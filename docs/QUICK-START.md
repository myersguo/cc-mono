# Quick Start Guide

This guide will help you get CC-Mono up and running in 5 minutes.

## Step 1: Build the Binary

```bash
cd /path/to/cc-mono/cmd/cc
go build -o cc
```

## Step 2: Set Up Configuration

### Option A: Use Default Location (~/.cc-mono)

```bash
# Create config directory
mkdir -p ~/.cc-mono

# Copy configuration files
cp /path/to/cc-mono/configs/models.json ~/.cc-mono/
cp /path/to/cc-mono/configs/providers.json ~/.cc-mono/

# Set your API key
export OPENAI_API_KEY=sk-your-api-key-here

# Run cc (it will use ~/.cc-mono by default)
./cc
```

### Option B: Use Custom Configuration Directory

```bash
# Create custom config directory
mkdir -p ~/my-configs

# Copy configuration files
cp /path/to/cc-mono/configs/models.json ~/my-configs/
cp /path/to/cc-mono/configs/providers.json ~/my-configs/

# Set your API key
export OPENAI_API_KEY=sk-your-api-key-here

# Run cc with custom config
./cc --config ~/my-configs
```

### Option C: Specify Full Paths

```bash
# Set your API key
export OPENAI_API_KEY=sk-your-api-key-here

# Run with full paths
./cc --models /path/to/models.json --providers /path/to/providers.json
```

## Step 3: Verify Configuration

Test that your configuration is working:

```bash
# List available models
./cc model list

# Expected output:
# Available models (3):
#   - gpt-4-turbo
#   - gemini-2.0-flash-exp
#   - claude-opus-4-5

# List with details
./cc model list -v
```

## Step 4: Start Chatting!

```bash
# Start interactive chat
./cc

# Or with specific model
./cc --model gpt-4-turbo

# Or with extensions
./cc --extensions logger,time-tracker
```

## Configuration File Details

### providers.json Structure

```json
{
  "default_provider": "openai",
  "providers": {
    "openai": {
      "base_url": "https://api.openai.com/v1",
      "api_key": "${OPENAI_API_KEY}",
      "default_model": "gpt-4-turbo"
    }
  }
}
```

**Key Points:**
- `${OPENAI_API_KEY}` will be replaced with the environment variable value
- `base_url` can be changed to use OpenAI-compatible services (DeepSeek, Ollama, etc.)
- `default_model` is used when no `--model` flag is specified

### models.json Structure

```json
{
  "models": [
    {
      "id": "gpt-4-turbo",
      "provider": "openai",
      "name": "GPT-4 Turbo",
      "context_window": 128000,
      "max_output": 4096,
      "input_cost_per_million": 10.0,
      "output_cost_per_million": 30.0,
      "supports_vision": true,
      "supports_tools": true
    }
  ]
}
```

**Key Points:**
- `id` must match the `default_model` in providers.json
- `provider` must match a provider name in providers.json
- Cost fields are in USD per million tokens

## Troubleshooting

### Error: "failed to load providers config"

**Problem:**
```
Error: failed to load providers config: failed to read providers config:
open configs/providers.json: no such file or directory
```

**Solution:**

The `--config` flag should point to a directory containing both `models.json` and `providers.json`:

```bash
# Wrong (looking for configs/providers.json relative to current directory)
./cc --config ~/configs

# Fix: Make sure files exist in the config directory
ls ~/configs/
# Should show: models.json  providers.json

# Or use absolute paths
./cc --models ~/configs/models.json --providers ~/configs/providers.json
```

### Error: "model not found"

**Problem:**
```
Error: failed to get model: model not found: my-model
```

**Solution:**

Check that the model exists in `models.json`:

```bash
./cc model list
```

Make sure the `--model` value matches a model `id` in the list.

### Error: "API key is required"

**Problem:**
```
Error: failed to create provider: API key is required
```

**Solution:**

Set the appropriate environment variable:

```bash
# For OpenAI
export OPENAI_API_KEY=sk-...

# Verify it's set
echo $OPENAI_API_KEY
```

Or edit `providers.json` to hardcode the key (not recommended for security):

```json
{
  "providers": {
    "openai": {
      "api_key": "sk-your-actual-key-here"
    }
  }
}
```

### Error: "provider not found in config"

**Problem:**
```
Error: provider openai not found in config
```

**Solution:**

Make sure `providers.json` includes the provider:

```bash
# Check providers.json content
cat ~/.cc-mono/providers.json
```

The file should have a `providers` object with the provider name as a key.

## Using Different Providers

### OpenAI

```bash
export OPENAI_API_KEY=sk-...
./cc --provider openai --model gpt-4-turbo
```

### DeepSeek

Add to `providers.json`:

```json
{
  "providers": {
    "deepseek": {
      "base_url": "https://api.deepseek.com/v1",
      "api_key": "${DEEPSEEK_API_KEY}",
      "default_model": "deepseek-chat"
    }
  }
}
```

Then:

```bash
export DEEPSEEK_API_KEY=sk-...
./cc --provider deepseek --model deepseek-chat
```

### Local Ollama

Add to `providers.json`:

```json
{
  "providers": {
    "ollama": {
      "base_url": "http://localhost:11434/v1",
      "api_key": "ollama",
      "default_model": "llama2"
    }
  }
}
```

Then:

```bash
./cc --provider ollama --model llama2
```

## Using Extensions

```bash
# List available extensions
./cc extension list -v

# Output:
# Available extensions (3):
#   - logger (v1.0.0)
#     Description: Logs all tool calls and results for debugging
#   - time-tracker (v1.0.0)
#     Description: Tracks execution time of tool calls
#   - content-filter (v1.0.0)
#     Description: Filters sensitive content from tool results

# Load one extension
./cc --extensions logger

# Load multiple extensions
./cc --extensions logger,time-tracker

# With verbose logging
./cc --extensions logger --verbose
```

## Next Steps

- Read the [CLI Reference](CLI.md) for complete command documentation
- Learn about [Extensions](EXTENSIONS.md) to customize behavior
- Check the [Architecture Guide](ARCHITECTURE.md) to understand the system design

## Common Workflows

### Development Workflow

```bash
# Build
cd cmd/cc && go build -o cc

# Test configuration
./cc model list

# Start coding session
./cc --extensions logger,time-tracker

# In the chat, you can:
# - Ask to read files
# - Request file modifications
# - Execute bash commands
# - Get coding assistance
```

### Using as a Library

See the main [README.md](../README.md) for library usage examples.

## Getting Help

```bash
# Show help
./cc --help

# Show help for a command
./cc chat --help
./cc model --help
./cc session --help
```
