"""SSH tools for the AI agent."""

import subprocess
import os
from typing import Optional, List
from dataclasses import dataclass, field

from ..base import Tool, ToolParameter, SafetyLevel


@dataclass
class SSHConnectTool(Tool):
    """Test SSH connection to a host."""
    
    name: str = "ssh_connect"
    description: str = "Test SSH connection to a remote host"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="host",
            type="string",
            description="Remote host (user@host or just host)",
            required=True
        ),
        ToolParameter(
            name="port",
            type="integer",
            description="SSH port",
            required=False,
            default=22
        ),
        ToolParameter(
            name="key",
            type="string",
            description="Path to SSH private key",
            required=False
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.SAFE
    
    def execute(self, host: str, port: int = 22, key: str = None) -> str:
        """Test SSH connection."""
        cmd = [
            "ssh",
            "-o", "BatchMode=yes",
            "-o", "ConnectTimeout=10",
            "-o", "StrictHostKeyChecking=no",
            "-p", str(port),
        ]
        
        if key:
            cmd.extend(["-i", key])
        
        cmd.extend([host, "echo", "Connection successful"])
        
        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=15
            )
            
            if result.returncode == 0:
                return f"SSH connection to {host}:{port} successful"
            else:
                return f"SSH connection failed: {result.stderr}"
        except subprocess.TimeoutExpired:
            return f"SSH connection to {host}:{port} timed out"
        except Exception as e:
            return f"Error: {str(e)}"


@dataclass
class SSHCommandTool(Tool):
    """Execute command on remote host via SSH."""
    
    name: str = "ssh_command"
    description: str = "Execute a command on a remote host via SSH"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="host",
            type="string",
            description="Remote host (user@host)",
            required=True
        ),
        ToolParameter(
            name="command",
            type="string",
            description="Command to execute",
            required=True
        ),
        ToolParameter(
            name="port",
            type="integer",
            description="SSH port",
            required=False,
            default=22
        ),
        ToolParameter(
            name="key",
            type="string",
            description="Path to SSH private key",
            required=False
        ),
        ToolParameter(
            name="timeout",
            type="integer",
            description="Command timeout in seconds",
            required=False,
            default=60
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.REQUIRES_CONFIRMATION
    
    def execute(
        self,
        host: str,
        command: str,
        port: int = 22,
        key: str = None,
        timeout: int = 60
    ) -> str:
        """Execute SSH command."""
        cmd = [
            "ssh",
            "-o", "BatchMode=yes",
            "-o", "ConnectTimeout=10",
            "-o", "StrictHostKeyChecking=no",
            "-p", str(port),
        ]
        
        if key:
            cmd.extend(["-i", key])
        
        cmd.extend([host, command])
        
        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=timeout
            )
            
            output = result.stdout
            if result.stderr:
                output += f"\nStderr: {result.stderr}"
            
            if result.returncode != 0:
                output += f"\nExit code: {result.returncode}"
            
            return output or "Command completed (no output)"
        except subprocess.TimeoutExpired:
            return f"Command timed out after {timeout} seconds"
        except Exception as e:
            return f"Error: {str(e)}"


@dataclass
class SCPTool(Tool):
    """Copy files via SCP."""
    
    name: str = "scp"
    description: str = "Copy files to/from remote host via SCP"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="source",
            type="string",
            description="Source path (local or user@host:path)",
            required=True
        ),
        ToolParameter(
            name="destination",
            type="string",
            description="Destination path (local or user@host:path)",
            required=True
        ),
        ToolParameter(
            name="port",
            type="integer",
            description="SSH port",
            required=False,
            default=22
        ),
        ToolParameter(
            name="key",
            type="string",
            description="Path to SSH private key",
            required=False
        ),
        ToolParameter(
            name="recursive",
            type="boolean",
            description="Copy directories recursively",
            required=False,
            default=False
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.REQUIRES_CONFIRMATION
    
    def execute(
        self,
        source: str,
        destination: str,
        port: int = 22,
        key: str = None,
        recursive: bool = False
    ) -> str:
        """Copy files via SCP."""
        cmd = [
            "scp",
            "-o", "StrictHostKeyChecking=no",
            "-P", str(port),
        ]
        
        if key:
            cmd.extend(["-i", key])
        
        if recursive:
            cmd.append("-r")
        
        cmd.extend([source, destination])
        
        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=300
            )
            
            if result.returncode == 0:
                return f"Successfully copied {source} to {destination}"
            else:
                return f"SCP failed: {result.stderr}"
        except subprocess.TimeoutExpired:
            return "SCP operation timed out"
        except Exception as e:
            return f"Error: {str(e)}"


