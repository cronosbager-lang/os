"""Connection Pool for MixOS IPC Client"""

import asyncio
import logging
from dataclasses import dataclass, field
from typing import Optional, Dict, List
from datetime import datetime, timedelta

logger = logging.getLogger(__name__)


@dataclass
class PooledConnection:
    """A pooled connection with metadata"""
    reader: asyncio.StreamReader
    writer: asyncio.StreamWriter
    service: str
    created_at: datetime = field(default_factory=datetime.now)
    last_used: datetime = field(default_factory=datetime.now)
    request_count: int = 0
    healthy: bool = True
    in_use: bool = False


class ConnectionPool:
    """Manages a pool of IPC connections"""

    def __init__(
        self,
        socket_path: str = "/run/mixos/ipc.sock",
        max_size: int = 10,
        idle_timeout: float = 300.0,  # 5 minutes
        cleanup_interval: float = 30.0,
    ):
        self.socket_path = socket_path
        self.max_size = max_size
        self.idle_timeout = timedelta(seconds=idle_timeout)
        self.cleanup_interval = cleanup_interval
        
        self._connections: Dict[str, PooledConnection] = {}
        self._lock = asyncio.Lock()
        self._cleanup_task: Optional[asyncio.Task] = None
        self._running = False

    async def start(self):
        """Start the connection pool"""
        self._running = True
        self._cleanup_task = asyncio.create_task(self._cleanup_loop())
        logger.info("Connection pool started")

    async def stop(self):
        """Stop the connection pool and close all connections"""
        self._running = False
        if self._cleanup_task:
            self._cleanup_task.cancel()
            try:
                await self._cleanup_task
            except asyncio.CancelledError:
                pass

        async with self._lock:
            for conn_id, conn in list(self._connections.items()):
                await self._close_connection(conn)
            self._connections.clear()
        
        logger.info("Connection pool stopped")

    async def acquire(self, service: str = "") -> Optional[PooledConnection]:
        """Acquire a connection from the pool"""
        async with self._lock:
            # Try to find an existing healthy connection
            for conn_id, conn in self._connections.items():
                if conn.service == service and conn.healthy and not conn.in_use:
                    conn.in_use = True
                    conn.last_used = datetime.now()
                    return conn

            # Create new connection if pool not full
            if len(self._connections) < self.max_size:
                conn = await self._create_connection(service)
                if conn:
                    conn_id = f"{id(conn.writer)}_{datetime.now().timestamp()}"
                    self._connections[conn_id] = conn
                    conn.in_use = True
                    return conn

        return None

    async def release(self, conn: PooledConnection):
        """Release a connection back to the pool"""
        async with self._lock:
            conn.in_use = False
            conn.last_used = datetime.now()
            conn.request_count += 1

    async def _create_connection(self, service: str) -> Optional[PooledConnection]:
        """Create a new connection"""
        try:
            reader, writer = await asyncio.open_unix_connection(self.socket_path)
            return PooledConnection(
                reader=reader,
                writer=writer,
                service=service,
            )
        except Exception as e:
            logger.error(f"Failed to create connection: {e}")
            return None

    async def _close_connection(self, conn: PooledConnection):
        """Close a connection"""
        try:
            conn.writer.close()
            await conn.writer.wait_closed()
        except Exception as e:
            logger.debug(f"Error closing connection: {e}")

    async def _cleanup_loop(self):
        """Periodically clean up idle connections"""
        while self._running:
            try:
                await asyncio.sleep(self.cleanup_interval)
                await self._cleanup()
            except asyncio.CancelledError:
                break
            except Exception as e:
                logger.error(f"Cleanup error: {e}")

    async def _cleanup(self):
        """Remove idle and unhealthy connections"""
        now = datetime.now()
        to_remove = []

        async with self._lock:
            for conn_id, conn in self._connections.items():
                # Remove idle connections
                if not conn.in_use and (now - conn.last_used) > self.idle_timeout:
                    to_remove.append(conn_id)
                    continue
                
                # Remove unhealthy connections
                if not conn.healthy and not conn.in_use:
                    to_remove.append(conn_id)

            for conn_id in to_remove:
                conn = self._connections.pop(conn_id)
                await self._close_connection(conn)
                logger.debug(f"Removed connection {conn_id}")

    def mark_unhealthy(self, conn: PooledConnection):
        """Mark a connection as unhealthy"""
        conn.healthy = False

    def mark_healthy(self, conn: PooledConnection):
        """Mark a connection as healthy"""
        conn.healthy = True

    @property
    def size(self) -> int:
        """Current pool size"""
        return len(self._connections)

    @property
    def available(self) -> int:
        """Number of available connections"""
        return sum(1 for c in self._connections.values() if not c.in_use and c.healthy)

    def stats(self) -> Dict:
        """Get pool statistics"""
        return {
            "total": len(self._connections),
            "available": self.available,
            "healthy": sum(1 for c in self._connections.values() if c.healthy),
            "in_use": sum(1 for c in self._connections.values() if c.in_use),
            "total_requests": sum(c.request_count for c in self._connections.values()),
        }
