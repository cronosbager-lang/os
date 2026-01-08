"""Language runtime tools for the AI agent."""

import subprocess
import os
import shutil
from typing import Optional, List, Dict
from dataclasses import dataclass, field

from ..base import Tool, ToolParameter, SafetyLevel


@dataclass
class PythonSetupTool(Tool):
    """Setup Python development environment."""
    
    name: str = "python_setup"
    description: str = "Setup Python development environment with virtualenv"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="path",
            type="string",
            description="Project path",
            required=True
        ),
        ToolParameter(
            name="python_version",
            type="string",
            description="Python version (e.g., '3.11')",
            required=False,
            default="3"
        ),
        ToolParameter(
            name="requirements",
            type="string",
            description="Path to requirements.txt",
            required=False
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.SAFE
    
    def execute(
        self,
        path: str,
        python_version: str = "3",
        requirements: str = None
    ) -> str:
        """Setup Python environment."""
        results = []
        
        # Create virtualenv
        venv_path = os.path.join(path, "venv")
        python_cmd = f"python{python_version}"
        
        try:
            # Check if Python is available
            result = subprocess.run(
                [python_cmd, "--version"],
                capture_output=True,
                text=True
            )
            if result.returncode != 0:
                python_cmd = "python3"
            
            # Create virtualenv
            result = subprocess.run(
                [python_cmd, "-m", "venv", venv_path],
                capture_output=True,
                text=True
            )
            
            if result.returncode != 0:
                return f"Error creating virtualenv: {result.stderr}"
            
            results.append(f"Created virtualenv at {venv_path}")
            
            # Install requirements if provided
            if requirements:
                req_path = os.path.join(path, requirements) if not os.path.isabs(requirements) else requirements
                if os.path.exists(req_path):
                    pip_path = os.path.join(venv_path, "bin", "pip")
                    result = subprocess.run(
                        [pip_path, "install", "-r", req_path],
                        capture_output=True,
                        text=True,
                        timeout=300
                    )
                    
                    if result.returncode == 0:
                        results.append("Installed requirements")
                    else:
                        results.append(f"Warning: Failed to install requirements: {result.stderr}")
            
            # Create activation hint
            results.append(f"\nTo activate: source {venv_path}/bin/activate")
            
            return "\n".join(results)
        except Exception as e:
            return f"Error: {str(e)}"


@dataclass
class NodeSetupTool(Tool):
    """Setup Node.js development environment."""
    
    name: str = "node_setup"
    description: str = "Setup Node.js project with npm/yarn"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="path",
            type="string",
            description="Project path",
            required=True
        ),
        ToolParameter(
            name="package_manager",
            type="string",
            description="Package manager: npm or yarn",
            required=False,
            default="npm"
        ),
        ToolParameter(
            name="install",
            type="boolean",
            description="Install dependencies",
            required=False,
            default=True
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.SAFE
    
    def execute(
        self,
        path: str,
        package_manager: str = "npm",
        install: bool = True
    ) -> str:
        """Setup Node.js environment."""
        results = []
        
        try:
            # Check if package.json exists
            package_json = os.path.join(path, "package.json")
            
            if not os.path.exists(package_json):
                # Initialize new project
                result = subprocess.run(
                    [package_manager, "init", "-y"],
                    cwd=path,
                    capture_output=True,
                    text=True
                )
                
                if result.returncode == 0:
                    results.append("Initialized new Node.js project")
                else:
                    return f"Error initializing project: {result.stderr}"
            
            # Install dependencies
            if install and os.path.exists(package_json):
                cmd = [package_manager, "install"]
                result = subprocess.run(
                    cmd,
                    cwd=path,
                    capture_output=True,
                    text=True,
                    timeout=300
                )
                
                if result.returncode == 0:
                    results.append("Installed dependencies")
                else:
                    results.append(f"Warning: Failed to install dependencies: {result.stderr}")
            
            return "\n".join(results) or "Node.js environment ready"
        except Exception as e:
            return f"Error: {str(e)}"


@dataclass
class GoSetupTool(Tool):
    """Setup Go development environment."""
    
    name: str = "go_setup"
    description: str = "Setup Go project with modules"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="path",
            type="string",
            description="Project path",
            required=True
        ),
        ToolParameter(
            name="module_name",
            type="string",
            description="Go module name",
            required=False
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.SAFE
    
    def execute(self, path: str, module_name: str = None) -> str:
        """Setup Go environment."""
        results = []
        
        try:
            go_mod = os.path.join(path, "go.mod")
            
            if not os.path.exists(go_mod):
                if not module_name:
                    module_name = os.path.basename(os.path.abspath(path))
                
                result = subprocess.run(
                    ["go", "mod", "init", module_name],
                    cwd=path,
                    capture_output=True,
                    text=True
                )
                
                if result.returncode == 0:
                    results.append(f"Initialized Go module: {module_name}")
                else:
                    return f"Error initializing module: {result.stderr}"
            
            # Download dependencies
            result = subprocess.run(
                ["go", "mod", "tidy"],
                cwd=path,
                capture_output=True,
                text=True,
                timeout=120
            )
            
            if result.returncode == 0:
                results.append("Dependencies synchronized")
            
            return "\n".join(results) or "Go environment ready"
        except Exception as e:
            return f"Error: {str(e)}"


@dataclass
class RustSetupTool(Tool):
    """Setup Rust development environment."""
    
    name: str = "rust_setup"
    description: str = "Setup Rust project with Cargo"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="path",
            type="string",
            description="Project path",
            required=True
        ),
        ToolParameter(
            name="project_type",
            type="string",
            description="Project type: bin or lib",
            required=False,
            default="bin"
        ),
        ToolParameter(
            name="name",
            type="string",
            description="Project name",
            required=False
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.SAFE
    
    def execute(
        self,
        path: str,
        project_type: str = "bin",
        name: str = None
    ) -> str:
        """Setup Rust environment."""
        results = []
        
        try:
            cargo_toml = os.path.join(path, "Cargo.toml")
            
            if not os.path.exists(cargo_toml):
                cmd = ["cargo", "init"]
                
                if project_type == "lib":
                    cmd.append("--lib")
                
                if name:
                    cmd.extend(["--name", name])
                
                result = subprocess.run(
                    cmd,
                    cwd=path,
                    capture_output=True,
                    text=True
                )
                
                if result.returncode == 0:
                    results.append("Initialized Rust project")
                else:
                    return f"Error initializing project: {result.stderr}"
            
            # Build to download dependencies
            result = subprocess.run(
                ["cargo", "build"],
                cwd=path,
                capture_output=True,
                text=True,
                timeout=300
            )
            
            if result.returncode == 0:
                results.append("Project built successfully")
            else:
                results.append(f"Build warning: {result.stderr}")
            
            return "\n".join(results) or "Rust environment ready"
        except Exception as e:
            return f"Error: {str(e)}"


@dataclass
class InstallDependenciesTool(Tool):
    """Install project dependencies."""
    
    name: str = "install_dependencies"
    description: str = "Detect project type and install dependencies"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="path",
            type="string",
            description="Project path",
            required=True
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.SAFE
    
    def execute(self, path: str) -> str:
        """Install dependencies based on project type."""
        results = []
        
        try:
            # Detect project type and install
            if os.path.exists(os.path.join(path, "requirements.txt")):
                # Python with requirements.txt
                result = subprocess.run(
                    ["pip", "install", "-r", "requirements.txt"],
                    cwd=path,
                    capture_output=True,
                    text=True,
                    timeout=300
                )
                results.append(f"Python: {'Installed' if result.returncode == 0 else 'Failed'}")
            
            if os.path.exists(os.path.join(path, "pyproject.toml")):
                # Python with pyproject.toml
                result = subprocess.run(
                    ["pip", "install", "-e", "."],
                    cwd=path,
                    capture_output=True,
                    text=True,
                    timeout=300
                )
                results.append(f"Python (pyproject): {'Installed' if result.returncode == 0 else 'Failed'}")
            
            if os.path.exists(os.path.join(path, "package.json")):
                # Node.js
                pm = "yarn" if os.path.exists(os.path.join(path, "yarn.lock")) else "npm"
                result = subprocess.run(
                    [pm, "install"],
                    cwd=path,
                    capture_output=True,
                    text=True,
                    timeout=300
                )
                results.append(f"Node.js ({pm}): {'Installed' if result.returncode == 0 else 'Failed'}")
            
            if os.path.exists(os.path.join(path, "go.mod")):
                # Go
                result = subprocess.run(
                    ["go", "mod", "download"],
                    cwd=path,
                    capture_output=True,
                    text=True,
                    timeout=120
                )
                results.append(f"Go: {'Installed' if result.returncode == 0 else 'Failed'}")
            
            if os.path.exists(os.path.join(path, "Cargo.toml")):
                # Rust
                result = subprocess.run(
                    ["cargo", "fetch"],
                    cwd=path,
                    capture_output=True,
                    text=True,
                    timeout=120
                )
                results.append(f"Rust: {'Installed' if result.returncode == 0 else 'Failed'}")
            
            if os.path.exists(os.path.join(path, "Gemfile")):
                # Ruby
                result = subprocess.run(
                    ["bundle", "install"],
                    cwd=path,
                    capture_output=True,
                    text=True,
                    timeout=300
                )
                results.append(f"Ruby: {'Installed' if result.returncode == 0 else 'Failed'}")
            
            if not results:
                return "No recognized dependency files found"
            
            return "\n".join(results)
        except Exception as e:
            return f"Error: {str(e)}"


@dataclass
class RunTestsTool(Tool):
    """Run project tests."""
    
    name: str = "run_tests"
    description: str = "Detect project type and run tests"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="path",
            type="string",
            description="Project path",
            required=True
        ),
        ToolParameter(
            name="verbose",
            type="boolean",
            description="Verbose output",
            required=False,
            default=False
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.SAFE
    
    def execute(self, path: str, verbose: bool = False) -> str:
        """Run tests based on project type."""
        try:
            # Python
            if os.path.exists(os.path.join(path, "pytest.ini")) or \
               os.path.exists(os.path.join(path, "tests")):
                cmd = ["pytest"]
                if verbose:
                    cmd.append("-v")
                result = subprocess.run(cmd, cwd=path, capture_output=True, text=True, timeout=300)
                return result.stdout + result.stderr
            
            # Node.js
            if os.path.exists(os.path.join(path, "package.json")):
                result = subprocess.run(
                    ["npm", "test"],
                    cwd=path,
                    capture_output=True,
                    text=True,
                    timeout=300
                )
                return result.stdout + result.stderr
            
            # Go
            if os.path.exists(os.path.join(path, "go.mod")):
                cmd = ["go", "test", "./..."]
                if verbose:
                    cmd.append("-v")
                result = subprocess.run(cmd, cwd=path, capture_output=True, text=True, timeout=300)
                return result.stdout + result.stderr
            
            # Rust
            if os.path.exists(os.path.join(path, "Cargo.toml")):
                result = subprocess.run(
                    ["cargo", "test"],
                    cwd=path,
                    capture_output=True,
                    text=True,
                    timeout=300
                )
                return result.stdout + result.stderr
            
            return "No test framework detected"
        except subprocess.TimeoutExpired:
            return "Error: Tests timed out"
        except Exception as e:
            return f"Error: {str(e)}"
