# MIXOS GO API Reference

## Overview

The MIXOS AI Agent exposes a REST API for programmatic interaction. The API server runs on `localhost:8765` by default.

## Base URL

```
http://localhost:8765/api
```

## Authentication

Currently, the API is accessible only from localhost without authentication. Future versions will support API keys.

## Endpoints

### Health Check

#### GET /api/health

Check if the agent is running.

**Response:**
```json
{
  "status": "ok"
}
```

### Status

#### GET /api/status

Get detailed agent status.

**Response:**
```json
{
  "status": "running",
  "model": "mix-small-1.1b-q4.gguf",
  "uptime": "2h 30m",
  "memory_usage": "1.2 GB",
  "requests_ok": 150
}
```

### Chat

#### POST /api/chat

Send a message to the agent.

**Request:**
```json
{
  "message": "Install Docker",
  "session_id": "optional-session-id",
  "stream": false
}
```

**Response:**
```json
{
  "response": "I'll help you install Docker...",
  "session_id": "abc123",
  "error": null
}
```

#### POST /api/chat/stream

Stream a chat response (Server-Sent Events).

**Request:**
```json
{
  "message": "Explain how containers work",
  "session_id": "optional-session-id"
}
```

**Response (SSE):**
```
data: {"content": "Containers are "}
data: {"content": "lightweight, isolated "}
data: {"content": "environments..."}
data: [DONE]
```

### Execute

#### POST /api/execute

Execute an autonomous task.

**Request:**
```json
{
  "task": "Setup Python development environment",
  "auto_confirm": false
}
```

**Response:**
```json
{
  "task_id": "task_1704672000",
  "status": "completed",
  "message": "Task completed successfully",
  "steps": [
    {
      "name": "Check Python installation",
      "status": "done",
      "output": "Python 3.11.5"
    },
    {
      "name": "Create virtual environment",
      "status": "done",
      "output": "Created venv at ~/venv"
    }
  ],
  "error": null
}
```

#### GET /api/execute/{task_id}

Get task execution status.

**Response:**
```json
{
  "task_id": "task_1704672000",
  "status": "in_progress",
  "message": "Installing packages...",
  "steps": [...],
  "error": null
}
```

#### POST /api/execute/{task_id}/cancel

Cancel a running task.

**Response:**
```json
{
  "status": "cancelled",
  "task_id": "task_1704672000"
}
```

### Tools

#### GET /api/tools

List available tools.

**Response:**
```json
[
  {
    "name": "install_package",
    "description": "Install a package using mix-pkg",
    "parameters": [
      {
        "name": "package_name",
        "type": "string",
        "description": "Name of the package",
        "required": true
      }
    ],
    "safety_level": "requires_confirmation"
  },
  ...
]
```

#### GET /api/tools/{tool_name}

Get information about a specific tool.

**Response:**
```json
{
  "name": "install_package",
  "description": "Install a package using mix-pkg",
  "parameters": [...],
  "safety_level": "requires_confirmation"
}
```

#### POST /api/tools/{tool_name}/execute

Execute a tool directly.

**Request:**
```json
{
  "package_name": "nginx"
}
```

**Response:**
```json
{
  "result": "Successfully installed nginx"
}
```

### Sessions

#### GET /api/sessions

List all sessions.

**Response:**
```json
[
  {
    "id": "abc123",
    "created_at": "2024-01-08T10:00:00Z",
    "updated_at": "2024-01-08T10:30:00Z",
    "message_count": 15
  }
]
```

#### GET /api/sessions/{session_id}

Get session information.

**Response:**
```json
{
  "id": "abc123",
  "created_at": "2024-01-08T10:00:00Z",
  "updated_at": "2024-01-08T10:30:00Z",
  "message_count": 15
}
```

#### GET /api/sessions/{session_id}/history

Get session conversation history.

**Query Parameters:**
- `limit` (optional): Maximum messages to return (default: 50)

**Response:**
```json
{
  "history": [
    {
      "role": "user",
      "content": "Install Docker",
      "timestamp": 1704672000
    },
    {
      "role": "assistant",
      "content": "I'll help you install Docker...",
      "timestamp": 1704672001
    }
  ]
}
```

#### DELETE /api/sessions/{session_id}

Delete a session.

**Response:**
```json
{
  "status": "deleted",
  "session_id": "abc123"
}
```

### Configuration

#### GET /api/config

Get agent configuration.

**Response:**
```json
{
  "agent": {
    "name": "Mix Agent",
    "version": "1.0.0"
  },
  "model": {
    "path": "/opt/mixos/ai/model/mix-small-1.1b-q4.gguf",
    "context_length": 4096
  },
  "inference": {
    "threads": 4
  }
}
```

