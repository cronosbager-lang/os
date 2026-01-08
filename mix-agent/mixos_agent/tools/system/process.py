"""Process execution tools."""

import subprocess
from dataclasses import dataclass, field
from typing import List

from ..base import Tool, ToolParameter, SafetyLevel


@dataclass
class ExecuteCommandTool(Tool):
    """Tool for executing shell commands."""
    name: str = "execute_command"
    description: str = "Execute a shell command"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="command",
            type="string",
            description="Command to execute",
            required=True,
        ),
        ToolParameter(
            name="timeout",
            type="integer",
            description="Timeout in seconds",
            required=False,
            default=60,
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.REQUIRES_CONFIRMATION
    
    def execute(self, command: str, timeout: int = 60) -> str:
        """Execute a shell command."""
        try:
            result = subprocess.run(
                command,
                shell=True,
                capture_output=True,
                text=True,
                timeout=timeout,
            )
            
            output = ""
            if result.stdout:
                output += result.stdout
            if result.stderr:
                output += f"\n[stderr]\n{result.stderr}"
            
            if result.returncode != 0:
                output += f"\n[exit code: {result.returncode}]"
            
            return output.strip() or "(no output)"
        except subprocess.TimeoutExpired:
            return f"Command timed out after {timeout} seconds"
        except Exception as e:
            return f"Error executing command: {e}"
