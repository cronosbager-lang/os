"""API models for the AI agent."""

from datetime import datetime
from typing import Optional, List, Dict, Any, Union
from pydantic import BaseModel, Field
from enum import Enum


# Enum models
class TaskStatus(str, Enum):
    """Task status enum."""
    PENDING = "pending"
    RUNNING = "running"
    IN_PROGRESS = "in_progress"
    COMPLETED = "completed"
    FAILED = "failed"
    CANCELLED = "cancelled"


class SafetyLevel(str, Enum):
    """Safety level enum."""
    SAFE = "safe"
    REQUIRES_CONFIRMATION = "requires_confirmation"
    DANGEROUS = "dangerous"


class MessageRole(str, Enum):
    """Message role enum."""
    USER = "user"
    ASSISTANT = "assistant"
    SYSTEM = "system"
    TOOL = "tool"


# Request models
class ChatRequest(BaseModel):
    """Chat request model."""
    message: str = Field(..., description="User message")
    session_id: Optional[str] = Field(None, description="Session ID for conversation continuity")
    stream: bool = Field(False, description="Enable streaming response")
    include_context: bool = Field(True, description="Include conversation context")
    max_tokens: int = Field(512, description="Maximum tokens in response")
    temperature: Optional[float] = Field(None, ge=0, le=2, description="Temperature override")


class ExecuteRequest(BaseModel):
    """Task execution request model."""
    task: str = Field(..., description="Task description")
    auto_confirm: bool = Field(False, description="Auto-confirm dangerous operations")
    timeout: int = Field(300, description="Task timeout in seconds")
    dry_run: bool = Field(False, description="Plan only, don't execute")


class ToolExecuteRequest(BaseModel):
    """Tool execution request model."""
    parameters: Dict[str, Any] = Field(default_factory=dict, description="Tool parameters")
    confirm: bool = Field(False, description="Confirm dangerous operations")


class ConfigUpdate(BaseModel):
    """Configuration update model."""
    model_path: Optional[str] = Field(None, description="Model path")
    temperature: Optional[float] = Field(None, ge=0, le=2, description="Temperature")
    top_p: Optional[float] = Field(None, ge=0, le=1, description="Top P")
    top_k: Optional[int] = Field(None, gt=0, description="Top K")
    max_tokens: Optional[int] = Field(None, gt=0, description="Max tokens")
    context_length: Optional[int] = Field(None, gt=0, description="Context length")
    threads: Optional[int] = Field(None, gt=0, description="Inference threads")
    system_prompt: Optional[str] = Field(None, description="System prompt")


# Response models
class ToolCallResult(BaseModel):
    """Tool call result model."""
    id: str = Field(..., description="Tool call ID")
    name: str = Field(..., description="Tool name")
    parameters: Dict[str, Any] = Field(default_factory=dict, description="Parameters used")
    result: Optional[str] = Field(None, description="Tool result")
    status: TaskStatus = Field(TaskStatus.COMPLETED, description="Call status")
    duration: float = Field(0.0, description="Execution duration in seconds")


class ChatResponse(BaseModel):
    """Chat response model."""
    response: str = Field(..., description="Assistant response")
    session_id: str = Field(..., description="Session ID")
    tool_calls: List[ToolCallResult] = Field(default_factory=list, description="Tool calls made")
    tokens_used: int = Field(0, description="Tokens used in response")
    processing_time: float = Field(0.0, description="Processing time in seconds")
    error: Optional[str] = Field(None, description="Error message if any")


class ExecuteStep(BaseModel):
    """Execution step model."""
    id: str = Field(..., description="Step ID")
    name: str = Field(..., description="Step name")
    description: str = Field("", description="Step description")
    status: TaskStatus = Field(TaskStatus.PENDING, description="Step status")
    output: str = Field("", description="Step output")
    started_at: Optional[datetime] = Field(None, description="Start time")
    completed_at: Optional[datetime] = Field(None, description="Completion time")
    duration: float = Field(0.0, description="Step duration in seconds")
    requires_confirmation: bool = Field(False, description="Requires user confirmation")


class ExecutePlan(BaseModel):
    """Execution plan model."""
    task_id: str = Field(..., description="Task ID")
    task: str = Field(..., description="Original task description")
    steps: List[ExecuteStep] = Field(default_factory=list, description="Planned steps")
    estimated_time: int = Field(0, description="Estimated time in seconds")
    risk_level: SafetyLevel = Field(SafetyLevel.SAFE, description="Overall risk level")


