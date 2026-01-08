"""MIXOS package management tools for the AI agent."""

import subprocess
from typing import Optional, List
from dataclasses import dataclass, field

from ..base import Tool, ToolParameter, SafetyLevel


@dataclass
class MixInstallTool(Tool):
    """Install packages using mix-pkg."""
    
    name: str = "install_package"
    description: str = "Install a package using the MIXOS package manager"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="package_name",
            type="string",
            description="Name of the package to install",
            required=True
        ),
        ToolParameter(
            name="version",
            type="string",
            description="Specific version to install",
            required=False
        ),
        ToolParameter(
            name="force",
            type="boolean",
            description="Force reinstallation",
            required=False,
            default=False
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.REQUIRES_CONFIRMATION
    
    def execute(
        self,
        package_name: str,
        version: str = None,
        force: bool = False
    ) -> str:
        """Install a package."""
        cmd = ["mix-pkg", "install"]
        
        if force:
            cmd.append("--force")
        
        cmd.append("-y")  # Auto-confirm
        
        pkg = package_name
        if version:
            pkg = f"{package_name}={version}"
        cmd.append(pkg)
        
        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=600
            )
            
            if result.returncode == 0:
                return f"Successfully installed {package_name}"
            else:
                return f"Error installing {package_name}: {result.stderr}"
        except FileNotFoundError:
            # Fallback to system package manager
            return self._fallback_install(package_name)
        except subprocess.TimeoutExpired:
            return "Error: Installation timed out"
        except Exception as e:
            return f"Error: {str(e)}"
    
    def _fallback_install(self, package_name: str) -> str:
        """Fallback to system package manager."""
        # Try apt
        try:
            result = subprocess.run(
                ["apt-get", "install", "-y", package_name],
                capture_output=True,
                text=True,
                timeout=600
            )
            if result.returncode == 0:
                return f"Installed {package_name} via apt"
        except:
            pass
        
        # Try pacman
        try:
            result = subprocess.run(
                ["pacman", "-S", "--noconfirm", package_name],
                capture_output=True,
                text=True,
                timeout=600
            )
            if result.returncode == 0:
                return f"Installed {package_name} via pacman"
        except:
            pass
        
        return f"Error: Could not install {package_name} - no package manager available"


@dataclass
class MixRemoveTool(Tool):
    """Remove packages using mix-pkg."""
    
    name: str = "remove_package"
    description: str = "Remove a package using the MIXOS package manager"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="package_name",
            type="string",
            description="Name of the package to remove",
            required=True
        ),
        ToolParameter(
            name="purge",
            type="boolean",
            description="Also remove configuration files",
            required=False,
            default=False
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.REQUIRES_CONFIRMATION
    
    def execute(self, package_name: str, purge: bool = False) -> str:
        """Remove a package."""
        cmd = ["mix-pkg", "remove", "-y", package_name]
        
        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=300
            )
            
            if result.returncode == 0:
                return f"Successfully removed {package_name}"
            else:
                return f"Error removing {package_name}: {result.stderr}"
        except FileNotFoundError:
            return "Error: mix-pkg not found"
        except Exception as e:
            return f"Error: {str(e)}"


@dataclass
class MixSearchTool(Tool):
    """Search for packages."""
    
    name: str = "search_package"
    description: str = "Search for packages in the MIXOS repository"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="query",
            type="string",
            description="Search query",
            required=True
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.SAFE
    
    def execute(self, query: str) -> str:
        """Search for packages."""
        try:
            result = subprocess.run(
                ["mix-pkg", "search", query],
                capture_output=True,
                text=True,
                timeout=60
            )
            
            return result.stdout or "No packages found"
        except FileNotFoundError:
            # Fallback to apt-cache
            try:
                result = subprocess.run(
                    ["apt-cache", "search", query],
                    capture_output=True,
                    text=True,
                    timeout=60
                )
                return result.stdout or "No packages found"
            except:
                return "Error: No package search available"
        except Exception as e:
            return f"Error: {str(e)}"


@dataclass
class MixUpdateTool(Tool):
    """Update package database."""
    
    name: str = "update_packages"
    description: str = "Update the package database"
    parameters: List[ToolParameter] = field(default_factory=list)
    safety_level: SafetyLevel = SafetyLevel.SAFE
    
    def execute(self) -> str:
        """Update package database."""
        try:
            result = subprocess.run(
                ["mix-pkg", "update"],
                capture_output=True,
                text=True,
                timeout=300
            )
            
            if result.returncode == 0:
                return "Package database updated"
            else:
                return f"Error updating: {result.stderr}"
        except FileNotFoundError:
            # Fallback
            try:
                result = subprocess.run(
                    ["apt-get", "update"],
                    capture_output=True,
                    text=True,
                    timeout=300
                )
                return "Package database updated (apt)"
            except:
                return "Error: No package manager available"
        except Exception as e:
            return f"Error: {str(e)}"


@dataclass
class MixUpgradeTool(Tool):
    """Upgrade all packages."""
    
    name: str = "upgrade_packages"
    description: str = "Upgrade all installed packages"
    parameters: List[ToolParameter] = field(default_factory=list)
    safety_level: SafetyLevel = SafetyLevel.REQUIRES_CONFIRMATION
    
    def execute(self) -> str:
        """Upgrade packages."""
        try:
            result = subprocess.run(
                ["mix-pkg", "upgrade", "-y"],
                capture_output=True,
                text=True,
                timeout=1800
            )
            
            return result.stdout or "Upgrade completed"
        except FileNotFoundError:
            return "Error: mix-pkg not found"
        except subprocess.TimeoutExpired:
            return "Error: Upgrade timed out"
        except Exception as e:
            return f"Error: {str(e)}"


@dataclass
class MixPackageInfoTool(Tool):
    """Get package information."""
    
    name: str = "package_info"
    description: str = "Get detailed information about a package"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="package_name",
            type="string",
            description="Package name",
            required=True
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.SAFE
    
    def execute(self, package_name: str) -> str:
        """Get package info."""
        try:
            result = subprocess.run(
                ["mix-pkg", "info", package_name],
                capture_output=True,
                text=True,
                timeout=30
            )
            
            return result.stdout or f"Package '{package_name}' not found"
        except FileNotFoundError:
            try:
                result = subprocess.run(
                    ["apt-cache", "show", package_name],
                    capture_output=True,
                    text=True,
                    timeout=30
                )
                return result.stdout or f"Package '{package_name}' not found"
            except:
                return "Error: No package info available"
        except Exception as e:
            return f"Error: {str(e)}"


@dataclass
class MixListInstalledTool(Tool):
    """List installed packages."""
    
    name: str = "list_installed"
    description: str = "List all installed packages"
    parameters: List[ToolParameter] = field(default_factory=list)
    safety_level: SafetyLevel = SafetyLevel.SAFE
    
    def execute(self) -> str:
        """List installed packages."""
        try:
            result = subprocess.run(
                ["mix-pkg", "list"],
                capture_output=True,
                text=True,
                timeout=60
            )
            
            return result.stdout or "No packages installed"
        except FileNotFoundError:
            try:
                result = subprocess.run(
                    ["dpkg", "-l"],
                    capture_output=True,
                    text=True,
                    timeout=60
                )
                return result.stdout
            except:
                return "Error: Cannot list packages"
        except Exception as e:
            return f"Error: {str(e)}"
