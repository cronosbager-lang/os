"""IPC Client for connecting to MixOS broker"""

import asyncio
import socket
import struct
import json
import logging
from typing import Optional, Callable, Dict, Any
from dataclasses import dataclass

from .messages import IPCMessage, MessageType

logger = logging.getLogger(__name__)


class IPCClient:
    """Async IPC client for MixOS services"""

    def __init__(
        self,
        service_name: str,
        socket_path: str = "/run/mixos/ipc.sock",
    ):
        self.service_name = service_name
        self.socket_path = socket_path
        self.reader: Optional[asyncio.StreamReader] = None
        self.writer: Optional[asyncio.StreamWriter] = None
        self.msg_counter = 0
        self.pending_requests: Dict[int, asyncio.Future] = {}
        self.handlers: Dict[str, Callable] = {}
        self._running = False
        self._receive_task: Optional[asyncio.Task] = None

    async def connect(self, max_retries: int = 30, retry_delay: float = 1.0) -> bool:
        """Connect to the IPC broker with retry"""
        for attempt in range(max_retries):
            try:
                self.reader, self.writer = await asyncio.open_unix_connection(
                    self.socket_path
                )
                logger.info(f"Connected to broker at {self.socket_path}")
                
                # Register with broker
                await self._register()
                
                # Start receive loop
                self._running = True
                self._receive_task = asyncio.create_task(self._receive_loop())
                
                return True
            except (ConnectionRefusedError, FileNotFoundError) as e:
                logger.warning(f"Connection attempt {attempt + 1} failed: {e}")
                if attempt < max_retries - 1:
                    await asyncio.sleep(retry_delay)
        
        logger.error("Failed to connect to broker")
        return False

    async def _register(self):
        """Register service with broker"""
        msg = IPCMessage(
            msg_type=MessageType.REQUEST,
            msg_id=self._next_msg_id(),
            source=self.service_name,
            target="broker",
            method="register",
            payload=json.dumps({
                "service": self.service_name,
                "type": "agent"
            }).encode('utf-8'),
        )
        await self._send_message(msg)
        response = await self._read_message()
        if response:
            logger.info(f"Registered as {self.service_name}")

    def _next_msg_id(self) -> int:
        self.msg_counter += 1
        return self.msg_counter

    async def _send_message(self, msg: IPCMessage):
        """Send message to broker"""
        if not self.writer:
            raise ConnectionError("Not connected")
        
        data = msg.to_bytes()
        self.writer.write(data)
        await self.writer.drain()

    async def _read_message(self) -> Optional[IPCMessage]:
        """Read message from broker"""
        if not self.reader:
            return None
        
        try:
            # Read length prefix
            length_bytes = await self.reader.readexactly(4)
            length = struct.unpack('>I', length_bytes)[0]
            
            # Read message
            msg_bytes = await self.reader.readexactly(length)
            return IPCMessage.from_json(msg_bytes.decode('utf-8'))
        except asyncio.IncompleteReadError:
            return None
        except Exception as e:
            logger.error(f"Error reading message: {e}")
            return None

    async def _receive_loop(self):
        """Background task to receive messages"""
        while self._running:
            msg = await self._read_message()
            if msg is None:
                logger.error("Connection lost")
                self._running = False
                break
            
            # Check if it's a response to a pending request
            if msg.msg_type == MessageType.RESPONSE and msg.msg_id in self.pending_requests:
                future = self.pending_requests.pop(msg.msg_id)
                future.set_result(msg)
            # Check if it's a request for us
            elif msg.msg_type == MessageType.REQUEST and msg.target == self.service_name:
                await self._handle_request(msg)
            # Check if it's an event
            elif msg.msg_type == MessageType.EVENT:
                await self._handle_event(msg)

    async def _handle_request(self, msg: IPCMessage):
        """Handle incoming request"""
        handler = self.handlers.get(msg.method)
        if handler:
            try:
                result = await handler(msg.payload)
                response = IPCMessage(
                    msg_type=MessageType.RESPONSE,
                    msg_id=msg.msg_id,
                    source=self.service_name,
                    target=msg.source,
                    method=msg.method,
                    payload=result if isinstance(result, bytes) else json.dumps(result).encode('utf-8'),
                )
                await self._send_message(response)
            except Exception as e:
                logger.error(f"Handler error: {e}")
                response = IPCMessage(
                    msg_type=MessageType.RESPONSE,
                    msg_id=msg.msg_id,
                    source=self.service_name,
                    target=msg.source,
                    method=msg.method,
                    error=str(e),
                )
                await self._send_message(response)
        else:
            logger.warning(f"No handler for method: {msg.method}")

    async def _handle_event(self, msg: IPCMessage):
        """Handle incoming event"""
        handler = self.handlers.get(f"event:{msg.method}")
        if handler:
            try:
                await handler(msg.payload)
            except Exception as e:
                logger.error(f"Event handler error: {e}")

    def register_handler(self, method: str, handler: Callable):
        """Register a handler for incoming requests"""
        self.handlers[method] = handler

    def register_event_handler(self, event: str, handler: Callable):
        """Register a handler for events"""
        self.handlers[f"event:{event}"] = handler

    async def call(
        self,
        target: str,
        method: str,
        payload: Any,
        timeout: float = 30.0,
    ) -> Optional[IPCMessage]:
        """Make an RPC call to another service"""
        msg_id = self._next_msg_id()
        
        msg = IPCMessage(
            msg_type=MessageType.REQUEST,
            msg_id=msg_id,
            source=self.service_name,
            target=target,
            method=method,
            payload=payload if isinstance(payload, bytes) else json.dumps(payload).encode('utf-8'),
        )
        
        # Create future for response
        future = asyncio.get_event_loop().create_future()
        self.pending_requests[msg_id] = future
        
        try:
            await self._send_message(msg)
            response = await asyncio.wait_for(future, timeout=timeout)
            return response
        except asyncio.TimeoutError:
            self.pending_requests.pop(msg_id, None)
            logger.error(f"Request timeout: {target}.{method}")
            return None

    async def send_event(self, event: str, payload: Any):
        """Send an event to all subscribers"""
        msg = IPCMessage(
            msg_type=MessageType.EVENT,
            msg_id=self._next_msg_id(),
            source=self.service_name,
            target="*",
            method=event,
            payload=payload if isinstance(payload, bytes) else json.dumps(payload).encode('utf-8'),
        )
        await self._send_message(msg)

    async def subscribe(self, events: list):
        """Subscribe to events"""
        msg = IPCMessage(
            msg_type=MessageType.REQUEST,
            msg_id=self._next_msg_id(),
            source=self.service_name,
            target="broker",
            method="subscribe",
            payload=",".join(events).encode('utf-8'),
        )
        await self._send_message(msg)

    async def close(self):
        """Close connection"""
        self._running = False
        if self._receive_task:
            self._receive_task.cancel()
            try:
                await self._receive_task
            except asyncio.CancelledError:
                pass
        if self.writer:
            self.writer.close()
            await self.writer.wait_closed()


