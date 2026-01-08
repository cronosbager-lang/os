"""MIXOS VM management tools for the AI agent."""

import subprocess
import json
from typing import Optional, List
from dataclasses import dataclass, field

from ..base import Tool, ToolParameter, SafetyLevel


@dataclass
class MixVMCreateTool(Tool):
    """Create a new virtual machine."""
    
    name: str = "vm_create"
    description: str = "Create a new virtual machine"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="name",
            type="string",
            description="VM name",
            required=True
        ),
        ToolParameter(
            name="cpus",
            type="integer",
            description="Number of CPUs",
            required=False,
            default=2
        ),
        ToolParameter(
            name="memory",
            type="integer",
            description="Memory in MB",
            required=False,
            default=2048
        ),
        ToolParameter(
            name="disk",
            type="integer",
            description="Disk size in GB",
            required=False,
            default=20
        ),
        ToolParameter(
            name="iso",
            type="string",
            description="Path to ISO file for installation",
            required=False
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.REQUIRES_CONFIRMATION
    
    def execute(
        self,
        name: str,
        cpus: int = 2,
        memory: int = 2048,
        disk: int = 20,
        iso: str = None
    ) -> str:
        """Create a VM."""
        cmd = [
            "mix", "vm", "create", name,
            "--cpus", str(cpus),
            "--memory", str(memory),
            "--disk", str(disk),
        ]
        
        if iso:
            cmd.extend(["--iso", iso])
        
        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=120
            )
            
            if result.returncode == 0:
                return f"Created VM '{name}' with {cpus} CPUs, {memory}MB RAM, {disk}GB disk"
            else:
                return f"Error creating VM: {result.stderr}"
        except FileNotFoundError:
            return "Error: mix CLI not found"
        except Exception as e:
            return f"Error: {str(e)}"


@dataclass
class MixVMStartTool(Tool):
    """Start a virtual machine."""
    
    name: str = "vm_start"
    description: str = "Start a virtual machine"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="name",
            type="string",
            description="VM name",
            required=True
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.SAFE
    
    def execute(self, name: str) -> str:
        """Start a VM."""
        try:
            result = subprocess.run(
                ["mix", "vm", "start", name],
                capture_output=True,
                text=True,
                timeout=60
            )
            
            if result.returncode == 0:
                return f"VM '{name}' started"
            else:
                return f"Error starting VM: {result.stderr}"
        except FileNotFoundError:
            return "Error: mix CLI not found"
        except Exception as e:
            return f"Error: {str(e)}"


@dataclass
class MixVMStopTool(Tool):
    """Stop a virtual machine."""
    
    name: str = "vm_stop"
    description: str = "Stop a virtual machine"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="name",
            type="string",
            description="VM name",
            required=True
        ),
        ToolParameter(
            name="force",
            type="boolean",
            description="Force stop (kill)",
            required=False,
            default=False
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.REQUIRES_CONFIRMATION
    
    def execute(self, name: str, force: bool = False) -> str:
        """Stop a VM."""
        cmd = ["mix", "vm", "stop", name]
        if force:
            cmd.append("--force")
        
        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=60
            )
            
            if result.returncode == 0:
                return f"VM '{name}' stopped"
            else:
                return f"Error stopping VM: {result.stderr}"
        except FileNotFoundError:
            return "Error: mix CLI not found"
        except Exception as e:
            return f"Error: {str(e)}"


@dataclass
class MixVMListTool(Tool):
    """List virtual machines."""
    
    name: str = "vm_list"
    description: str = "List all virtual machines"
    parameters: List[ToolParameter] = field(default_factory=list)
    safety_level: SafetyLevel = SafetyLevel.SAFE
    
    def execute(self) -> str:
        """List VMs."""
        try:
            result = subprocess.run(
                ["mix", "vm", "list"],
                capture_output=True,
                text=True,
                timeout=30
            )
            
            return result.stdout or "No VMs found"
        except FileNotFoundError:
            return "Error: mix CLI not found"
        except Exception as e:
            return f"Error: {str(e)}"


