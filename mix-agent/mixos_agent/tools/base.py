"""Base classes for tools."""

from abc import ABC, abstractmethod
from dataclasses import dataclass, field
from typing import List, Any, Optional
from enum import Enum


class SafetyLevel(Enum):
    SAFE = "safe"
    REQUIRES_CONFIRMATION = "requires_confirmation"
    DANGEROUS = "dangerous"


@dataclass
class ToolParameter:
    """Represents a tool parameter."""
    name: str
    type: str
    description: str
    required: bool = True
    default: Any = None


@dataclass
class Tool(ABC):
    """Base class for all tools."""
    name: str
    description: str
    parameters: List[ToolParameter] = field(default_factory=list)
    safety_level: SafetyLevel = SafetyLevel.SAFE
    
    @abstractmethod
    def execute(self, **kwargs) -> str:
        """Execute the tool with given parameters."""
        pass
    
    def validate_params(self, **kwargs) -> bool:
        """Validate that required parameters are provided."""
        for param in self.parameters:
            if param.required and param.name not in kwargs:
                return False
        return True
    
    def to_dict(self) -> dict:
        """Convert tool to dictionary for JSON schema."""
        return {
            "name": self.name,
            "description": self.description,
            "parameters": {
                "type": "object",
                "properties": {
                    p.name: {
                        "type": p.type,
                        "description": p.description,
                    }
                    for p in self.parameters
                },
                "required": [p.name for p in self.parameters if p.required],
            },
            "safety_level": self.safety_level.value,
        }
