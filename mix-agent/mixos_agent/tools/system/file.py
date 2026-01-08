"""File operation tools."""

import os
from pathlib import Path
from dataclasses import dataclass, field
from typing import List

from ..base import Tool, ToolParameter, SafetyLevel


@dataclass
class ReadFileTool(Tool):
    """Tool for reading files."""
    name: str = "read_file"
    description: str = "Read contents of a file"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="path",
            type="string",
            description="Path to the file",
            required=True,
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.SAFE
    
    def execute(self, path: str) -> str:
        """Read a file."""
        try:
            file_path = Path(path).expanduser()
            
            if not file_path.exists():
                return f"File not found: {path}"
            
            if not file_path.is_file():
                return f"Not a file: {path}"
            
            # Limit file size
            if file_path.stat().st_size > 1024 * 1024:  # 1MB
                return "File too large to read (>1MB)"
            
            return file_path.read_text()
        except PermissionError:
            return f"Permission denied: {path}"
        except Exception as e:
            return f"Error reading file: {e}"


@dataclass
class WriteFileTool(Tool):
    """Tool for writing files."""
    name: str = "write_file"
    description: str = "Write content to a file"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="path",
            type="string",
            description="Path to the file",
            required=True,
        ),
        ToolParameter(
            name="content",
            type="string",
            description="Content to write",
            required=True,
        ),
        ToolParameter(
            name="append",
            type="boolean",
            description="Append to file instead of overwriting",
            required=False,
            default=False,
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.REQUIRES_CONFIRMATION
    
    def execute(self, path: str, content: str, append: bool = False) -> str:
        """Write to a file."""
        try:
            file_path = Path(path).expanduser()
            
            # Create parent directories if needed
            file_path.parent.mkdir(parents=True, exist_ok=True)
            
            mode = "a" if append else "w"
            with open(file_path, mode) as f:
                f.write(content)
            
            action = "Appended to" if append else "Wrote to"
            return f"{action} {path}"
        except PermissionError:
            return f"Permission denied: {path}"
        except Exception as e:
            return f"Error writing file: {e}"


@dataclass
class CreateDirectoryTool(Tool):
    """Tool for creating directories."""
    name: str = "create_directory"
    description: str = "Create a directory"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="path",
            type="string",
            description="Path to the directory",
            required=True,
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.SAFE
    
    def execute(self, path: str) -> str:
        """Create a directory."""
        try:
            dir_path = Path(path).expanduser()
            dir_path.mkdir(parents=True, exist_ok=True)
            return f"Created directory: {path}"
        except PermissionError:
            return f"Permission denied: {path}"
        except Exception as e:
            return f"Error creating directory: {e}"
