"""Service management tools."""

import subprocess
from dataclasses import dataclass, field
from typing import List

from ..base import Tool, ToolParameter, SafetyLevel


@dataclass
class ServiceControlTool(Tool):
    """Tool for controlling systemd services."""
    name: str = "service_control"
    description: str = "Control systemd services (start, stop, restart, status)"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="service_name",
            type="string",
            description="Name of the service",
            required=True,
        ),
        ToolParameter(
            name="action",
            type="string",
            description="Action to perform: start, stop, restart, status, enable, disable",
            required=True,
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.REQUIRES_CONFIRMATION
    
    def execute(self, service_name: str, action: str) -> str:
        """Control a systemd service."""
        valid_actions = ["start", "stop", "restart", "status", "enable", "disable"]
        
        if action not in valid_actions:
            return f"Invalid action '{action}'. Valid actions: {', '.join(valid_actions)}"
        
        cmd = ["systemctl", action, service_name]
        
        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=30,
            )
            
            if action == "status":
                return result.stdout or result.stderr
            
            if result.returncode == 0:
                return f"Successfully {action}ed {service_name}"
            else:
                return f"Failed to {action} {service_name}: {result.stderr}"
        except Exception as e:
            return f"Error controlling service: {e}"
