"""Session management for the AI agent."""

from typing import Dict, Any, Optional, List
from dataclasses import dataclass, field
import json
import os
import time
import uuid
from pathlib import Path

from .context import ContextWindow, Message


@dataclass
class Session:
    """Represents a conversation session."""
    id: str
    created_at: float
    updated_at: float
    context: ContextWindow
    metadata: Dict[str, Any] = field(default_factory=dict)
    
    def to_dict(self) -> Dict[str, Any]:
        return {
            "id": self.id,
            "created_at": self.created_at,
            "updated_at": self.updated_at,
            "context": json.loads(self.context.to_json()),
            "metadata": self.metadata,
        }
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "Session":
        context = ContextWindow.from_json(json.dumps(data["context"]))
        return cls(
            id=data["id"],
            created_at=data["created_at"],
            updated_at=data["updated_at"],
            context=context,
            metadata=data.get("metadata", {}),
        )


class SessionManager:
    """Manages conversation sessions."""
    
    def __init__(
        self,
        storage_path: str = None,
        max_sessions: int = 100,
        session_timeout: int = 3600 * 24,  # 24 hours
        default_max_tokens: int = 4096,
        default_system_prompt: str = "",
    ):
        self.storage_path = storage_path or os.path.expanduser("~/.mixos/agent/sessions")
        self.max_sessions = max_sessions
        self.session_timeout = session_timeout
        self.default_max_tokens = default_max_tokens
        self.default_system_prompt = default_system_prompt
        
        self.sessions: Dict[str, Session] = {}
        
        # Create storage directory
        os.makedirs(self.storage_path, exist_ok=True)
        
        # Load existing sessions
        self._load_sessions()
    
    def create_session(
        self,
        session_id: str = None,
        system_prompt: str = None,
        max_tokens: int = None,
        metadata: Dict[str, Any] = None,
    ) -> Session:
        """Create a new session."""
        if session_id is None:
            session_id = str(uuid.uuid4())
        
        if session_id in self.sessions:
            return self.sessions[session_id]
        
        context = ContextWindow(
            max_tokens=max_tokens or self.default_max_tokens,
            system_prompt=system_prompt or self.default_system_prompt,
        )
        
        now = time.time()
        session = Session(
            id=session_id,
            created_at=now,
            updated_at=now,
            context=context,
            metadata=metadata or {},
        )
        
        self.sessions[session_id] = session
        
        # Cleanup old sessions if needed
        self._cleanup_old_sessions()
        
        return session
    
    def get_session(self, session_id: str) -> Optional[Session]:
        """Get a session by ID."""
        session = self.sessions.get(session_id)
        
        if session:
            # Check if session has expired
            if time.time() - session.updated_at > self.session_timeout:
                self.delete_session(session_id)
                return None
            
            # Update last access time
            session.updated_at = time.time()
        
        return session
    
    def get_or_create_session(
        self,
        session_id: str = None,
        **kwargs,
    ) -> Session:
        """Get existing session or create new one."""
        if session_id:
            session = self.get_session(session_id)
            if session:
                return session
        
        return self.create_session(session_id=session_id, **kwargs)
    
    def delete_session(self, session_id: str) -> bool:
        """Delete a session."""
        if session_id in self.sessions:
            del self.sessions[session_id]
            
            # Remove from disk
            session_file = os.path.join(self.storage_path, f"{session_id}.json")
            if os.path.exists(session_file):
                os.remove(session_file)
            
            return True
        return False
    
    def list_sessions(self) -> List[Dict[str, Any]]:
        """List all sessions."""
        return [
            {
                "id": s.id,
                "created_at": s.created_at,
                "updated_at": s.updated_at,
                "message_count": len(s.context.messages),
                "metadata": s.metadata,
            }
            for s in self.sessions.values()
        ]
    
    def save_session(self, session_id: str):
        """Save a session to disk."""
        session = self.sessions.get(session_id)
        if not session:
            return
        
        session_file = os.path.join(self.storage_path, f"{session_id}.json")
        with open(session_file, "w") as f:
            json.dump(session.to_dict(), f, indent=2)
    
    def save_all_sessions(self):
        """Save all sessions to disk."""
        for session_id in self.sessions:
            self.save_session(session_id)
    
    def _load_sessions(self):
        """Load sessions from disk."""
        if not os.path.exists(self.storage_path):
            return
        
        for filename in os.listdir(self.storage_path):
            if filename.endswith(".json"):
                filepath = os.path.join(self.storage_path, filename)
                try:
                    with open(filepath) as f:
                        data = json.load(f)
                    session = Session.from_dict(data)
                    
                    # Check if session has expired
                    if time.time() - session.updated_at <= self.session_timeout:
                        self.sessions[session.id] = session
                    else:
                        # Remove expired session file
                        os.remove(filepath)
                except Exception as e:
                    print(f"Error loading session {filename}: {e}")
    
    def _cleanup_old_sessions(self):
        """Remove old sessions if over limit."""
        if len(self.sessions) <= self.max_sessions:
            return
        
        # Sort by last update time
        sorted_sessions = sorted(
            self.sessions.items(),
            key=lambda x: x[1].updated_at,
        )
        
        # Remove oldest sessions
        to_remove = len(self.sessions) - self.max_sessions
        for session_id, _ in sorted_sessions[:to_remove]:
            self.delete_session(session_id)


