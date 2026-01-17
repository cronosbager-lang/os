"""IPC Client for connecting to MixOS broker"""

import asyncio
import struct
import json
import logging
import time
from typing import Optional, Callable, Dict, Any, List
from dataclasses import dataclass, field
from enum import Enum

from .messages import IPCMessage, MessageType

logger = logging.getLogger(__name__)


class ConnectionState(Enum):
    """Connection state"""
    DISCONNECTED = "disconnected"
    CONNECTING = "connecting"
    CONNECTED = "connected"
    RECONNECTING = "reconnecting"


@dataclass
class ClientConfig:
    """IPC Client configuration"""
    socket_path: str = "/run/mixos/ipc.sock"
    connect_timeout: float = 5.0
    request_timeout: float = 30.0
    reconnect_delay: float = 1.0
    max_reconnect_delay: float = 30.0
    max_retries: int = 3
    heartbeat_interval: float = 30.0
    heartbeat_timeout: float = 10.0


@dataclass
class RetryConfig:
    """Retry configuration with exponential backoff"""
    max_retries: int = 3
    initial_delay: float = 0.1
    max_delay: float = 10.0
    exponential_base: float = 2.0
    
    def get_delay(self, attempt: int) -> float:
        """Calculate delay for given attempt"""
        delay = self.initial_delay * (self.exponential_base ** attempt)
        return min(delay, self.max_delay)


