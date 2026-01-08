"""Context window management for the AI agent."""

from typing import List, Dict, Any, Optional
from dataclasses import dataclass, field
import json
import time


@dataclass
class Message:
    """Represents a conversation message."""
    role: str  # "system", "user", "assistant", "tool"
    content: str
    timestamp: float = field(default_factory=time.time)
    metadata: Dict[str, Any] = field(default_factory=dict)
    
    def to_dict(self) -> Dict[str, Any]:
        return {
            "role": self.role,
            "content": self.content,
            "timestamp": self.timestamp,
            "metadata": self.metadata,
        }
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "Message":
        return cls(
            role=data["role"],
            content=data["content"],
            timestamp=data.get("timestamp", time.time()),
            metadata=data.get("metadata", {}),
        )
    
    def token_estimate(self) -> int:
        """Estimate token count (rough approximation)."""
        # Rough estimate: ~4 characters per token
        return len(self.content) // 4 + 1


class ContextWindow:
    """Manages the context window for LLM inference."""
    
    def __init__(
        self,
        max_tokens: int = 4096,
        system_prompt: str = "",
        reserve_tokens: int = 512,  # Reserve for response
    ):
        self.max_tokens = max_tokens
        self.reserve_tokens = reserve_tokens
        self.system_prompt = system_prompt
        self.messages: List[Message] = []
        
        # Add system prompt as first message
        if system_prompt:
            self.messages.append(Message(
                role="system",
                content=system_prompt,
            ))
    
    @property
    def available_tokens(self) -> int:
        """Get available tokens for new content."""
        used = sum(m.token_estimate() for m in self.messages)
        return self.max_tokens - used - self.reserve_tokens
    
    @property
    def token_count(self) -> int:
        """Get current token count."""
        return sum(m.token_estimate() for m in self.messages)
    
    def add_message(self, role: str, content: str, metadata: Dict[str, Any] = None) -> bool:
        """Add a message to the context."""
        message = Message(
            role=role,
            content=content,
            metadata=metadata or {},
        )
        
        # Check if we need to truncate
        while self.available_tokens < message.token_estimate() and len(self.messages) > 1:
            self._truncate_oldest()
        
        self.messages.append(message)
        return True
    
    def add_user_message(self, content: str) -> bool:
        """Add a user message."""
        return self.add_message("user", content)
    
    def add_assistant_message(self, content: str) -> bool:
        """Add an assistant message."""
        return self.add_message("assistant", content)
    
    def add_tool_result(self, content: str, tool_name: str = None) -> bool:
        """Add a tool result."""
        return self.add_message("tool", content, {"tool_name": tool_name})
    
    def _truncate_oldest(self):
        """Remove the oldest non-system message."""
        for i, msg in enumerate(self.messages):
            if msg.role != "system":
                self.messages.pop(i)
                return
    
    def get_messages(self) -> List[Dict[str, str]]:
        """Get messages in format suitable for LLM."""
        return [{"role": m.role, "content": m.content} for m in self.messages]
    
    def get_last_n_messages(self, n: int) -> List[Message]:
        """Get the last N messages."""
        return self.messages[-n:] if n < len(self.messages) else self.messages
    
    def clear(self, keep_system: bool = True):
        """Clear the context."""
        if keep_system and self.messages and self.messages[0].role == "system":
            self.messages = [self.messages[0]]
        else:
            self.messages = []
    
    def summarize(self) -> str:
        """Generate a summary of the conversation."""
        if len(self.messages) <= 2:
            return ""
        
        # Simple summary: extract key points
        user_messages = [m.content for m in self.messages if m.role == "user"]
        assistant_messages = [m.content for m in self.messages if m.role == "assistant"]
        
        summary = "Previous conversation summary:\n"
        summary += f"- {len(user_messages)} user messages\n"
        summary += f"- {len(assistant_messages)} assistant responses\n"
        
        # Include last few exchanges
        if user_messages:
            summary += f"- Last user request: {user_messages[-1][:100]}...\n"
        
        return summary
    
    def compress(self, target_tokens: int = None):
        """Compress context to fit within target tokens."""
        if target_tokens is None:
            target_tokens = self.max_tokens // 2
        
        while self.token_count > target_tokens and len(self.messages) > 2:
            self._truncate_oldest()
    
    def to_json(self) -> str:
        """Serialize context to JSON."""
        return json.dumps({
            "max_tokens": self.max_tokens,
            "reserve_tokens": self.reserve_tokens,
            "system_prompt": self.system_prompt,
            "messages": [m.to_dict() for m in self.messages],
        })
    
    @classmethod
    def from_json(cls, data: str) -> "ContextWindow":
        """Deserialize context from JSON."""
        obj = json.loads(data)
        context = cls(
            max_tokens=obj["max_tokens"],
            system_prompt="",  # Will be added from messages
            reserve_tokens=obj["reserve_tokens"],
        )
        context.messages = [Message.from_dict(m) for m in obj["messages"]]
        return context