@dataclass
class MixVMDestroyTool(Tool):
    """Destroy a virtual machine."""
    
    name: str = "vm_destroy"
    description: str = "Destroy a virtual machine and its disk"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="name",
            type="string",
            description="VM name",
            required=True
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.DANGEROUS
    
    def execute(self, name: str) -> str:
        """Destroy a VM."""
        try:
            result = subprocess.run(
                ["mix", "vm", "destroy", name, "-y"],
                capture_output=True,
                text=True,
                timeout=60
            )
            
            if result.returncode == 0:
                return f"VM '{name}' destroyed"
            else:
                return f"Error destroying VM: {result.stderr}"
        except FileNotFoundError:
            return "Error: mix CLI not found"
        except Exception as e:
            return f"Error: {str(e)}"


@dataclass
class MixVMSSHTool(Tool):
    """SSH into a virtual machine."""
    
    name: str = "vm_ssh"
    description: str = "Get SSH command for a virtual machine"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="name",
            type="string",
            description="VM name",
            required=True
        ),
        ToolParameter(
            name="user",
            type="string",
            description="SSH user",
            required=False,
            default="root"
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.SAFE
    
    def execute(self, name: str, user: str = "root") -> str:
        """Get SSH command for VM."""
        try:
            # Get VM info to find SSH port
            result = subprocess.run(
                ["mix", "vm", "info", name, "--json"],
                capture_output=True,
                text=True,
                timeout=30
            )
            
            if result.returncode != 0:
                return f"Error: VM '{name}' not found"
            
            try:
                info = json.loads(result.stdout)
                ssh_port = info.get("ssh_port", 22)
                return f"SSH command: ssh -p {ssh_port} {user}@localhost"
            except json.JSONDecodeError:
                return f"SSH command: mix vm ssh {name}"
        except FileNotFoundError:
            return "Error: mix CLI not found"
        except Exception as e:
            return f"Error: {str(e)}"


@dataclass
class MixVMSnapshotTool(Tool):
    """Manage VM snapshots."""
    
    name: str = "vm_snapshot"
    description: str = "Create or restore VM snapshots"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="action",
            type="string",
            description="Action: create, restore, list",
            required=True
        ),
        ToolParameter(
            name="vm_name",
            type="string",
            description="VM name",
            required=True
        ),
        ToolParameter(
            name="snapshot_name",
            type="string",
            description="Snapshot name (for create/restore)",
            required=False
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.REQUIRES_CONFIRMATION
    
    def execute(
        self,
        action: str,
        vm_name: str,
        snapshot_name: str = None
    ) -> str:
        """Manage snapshots."""
        try:
            if action == "list":
                result = subprocess.run(
                    ["mix", "vm", "snapshot", "list", vm_name],
                    capture_output=True,
                    text=True,
                    timeout=30
                )
                return result.stdout or "No snapshots found"
            
            elif action == "create":
                if not snapshot_name:
                    return "Error: snapshot_name required for create"
                result = subprocess.run(
                    ["mix", "vm", "snapshot", "create", vm_name, snapshot_name],
                    capture_output=True,
                    text=True,
                    timeout=120
                )
                if result.returncode == 0:
                    return f"Created snapshot '{snapshot_name}' for VM '{vm_name}'"
                return f"Error: {result.stderr}"
            
            elif action == "restore":
                if not snapshot_name:
                    return "Error: snapshot_name required for restore"
                result = subprocess.run(
                    ["mix", "vm", "snapshot", "restore", vm_name, snapshot_name],
                    capture_output=True,
                    text=True,
                    timeout=120
                )
                if result.returncode == 0:
                    return f"Restored VM '{vm_name}' to snapshot '{snapshot_name}'"
                return f"Error: {result.stderr}"
            
            else:
                return f"Unknown action: {action}"
        except FileNotFoundError:
            return "Error: mix CLI not found"
        except Exception as e:
            return f"Error: {str(e)}"