@dataclass
class SSHKeyGenTool(Tool):
    """Generate SSH key pair."""
    
    name: str = "ssh_keygen"
    description: str = "Generate an SSH key pair"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="path",
            type="string",
            description="Path for the key file",
            required=True
        ),
        ToolParameter(
            name="type",
            type="string",
            description="Key type (ed25519, rsa)",
            required=False,
            default="ed25519"
        ),
        ToolParameter(
            name="comment",
            type="string",
            description="Key comment",
            required=False
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.SAFE
    
    def execute(
        self,
        path: str,
        type: str = "ed25519",
        comment: str = None
    ) -> str:
        """Generate SSH key."""
        # Expand path
        path = os.path.expanduser(path)
        
        # Check if key already exists
        if os.path.exists(path):
            return f"Error: Key already exists at {path}"
        
        # Create directory if needed
        os.makedirs(os.path.dirname(path), exist_ok=True)
        
        cmd = [
            "ssh-keygen",
            "-t", type,
            "-f", path,
            "-N", "",  # No passphrase
        ]
        
        if comment:
            cmd.extend(["-C", comment])
        
        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True
            )
            
            if result.returncode == 0:
                # Read public key
                with open(f"{path}.pub") as f:
                    pubkey = f.read().strip()
                
                return f"Generated {type} key pair:\n  Private: {path}\n  Public: {path}.pub\n\nPublic key:\n{pubkey}"
            else:
                return f"Error generating key: {result.stderr}"
        except Exception as e:
            return f"Error: {str(e)}"


@dataclass
class SSHCopyIdTool(Tool):
    """Copy SSH public key to remote host."""
    
    name: str = "ssh_copy_id"
    description: str = "Copy SSH public key to a remote host for passwordless login"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="host",
            type="string",
            description="Remote host (user@host)",
            required=True
        ),
        ToolParameter(
            name="key",
            type="string",
            description="Path to public key file",
            required=False,
            default="~/.ssh/id_ed25519.pub"
        ),
        ToolParameter(
            name="port",
            type="integer",
            description="SSH port",
            required=False,
            default=22
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.REQUIRES_CONFIRMATION
    
    def execute(self, host: str, key: str = "~/.ssh/id_ed25519.pub", port: int = 22) -> str:
        """Copy SSH key to remote host."""
        key = os.path.expanduser(key)
        
        if not os.path.exists(key):
            return f"Error: Public key not found at {key}"
        
        cmd = [
            "ssh-copy-id",
            "-i", key,
            "-p", str(port),
            "-o", "StrictHostKeyChecking=no",
            host
        ]
        
        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=60
            )
            
            if result.returncode == 0:
                return f"Successfully copied public key to {host}"
            else:
                return f"Error: {result.stderr}"
        except subprocess.TimeoutExpired:
            return "Operation timed out"
        except Exception as e:
            return f"Error: {str(e)}"


@dataclass
class SSHConfigTool(Tool):
    """Manage SSH config entries."""
    
    name: str = "ssh_config"
    description: str = "Add or list SSH config entries"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="action",
            type="string",
            description="Action: list, add",
            required=True
        ),
        ToolParameter(
            name="name",
            type="string",
            description="Host alias (for add)",
            required=False
        ),
        ToolParameter(
            name="hostname",
            type="string",
            description="Actual hostname (for add)",
            required=False
        ),
        ToolParameter(
            name="user",
            type="string",
            description="Username (for add)",
            required=False
        ),
        ToolParameter(
            name="port",
            type="integer",
            description="Port (for add)",
            required=False,
            default=22
        ),
        ToolParameter(
            name="key",
            type="string",
            description="Identity file path (for add)",
            required=False
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.SAFE
    
    def execute(
        self,
        action: str,
        name: str = None,
        hostname: str = None,
        user: str = None,
        port: int = 22,
        key: str = None
    ) -> str:
        """Manage SSH config."""
        config_path = os.path.expanduser("~/.ssh/config")
        
        if action == "list":
            if not os.path.exists(config_path):
                return "No SSH config file found"
            
            with open(config_path) as f:
                return f.read()
        
        elif action == "add":
            if not name or not hostname:
                return "Error: name and hostname are required for add"
            
            # Create .ssh directory if needed
            os.makedirs(os.path.dirname(config_path), exist_ok=True)
            
            # Build config entry
            entry = f"\nHost {name}\n"
            entry += f"    HostName {hostname}\n"
            if user:
                entry += f"    User {user}\n"
            if port != 22:
                entry += f"    Port {port}\n"
            if key:
                entry += f"    IdentityFile {key}\n"
            
            # Append to config
            with open(config_path, "a") as f:
                f.write(entry)
            
            return f"Added SSH config entry for '{name}'"
        
        else:
            return f"Unknown action: {action}"
