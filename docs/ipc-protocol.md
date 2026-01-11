# MixOS IPC Protocol Specification

## Overview

MixOS uses Unix domain sockets for inter-process communication between services. The protocol supports both JSON and Protobuf encoding.

## Socket Location

```
/run/mixos/ipc.sock
```

## Message Format

### Wire Format

```
┌─────────────────────────────────────────┐
│  Length (4 bytes, big-endian uint32)    │
├─────────────────────────────────────────┤
│  Payload (JSON or Protobuf)             │
└─────────────────────────────────────────┘
```

### Message Structure

```json
{
  "version": 1,
  "msg_type": "REQUEST",
  "msg_id": 12345,
  "source": "service_name",
  "target": "target_service",
  "method": "method_name",
  "payload": "...",
  "timestamp": 1704931200000,
  "error": null
}
```

### Fields

| Field | Type | Description |
|-------|------|-------------|
| version | uint32 | Protocol version (currently 1) |
| msg_type | enum | REQUEST, RESPONSE, EVENT, STREAM |
| msg_id | uint64 | Unique message identifier |
| source | string | Source service name |
| target | string | Target service name ("*" for broadcast) |
| method | string | RPC method name |
| payload | bytes | Method-specific payload (JSON encoded) |
| timestamp | uint64 | Unix timestamp in milliseconds |
| error | string? | Error message (null if success) |

## Message Types

### REQUEST
Client-to-service RPC call.

```json
{
  "msg_type": "REQUEST",
  "source": "agent",
  "target": "pkgmgr",
  "method": "package",
  "payload": "{\"action\": \"install\", \"packages\": [\"vim\"]}"
}
```

### RESPONSE
Service response to REQUEST.

```json
{
  "msg_type": "RESPONSE",
  "msg_id": 12345,
  "source": "pkgmgr",
  "target": "agent",
  "method": "package",
  "payload": "{\"success\": true}"
}
```

### EVENT
Broadcast event to subscribers.

```json
{
  "msg_type": "EVENT",
  "source": "builder",
  "target": "*",
  "method": "build.completed",
  "payload": "{\"package\": \"vim\", \"hash\": \"abc123\"}"
}
```

### STREAM
Streaming data (for logs, progress, etc.).

```json
{
  "msg_type": "STREAM",
  "source": "builder",
  "target": "agent",
  "method": "build.log",
  "payload": "Compiling main.c..."
}
```

## Service Registration

Services must register with the broker on connection:

```json
{
  "msg_type": "REQUEST",
  "source": "pkgmgr",
  "target": "broker",
  "method": "register",
  "payload": "{\"service\": \"pkgmgr\", \"type\": \"core\"}"
}
```

Response:
```json
{
  "msg_type": "RESPONSE",
  "source": "broker",
  "target": "pkgmgr",
  "method": "register",
  "payload": "{\"status\": \"ok\"}"
}
```

## Event Subscription

Subscribe to events:

```json
{
  "msg_type": "REQUEST",
  "source": "agent",
  "target": "broker",
  "method": "subscribe",
  "payload": "build.*,package.*"
}
```

## Service APIs

### Package Manager (pkgmgr)

#### package
```json
// Request
{
  "action": "install",
  "packages": ["vim", "git"],
  "options": {}
}

// Response
{
  "success": true,
  "packages": [
    {"name": "vim", "version": "9.0", "installed": true}
  ]
}
```

Actions: `install`, `remove`, `query`, `list`

### Build Executor (builder)

#### build
```json
// Request
{
  "package_name": "hello",
  "source_path": "/src/hello",
  "output_path": "/store",
  "env": {"CC": "gcc"},
  "build_args": []
}

// Response
{
  "success": true,
  "artifact_path": "/store/abc123",
  "artifact_hash": "abc123...",
  "logs": ["Compiling...", "Done"]
}
```

#### status
```json
// Request
"build-123"

// Response
{
  "job_id": "build-123",
  "package_name": "hello",
  "status": "building",
  "progress": 50
}
```

### Dependency Resolver (resolver)

#### resolve
```json
// Request
{
  "packages": ["vim"],
  "include_optional": false
}

// Response
{
  "success": true,
  "graph": [
    {"name": "libc", "version": "1.0", "dependencies": []},
    {"name": "vim", "version": "9.0", "dependencies": ["libc"]}
  ],
  "install_order": ["libc", "vim"]
}
```

### Artifact Cache (cache)

#### cache
```json
// GET Request
{
  "action": "get",
  "key": "build:vim:9.0"
}

// GET Response
{
  "success": true,
  "hit": true,
  "value": "..."
}

// PUT Request
{
  "action": "put",
  "key": "build:vim:9.0",
  "value": "...",
  "ttl": 86400
}

// STATS Request
{
  "action": "stats"
}

// STATS Response
{
  "success": true,
  "stats": {
    "total_entries": 100,
    "total_size": 1048576,
    "hit_count": 500,
    "miss_count": 50
  }
}
```

## Error Handling

Errors are returned in the `error` field:

```json
{
  "msg_type": "RESPONSE",
  "error": "Package not found: nonexistent"
}
```

## Implementation Examples

### Go Client
```go
// Connect
conn, _ := net.Dial("unix", "/run/mixos/ipc.sock")

// Send message
msg := IPCMessage{...}
data, _ := json.Marshal(msg)
binary.Write(conn, binary.BigEndian, uint32(len(data)))
conn.Write(data)

// Read response
var length uint32
binary.Read(conn, binary.BigEndian, &length)
buf := make([]byte, length)
conn.Read(buf)
```

### Python Client
```python
import socket
import struct
import json

sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
sock.connect("/run/mixos/ipc.sock")

# Send
data = json.dumps(msg).encode()
sock.send(struct.pack(">I", len(data)) + data)

# Receive
length = struct.unpack(">I", sock.recv(4))[0]
response = json.loads(sock.recv(length))
```

### OCaml Client
```ocaml
let read_message fd =
  let len_buf = Bytes.create 4 in
  let _ = Unix.read fd len_buf 0 4 in
  let len = (* decode big-endian *) in
  let msg_buf = Bytes.create len in
  let _ = Unix.read fd msg_buf 0 len in
  Yojson.Safe.from_string (Bytes.to_string msg_buf)
```

## Protobuf Schema

See `bin/pkg/ipc/proto/ipc.proto` for the Protobuf schema definition.
