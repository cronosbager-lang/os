"""Memory manager for conversation history."""

from typing import Dict, List, Optional
from collections import defaultdict
import json
from pathlib import Path
from loguru import logger

from ..config.loader import MemoryConfig


class MemoryManager:
    """Manages conversation history and long-term memory."""
    
    def __init__(self, config: MemoryConfig):
        self.config = config
        self.max_history = config.max_conversation_history
        
        # In-memory conversation storage
        self._conversations: Dict[str, List[Dict[str, str]]] = defaultdict(list)
        
        # Long-term memory (simple file-based for now)
        self._memory_path = Path(config.memory_db_path)
        self._memory_path.parent.mkdir(parents=True, exist_ok=True)
        
        self._load_memory()
    
    def _load_memory(self):
        """Load long-term memory from disk."""
        if self._memory_path.exists():
            try:
                with open(self._memory_path) as f:
                    data = json.load(f)
                    self._long_term = data.get("long_term", {})
            except Exception as e:
                logger.warning(f"Failed to load memory: {e}")
                self._long_term = {}
        else:
            self._long_term = {}
    
    def _save_memory(self):
        """Save long-term memory to disk."""
        try:
            with open(self._memory_path, "w") as f:
                json.dump({"long_term": self._long_term}, f)
        except Exception as e:
            logger.warning(f"Failed to save memory: {e}")
    
    def get_context(self, session_id: Optional[str] = None) -> List[Dict[str, str]]:
        """Get conversation context for a session."""
        if session_id is None:
            session_id = "default"
        
        return list(self._conversations[session_id])
    
    def add_message(
        self,
        session_id: Optional[str],
        role: str,
        content: str,
    ):
        """Add a message to conversation history."""
        if session_id is None:
            session_id = "default"
        
        self._conversations[session_id].append({
            "role": role,
            "content": content,
        })
        
        # Trim to max history
        if len(self._conversations[session_id]) > self.max_history:
            self._conversations[session_id] = \
                self._conversations[session_id][-self.max_history:]
    
    def clear_session(self, session_id: str):
        """Clear conversation history for a session."""
        if session_id in self._conversations:
            del self._conversations[session_id]
    
    def store_long_term(self, key: str, value: str):
        """Store information in long-term memory."""
        self._long_term[key] = value
        self._save_memory()
    
    def retrieve_long_term(self, key: str) -> Optional[str]:
        """Retrieve information from long-term memory."""
        return self._long_term.get(key)
    
    def search_long_term(self, query: str) -> List[str]:
        """Search long-term memory (simple keyword search)."""
        results = []
        query_lower = query.lower()
        
        for key, value in self._long_term.items():
            if query_lower in key.lower() or query_lower in value.lower():
                results.append(f"{key}: {value}")
        
        return results