class SlidingWindowContext(ContextWindow):
    """Context window with sliding window strategy."""
    
    def __init__(
        self,
        max_tokens: int = 4096,
        system_prompt: str = "",
        window_size: int = 10,  # Number of recent messages to keep
    ):
        super().__init__(max_tokens, system_prompt)
        self.window_size = window_size
    
    def add_message(self, role: str, content: str, metadata: Dict[str, Any] = None) -> bool:
        """Add message with sliding window."""
        message = Message(
            role=role,
            content=content,
            metadata=metadata or {},
        )
        
        self.messages.append(message)
        
        # Keep only window_size recent messages (plus system)
        non_system = [m for m in self.messages if m.role != "system"]
        if len(non_system) > self.window_size:
            # Keep system message and last window_size messages
            system_msgs = [m for m in self.messages if m.role == "system"]
            recent = non_system[-self.window_size:]
            self.messages = system_msgs + recent
        
        return True


class SummarizingContext(ContextWindow):
    """Context window that summarizes old messages."""
    
    def __init__(
        self,
        max_tokens: int = 4096,
        system_prompt: str = "",
        summarize_threshold: int = 20,  # Summarize after this many messages
    ):
        super().__init__(max_tokens, system_prompt)
        self.summarize_threshold = summarize_threshold
        self.summaries: List[str] = []
    
    def add_message(self, role: str, content: str, metadata: Dict[str, Any] = None) -> bool:
        """Add message with automatic summarization."""
        result = super().add_message(role, content, metadata)
        
        # Check if we need to summarize
        non_system = [m for m in self.messages if m.role != "system"]
        if len(non_system) >= self.summarize_threshold:
            self._summarize_and_compress()
        
        return result
    
    def _summarize_and_compress(self):
        """Summarize older messages and compress context."""
        # Keep system message
        system_msgs = [m for m in self.messages if m.role == "system"]
        non_system = [m for m in self.messages if m.role != "system"]
        
        # Summarize older half
        mid = len(non_system) // 2
        to_summarize = non_system[:mid]
        to_keep = non_system[mid:]
        
        # Create summary
        summary = self._create_summary(to_summarize)
        self.summaries.append(summary)
        
        # Rebuild messages with summary
        summary_msg = Message(
            role="system",
            content=f"[Previous conversation summary: {summary}]",
            metadata={"is_summary": True},
        )
        
        self.messages = system_msgs + [summary_msg] + to_keep
    
    def _create_summary(self, messages: List[Message]) -> str:
        """Create a summary of messages."""
        # Simple extractive summary
        user_msgs = [m.content[:100] for m in messages if m.role == "user"]
        
        if not user_msgs:
            return "No significant conversation"
        
        return f"User discussed: {'; '.join(user_msgs[:3])}"
