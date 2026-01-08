"""Tool registry for managing available tools."""

from typing import Dict, List, Optional
from loguru import logger

from .base import Tool
from .system.package import InstallPackageTool, RemovePackageTool
from .system.service import ServiceControlTool
from .system.file import ReadFileTool, WriteFileTool, CreateDirectoryTool
from .system.process import ExecuteCommandTool


class ToolRegistry:
    """Registry for managing available tools."""
    
    def __init__(self):
        self._tools: Dict[str, Tool] = {}
        self._register_default_tools()
    
    def _register_default_tools(self):
        """Register default system tools."""
        default_tools = [
            InstallPackageTool(),
            RemovePackageTool(),
            ServiceControlTool(),
            ReadFileTool(),
            WriteFileTool(),
            CreateDirectoryTool(),
            ExecuteCommandTool(),
        ]
        
        for tool in default_tools:
            self.register(tool)
    
    def register(self, tool: Tool):
        """Register a tool."""
        logger.debug(f"Registering tool: {tool.name}")
        self._tools[tool.name] = tool
    
    def unregister(self, name: str):
        """Unregister a tool."""
        if name in self._tools:
            del self._tools[name]
    
    def get_tool(self, name: str) -> Optional[Tool]:
        """Get a tool by name."""
        return self._tools.get(name)
    
    def list_tools(self) -> List[Tool]:
        """List all registered tools."""
        return list(self._tools.values())
    
    def get_tool_names(self) -> List[str]:
        """Get names of all registered tools."""
        return list(self._tools.keys())