class IPCClient:
    """Async IPC client for MixOS services with auto-reconnect and heartbeat"""

    def __init__(
        self,
        service_name: str,
        socket_path: str = "/run/mixos/ipc.sock",
        config: Optional[ClientConfig] = None,
    ):
        self.service_name = service_name
        self.socket_path = socket_path
        self.config = config or ClientConfig(socket_path=socket_path)
        
        self.reader: Optional[asyncio.StreamReader] = None
        self.writer: Optional[asyncio.StreamWriter] = None
        self.msg_counter = 0
        self.pending_requests: Dict[int, asyncio.Future] = {}
        self.handlers: Dict[str, Callable] = {}
        self._running = False
        self._receive_task: Optional[asyncio.Task] = None
        self._heartbeat_task: Optional[asyncio.Task] = None
        self._reconnect_task: Optional[asyncio.Task] = None
        
        # Connection state
        self._state = ConnectionState.DISCONNECTED
        self._state_lock = asyncio.Lock()
        self._last_heartbeat: Optional[float] = None
        self._missed_heartbeats = 0
        
        # Retry configuration
        self._retry_config = RetryConfig(
            max_retries=self.config.max_retries,
            initial_delay=self.config.reconnect_delay,
            max_delay=self.config.max_reconnect_delay,
        )
        
        # Event callbacks
        self._on_connect_callbacks: List[Callable] = []
        self._on_disconnect_callbacks: List[Callable] = []

    @property
    def state(self) -> ConnectionState:
        """Current connection state"""
        return self._state

    @property
    def is_connected(self) -> bool:
        """Check if connected"""
        return self._state == ConnectionState.CONNECTED

    def on_connect(self, callback: Callable):
        """Register callback for connection events"""
        self._on_connect_callbacks.append(callback)

    def on_disconnect(self, callback: Callable):
        """Register callback for disconnection events"""
        self._on_disconnect_callbacks.append(callback)

    async def _set_state(self, state: ConnectionState):
        """Set connection state"""
        async with self._state_lock:
            old_state = self._state
            self._state = state
            
            if old_state != state:
                logger.info(f"Connection state: {old_state.value} -> {state.value}")
                
                if state == ConnectionState.CONNECTED:
                    for callback in self._on_connect_callbacks:
                        try:
                            if asyncio.iscoroutinefunction(callback):
                                await callback()
                            else:
                                callback()
                        except Exception as e:
                            logger.error(f"Connect callback error: {e}")
                            
                elif state == ConnectionState.DISCONNECTED:
                    for callback in self._on_disconnect_callbacks:
                        try:
                            if asyncio.iscoroutinefunction(callback):
                                await callback()
                            else:
                                callback()
                        except Exception as e:
                            logger.error(f"Disconnect callback error: {e}")

    async def connect(self, max_retries: int = 30, retry_delay: float = 1.0) -> bool:
        """Connect to the IPC broker with retry"""
        await self._set_state(ConnectionState.CONNECTING)
        
        for attempt in range(max_retries):
            try:
                # Connect with timeout
                self.reader, self.writer = await asyncio.wait_for(
                    asyncio.open_unix_connection(self.socket_path),
                    timeout=self.config.connect_timeout
                )
                logger.info(f"Connected to broker at {self.socket_path}")
                
                # Register with broker
                await self._register()
                
                # Start background tasks
                self._running = True
                self._receive_task = asyncio.create_task(self._receive_loop())
                self._heartbeat_task = asyncio.create_task(self._heartbeat_loop())
                
                await self._set_state(ConnectionState.CONNECTED)
                return True
                
            except asyncio.TimeoutError:
                logger.warning(f"Connection attempt {attempt + 1} timed out")
            except (ConnectionRefusedError, FileNotFoundError, OSError) as e:
                logger.warning(f"Connection attempt {attempt + 1} failed: {e}")
            
            if attempt < max_retries - 1:
                delay = self._retry_config.get_delay(attempt)
                await asyncio.sleep(delay)
        
        await self._set_state(ConnectionState.DISCONNECTED)
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
            self._last_heartbeat = time.time()

    async def _heartbeat_loop(self):
        """Send periodic heartbeats to the server"""
        while self._running:
            try:
                await asyncio.sleep(self.config.heartbeat_interval)
                
                if not self._running or self._state != ConnectionState.CONNECTED:
                    break
                
                # Send ping
                try:
                    response = await self.call_with_timeout(
                        "init", "ping", b"ping",
                        timeout=self.config.heartbeat_timeout
                    )
                    if response:
                        self._last_heartbeat = time.time()
                        self._missed_heartbeats = 0
                    else:
                        self._missed_heartbeats += 1
                except Exception as e:
                    logger.warning(f"Heartbeat failed: {e}")
                    self._missed_heartbeats += 1
                
                # Check for too many missed heartbeats
                if self._missed_heartbeats >= 3:
                    logger.error("Too many missed heartbeats, reconnecting...")
                    await self._handle_disconnect()
                    
            except asyncio.CancelledError:
                break
            except Exception as e:
                logger.error(f"Heartbeat loop error: {e}")

    async def _handle_disconnect(self):
        """Handle disconnection and attempt reconnection"""
        if self._state == ConnectionState.RECONNECTING:
            return
            
        await self._set_state(ConnectionState.RECONNECTING)
        
        # Cancel pending requests
        for msg_id, future in list(self.pending_requests.items()):
            if not future.done():
                future.set_exception(ConnectionError("Disconnected"))
            del self.pending_requests[msg_id]
        
        # Close existing connection
        if self.writer:
            try:
                self.writer.close()
                await self.writer.wait_closed()
            except Exception:
                pass
            self.writer = None
            self.reader = None
        
        # Attempt reconnection
        self._reconnect_task = asyncio.create_task(self._reconnect())

    async def _reconnect(self):
        """Attempt to reconnect with exponential backoff"""
        attempt = 0
        while self._running:
            try:
                delay = self._retry_config.get_delay(attempt)
                logger.info(f"Reconnection attempt {attempt + 1} in {delay:.1f}s...")
                await asyncio.sleep(delay)
                
                # Try to connect
                self.reader, self.writer = await asyncio.wait_for(
                    asyncio.open_unix_connection(self.socket_path),
                    timeout=self.config.connect_timeout
                )
                
                # Re-register
                await self._register()
                
                # Restart receive loop
                if self._receive_task:
                    self._receive_task.cancel()
                self._receive_task = asyncio.create_task(self._receive_loop())
                
                await self._set_state(ConnectionState.CONNECTED)
                self._missed_heartbeats = 0
                logger.info("Reconnected successfully")
                return
                
            except asyncio.CancelledError:
                break
            except Exception as e:
                logger.warning(f"Reconnection attempt {attempt + 1} failed: {e}")
                attempt += 1
        
        await self._set_state(ConnectionState.DISCONNECTED)

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
            
            # Sanity check on message size (max 16MB)
            if length > 16 * 1024 * 1024:
                logger.error(f"Message too large: {length} bytes")
                return None
            
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
            try:
                msg = await self._read_message()
                if msg is None:
                    if self._running:
                        logger.error("Connection lost")
                        await self._handle_disconnect()
                    break
                
                # Handle ping (heartbeat from server)
                if msg.method == "ping":
                    await self._send_pong(msg)
                    continue
                
                # Check if it's a response to a pending request
                if msg.msg_type == MessageType.RESPONSE and msg.msg_id in self.pending_requests:
                    future = self.pending_requests.pop(msg.msg_id)
                    if not future.done():
                        future.set_result(msg)
                # Check if it's a request for us
                elif msg.msg_type == MessageType.REQUEST and msg.target == self.service_name:
                    asyncio.create_task(self._handle_request(msg))
                # Check if it's an event
                elif msg.msg_type == MessageType.EVENT:
                    asyncio.create_task(self._handle_event(msg))
                    
            except asyncio.CancelledError:
                break
            except Exception as e:
                logger.error(f"Receive loop error: {e}")
                if self._running:
                    await self._handle_disconnect()
                break

    async def _send_pong(self, ping_msg: IPCMessage):
        """Respond to a ping with a pong"""
        pong = IPCMessage(
            msg_type=MessageType.RESPONSE,
            msg_id=ping_msg.msg_id,
            source=self.service_name,
            target=ping_msg.source,
            method="pong",
            payload=b"pong",
        )
        try:
            await self._send_message(pong)
        except Exception as e:
            logger.error(f"Failed to send pong: {e}")

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
        return await self.call_with_timeout(target, method, payload, timeout)

    async def call_with_timeout(
        self,
        target: str,
        method: str,
        payload: Any,
        timeout: float = 30.0,
    ) -> Optional[IPCMessage]:
        """Make an RPC call with explicit timeout"""
        if not self.is_connected:
            raise ConnectionError("Not connected")
        
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
        loop = asyncio.get_running_loop()
        future = loop.create_future()
        self.pending_requests[msg_id] = future
        
        try:
            await self._send_message(msg)
            response = await asyncio.wait_for(future, timeout=timeout)
            return response
        except asyncio.TimeoutError:
            self.pending_requests.pop(msg_id, None)
            logger.error(f"Request timeout: {target}.{method}")
            return None
        except ConnectionError:
            self.pending_requests.pop(msg_id, None)
            raise

    async def call_with_retry(
        self,
        target: str,
        method: str,
        payload: Any,
        timeout: float = 30.0,
        max_retries: int = 3,
    ) -> Optional[IPCMessage]:
        """Make an RPC call with retry logic"""
        last_error = None
        
        for attempt in range(max_retries):
            try:
                return await self.call_with_timeout(target, method, payload, timeout)
            except ConnectionError as e:
                last_error = e
                if attempt < max_retries - 1:
                    delay = self._retry_config.get_delay(attempt)
                    logger.warning(f"Call failed, retrying in {delay:.1f}s: {e}")
                    await asyncio.sleep(delay)
            except Exception as e:
                last_error = e
                break
        
        if last_error:
            logger.error(f"Call failed after {max_retries} attempts: {last_error}")
        return None

    async def call_json(
        self,
        target: str,
        method: str,
        request: Any,
        response_type: type = dict,
        timeout: float = 30.0,
    ) -> Optional[Any]:
        """Make an RPC call with JSON serialization"""
        payload = json.dumps(request).encode('utf-8')
        response = await self.call_with_timeout(target, method, payload, timeout)
        
        if response is None:
            return None
        
        if response.error:
            raise Exception(f"Service error: {response.error}")
        
        try:
            data = json.loads(response.payload.decode('utf-8'))
            return data
        except json.JSONDecodeError as e:
            logger.error(f"Failed to decode response: {e}")
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
        
        # Cancel background tasks
        for task in [self._receive_task, self._heartbeat_task, self._reconnect_task]:
            if task:
                task.cancel()
                try:
                    await task
                except asyncio.CancelledError:
                    pass
        
        # Cancel pending requests
        for msg_id, future in list(self.pending_requests.items()):
            if not future.done():
                future.cancel()
        self.pending_requests.clear()
        
        # Close connection
        if self.writer:
            try:
                self.writer.close()
                await self.writer.wait_closed()
            except Exception:
                pass
            self.writer = None
            self.reader = None
        
        await self._set_state(ConnectionState.DISCONNECTED)

    def get_stats(self) -> Dict[str, Any]:
        """Get client statistics"""
        return {
            "state": self._state.value,
            "service_name": self.service_name,
            "pending_requests": len(self.pending_requests),
            "last_heartbeat": self._last_heartbeat,
            "missed_heartbeats": self._missed_heartbeats,
            "message_count": self.msg_counter,
        }


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
