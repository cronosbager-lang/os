"""IPC Message definitions for MixOS"""

from dataclasses import dataclass, field
from enum import IntEnum
from typing import Optional, Dict, Any
import json
import time


class MessageType(IntEnum):
    REQUEST = 0
    RESPONSE = 1
    EVENT = 2
    STREAM = 3


@dataclass
class IPCMessage:
    """IPC Message structure matching protobuf definition"""
    version: int = 1
    msg_type: MessageType = MessageType.REQUEST
    msg_id: int = 0
    source: str = ""
    target: str = ""
    method: str = ""
    payload: bytes = b""
    timestamp: int = 0
    error: Optional[str] = None

    def __post_init__(self):
        if self.timestamp == 0:
            self.timestamp = int(time.time() * 1000)

    def to_json(self) -> str:
        """Serialize to JSON"""
        data = {
            "version": self.version,
            "msg_type": self.msg_type.name if isinstance(self.msg_type, MessageType) else self.msg_type,
            "msg_id": self.msg_id,
            "source": self.source,
            "target": self.target,
            "method": self.method,
            "payload": self.payload.decode('utf-8') if isinstance(self.payload, bytes) else self.payload,
            "timestamp": self.timestamp,
        }
        if self.error:
            data["error"] = self.error
        return json.dumps(data)

    @classmethod
    def from_json(cls, json_str: str) -> 'IPCMessage':
        """Deserialize from JSON"""
        data = json.loads(json_str)
        msg_type = data.get("msg_type", "REQUEST")
        if isinstance(msg_type, str):
            msg_type = MessageType[msg_type]
        
        payload = data.get("payload", "")
        if isinstance(payload, str):
            payload = payload.encode('utf-8')
        
        return cls(
            version=data.get("version", 1),
            msg_type=msg_type,
            msg_id=data.get("msg_id", 0),
            source=data.get("source", ""),
            target=data.get("target", ""),
            method=data.get("method", ""),
            payload=payload,
            timestamp=data.get("timestamp", 0),
            error=data.get("error"),
        )

    def to_bytes(self) -> bytes:
        """Serialize to bytes with length prefix"""
        json_bytes = self.to_json().encode('utf-8')
        length = len(json_bytes)
        length_bytes = length.to_bytes(4, byteorder='big')
        return length_bytes + json_bytes


# Specific message types for services

@dataclass
class PackageRequest:
    action: str  # "install", "remove", "query", "list"
    packages: list = field(default_factory=list)
    options: Dict[str, str] = field(default_factory=dict)

    def to_json(self) -> str:
        return json.dumps({
            "action": self.action,
            "packages": self.packages,
            "options": self.options,
        })


@dataclass
class BuildRequest:
    package_name: str
    source_path: str
    output_path: str
    env: Dict[str, str] = field(default_factory=dict)
    build_args: list = field(default_factory=list)

    def to_json(self) -> str:
        return json.dumps({
            "package_name": self.package_name,
            "source_path": self.source_path,
            "output_path": self.output_path,
            "env": self.env,
            "build_args": self.build_args,
        })


@dataclass
class ResolveRequest:
    packages: list
    include_optional: bool = False

    def to_json(self) -> str:
        return json.dumps({
            "packages": self.packages,
            "include_optional": self.include_optional,
        })


@dataclass
class CacheRequest:
    action: str  # "get", "put", "delete", "gc", "stats"
    key: str = ""
    value: Optional[str] = None
    ttl: int = 0

    def to_json(self) -> str:
        data = {
            "action": self.action,
            "key": self.key,
            "ttl": self.ttl,
        }
        if self.value is not None:
            data["value"] = self.value
        return json.dumps(data)


@dataclass 
class AgentRequest:
    action: str  # "chat", "execute", "status"
    prompt: str = ""
    context: Dict[str, str] = field(default_factory=dict)

    def to_json(self) -> str:
        return json.dumps({
            "action": self.action,
            "prompt": self.prompt,
            "context": self.context,
        })
