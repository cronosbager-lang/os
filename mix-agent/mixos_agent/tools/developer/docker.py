"""Docker tools for the AI agent."""

import subprocess
import os
import json
from typing import Optional, List, Dict
from dataclasses import dataclass, field

from ..base import Tool, ToolParameter, SafetyLevel


@dataclass
class DockerRunTool(Tool):
    """Run a Docker container."""
    
    name: str = "docker_run"
    description: str = "Run a Docker container"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="image",
            type="string",
            description="Docker image to run",
            required=True
        ),
        ToolParameter(
            name="name",
            type="string",
            description="Container name",
            required=False
        ),
        ToolParameter(
            name="ports",
            type="string",
            description="Port mappings (e.g., '8080:80,443:443')",
            required=False
        ),
        ToolParameter(
            name="volumes",
            type="string",
            description="Volume mappings (e.g., '/host:/container')",
            required=False
        ),
        ToolParameter(
            name="env",
            type="string",
            description="Environment variables (e.g., 'KEY=value,KEY2=value2')",
            required=False
        ),
        ToolParameter(
            name="detach",
            type="boolean",
            description="Run in background",
            required=False,
            default=True
        ),
        ToolParameter(
            name="command",
            type="string",
            description="Command to run in container",
            required=False
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.REQUIRES_CONFIRMATION
    
    def execute(
        self,
        image: str,
        name: str = None,
        ports: str = None,
        volumes: str = None,
        env: str = None,
        detach: bool = True,
        command: str = None
    ) -> str:
        """Run a Docker container."""
        cmd = ["docker", "run"]
        
        if detach:
            cmd.append("-d")
        
        if name:
            cmd.extend(["--name", name])
        
        if ports:
            for port in ports.split(","):
                cmd.extend(["-p", port.strip()])
        
        if volumes:
            for vol in volumes.split(","):
                cmd.extend(["-v", vol.strip()])
        
        if env:
            for e in env.split(","):
                cmd.extend(["-e", e.strip()])
        
        cmd.append(image)
        
        if command:
            cmd.extend(command.split())
        
        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=120
            )
            
            if result.returncode != 0:
                return f"Error running container: {result.stderr}"
            
            container_id = result.stdout.strip()[:12]
            return f"Container started: {container_id}"
        except subprocess.TimeoutExpired:
            return "Error: Operation timed out"
        except Exception as e:
            return f"Error: {str(e)}"


@dataclass
class DockerBuildTool(Tool):
    """Build a Docker image."""
    
    name: str = "docker_build"
    description: str = "Build a Docker image from a Dockerfile"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="path",
            type="string",
            description="Build context path",
            required=True
        ),
        ToolParameter(
            name="tag",
            type="string",
            description="Image tag (e.g., 'myapp:latest')",
            required=True
        ),
        ToolParameter(
            name="dockerfile",
            type="string",
            description="Dockerfile path (relative to context)",
            required=False
        ),
        ToolParameter(
            name="build_args",
            type="string",
            description="Build arguments (e.g., 'ARG1=val1,ARG2=val2')",
            required=False
        ),
        ToolParameter(
            name="no_cache",
            type="boolean",
            description="Build without cache",
            required=False,
            default=False
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.REQUIRES_CONFIRMATION
    
    def execute(
        self,
        path: str,
        tag: str,
        dockerfile: str = None,
        build_args: str = None,
        no_cache: bool = False
    ) -> str:
        """Build a Docker image."""
        cmd = ["docker", "build", "-t", tag]
        
        if dockerfile:
            cmd.extend(["-f", dockerfile])
        
        if build_args:
            for arg in build_args.split(","):
                cmd.extend(["--build-arg", arg.strip()])
        
        if no_cache:
            cmd.append("--no-cache")
        
        cmd.append(path)
        
        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=600
            )
            
            if result.returncode != 0:
                return f"Error building image: {result.stderr}"
            
            return f"Successfully built image: {tag}"
        except subprocess.TimeoutExpired:
            return "Error: Build operation timed out"
        except Exception as e:
            return f"Error: {str(e)}"


