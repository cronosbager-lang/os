"""WebSocket support for the AI agent."""

from typing import Dict, Set, Optional, Any
from fastapi import WebSocket, WebSocketDisconnect
import asyncio
import json
from dataclasses import dataclass
from enum import Enum


class WSMessageType(Enum):
    """WebSocket message types."""
    CHAT = "chat"
    CHAT_RESPONSE = "chat_response"
    CHAT_STREAM = "chat_stream"
    TOOL_CALL = "tool_call"
    TOOL_RESULT = "tool_result"
    STATUS = "status"
    ERROR = "error"
    PING = "ping"
    PONG = "pong"


@dataclass
class WSMessage:
    """WebSocket message."""
    type: WSMessageType
    data: Dict[str, Any]
    id: Optional[str] = None
    
    def to_json(self) -> str:
        return json.dumps({
            "type": self.type.value,
            "data": self.data,
            "id": self.id,
        })
    
    @classmethod
    def from_json(cls, data: str) -> "WSMessage":
        obj = json.loads(data)
        return cls(
            type=WSMessageType(obj["type"]),
            data=obj.get("data", {}),
            id=obj.get("id"),
        )


class ConnectionManager:
    """Manages WebSocket connections."""
    
    def __init__(self):
        self.active_connections: Dict[str, WebSocket] = {}
        self.session_connections: Dict[str, Set[str]] = {}  # session_id -> connection_ids
    
    async def connect(self, websocket: WebSocket, connection_id: str):
        """Accept a new connection."""
        await websocket.accept()
        self.active_connections[connection_id] = websocket
    
    def disconnect(self, connection_id: str):
        """Remove a connection."""
        if connection_id in self.active_connections:
            del self.active_connections[connection_id]
        
        # Remove from session mappings
        for session_id, connections in list(self.session_connections.items()):
            connections.discard(connection_id)
            if not connections:
                del self.session_connections[session_id]
    
    def associate_session(self, connection_id: str, session_id: str):
        """Associate a connection with a session."""
        if session_id not in self.session_connections:
            self.session_connections[session_id] = set()
        self.session_connections[session_id].add(connection_id)
    
    async def send_message(self, connection_id: str, message: WSMessage):
        """Send a message to a specific connection."""
        if connection_id in self.active_connections:
            websocket = self.active_connections[connection_id]
            await websocket.send_text(message.to_json())
    
    async def broadcast_to_session(self, session_id: str, message: WSMessage):
        """Broadcast a message to all connections in a session."""
        if session_id in self.session_connections:
            for connection_id in self.session_connections[session_id]:
                await self.send_message(connection_id, message)
    
    async def broadcast_all(self, message: WSMessage):
        """Broadcast a message to all connections."""
        for connection_id in self.active_connections:
            await self.send_message(connection_id, message)


class WebSocketHandler:
    """Handles WebSocket communication."""
    
    def __init__(self, agent):
        self.agent = agent
        self.manager = ConnectionManager()
    
    async def handle_connection(self, websocket: WebSocket, connection_id: str):
        """Handle a WebSocket connection."""
        await self.manager.connect(websocket, connection_id)
        
        try:
            while True:
                # Receive message
                data = await websocket.receive_text()
                
                try:
                    message = WSMessage.from_json(data)
                    await self.handle_message(connection_id, message)
                except json.JSONDecodeError:
                    await self.send_error(connection_id, "Invalid JSON")
                except Exception as e:
                    await self.send_error(connection_id, str(e))
        
        except WebSocketDisconnect:
            self.manager.disconnect(connection_id)
    
    async def handle_message(self, connection_id: str, message: WSMessage):
        """Handle an incoming message."""
        if message.type == WSMessageType.PING:
            await self.manager.send_message(
                connection_id,
                WSMessage(type=WSMessageType.PONG, data={}, id=message.id),
            )
        
        elif message.type == WSMessageType.CHAT:
            await self.handle_chat(connection_id, message)
        
        elif message.type == WSMessageType.CHAT_STREAM:
            await self.handle_chat_stream(connection_id, message)
        
        elif message.type == WSMessageType.STATUS:
            await self.handle_status(connection_id, message)
        
        else:
            await self.send_error(connection_id, f"Unknown message type: {message.type}")
    
    async def handle_chat(self, connection_id: str, message: WSMessage):
        """Handle a chat message."""
        user_message = message.data.get("message", "")
        session_id = message.data.get("session_id")
        
        if session_id:
            self.manager.associate_session(connection_id, session_id)
        
        try:
            response = await self.agent.chat(
                message=user_message,
                session_id=session_id,
            )
            
            await self.manager.send_message(
                connection_id,
                WSMessage(
                    type=WSMessageType.CHAT_RESPONSE,
                    data={
                        "response": response.content,
                        "session_id": response.session_id,
                        "tool_calls": response.tool_calls,
                    },
                    id=message.id,
                ),
            )
        except Exception as e:
            await self.send_error(connection_id, str(e), message.id)
    
    async def handle_chat_stream(self, connection_id: str, message: WSMessage):
        """Handle a streaming chat message."""
        user_message = message.data.get("message", "")
        session_id = message.data.get("session_id")
        
        if session_id:
            self.manager.associate_session(connection_id, session_id)
        
        try:
            async for chunk in self.agent.chat_stream(
                message=user_message,
                session_id=session_id,
            ):
                await self.manager.send_message(
                    connection_id,
                    WSMessage(
                        type=WSMessageType.CHAT_STREAM,
                        data={"chunk": chunk, "done": False},
                        id=message.id,
                    ),
                )
            
            # Send completion message
            await self.manager.send_message(
                connection_id,
                WSMessage(
                    type=WSMessageType.CHAT_STREAM,
                    data={"chunk": "", "done": True},
                    id=message.id,
                ),
            )
        except Exception as e:
            await self.send_error(connection_id, str(e), message.id)
    
    async def handle_status(self, connection_id: str, message: WSMessage):
        """Handle a status request."""
        status = self.agent.get_status()
        
        await self.manager.send_message(
            connection_id,
            WSMessage(
                type=WSMessageType.STATUS,
                data=status,
                id=message.id,
            ),
        )
    
    async def send_error(self, connection_id: str, error: str, message_id: str = None):
        """Send an error message."""
        await self.manager.send_message(
            connection_id,
            WSMessage(
                type=WSMessageType.ERROR,
                data={"error": error},
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
                },
            ),
        )
    
    async def notify_tool_result(self, session_id: str, tool_name: str, result: str, success: bool):
        """Notify clients about a tool result."""
        await self.manager.broadcast_to_session(
            session_id,
            WSMessage(
                type=WSMessageType.TOOL_RESULT,
                data={
                    "tool": tool_name,
                    "result": result,
                    "success": success,
                },
            ),
        )


def create_websocket_endpoint(handler: WebSocketHandler):
    """Create a WebSocket endpoint function."""
    async def websocket_endpoint(websocket: WebSocket):
        import uuid
        connection_id = str(uuid.uuid4())
        await handler.handle_connection(websocket, connection_id)
    
    return websocket_endpoint
