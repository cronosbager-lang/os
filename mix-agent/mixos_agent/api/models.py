"""API models for the AI agent."""

from typing import Optional, List, Dict, Any
from pydantic import BaseModel, Field
from enum import Enum


# Request models
class ChatRequest(BaseModel):
    """Chat request model."""
    message: str = Field(..., description="User message")
    session_id: Optional[str] = Field(None, description="Session ID for conversation continuity")
    stream: bool = Field(False, description="Enable streaming response")


class ExecuteRequest(BaseModel):
    """Task execution request model."""
    task: str = Field(..., description="Task description")
    auto_confirm: bool = Field(False, description="Auto-confirm dangerous operations")
    timeout: Optional[int] = Field(None, description="Task timeout in seconds")


class ToolExecuteRequest(BaseModel):
    """Tool execution request model."""
    parameters: Dict[str, Any] = Field(default_factory=dict, description="Tool parameters")


class ConfigUpdate(BaseModel):
    """Configuration update model."""
    model: Optional[str] = Field(None, description="Model name")
    temperature: Optional[float] = Field(None, ge=0, le=2, description="Temperature")
    max_tokens: Optional[int] = Field(None, gt=0, description="Max tokens")
    system_prompt: Optional[str] = Field(None, description="System prompt")


# Response models
class ChatResponse(BaseModel):
    """Chat response model."""
    response: str = Field(..., description="Assistant response")
    session_id: str = Field(..., description="Session ID")
    tool_calls: Optional[List[Dict[str, Any]]] = Field(None, description="Tool calls made")


class ExecuteStep(BaseModel):
    """Execution step model."""
    name: str = Field(..., description="Step name")
    status: str = Field(..., description="Step status")
    output: Optional[str] = Field(None, description="Step output")
    duration: Optional[float] = Field(None, description="Step duration in seconds")


class ExecuteResponse(BaseModel):
    """Task execution response model."""
    task_id: str = Field(..., description="Task ID")
    status: str = Field(..., description="Task status")
    message: Optional[str] = Field(None, description="Status message")
    steps: Optional[List[ExecuteStep]] = Field(None, description="Execution steps")
    error: Optional[str] = Field(None, description="Error message if failed")


class ToolParameter(BaseModel):
    """Tool parameter model."""
    name: str = Field(..., description="Parameter name")
    type: str = Field(..., description="Parameter type")
    description: str = Field(..., description="Parameter description")
    required: bool = Field(True, description="Whether parameter is required")
    default: Optional[Any] = Field(None, description="Default value")


class ToolInfo(BaseModel):
    """Tool information model."""
    name: str = Field(..., description="Tool name")
    description: str = Field(..., description="Tool description")
    parameters: List[Dict[str, Any]] = Field(default_factory=list, description="Tool parameters")
    safety_level: str = Field(..., description="Safety level")


class SessionInfo(BaseModel):
    """Session information model."""
    id: str = Field(..., description="Session ID")
    created_at: float = Field(..., description="Creation timestamp")
    updated_at: float = Field(..., description="Last update timestamp")
    message_count: int = Field(..., description="Number of messages")
    metadata: Optional[Dict[str, Any]] = Field(None, description="Session metadata")


class StatusResponse(BaseModel):
    """Agent status response model."""
    status: str = Field(..., description="Agent status")
    model: str = Field(..., description="Model name")
    uptime: float = Field(..., description="Uptime in seconds")
    memory_usage: float = Field(..., description="Memory usage in MB")
    active_sessions: int = Field(0, description="Number of active sessions")
    total_requests: int = Field(0, description="Total requests processed")


class ErrorResponse(BaseModel):
    """Error response model."""
    error: str = Field(..., description="Error message")
    code: Optional[str] = Field(None, description="Error code")
    details: Optional[Dict[str, Any]] = Field(None, description="Error details")


# Enum models
class TaskStatus(str, Enum):
    """Task status enum."""
    PENDING = "pending"
    RUNNING = "running"
    COMPLETED = "completed"
    FAILED = "failed"
    CANCELLED = "cancelled"


class SafetyLevel(str, Enum):
    """Safety level enum."""
    SAFE = "safe"
    REQUIRES_CONFIRMATION = "requires_confirmation"
    DANGEROUS = "dangerous"


# WebSocket models
class WSChatMessage(BaseModel):
    """WebSocket chat message model."""
    type: str = Field("chat", description="Message type")
    message: str = Field(..., description="User message")
    session_id: Optional[str] = Field(None, description="Session ID")


class WSChatResponse(BaseModel):
    """WebSocket chat response model."""
    type: str = Field("chat_response", description="Message type")
    response: str = Field(..., description="Assistant response")
    session_id: str = Field(..., description="Session ID")
    done: bool = Field(True, description="Whether response is complete")


class WSStreamChunk(BaseModel):
    """WebSocket stream chunk model."""
    type: str = Field("stream", description="Message type")
    chunk: str = Field(..., description="Response chunk")
    done: bool = Field(False, description="Whether stream is complete")


class WSError(BaseModel):
    """WebSocket error model."""
    type: str = Field("error", description="Message type")
    error: str = Field(..., description="Error message")
    code: Optional[str] = Field(None, description="Error code")
