"""API routes for the AI agent."""

from typing import Optional, Dict, Any
from fastapi import APIRouter, HTTPException, Depends, BackgroundTasks
from fastapi.responses import StreamingResponse
from pydantic import BaseModel
import asyncio
import json

from .models import (
    ChatRequest, ChatResponse,
    ExecuteRequest, ExecuteResponse,
    StatusResponse, ToolInfo,
    SessionInfo, ConfigUpdate,
)


# Create routers
chat_router = APIRouter(prefix="/chat", tags=["chat"])
execute_router = APIRouter(prefix="/execute", tags=["execute"])
tools_router = APIRouter(prefix="/tools", tags=["tools"])
session_router = APIRouter(prefix="/sessions", tags=["sessions"])
config_router = APIRouter(prefix="/config", tags=["config"])
health_router = APIRouter(prefix="/health", tags=["health"])


# Dependency to get agent instance
def get_agent():
    """Get the agent instance."""
    from ..agent import Agent
    # This would be injected by the application
    return Agent.get_instance()


# Chat routes
@chat_router.post("", response_model=ChatResponse)
async def chat(request: ChatRequest, agent=Depends(get_agent)):
    """Send a message to the agent."""
    try:
        response = await agent.chat(
            message=request.message,
            session_id=request.session_id,
        )
        
        return ChatResponse(
            response=response.content,
            session_id=response.session_id,
            tool_calls=response.tool_calls,
        )
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@chat_router.post("/stream")
async def chat_stream(request: ChatRequest, agent=Depends(get_agent)):
    """Stream a chat response."""
    async def generate():
        try:
            async for chunk in agent.chat_stream(
                message=request.message,
                session_id=request.session_id,
            ):
                yield f"data: {json.dumps({'content': chunk})}\n\n"
            yield "data: [DONE]\n\n"
        except Exception as e:
            yield f"data: {json.dumps({'error': str(e)})}\n\n"
    
    return StreamingResponse(
        generate(),
        media_type="text/event-stream",
    )


# Execute routes
@execute_router.post("", response_model=ExecuteResponse)
async def execute_task(
    request: ExecuteRequest,
    background_tasks: BackgroundTasks,
    agent=Depends(get_agent),
):
    """Execute an autonomous task."""
    try:
        # Start task execution
        task_id = await agent.start_task(
            task=request.task,
            auto_confirm=request.auto_confirm,
        )
        
        return ExecuteResponse(
            task_id=task_id,
            status="started",
            message=f"Task {task_id} started",
        )
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@execute_router.get("/{task_id}", response_model=ExecuteResponse)
async def get_task_status(task_id: str, agent=Depends(get_agent)):
    """Get task execution status."""
    try:
        status = await agent.get_task_status(task_id)
        
        if status is None:
            raise HTTPException(status_code=404, detail="Task not found")
        
        return ExecuteResponse(
            task_id=task_id,
            status=status.status,
            message=status.message,
            steps=status.steps,
        )
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@execute_router.post("/{task_id}/cancel")
async def cancel_task(task_id: str, agent=Depends(get_agent)):
    """Cancel a running task."""
    try:
        success = await agent.cancel_task(task_id)
        
        if not success:
            raise HTTPException(status_code=404, detail="Task not found or already completed")
        
        return {"status": "cancelled", "task_id": task_id}
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


# Tools routes
@tools_router.get("", response_model=list[ToolInfo])
async def list_tools(agent=Depends(get_agent)):
    """List available tools."""
    tools = agent.get_tools()
    return [
        ToolInfo(
            name=tool.name,
            description=tool.description,
            parameters=[p.__dict__ for p in tool.parameters],
            safety_level=tool.safety_level.value,
        )
        for tool in tools
    ]


@tools_router.get("/{tool_name}", response_model=ToolInfo)
async def get_tool(tool_name: str, agent=Depends(get_agent)):
    """Get tool information."""
    tool = agent.get_tool(tool_name)
    
    if tool is None:
        raise HTTPException(status_code=404, detail="Tool not found")
    
    return ToolInfo(
        name=tool.name,
        description=tool.description,
        parameters=[p.__dict__ for p in tool.parameters],
        safety_level=tool.safety_level.value,
    )


@tools_router.post("/{tool_name}/execute")
async def execute_tool(
    tool_name: str,
    parameters: Dict[str, Any],
    agent=Depends(get_agent),
):
    """Execute a tool directly."""
    tool = agent.get_tool(tool_name)
    
    if tool is None:
        raise HTTPException(status_code=404, detail="Tool not found")
    
    try:
        result = await agent.execute_tool(tool_name, parameters)
        return {"result": result}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


# Session routes
@session_router.get("", response_model=list[SessionInfo])
async def list_sessions(agent=Depends(get_agent)):
    """List all sessions."""
    sessions = agent.list_sessions()
    return [
        SessionInfo(
            id=s["id"],
            created_at=s["created_at"],
            updated_at=s["updated_at"],
            message_count=s["message_count"],
        )
        for s in sessions
    ]


@session_router.get("/{session_id}", response_model=SessionInfo)
async def get_session(session_id: str, agent=Depends(get_agent)):
    """Get session information."""
    session = agent.get_session(session_id)
    
    if session is None:
        raise HTTPException(status_code=404, detail="Session not found")
    
    return SessionInfo(
        id=session.id,
        created_at=session.created_at,
        updated_at=session.updated_at,
        message_count=len(session.context.messages),
    )


@session_router.delete("/{session_id}")
async def delete_session(session_id: str, agent=Depends(get_agent)):
    """Delete a session."""
    success = agent.delete_session(session_id)
    
    if not success:
        raise HTTPException(status_code=404, detail="Session not found")
    
    return {"status": "deleted", "session_id": session_id}


@session_router.get("/{session_id}/history")
async def get_session_history(
    session_id: str,
    limit: int = 50,
    agent=Depends(get_agent),
):
    """Get session conversation history."""
    history = agent.get_session_history(session_id, limit=limit)
    return {"history": history}


# Config routes
@config_router.get("")
async def get_config(agent=Depends(get_agent)):
    """Get agent configuration."""
    return agent.get_config()


@config_router.patch("")
async def update_config(update: ConfigUpdate, agent=Depends(get_agent)):
    """Update agent configuration."""
    try:
        agent.update_config(update.dict(exclude_unset=True))
        return {"status": "updated"}
    except Exception as e:
        raise HTTPException(status_code=400, detail=str(e))


# Health routes
@health_router.get("")
async def health_check():
    """Health check endpoint."""
    return {"status": "healthy"}


@health_router.get("/status", response_model=StatusResponse)
async def get_status(agent=Depends(get_agent)):
    """Get detailed agent status."""
    status = agent.get_status()
    
    return StatusResponse(
        status=status["status"],
        model=status.get("model", "unknown"),
        uptime=status.get("uptime", 0),
        memory_usage=status.get("memory_usage", 0),
        active_sessions=status.get("active_sessions", 0),
        total_requests=status.get("total_requests", 0),
    )


@health_router.get("/metrics")
async def get_metrics(agent=Depends(get_agent)):
    """Get agent metrics."""
    return agent.get_metrics()


# Create main router
def create_api_router() -> APIRouter:
    """Create the main API router."""
    router = APIRouter(prefix="/api")
    
    router.include_router(chat_router)
    router.include_router(execute_router)
    router.include_router(tools_router)
    router.include_router(session_router)
    router.include_router(config_router)
    router.include_router(health_router)
    
    return router
