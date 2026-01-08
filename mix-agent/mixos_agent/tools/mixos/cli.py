"""MIXOS CLI tools for the AI agent."""

import subprocess
import json
from typing import Optional, List
from dataclasses import dataclass, field

from ..base import Tool, ToolParameter, SafetyLevel


@dataclass
class MixCLITool(Tool):
    """Execute mix CLI commands."""
    
    name: str = "mix_cli"
    description: str = "Execute a mix CLI command"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="command",
            type="string",
            description="Mix command to execute (e.g., 'status', 'install docker')",
            required=True
        ),
        ToolParameter(
            name="args",
            type="string",
            description="Additional arguments",
            required=False
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.SAFE
    
    def execute(self, command: str, args: str = "") -> str:
        """Execute mix CLI command."""
        cmd = ["mix"] + command.split()
        
        if args:
            cmd.extend(args.split())
        
        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=300
            )
            
            output = result.stdout
            if result.stderr:
                output += f"\n{result.stderr}"
            
            return output or "Command completed"
        except FileNotFoundError:
            return "Error: mix CLI not found. Is MIXOS installed?"
        except subprocess.TimeoutExpired:
            return "Error: Command timed out"
        except Exception as e:
            return f"Error: {str(e)}"


@dataclass
class MixStatusTool(Tool):
    """Get MIXOS system status."""
    
    name: str = "mix_status"
    description: str = "Get MIXOS system status"
    parameters: List[ToolParameter] = field(default_factory=list)
    safety_level: SafetyLevel = SafetyLevel.SAFE
    
    def execute(self) -> str:
        """Get system status."""
        try:
            result = subprocess.run(
                ["mix", "status"],
                capture_output=True,
                text=True,
                timeout=30
            )
            
            return result.stdout or result.stderr or "Status unavailable"
        except FileNotFoundError:
            # Fallback to basic system info
            return self._get_basic_status()
        except Exception as e:
            return f"Error: {str(e)}"
    
    def _get_basic_status(self) -> str:
        """Get basic system status without mix CLI."""
        import os
        import platform
        
        status = []
        status.append(f"OS: {platform.system()} {platform.release()}")
        status.append(f"Architecture: {platform.machine()}")
        
        # Memory info
        try:
            with open("/proc/meminfo") as f:
                for line in f:
                    if line.startswith("MemTotal:"):
                        total = int(line.split()[1]) // 1024
                        status.append(f"Memory: {total} MB")
                        break
        except:
            pass
        
        # CPU info
        try:
            with open("/proc/cpuinfo") as f:
                cores = sum(1 for line in f if line.startswith("processor"))
                status.append(f"CPU Cores: {cores}")
        except:
            pass
        
        # Disk usage
        try:
            stat = os.statvfs("/")
            total = stat.f_blocks * stat.f_frsize // (1024**3)
            free = stat.f_bavail * stat.f_frsize // (1024**3)
            status.append(f"Disk: {free}GB free / {total}GB total")
        except:
            pass
        
        return "\n".join(status)


@dataclass
class MixConfigTool(Tool):
    """Manage MIXOS configuration."""
    
    name: str = "mix_config"
    description: str = "Get or set MIXOS configuration"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="action",
            type="string",
            description="Action: get, set, list",
            required=True
        ),
        ToolParameter(
            name="key",
            type="string",
            description="Configuration key",
            required=False
        ),
        ToolParameter(
            name="value",
            type="string",
            description="Value to set",
            required=False
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.SAFE
    
    def execute(self, action: str, key: str = None, value: str = None) -> str:
        """Manage configuration."""
        try:
            if action == "list":
                result = subprocess.run(
                    ["mix", "config", "list"],
                    capture_output=True,
                    text=True
                )
                return result.stdout or "No configuration found"
            
            elif action == "get":
                if not key:
                    return "Error: key is required for get"
                result = subprocess.run(
                    ["mix", "config", "get", key],
                    capture_output=True,
                    text=True
                )
                return result.stdout.strip() or f"Key '{key}' not found"
            
            elif action == "set":
                if not key or not value:
                    return "Error: key and value are required for set"
                result = subprocess.run(
                    ["mix", "config", "set", key, value],
                    capture_output=True,
                    text=True
                )
                if result.returncode == 0:
                    return f"Set {key} = {value}"
                return f"Error: {result.stderr}"
            
            else:
                return f"Unknown action: {action}"
        except FileNotFoundError:
            return "Error: mix CLI not found"
        except Exception as e:
            return f"Error: {str(e)}"


@dataclass
class MixServiceTool(Tool):
    """Manage MIXOS services."""
    
    name: str = "mix_service"
    description: str = "Manage MIXOS services (agent, docker, etc.)"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="action",
            type="string",
            description="Action: start, stop, restart, status",
            required=True
        ),
        ToolParameter(
            name="service",
            type="string",
            description="Service name (agent, docker)",
            required=True
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.REQUIRES_CONFIRMATION
    
    def execute(self, action: str, service: str) -> str:
        """Manage service."""
        # Map service names to systemd units
        service_map = {
            "agent": "mixos-agent",
            "docker": "docker",
        }
        
        unit = service_map.get(service, service)
        
        try:
            if action == "status":
                result = subprocess.run(
                    ["systemctl", "status", unit],
                    capture_output=True,
                    text=True
                )
                return result.stdout or result.stderr
            
            elif action in ["start", "stop", "restart"]:
                result = subprocess.run(
                    ["systemctl", action, unit],
                    capture_output=True,
                    text=True
                )
                if result.returncode == 0:
                    return f"Service {service} {action}ed"
                return f"Error: {result.stderr}"
            
            else:
                return f"Unknown action: {action}"
        except Exception as e:
            return f"Error: {str(e)}"
