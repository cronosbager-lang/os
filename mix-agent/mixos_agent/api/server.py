"""FastAPI server for the agent API."""

from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
from typing import Optional, List
import uuid

from ..core.agent import Agent
from ..config.loader import AgentConfig


class ChatRequest(BaseModel):
    message: str
    session_id: Optional[str] = None
    stream: bool = False


class ChatResponse(BaseModel):
    response: str
    session_id: str
    error: Optional[str] = None


class ExecuteRequest(BaseModel):
    task: str
    auto_confirm: bool = False


class ExecuteResponse(BaseModel):
    task_id: str
    status: str
    message: str
    steps: List[dict] = []
    error: Optional[str] = None


class StatusResponse(BaseModel):
    status: str
    model: str
    uptime: str
    memory_usage: str
    requests_ok: int


def create_app(config: AgentConfig, agent: Agent) -> FastAPI:
    """Create FastAPI application."""
    app = FastAPI(
        title="MIXOS Agent API",
        description="API for MIXOS AI Agent",
        version="1.0.0",
    )
    
    # Add CORS middleware
    if config.api.enable_cors:
        app.add_middleware(
            CORSMiddleware,
            allow_origins=["*"],
            allow_credentials=True,
            allow_methods=["*"],
            allow_headers=["*"],
        )
    
    @app.get("/api/status", response_model=StatusResponse)
    async def get_status():
        """Get agent status."""
        return agent.get_status()
    
    @app.post("/api/chat", response_model=ChatResponse)
    async def chat(request: ChatRequest):
        """Chat with the agent."""
        session_id = request.session_id or str(uuid.uuid4())
        
        try:
            response = agent.chat(request.message, session_id)
            return ChatResponse(
                response=response,
                session_id=session_id,
            )
        except Exception as e:
            return ChatResponse(
                response="",
                session_id=session_id,
                error=str(e),
            )
    
    @app.post("/api/execute", response_model=ExecuteResponse)
    async def execute(request: ExecuteRequest):
        """Execute an autonomous task."""
        try:
            result = agent.execute_task(request.task, request.auto_confirm)
            return ExecuteResponse(**result)
        except Exception as e:
            return ExecuteResponse(
                task_id="",
                status="failed",
                message="",
                error=str(e),
            )
    
    @app.get("/api/health")
    async def health():
        """Health check endpoint."""
        return {"status": "ok"}
    
    return app