class ConversationHistory:
    """Manages conversation history across sessions."""
    
    def __init__(self, storage_path: str = None):
        self.storage_path = storage_path or os.path.expanduser("~/.mixos/agent/history")
        os.makedirs(self.storage_path, exist_ok=True)
    
    def add_exchange(
        self,
        session_id: str,
        user_message: str,
        assistant_response: str,
        metadata: Dict[str, Any] = None,
    ):
        """Add a conversation exchange to history."""
        history_file = os.path.join(self.storage_path, f"{session_id}.jsonl")
        
        exchange = {
            "timestamp": time.time(),
            "user": user_message,
            "assistant": assistant_response,
            "metadata": metadata or {},
        }
        
        with open(history_file, "a") as f:
            f.write(json.dumps(exchange) + "\n")
    
    def get_history(
        self,
        session_id: str,
        limit: int = None,
    ) -> List[Dict[str, Any]]:
        """Get conversation history for a session."""
        history_file = os.path.join(self.storage_path, f"{session_id}.jsonl")
        
        if not os.path.exists(history_file):
            return []
        
        exchanges = []
        with open(history_file) as f:
            for line in f:
                if line.strip():
                    exchanges.append(json.loads(line))
        
        if limit:
            exchanges = exchanges[-limit:]
        
        return exchanges
    
    def search_history(
        self,
        query: str,
        session_id: str = None,
        limit: int = 10,
    ) -> List[Dict[str, Any]]:
        """Search conversation history."""
        results = []
        query_lower = query.lower()
        
        # Get files to search
        if session_id:
            files = [os.path.join(self.storage_path, f"{session_id}.jsonl")]
        else:
            files = [
                os.path.join(self.storage_path, f)
                for f in os.listdir(self.storage_path)
                if f.endswith(".jsonl")
            ]
        
        for filepath in files:
            if not os.path.exists(filepath):
                continue
            
            session = os.path.basename(filepath).replace(".jsonl", "")
            
            with open(filepath) as f:
                for line in f:
                    if line.strip():
                        exchange = json.loads(line)
                        
                        # Simple text search
                        if (query_lower in exchange.get("user", "").lower() or
                            query_lower in exchange.get("assistant", "").lower()):
                            exchange["session_id"] = session
                            results.append(exchange)
        
        # Sort by timestamp (most recent first)
        results.sort(key=lambda x: x.get("timestamp", 0), reverse=True)
        
        return results[:limit]
    
    def clear_history(self, session_id: str = None):
        """Clear conversation history."""
        if session_id:
            history_file = os.path.join(self.storage_path, f"{session_id}.jsonl")
            if os.path.exists(history_file):
                os.remove(history_file)
        else:
            for filename in os.listdir(self.storage_path):
                if filename.endswith(".jsonl"):
                    os.remove(os.path.join(self.storage_path, filename))
