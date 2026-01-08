"""WebSocket support for the AI agent."""

from typing import Dict, Set, Optional, Any, Callable, Awaitable
from fastapi import WebSocket, WebSocketDisconnect
import asyncio
import json
import time
import uuid
from dataclasses import dataclass, field
from datetime import datetime
from enum import Enum
from loguru import logger


class WSMessageType(Enum):
    """WebSocket message types."""
    # Chat
    CHAT = "chat"
    CHAT_RESPONSE = "chat_response"
    CHAT_STREAM = "chat_stream"
    CHAT_STREAM_END = "chat_stream_end"
    
    # Execute
    EXECUTE = "execute"
    EXECUTE_PLAN = "execute_plan"
    EXECUTE_STATUS = "execute_status"
    EXECUTE_STEP = "execute_step"
    EXECUTE_COMPLETE = "execute_complete"
    EXECUTE_CANCEL = "execute_cancel"
    
    # Tools
    TOOL_CALL = "tool_call"
    TOOL_RESULT = "tool_result"
    TOOL_CONFIRM = "tool_confirm"
    TOOL_CONFIRM_RESPONSE = "tool_confirm_response"
    
    # System
    STATUS = "status"
    METRICS = "metrics"
    ERROR = "error"
    PING = "ping"
    PONG = "pong"
    
    # Session
    SESSION_START = "session_start"
    SESSION_END = "session_end"


@dataclass
class WSMessage:
    """WebSocket message."""
    type: WSMessageType
    data: Dict[str, Any]
    id: Optional[str] = None
    timestamp: datetime = field(default_factory=datetime.utcnow)
    
    def to_json(self) -> str:
        return json.dumps({
            "type": self.type.value,
            "data": self.data,
            "id": self.id,
            "timestamp": self.timestamp.isoformat(),
        })
    
    @classmethod
    def from_json(cls, data: str) -> "WSMessage":
        obj = json.loads(data)
        return cls(
            type=WSMessageType(obj["type"]),
            data=obj.get("data", {}),
            id=obj.get("id"),
            timestamp=datetime.fromisoformat(obj["timestamp"]) if "timestamp" in obj else datetime.utcnow(),
        )


@dataclass
class ConnectionInfo:
    """Information about a WebSocket connection."""
    id: str
    websocket: WebSocket
    session_id: Optional[str] = None
    connected_at: datetime = field(default_factory=datetime.utcnow)
    last_activity: datetime = field(default_factory=datetime.utcnow)
    message_count: int = 0
    metadata: Dict[str, Any] = field(default_factory=dict)