class ExecuteResponse(BaseModel):
    """Task execution response model."""
    task_id: str = Field(..., description="Task ID")
    status: TaskStatus = Field(..., description="Task status")
    message: str = Field("", description="Status message")
    plan: Optional[ExecutePlan] = Field(None, description="Execution plan")
    steps: List[ExecuteStep] = Field(default_factory=list, description="Execution steps")
    started_at: Optional[datetime] = Field(None, description="Start time")
    completed_at: Optional[datetime] = Field(None, description="Completion time")
    error: Optional[str] = Field(None, description="Error message if failed")


class ToolParameter(BaseModel):
    """Tool parameter model."""
    name: str = Field(..., description="Parameter name")
    type: str = Field(..., description="Parameter type")
    description: str = Field(..., description="Parameter description")
    required: bool = Field(True, description="Whether parameter is required")
    default: Optional[Any] = Field(None, description="Default value")
    enum: Optional[List[str]] = Field(None, description="Allowed values")


class ToolInfo(BaseModel):
    """Tool information model."""
    name: str = Field(..., description="Tool name")
    description: str = Field(..., description="Tool description")
    parameters: List[ToolParameter] = Field(default_factory=list, description="Tool parameters")
    safety_level: SafetyLevel = Field(SafetyLevel.SAFE, description="Safety level")
    examples: List[Dict[str, Any]] = Field(default_factory=list, description="Usage examples")


class ToolExecuteResponse(BaseModel):
    """Tool execution response model."""
    tool_name: str = Field(..., description="Tool name")
    result: str = Field(..., description="Execution result")
    success: bool = Field(True, description="Whether execution succeeded")
    execution_time: float = Field(0.0, description="Execution time in seconds")
    error: Optional[str] = Field(None, description="Error message if failed")


class ChatMessage(BaseModel):
    """Chat message model."""
    role: MessageRole = Field(..., description="Message role")
    content: str = Field(..., description="Message content")
    timestamp: datetime = Field(default_factory=datetime.utcnow, description="Message timestamp")
    tool_call_id: Optional[str] = Field(None, description="Associated tool call ID")


class SessionInfo(BaseModel):
    """Session information model."""
    id: str = Field(..., description="Session ID")
    created_at: datetime = Field(..., description="Creation timestamp")
    updated_at: datetime = Field(..., description="Last update timestamp")
    message_count: int = Field(0, description="Number of messages")
    last_message: Optional[str] = Field(None, description="Last message preview")
    metadata: Dict[str, Any] = Field(default_factory=dict, description="Session metadata")


class SessionHistory(BaseModel):
    """Session history model."""
    session_id: str = Field(..., description="Session ID")
    messages: List[ChatMessage] = Field(default_factory=list, description="Messages")
    total_count: int = Field(0, description="Total message count")


class SystemMetrics(BaseModel):
    """System metrics model."""
    cpu_percent: float = Field(0.0, description="CPU usage percentage")
    memory_percent: float = Field(0.0, description="Memory usage percentage")
    memory_used_mb: float = Field(0.0, description="Memory used in MB")
    memory_total_mb: float = Field(0.0, description="Total memory in MB")
    disk_percent: float = Field(0.0, description="Disk usage percentage")
    load_average: List[float] = Field(default_factory=list, description="Load average (1, 5, 15 min)")


class ModelInfo(BaseModel):
    """Model information model."""
    name: str = Field(..., description="Model name")
    path: str = Field(..., description="Model path")
    size_mb: float = Field(0.0, description="Model size in MB")
    context_length: int = Field(4096, description="Context length")
    loaded: bool = Field(False, description="Whether model is loaded")


class StatusResponse(BaseModel):
    """Agent status response model."""
    status: str = Field(..., description="Agent status")
    version: str = Field("1.0.0", description="Agent version")
    model: Union[str, ModelInfo] = Field(..., description="Model info")
    uptime: str = Field(..., description="Uptime string")
    uptime_seconds: float = Field(0.0, description="Uptime in seconds")
    memory_usage: str = Field(..., description="Memory usage string")
    memory_usage_mb: float = Field(0.0, description="Memory usage in MB")
    active_sessions: int = Field(0, description="Number of active sessions")
    total_requests: int = Field(0, description="Total requests processed")
    requests_per_minute: float = Field(0.0, description="Requests per minute")
    system_metrics: Optional[SystemMetrics] = Field(None, description="System metrics")


class HealthCheck(BaseModel):
    """Health check result model."""
    name: str = Field(..., description="Check name")
    status: str = Field(..., description="Check status")
    message: str = Field("", description="Check message")
    duration_ms: float = Field(0.0, description="Check duration in ms")


class HealthResponse(BaseModel):
    """Health response model."""
    status: str = Field(..., description="Overall health status")
    timestamp: datetime = Field(default_factory=datetime.utcnow, description="Check timestamp")
    checks: List[HealthCheck] = Field(default_factory=list, description="Individual checks")