@dataclass
class DockerComposeTool(Tool):
    """Manage Docker Compose services."""
    
    name: str = "docker_compose"
    description: str = "Manage Docker Compose services"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="action",
            type="string",
            description="Action: up, down, restart, logs, ps",
            required=True
        ),
        ToolParameter(
            name="service",
            type="string",
            description="Service name (optional)",
            required=False
        ),
        ToolParameter(
            name="path",
            type="string",
            description="Path to docker-compose.yml directory",
            required=False
        ),
        ToolParameter(
            name="detach",
            type="boolean",
            description="Run in background (for 'up')",
            required=False,
            default=True
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.REQUIRES_CONFIRMATION
    
    def execute(
        self,
        action: str,
        service: str = None,
        path: str = None,
        detach: bool = True
    ) -> str:
        """Manage Docker Compose services."""
        cmd = ["docker", "compose"]
        
        if action == "up":
            cmd.append("up")
            if detach:
                cmd.append("-d")
        elif action == "down":
            cmd.append("down")
        elif action == "restart":
            cmd.append("restart")
        elif action == "logs":
            cmd.extend(["logs", "--tail", "100"])
        elif action == "ps":
            cmd.append("ps")
        else:
            return f"Error: Unknown action '{action}'"
        
        if service:
            cmd.append(service)
        
        try:
            result = subprocess.run(
                cmd,
                cwd=path,
                capture_output=True,
                text=True,
                timeout=300
            )
            
            if result.returncode != 0:
                return f"Error: {result.stderr}"
            
            return result.stdout or f"Action '{action}' completed"
        except subprocess.TimeoutExpired:
            return "Error: Operation timed out"
        except Exception as e:
            return f"Error: {str(e)}"


@dataclass
class DockerListTool(Tool):
    """List Docker containers."""
    
    name: str = "docker_list"
    description: str = "List Docker containers"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="all",
            type="boolean",
            description="Show all containers (including stopped)",
            required=False,
            default=False
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.SAFE
    
    def execute(self, all: bool = False) -> str:
        """List containers."""
        cmd = ["docker", "ps", "--format", "table {{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Status}}"]
        
        if all:
            cmd.append("-a")
        
        try:
            result = subprocess.run(cmd, capture_output=True, text=True)
            return result.stdout or "No containers found"
        except Exception as e:
            return f"Error: {str(e)}"


@dataclass
class DockerStopTool(Tool):
    """Stop a Docker container."""
    
    name: str = "docker_stop"
    description: str = "Stop a running Docker container"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="container",
            type="string",
            description="Container ID or name",
            required=True
        ),
        ToolParameter(
            name="timeout",
            type="integer",
            description="Timeout in seconds before killing",
            required=False,
            default=10
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.REQUIRES_CONFIRMATION
    
    def execute(self, container: str, timeout: int = 10) -> str:
        """Stop a container."""
        cmd = ["docker", "stop", "-t", str(timeout), container]
        
        try:
            result = subprocess.run(cmd, capture_output=True, text=True, timeout=timeout + 30)
            
            if result.returncode != 0:
                return f"Error stopping container: {result.stderr}"
            
            return f"Container {container} stopped"
        except Exception as e:
            return f"Error: {str(e)}"


@dataclass
class DockerLogsTool(Tool):
    """Get Docker container logs."""
    
    name: str = "docker_logs"
    description: str = "Get logs from a Docker container"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="container",
            type="string",
            description="Container ID or name",
            required=True
        ),
        ToolParameter(
            name="tail",
            type="integer",
            description="Number of lines to show",
            required=False,
            default=100
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.SAFE
    
    def execute(self, container: str, tail: int = 100) -> str:
        """Get container logs."""
        cmd = ["docker", "logs", "--tail", str(tail), container]
        
        try:
            result = subprocess.run(cmd, capture_output=True, text=True)
            return result.stdout + result.stderr or "No logs available"
        except Exception as e:
            return f"Error: {str(e)}"


@dataclass
class DockerExecTool(Tool):
    """Execute a command in a Docker container."""
    
    name: str = "docker_exec"
    description: str = "Execute a command inside a running container"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="container",
            type="string",
            description="Container ID or name",
            required=True
        ),
        ToolParameter(
            name="command",
            type="string",
            description="Command to execute",
            required=True
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.REQUIRES_CONFIRMATION
    
    def execute(self, container: str, command: str) -> str:
        """Execute command in container."""
        cmd = ["docker", "exec", container] + command.split()
        
        try:
            result = subprocess.run(cmd, capture_output=True, text=True, timeout=60)
            
            if result.returncode != 0:
                return f"Error: {result.stderr}"
            
            return result.stdout or "Command completed"
        except subprocess.TimeoutExpired:
            return "Error: Command timed out"
        except Exception as e:
            return f"Error: {str(e)}"


@dataclass
class DockerImagesTool(Tool):
    """List Docker images."""
    
    name: str = "docker_images"
    description: str = "List Docker images"
    parameters: List[ToolParameter] = field(default_factory=list)
    safety_level: SafetyLevel = SafetyLevel.SAFE
    
    def execute(self) -> str:
        """List images."""
        cmd = ["docker", "images", "--format", "table {{.Repository}}\t{{.Tag}}\t{{.Size}}"]
        
        try:
            result = subprocess.run(cmd, capture_output=True, text=True)
            return result.stdout or "No images found"
        except Exception as e:
            return f"Error: {str(e)}"


@dataclass
class DockerPullTool(Tool):
    """Pull a Docker image."""
    
    name: str = "docker_pull"
    description: str = "Pull a Docker image from registry"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="image",
            type="string",
            description="Image name (e.g., 'nginx:latest')",
            required=True
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.SAFE
    
    def execute(self, image: str) -> str:
        """Pull an image."""
        cmd = ["docker", "pull", image]
        
        try:
            result = subprocess.run(cmd, capture_output=True, text=True, timeout=300)
            
            if result.returncode != 0:
                return f"Error pulling image: {result.stderr}"
            
            return f"Successfully pulled {image}"
        except subprocess.TimeoutExpired:
            return "Error: Pull operation timed out"
        except Exception as e:
            return f"Error: {str(e)}"