# Convenience functions for common operations

async def call_pkgmgr(client: IPCClient, action: str, packages: list = None, options: dict = None):
    """Call package manager service"""
    payload = {
        "action": action,
        "packages": packages or [],
        "options": options or {},
    }
    response = await client.call("pkgmgr", "package", payload)
    if response and not response.error:
        return json.loads(response.payload.decode('utf-8'))
    return None


async def call_builder(client: IPCClient, package_name: str, source_path: str, output_path: str, env: dict = None):
    """Call build executor service"""
    payload = {
        "package_name": package_name,
        "source_path": source_path,
        "output_path": output_path,
        "env": env or {},
        "build_args": [],
    }
    response = await client.call("builder", "build", payload, timeout=300.0)
    if response and not response.error:
        return json.loads(response.payload.decode('utf-8'))
    return None


async def call_resolver(client: IPCClient, packages: list, include_optional: bool = False):
    """Call dependency resolver service"""
    payload = {
        "packages": packages,
        "include_optional": include_optional,
    }
    response = await client.call("resolver", "resolve", payload)
    if response and not response.error:
        return json.loads(response.payload.decode('utf-8'))
    return None


async def call_cache(client: IPCClient, action: str, key: str = "", value: str = None, ttl: int = 0):
    """Call cache service"""
    payload = {
        "action": action,
        "key": key,
        "ttl": ttl,
    }
    if value is not None:
        payload["value"] = value
    response = await client.call("cache", "cache", payload)
    if response and not response.error:
        return json.loads(response.payload.decode('utf-8'))
    return None