class MetricsResponse(BaseModel):
    """Metrics response model."""
    uptime_seconds: float = Field(0.0, description="Uptime in seconds")
    total_requests: int = Field(0, description="Total requests")
    total_tokens: int = Field(0, description="Total tokens processed")
    average_response_time: float = Field(0.0, description="Average response time in seconds")
    active_sessions: int = Field(0, description="Active sessions")
    tool_calls: int = Field(0, description="Total tool calls")
    errors: int = Field(0, description="Total errors")
    system: SystemMetrics = Field(default_factory=SystemMetrics, description="System metrics")


class ErrorResponse(BaseModel):
    """Error response model."""
    error: str = Field(..., description="Error message")
    code: str = Field("UNKNOWN_ERROR", description="Error code")
    details: Optional[Dict[str, Any]] = Field(None, description="Error details")
    timestamp: datetime = Field(default_factory=datetime.utcnow, description="Error timestamp")


# WebSocket models
class WSMessageType(str, Enum):
    """WebSocket message type enum."""
    CHAT = "chat"
    CHAT_RESPONSE = "chat_response"
    STREAM = "stream"
    STREAM_END = "stream_end"
    EXECUTE = "execute"
    EXECUTE_STATUS = "execute_status"
    EXECUTE_STEP = "execute_step"
    EXECUTE_COMPLETE = "execute_complete"
    STATUS = "status"
    ERROR = "error"
    PING = "ping"
    PONG = "pong"


class WSMessage(BaseModel):
    """Base WebSocket message model."""
    type: WSMessageType = Field(..., description="Message type")
    data: Dict[str, Any] = Field(default_factory=dict, description="Message data")
    timestamp: datetime = Field(default_factory=datetime.utcnow, description="Message timestamp")


class WSChatMessage(BaseModel):
    """WebSocket chat message model."""
    type: WSMessageType = Field(WSMessageType.CHAT, description="Message type")
    message: str = Field(..., description="User message")
    session_id: Optional[str] = Field(None, description="Session ID")
    stream: bool = Field(True, description="Enable streaming")


class WSChatResponse(BaseModel):
    """WebSocket chat response model."""
    type: WSMessageType = Field(WSMessageType.CHAT_RESPONSE, description="Message type")
    response: str = Field(..., description="Assistant response")
    session_id: str = Field(..., description="Session ID")
    tool_calls: List[ToolCallResult] = Field(default_factory=list, description="Tool calls")
    done: bool = Field(True, description="Whether response is complete")


class WSStreamChunk(BaseModel):
    """WebSocket stream chunk model."""
    type: WSMessageType = Field(WSMessageType.STREAM, description="Message type")
    chunk: str = Field(..., description="Response chunk")
    session_id: str = Field(..., description="Session ID")
    done: bool = Field(False, description="Whether stream is complete")


class WSExecuteMessage(BaseModel):
    """WebSocket execute message model."""
    type: WSMessageType = Field(WSMessageType.EXECUTE, description="Message type")
    task: str = Field(..., description="Task to execute")
    auto_confirm: bool = Field(False, description="Auto-confirm dangerous operations")


class WSExecuteStatus(BaseModel):
    """WebSocket execute status model."""
    type: WSMessageType = Field(WSMessageType.EXECUTE_STATUS, description="Message type")
    task_id: str = Field(..., description="Task ID")
    status: TaskStatus = Field(..., description="Task status")
    message: str = Field("", description="Status message")
    progress: float = Field(0.0, description="Progress percentage")


class WSExecuteStep(BaseModel):
    """WebSocket execute step model."""
    type: WSMessageType = Field(WSMessageType.EXECUTE_STEP, description="Message type")
    task_id: str = Field(..., description="Task ID")
    step: ExecuteStep = Field(..., description="Step details")


class WSError(BaseModel):
    """WebSocket error model."""
    type: WSMessageType = Field(WSMessageType.ERROR, description="Message type")
    error: str = Field(..., description="Error message")
    code: str = Field("UNKNOWN_ERROR", description="Error code")
    details: Optional[Dict[str, Any]] = Field(None, description="Error details")


class WSPing(BaseModel):
    """WebSocket ping model."""
    type: WSMessageType = Field(WSMessageType.PING, description="Message type")
    timestamp: datetime = Field(default_factory=datetime.utcnow, description="Ping timestamp")


class WSPong(BaseModel):
    """WebSocket pong model."""
    type: WSMessageType = Field(WSMessageType.PONG, description="Message type")
    timestamp: datetime = Field(default_factory=datetime.utcnow, description="Pong timestamp")
    latency_ms: float = Field(0.0, description="Latency in milliseconds")