#### PATCH /api/config

Update agent configuration.

**Request:**
```json
{
  "model": {
    "temperature": 0.8
  }
}
```

**Response:**
```json
{
  "status": "updated"
}
```

### Metrics

#### GET /api/health/metrics

Get agent metrics.

**Response:**
```json
{
  "requests_total": 1500,
  "requests_success": 1480,
  "requests_failed": 20,
  "average_response_time_ms": 250,
  "active_sessions": 5,
  "memory_usage_bytes": 1288490188,
  "cpu_usage_percent": 15.5
}
```

## WebSocket API

### Connection

```
ws://localhost:8765/ws
```

### Message Types

#### Chat Message

**Send:**
```json
{
  "type": "chat",
  "message": "Hello",
  "session_id": "optional"
}
```

**Receive:**
```json
{
  "type": "chat_response",
  "content": "Hello! How can I help?",
  "session_id": "abc123"
}
```

#### Streaming Chat

**Send:**
```json
{
  "type": "chat_stream",
  "message": "Explain Docker",
  "session_id": "optional"
}
```

**Receive (multiple):**
```json
{"type": "chat_stream", "content": "Docker is "}
{"type": "chat_stream", "content": "a platform for "}
{"type": "chat_stream_end", "session_id": "abc123"}
```

#### Execute Task

**Send:**
```json
{
  "type": "execute",
  "task": "Install nginx",
  "auto_confirm": true
}
```

**Receive (multiple):**
```json
{"type": "execute_plan", "task_id": "task_123", "steps": [...]}
{"type": "execute_step", "step": "Installing nginx", "status": "running"}
{"type": "execute_step", "step": "Installing nginx", "status": "done"}
{"type": "execute_complete", "task_id": "task_123", "status": "completed"}
```

## Error Handling

### Error Response Format

```json
{
  "error": {
    "code": "TOOL_NOT_FOUND",
    "message": "Tool 'unknown_tool' not found",
    "details": {}
  }
}
```

### Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `INVALID_REQUEST` | 400 | Malformed request |
| `UNAUTHORIZED` | 401 | Authentication required |
| `FORBIDDEN` | 403 | Operation not allowed |
| `NOT_FOUND` | 404 | Resource not found |
| `TOOL_NOT_FOUND` | 404 | Tool does not exist |
| `SESSION_NOT_FOUND` | 404 | Session does not exist |
| `TASK_NOT_FOUND` | 404 | Task does not exist |
| `VALIDATION_ERROR` | 422 | Invalid parameters |
| `INTERNAL_ERROR` | 500 | Server error |
| `MODEL_ERROR` | 500 | AI model error |
| `EXECUTION_ERROR` | 500 | Tool execution failed |

## Rate Limiting

Currently no rate limiting is enforced for localhost connections.

## Examples

### Python

```python
import requests

BASE_URL = "http://localhost:8765/api"

# Chat
response = requests.post(f"{BASE_URL}/chat", json={
    "message": "Install Docker"
})
print(response.json()["response"])

# Execute task
response = requests.post(f"{BASE_URL}/execute", json={
    "task": "Setup Python environment",
    "auto_confirm": True
})
result = response.json()
print(f"Status: {result['status']}")
for step in result['steps']:
    print(f"  {step['name']}: {step['status']}")
```

### curl

```bash
# Health check
curl http://localhost:8765/api/health

# Chat
curl -X POST http://localhost:8765/api/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello"}'

# Execute task
curl -X POST http://localhost:8765/api/execute \
  -H "Content-Type: application/json" \
  -d '{"task": "Install nginx", "auto_confirm": true}'

# List tools
curl http://localhost:8765/api/tools
```

### JavaScript

```javascript
const BASE_URL = 'http://localhost:8765/api';

// Chat
async function chat(message) {
  const response = await fetch(`${BASE_URL}/chat`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ message })
  });
  return response.json();
}

// WebSocket streaming
const ws = new WebSocket('ws://localhost:8765/ws');
ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log(data);
};
ws.send(JSON.stringify({
  type: 'chat_stream',
  message: 'Explain containers'
}));
```

## SDK

A Python SDK is available:

```python
from mixos_agent.client import AgentClient

client = AgentClient()

# Chat
response = client.chat("Install Docker")
print(response)

# Execute
result = client.execute("Setup Python environment")
print(result.status)

# Stream
for chunk in client.chat_stream("Explain Docker"):
    print(chunk, end="")
```
