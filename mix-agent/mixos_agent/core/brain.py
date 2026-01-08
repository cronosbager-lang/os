"""Brain module - handles reasoning and decision making."""

from typing import List, Dict, Any, Optional, Tuple
import re
import json
from loguru import logger

from ..inference.engine import InferenceEngine
from ..tools.registry import ToolRegistry
from ..memory.manager import MemoryManager


SYSTEM_PROMPT = """You are Mix Agent, an AI assistant integrated into MIXOS GO operating system.

Your capabilities:
- System administration (package management, service control, file operations)
- Development environment setup
- Docker container management
- Troubleshooting and diagnostics

When you need to perform an action, use tool calls in this format:
<tool_call>tool_name</tool_call>
<tool_params>{"param": "value"}</tool_params>

Available tools:
{tools}

Guidelines:
- Be concise and helpful
- Explain what you're doing before executing commands
- Ask for confirmation before destructive operations
- Provide clear error messages and suggestions
"""


class Brain:
    """Handles reasoning and decision making."""
    
    def __init__(
        self,
        inference: InferenceEngine,
        tools: ToolRegistry,
        memory: MemoryManager,
    ):
        self.inference = inference
        self.tools = tools
        self.memory = memory
        
        # Build system prompt with available tools
        tools_desc = self._build_tools_description()
        self.system_prompt = SYSTEM_PROMPT.format(tools=tools_desc)
    
    def _build_tools_description(self) -> str:
        """Build description of available tools."""
        descriptions = []
        for tool in self.tools.list_tools():
            desc = f"- {tool.name}: {tool.description}"
            if tool.parameters:
                params = ", ".join(f"{p.name}: {p.type}" for p in tool.parameters)
                desc += f"\n  Parameters: {params}"
            descriptions.append(desc)
        return "\n".join(descriptions)
    
    def think(self, context: List[Dict[str, str]]) -> str:
        """Process context and generate response."""
        # Build messages for inference
        messages = [{"role": "system", "content": self.system_prompt}]
        messages.extend(context)
        
        # Generate response
        response = self.inference.generate(messages)
        
        return response
    
    def has_tool_call(self, response: str) -> bool:
        """Check if response contains a tool call."""
        return "<tool_call>" in response and "</tool_call>" in response
    
    def parse_tool_call(self, response: str) -> Tuple[str, Dict[str, Any]]:
        """Parse tool call from response."""
        # Extract tool name
        tool_match = re.search(r"<tool_call>(\w+)</tool_call>", response)
        if not tool_match:
            raise ValueError("No tool call found in response")
        
        tool_name = tool_match.group(1)
        
        # Extract parameters
        params_match = re.search(r"<tool_params>(.*?)</tool_params>", response, re.DOTALL)
        if params_match:
            try:
                params = json.loads(params_match.group(1))
            except json.JSONDecodeError:
                params = {}
        else:
            params = {}
        
        return tool_name, params
    
    def classify_intent(self, message: str) -> Dict[str, Any]:
        """Classify the intent of a user message."""
        # Simple keyword-based classification
        # In production, this would use the model
        
        message_lower = message.lower()
        
        intents = {
            "install": ["install", "add", "get"],
            "remove": ["remove", "uninstall", "delete"],
            "update": ["update", "upgrade"],
            "search": ["search", "find", "look for"],
            "status": ["status", "check", "show"],
            "help": ["help", "how to", "what is"],
            "setup": ["setup", "configure", "create"],
            "docker": ["docker", "container"],
            "service": ["service", "systemctl", "start", "stop", "restart"],
        }
        
        for intent, keywords in intents.items():
            if any(kw in message_lower for kw in keywords):
                return {
                    "intent": intent,
                    "confidence": 0.8,
                    "entities": self._extract_entities(message),
                }
        
        return {
            "intent": "general",
            "confidence": 0.5,
            "entities": {},
        }
    
    def _extract_entities(self, message: str) -> Dict[str, Any]:
        """Extract entities from message."""
        entities = {}
        
        # Extract package names (simple heuristic)
        words = message.split()
        for i, word in enumerate(words):
            if word.lower() in ["install", "remove", "search"]:
                if i + 1 < len(words):
                    entities["package"] = words[i + 1]
        
        return entities
