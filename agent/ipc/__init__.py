"""MixOS IPC Client Library"""

from .client import (
    IPCClient,
    ClientConfig,
    ConnectionState,
    RetryConfig,
)
from .messages import (
    IPCMessage,
    MessageType,
    PackageRequest,
    BuildRequest,
    ResolveRequest,
    CacheRequest,
    AgentRequest,
)
from .pool import ConnectionPool, PooledConnection

__all__ = [
    "IPCClient",
    "ClientConfig",
    "ConnectionState",
    "RetryConfig",
    "IPCMessage",
    "MessageType",
    "PackageRequest",
    "BuildRequest",
    "ResolveRequest",
    "CacheRequest",
    "AgentRequest",
    "ConnectionPool",
    "PooledConnection",
]