class ConnectionManager:
    """Manages WebSocket connections."""
    
    def __init__(self):
        self.connections: Dict[str, ConnectionInfo] = {}
        self.session_connections: Dict[str, Set[str]] = {}  # session_id -> connection_ids
        self.pending_confirmations: Dict[str, asyncio.Future] = {}  # confirmation_id -> future
        self._lock = asyncio.Lock()
    
    @property
    def active_connections(self) -> Dict[str, WebSocket]:
        """Get active connections (backward compatibility)."""
        return {cid: info.websocket for cid, info in self.connections.items()}
    
    @property
    def connection_count(self) -> int:
        """Get number of active connections."""
        return len(self.connections)
    
    async def connect(self, websocket: WebSocket, connection_id: str, metadata: Dict[str, Any] = None):
        """Accept a new connection."""
        await websocket.accept()
        async with self._lock:
            self.connections[connection_id] = ConnectionInfo(
                id=connection_id,
                websocket=websocket,
                metadata=metadata or {},
            )
        logger.info(f"WebSocket connected: {connection_id}")
    
    async def disconnect(self, connection_id: str):
        """Remove a connection."""
        async with self._lock:
            if connection_id in self.connections:
                info = self.connections[connection_id]
                del self.connections[connection_id]
                
                # Remove from session mappings
                if info.session_id and info.session_id in self.session_connections:
                    self.session_connections[info.session_id].discard(connection_id)
                    if not self.session_connections[info.session_id]:
                        del self.session_connections[info.session_id]
        
        logger.info(f"WebSocket disconnected: {connection_id}")
    
    def associate_session(self, connection_id: str, session_id: str):
        """Associate a connection with a session."""
        if connection_id in self.connections:
            self.connections[connection_id].session_id = session_id
            
            if session_id not in self.session_connections:
                self.session_connections[session_id] = set()
            self.session_connections[session_id].add(connection_id)
    
    def get_connection(self, connection_id: str) -> Optional[ConnectionInfo]:
        """Get connection info."""
        return self.connections.get(connection_id)
    
    def get_session_connections(self, session_id: str) -> Set[str]:
        """Get all connection IDs for a session."""
        return self.session_connections.get(session_id, set())
    
    async def send_message(self, connection_id: str, message: WSMessage):
        """Send a message to a specific connection."""
        if connection_id in self.connections:
            info = self.connections[connection_id]
            try:
                await info.websocket.send_text(message.to_json())
                info.last_activity = datetime.utcnow()
                info.message_count += 1
            except Exception as e:
                logger.error(f"Failed to send message to {connection_id}: {e}")
                await self.disconnect(connection_id)
    
    async def broadcast_to_session(self, session_id: str, message: WSMessage):
        """Broadcast a message to all connections in a session."""
        if session_id in self.session_connections:
            tasks = [
                self.send_message(cid, message)
                for cid in self.session_connections[session_id]
            ]
            await asyncio.gather(*tasks, return_exceptions=True)
    
    async def broadcast_all(self, message: WSMessage):
        """Broadcast a message to all connections."""
        tasks = [
            self.send_message(cid, message)
            for cid in self.connections
        ]
        await asyncio.gather(*tasks, return_exceptions=True)
    
    async def request_confirmation(
        self,
        connection_id: str,
        tool_name: str,
        parameters: Dict[str, Any],
        timeout: float = 60.0,
    ) -> bool:
        """Request confirmation from user for a tool call."""
        confirmation_id = str(uuid.uuid4())
        future = asyncio.get_event_loop().create_future()
        self.pending_confirmations[confirmation_id] = future
        
        await self.send_message(
            connection_id,
            WSMessage(
                type=WSMessageType.TOOL_CONFIRM,
                data={
                    "confirmation_id": confirmation_id,
                    "tool": tool_name,
                    "parameters": parameters,
                },
            ),
        )
        
        try:
            result = await asyncio.wait_for(future, timeout=timeout)
            return result
        except asyncio.TimeoutError:
            return False
        finally:
            self.pending_confirmations.pop(confirmation_id, None)
    
    def resolve_confirmation(self, confirmation_id: str, confirmed: bool):
        """Resolve a pending confirmation."""
        if confirmation_id in self.pending_confirmations:
            future = self.pending_confirmations[confirmation_id]
            if not future.done():
                future.set_result(confirmed)
    
    def get_stats(self) -> Dict[str, Any]:
        """Get connection statistics."""
        return {
            "total_connections": len(self.connections),
            "total_sessions": len(self.session_connections),
            "connections": [
                {
                    "id": info.id,
                    "session_id": info.session_id,
                    "connected_at": info.connected_at.isoformat(),
                    "message_count": info.message_count,
                }
                for info in self.connections.values()
            ],
        }


