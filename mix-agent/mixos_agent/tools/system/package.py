"""Package management tools."""

import subprocess
from dataclasses import dataclass, field
from typing import List

from ..base import Tool, ToolParameter, SafetyLevel


@dataclass
class InstallPackageTool(Tool):
    """Tool for installing packages."""
    name: str = "install_package"
    description: str = "Install a package using mix-pkg"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="package_name",
            type="string",
            description="Name of the package to install",
            required=True,
        ),
        ToolParameter(
            name="version",
            type="string",
            description="Specific version to install",
            required=False,
        ),
        ToolParameter(
            name="force",
            type="boolean",
            description="Force reinstallation",
            required=False,
            default=False,
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.REQUIRES_CONFIRMATION
    
    def execute(self, package_name: str, version: str = None, force: bool = False) -> str:
        """Install a package."""
        cmd = ["mix-pkg", "install", "-y"]
        
        if force:
            cmd.append("--force")
        
        if version:
            cmd.append(f"{package_name}={version}")
        else:
            cmd.append(package_name)
        
        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=300,
            )
            
            if result.returncode == 0:
                return f"Successfully installed {package_name}"
            else:
                return f"Failed to install {package_name}: {result.stderr}"
        except subprocess.TimeoutExpired:
            return f"Installation of {package_name} timed out"
        except Exception as e:
            return f"Error installing {package_name}: {e}"


@dataclass
class RemovePackageTool(Tool):
    """Tool for removing packages."""
    name: str = "remove_package"
    description: str = "Remove a package using mix-pkg"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="package_name",
            type="string",
            description="Name of the package to remove",
            required=True,
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.REQUIRES_CONFIRMATION
    
    def execute(self, package_name: str) -> str:
        """Remove a package."""
        cmd = ["mix-pkg", "remove", "-y", package_name]
        
        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=120,
            )
            
            if result.returncode == 0:
                return f"Successfully removed {package_name}"
            else:
                return f"Failed to remove {package_name}: {result.stderr}"
        except Exception as e:
            return f"Error removing {package_name}: {e}"
