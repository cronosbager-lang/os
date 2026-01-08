"""Executor module - handles command and tool execution."""

from typing import Dict, Any
from dataclasses import dataclass
import subprocess
from loguru import logger

from ..tools.registry import ToolRegistry
from ..safety.validator import SafetyValidator


@dataclass
class ExecutionStep:
    """Represents a single execution step."""
    name: str
    action: str
    tool: str
    params: Dict[str, Any]
    requires_confirmation: bool = False


class Executor:
    """Handles execution of tools and commands."""
    
    def __init__(self, tools: ToolRegistry, safety: SafetyValidator):
        self.tools = tools
        self.safety = safety
    
    def execute(self, step: ExecutionStep) -> str:
        """Execute a single step."""
        logger.info(f"Executing step: {step.name}")
        
        # Validate safety
        if not self.safety.is_safe(step.action):
            raise ValueError(f"Action '{step.action}' is not allowed")
        
        # Execute tool
        return self.execute_tool(step.tool, step.params)
    
    def execute_tool(self, tool_name: str, params: Dict[str, Any]) -> str:
        """Execute a tool by name."""
        tool = self.tools.get_tool(tool_name)
        if not tool:
            raise ValueError(f"Tool '{tool_name}' not found")
        
        logger.debug(f"Executing tool: {tool_name} with params: {params}")
        
        try:
            result = tool.execute(**params)
            return result
        except Exception as e:
            logger.error(f"Tool execution failed: {e}")
            raise
    
    def execute_command(
        self,
        command: str,
        timeout: int = 60,
        capture_output: bool = True,
    ) -> Dict[str, Any]:
        """Execute a shell command."""
        # Safety check
        if not self.safety.is_command_safe(command):
            raise ValueError(f"Command is not allowed: {command}")
        
        logger.debug(f"Executing command: {command}")
        
        try:
            result = subprocess.run(
                command,
                shell=True,
                capture_output=capture_output,
                text=True,
                timeout=timeout,
            )
            
            return {
                "returncode": result.returncode,
                "stdout": result.stdout,
                "stderr": result.stderr,
                "success": result.returncode == 0,
            }
        except subprocess.TimeoutExpired:
            return {
                "returncode": -1,
                "stdout": "",
                "stderr": "Command timed out",
                "success": False,
            }
        except Exception as e:
            return {
                "returncode": -1,
                "stdout": "",
                "stderr": str(e),
                "success": False,
            }