class WebSocketHandler:
    """Handles WebSocket communication."""
    
    def __init__(self, agent):
        self.agent = agent
        self.manager = ConnectionManager()
        self.active_tasks: Dict[str, asyncio.Task] = {}
        self._heartbeat_task: Optional[asyncio.Task] = None
        self._heartbeat_interval = 30  # seconds
    
    async def start(self):
        """Start background tasks."""
        self._heartbeat_task = asyncio.create_task(self._heartbeat_loop())
    
    async def stop(self):
        """Stop background tasks."""
        if self._heartbeat_task:
            self._heartbeat_task.cancel()
            try:
                await self._heartbeat_task
            except asyncio.CancelledError:
                pass
        
        # Cancel all active tasks
        for task in self.active_tasks.values():
            task.cancel()
    
    async def _heartbeat_loop(self):
        """Send periodic heartbeats to all connections."""
        while True:
            try:
                await asyncio.sleep(self._heartbeat_interval)
                await self.manager.broadcast_all(
                    WSMessage(type=WSMessageType.PING, data={})
                )
            except asyncio.CancelledError:
                break
            except Exception as e:
                logger.error(f"Heartbeat error: {e}")
    
    async def handle_connection(self, websocket: WebSocket, connection_id: str):
        """Handle a WebSocket connection."""
        await self.manager.connect(websocket, connection_id)
        
        # Send session start message
        session_id = str(uuid.uuid4())
        self.manager.associate_session(connection_id, session_id)
        
        await self.manager.send_message(
            connection_id,
            WSMessage(
                type=WSMessageType.SESSION_START,
                data={"session_id": session_id, "connection_id": connection_id},
            ),
        )
        
        try:
            while True:
                data = await websocket.receive_text()
                
                try:
                    message = WSMessage.from_json(data)
                    await self.handle_message(connection_id, message)
                except json.JSONDecodeError:
                    await self.send_error(connection_id, "Invalid JSON", code="INVALID_JSON")
                except ValueError as e:
                    await self.send_error(connection_id, str(e), code="INVALID_MESSAGE")
                except Exception as e:
                    logger.exception(f"Error handling message: {e}")
                    await self.send_error(connection_id, str(e), code="INTERNAL_ERROR")
        
        except WebSocketDisconnect:
            await self.manager.disconnect(connection_id)
            # Cancel any active tasks for this connection
            if connection_id in self.active_tasks:
                self.active_tasks[connection_id].cancel()
                del self.active_tasks[connection_id]
    
    async def handle_message(self, connection_id: str, message: WSMessage):
        """Handle an incoming message."""
        handlers = {
            WSMessageType.PING: self.handle_ping,
            WSMessageType.PONG: self.handle_pong,
            WSMessageType.CHAT: self.handle_chat,
            WSMessageType.CHAT_STREAM: self.handle_chat_stream,
            WSMessageType.EXECUTE: self.handle_execute,
            WSMessageType.EXECUTE_CANCEL: self.handle_execute_cancel,
            WSMessageType.TOOL_CONFIRM_RESPONSE: self.handle_tool_confirm_response,
            WSMessageType.STATUS: self.handle_status,
            WSMessageType.METRICS: self.handle_metrics,
        }
        
        handler = handlers.get(message.type)
        if handler:
            await handler(connection_id, message)
        else:
            await self.send_error(
                connection_id,
                f"Unknown message type: {message.type.value}",
                code="UNKNOWN_MESSAGE_TYPE",
                message_id=message.id,
            )
    
    async def handle_ping(self, connection_id: str, message: WSMessage):
        """Handle ping message."""
        await self.manager.send_message(
            connection_id,
            WSMessage(
                type=WSMessageType.PONG,
                data={"timestamp": datetime.utcnow().isoformat()},
                id=message.id,
            ),
        )
    
    async def handle_pong(self, connection_id: str, message: WSMessage):
        """Handle pong message (update last activity)."""
        info = self.manager.get_connection(connection_id)
        if info:
            info.last_activity = datetime.utcnow()
    
    async def handle_chat(self, connection_id: str, message: WSMessage):
        """Handle a chat message."""
        user_message = message.data.get("message", "")
        session_id = message.data.get("session_id")
        
        if not user_message:
            await self.send_error(connection_id, "Message is required", code="MISSING_MESSAGE", message_id=message.id)
            return
        
        if session_id:
            self.manager.associate_session(connection_id, session_id)
        else:
            info = self.manager.get_connection(connection_id)
            session_id = info.session_id if info else None
        
        start_time = time.time()
        
        try:
            response = self.agent.chat(
                message=user_message,
                session_id=session_id,
            )
            
            processing_time = time.time() - start_time
            
            await self.manager.send_message(
                connection_id,
                WSMessage(
                    type=WSMessageType.CHAT_RESPONSE,
                    data={
                        "response": response,
                        "session_id": session_id,
                        "processing_time": processing_time,
                    },
                    id=message.id,
                ),
            )
        except Exception as e:
            logger.exception(f"Chat error: {e}")
            await self.send_error(connection_id, str(e), code="CHAT_ERROR", message_id=message.id)
    
    async def handle_chat_stream(self, connection_id: str, message: WSMessage):
        """Handle a streaming chat message."""
        user_message = message.data.get("message", "")
        session_id = message.data.get("session_id")
        
        if not user_message:
            await self.send_error(connection_id, "Message is required", code="MISSING_MESSAGE", message_id=message.id)
            return
        
        if session_id:
            self.manager.associate_session(connection_id, session_id)
        else:
            info = self.manager.get_connection(connection_id)
            session_id = info.session_id if info else None
        
        try:
            # Check if agent supports streaming
            if hasattr(self.agent, 'chat_stream'):
                async for chunk in self.agent.chat_stream(
                    message=user_message,
                    session_id=session_id,
                ):
                    await self.manager.send_message(
                        connection_id,
                        WSMessage(
                            type=WSMessageType.CHAT_STREAM,
                            data={"chunk": chunk, "session_id": session_id, "done": False},
                            id=message.id,
                        ),
                    )
            else:
                # Fallback to non-streaming
                response = self.agent.chat(message=user_message, session_id=session_id)
                await self.manager.send_message(
                    connection_id,
                    WSMessage(
                        type=WSMessageType.CHAT_STREAM,
                        data={"chunk": response, "session_id": session_id, "done": False},
                        id=message.id,
                    ),
                )
            
            # Send completion message
            await self.manager.send_message(
                connection_id,
                WSMessage(
                    type=WSMessageType.CHAT_STREAM_END,
                    data={"session_id": session_id, "done": True},
                    id=message.id,
                ),
            )
        except Exception as e:
            logger.exception(f"Stream error: {e}")
            await self.send_error(connection_id, str(e), code="STREAM_ERROR", message_id=message.id)
    
    async def handle_execute(self, connection_id: str, message: WSMessage):
        """Handle task execution request."""
        task_description = message.data.get("task", "")
        auto_confirm = message.data.get("auto_confirm", False)
        
        if not task_description:
            await self.send_error(connection_id, "Task is required", code="MISSING_TASK", message_id=message.id)
            return
        
        task_id = str(uuid.uuid4())
        
        # Send initial status
        await self.manager.send_message(
            connection_id,
            WSMessage(
                type=WSMessageType.EXECUTE_STATUS,
                data={
                    "task_id": task_id,
                    "status": "started",
                    "message": "Task execution started",
                },
                id=message.id,
            ),
        )
        
        # Create execution task
        async def execute_task():
            try:
                result = self.agent.execute_task(task_description, auto_confirm)
                
                # Send step updates
                for step in result.get("steps", []):
                    await self.manager.send_message(
                        connection_id,
                        WSMessage(
                            type=WSMessageType.EXECUTE_STEP,
                            data={"task_id": task_id, "step": step},
                        ),
                    )
                
                # Send completion
                await self.manager.send_message(
                    connection_id,
                    WSMessage(
                        type=WSMessageType.EXECUTE_COMPLETE,
                        data={
                            "task_id": task_id,
                            "status": result.get("status", "completed"),
                            "message": result.get("message", ""),
                            "steps": result.get("steps", []),
                        },
                        id=message.id,
                    ),
                )
            except asyncio.CancelledError:
                await self.manager.send_message(
                    connection_id,
                    WSMessage(
                        type=WSMessageType.EXECUTE_COMPLETE,
                        data={
                            "task_id": task_id,
                            "status": "cancelled",
                            "message": "Task was cancelled",
                        },
                        id=message.id,
                    ),
                )
            except Exception as e:
                logger.exception(f"Execute error: {e}")
                await self.manager.send_message(
                    connection_id,
                    WSMessage(
                        type=WSMessageType.EXECUTE_COMPLETE,
                        data={
                            "task_id": task_id,
                            "status": "failed",
                            "message": str(e),
                        },
                        id=message.id,
                    ),
                )
            finally:
                self.active_tasks.pop(task_id, None)
        
        # Start task
        task = asyncio.create_task(execute_task())
        self.active_tasks[task_id] = task
    
    async def handle_execute_cancel(self, connection_id: str, message: WSMessage):
        """Handle task cancellation request."""
        task_id = message.data.get("task_id")
        
        if task_id and task_id in self.active_tasks:
            self.active_tasks[task_id].cancel()
            await self.manager.send_message(
                connection_id,
                WSMessage(
                    type=WSMessageType.EXECUTE_STATUS,
                    data={
                        "task_id": task_id,
                        "status": "cancelling",
                        "message": "Task cancellation requested",
                    },
                    id=message.id,
                ),
            )
        else:
            await self.send_error(
                connection_id,
                f"Task not found: {task_id}",
                code="TASK_NOT_FOUND",
                message_id=message.id,
            )
    
    async def handle_tool_confirm_response(self, connection_id: str, message: WSMessage):
        """Handle tool confirmation response."""
        confirmation_id = message.data.get("confirmation_id")
        confirmed = message.data.get("confirmed", False)
        
        if confirmation_id:
            self.manager.resolve_confirmation(confirmation_id, confirmed)
    
    async def handle_status(self, connection_id: str, message: WSMessage):
        """Handle a status request."""
        status = self.agent.get_status()
        status["websocket"] = self.manager.get_stats()
        
        await self.manager.send_message(
            connection_id,
            WSMessage(
                type=WSMessageType.STATUS,
                data=status,
                id=message.id,
            ),
        )
    
    async def handle_metrics(self, connection_id: str, message: WSMessage):
        """Handle metrics request."""
        metrics = {
            "agent": self.agent.get_status() if hasattr(self.agent, 'get_status') else {},
            "websocket": self.manager.get_stats(),
            "active_tasks": len(self.active_tasks),
        }
        
        await self.manager.send_message(
            connection_id,
            WSMessage(
                type=WSMessageType.METRICS,
                data=metrics,
                id=message.id,
            ),
        )
    
    async def send_error(
        self,
        connection_id: str,
        error: str,
        code: str = "UNKNOWN_ERROR",
        message_id: str = None,
    ):
        """Send an error message."""
        await self.manager.send_message(
            connection_id,
            WSMessage(
                type=WSMessageType.ERROR,
                data={"error": error, "code": code},
                id=message_id,
            ),
        )
    
    async def notify_tool_call(self, session_id: str, tool_name: str, parameters: Dict):
        """Notify clients about a tool call."""
        await self.manager.broadcast_to_session(
            session_id,
            WSMessage(
                type=WSMessageType.TOOL_CALL,
                data={
                    "tool": tool_name,
                    "parameters": parameters,
                    "timestamp": datetime.utcnow().isoformat(),
                },
            ),
        )
    
    async def notify_tool_result(
        self,
        session_id: str,
        tool_name: str,
        result: str,
        success: bool,
        duration: float = 0.0,
    ):
        """Notify clients about a tool result."""
        await self.manager.broadcast_to_session(
            session_id,
            WSMessage(
                type=WSMessageType.TOOL_RESULT,
                data={
                    "tool": tool_name,
                    "result": result,
                    "success": success,
                    "duration": duration,
                    "timestamp": datetime.utcnow().isoformat(),
                },
            ),
        )
    
    async def request_tool_confirmation(
        self,
        connection_id: str,
        tool_name: str,
        parameters: Dict[str, Any],
        timeout: float = 60.0,
    ) -> bool:
        """Request user confirmation for a tool call."""
        return await self.manager.request_confirmation(
            connection_id, tool_name, parameters, timeout
        )


def create_websocket_endpoint(handler: WebSocketHandler):
    """Create a WebSocket endpoint function."""
    async def websocket_endpoint(websocket: WebSocket):
        connection_id = str(uuid.uuid4())
        await handler.handle_connection(websocket, connection_id)
    
    return websocket_endpoint


# Global handler instance (set by application)
_ws_handler: Optional[WebSocketHandler] = None


def get_ws_handler() -> Optional[WebSocketHandler]:
    """Get the global WebSocket handler."""
    return _ws_handler


def set_ws_handler(handler: WebSocketHandler):
    """Set the global WebSocket handler."""
    global _ws_handler
    _ws_handler = handler
