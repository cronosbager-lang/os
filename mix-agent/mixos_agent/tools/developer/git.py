"""Git tools for the AI agent."""

import subprocess
import os
from typing import Optional, List
from dataclasses import dataclass, field

from ..base import Tool, ToolParameter, SafetyLevel


@dataclass
class GitTool(Tool):
    """Execute git commands."""
    
    name: str = "git"
    description: str = "Execute a git command"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="command",
            type="string",
            description="Git command to execute (e.g., 'status', 'log', 'diff')",
            required=True
        ),
        ToolParameter(
            name="args",
            type="string",
            description="Additional arguments for the command",
            required=False
        ),
        ToolParameter(
            name="cwd",
            type="string",
            description="Working directory",
            required=False
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.SAFE
    
    def execute(self, command: str, args: str = "", cwd: str = None) -> str:
        """Execute a git command."""
        cmd = ["git", command]
        if args:
            cmd.extend(args.split())
        
        try:
            result = subprocess.run(
                cmd,
                cwd=cwd,
                capture_output=True,
                text=True,
                timeout=60
            )
            
            if result.returncode != 0:
                return f"Error: {result.stderr}"
            
            return result.stdout or "Command completed successfully"
        except subprocess.TimeoutExpired:
            return "Error: Command timed out"
        except Exception as e:
            return f"Error: {str(e)}"


@dataclass
class GitCloneTool(Tool):
    """Clone a git repository."""
    
    name: str = "git_clone"
    description: str = "Clone a git repository"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="url",
            type="string",
            description="Repository URL to clone",
            required=True
        ),
        ToolParameter(
            name="path",
            type="string",
            description="Destination path",
            required=False
        ),
        ToolParameter(
            name="branch",
            type="string",
            description="Branch to clone",
            required=False
        ),
        ToolParameter(
            name="depth",
            type="integer",
            description="Clone depth (shallow clone)",
            required=False
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.SAFE
    
    def execute(
        self,
        url: str,
        path: str = None,
        branch: str = None,
        depth: int = None
    ) -> str:
        """Clone a git repository."""
        cmd = ["git", "clone"]
        
        if branch:
            cmd.extend(["-b", branch])
        
        if depth:
            cmd.extend(["--depth", str(depth)])
        
        cmd.append(url)
        
        if path:
            cmd.append(path)
        
        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=300
            )
            
            if result.returncode != 0:
                return f"Error cloning repository: {result.stderr}"
            
            return f"Successfully cloned {url}"
        except subprocess.TimeoutExpired:
            return "Error: Clone operation timed out"
        except Exception as e:
            return f"Error: {str(e)}"


@dataclass
class GitCommitTool(Tool):
    """Create a git commit."""
    
    name: str = "git_commit"
    description: str = "Stage files and create a commit"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="message",
            type="string",
            description="Commit message",
            required=True
        ),
        ToolParameter(
            name="files",
            type="string",
            description="Files to stage (space-separated, or '.' for all)",
            required=False,
            default="."
        ),
        ToolParameter(
            name="cwd",
            type="string",
            description="Working directory",
            required=False
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.REQUIRES_CONFIRMATION
    
    def execute(self, message: str, files: str = ".", cwd: str = None) -> str:
        """Stage files and create a commit."""
        try:
            # Stage files
            stage_cmd = ["git", "add"] + files.split()
            result = subprocess.run(
                stage_cmd,
                cwd=cwd,
                capture_output=True,
                text=True
            )
            
            if result.returncode != 0:
                return f"Error staging files: {result.stderr}"
            
            # Create commit
            commit_cmd = ["git", "commit", "-m", message]
            result = subprocess.run(
                commit_cmd,
                cwd=cwd,
                capture_output=True,
                text=True
            )
            
            if result.returncode != 0:
                if "nothing to commit" in result.stdout:
                    return "Nothing to commit"
                return f"Error creating commit: {result.stderr}"
            
            return f"Created commit: {message}"
        except Exception as e:
            return f"Error: {str(e)}"


@dataclass
class GitPushTool(Tool):
    """Push commits to remote."""
    
    name: str = "git_push"
    description: str = "Push commits to remote repository"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="remote",
            type="string",
            description="Remote name",
            required=False,
            default="origin"
        ),
        ToolParameter(
            name="branch",
            type="string",
            description="Branch to push",
            required=False
        ),
        ToolParameter(
            name="force",
            type="boolean",
            description="Force push",
            required=False,
            default=False
        ),
        ToolParameter(
            name="cwd",
            type="string",
            description="Working directory",
            required=False
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.REQUIRES_CONFIRMATION
    
    def execute(
        self,
        remote: str = "origin",
        branch: str = None,
        force: bool = False,
        cwd: str = None
    ) -> str:
        """Push commits to remote."""
        cmd = ["git", "push"]
        
        if force:
            cmd.append("--force")
        
        cmd.append(remote)
        
        if branch:
            cmd.append(branch)
        
        try:
            result = subprocess.run(
                cmd,
                cwd=cwd,
                capture_output=True,
                text=True,
                timeout=120
            )
            
            if result.returncode != 0:
                return f"Error pushing: {result.stderr}"
            
            return "Successfully pushed to remote"
        except subprocess.TimeoutExpired:
            return "Error: Push operation timed out"
        except Exception as e:
            return f"Error: {str(e)}"


@dataclass
class GitStatusTool(Tool):
    """Get git repository status."""
    
    name: str = "git_status"
    description: str = "Get the status of a git repository"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="cwd",
            type="string",
            description="Working directory",
            required=False
        ),
        ToolParameter(
            name="short",
            type="boolean",
            description="Show short status",
            required=False,
            default=False
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.SAFE
    
    def execute(self, cwd: str = None, short: bool = False) -> str:
        """Get repository status."""
        cmd = ["git", "status"]
        if short:
            cmd.append("-s")
        
        try:
            result = subprocess.run(
                cmd,
                cwd=cwd,
                capture_output=True,
                text=True
            )
            
            if result.returncode != 0:
                return f"Error: {result.stderr}"
            
            return result.stdout or "Working tree clean"
        except Exception as e:
            return f"Error: {str(e)}"


@dataclass
class GitBranchTool(Tool):
    """Manage git branches."""
    
    name: str = "git_branch"
    description: str = "List, create, or switch branches"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="action",
            type="string",
            description="Action: list, create, switch, delete",
            required=True
        ),
        ToolParameter(
            name="name",
            type="string",
            description="Branch name (for create/switch/delete)",
            required=False
        ),
        ToolParameter(
            name="cwd",
            type="string",
            description="Working directory",
            required=False
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.SAFE
    
    def execute(self, action: str, name: str = None, cwd: str = None) -> str:
        """Manage branches."""
        try:
            if action == "list":
                cmd = ["git", "branch", "-a"]
            elif action == "create":
                if not name:
                    return "Error: Branch name required"
                cmd = ["git", "checkout", "-b", name]
            elif action == "switch":
                if not name:
                    return "Error: Branch name required"
                cmd = ["git", "checkout", name]
            elif action == "delete":
                if not name:
                    return "Error: Branch name required"
                cmd = ["git", "branch", "-d", name]
            else:
                return f"Error: Unknown action '{action}'"
            
            result = subprocess.run(
                cmd,
                cwd=cwd,
                capture_output=True,
                text=True
            )
            
            if result.returncode != 0:
                return f"Error: {result.stderr}"
            
            return result.stdout or "Operation completed"
        except Exception as e:
            return f"Error: {str(e)}"


@dataclass
class GitDiffTool(Tool):
    """Show git diff."""
    
    name: str = "git_diff"
    description: str = "Show changes between commits, commit and working tree, etc."
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="target",
            type="string",
            description="Target to diff against (commit, branch, or file)",
            required=False
        ),
        ToolParameter(
            name="staged",
            type="boolean",
            description="Show staged changes",
            required=False,
            default=False
        ),
        ToolParameter(
            name="cwd",
            type="string",
            description="Working directory",
            required=False
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.SAFE
    
    def execute(self, target: str = None, staged: bool = False, cwd: str = None) -> str:
        """Show diff."""
        cmd = ["git", "diff"]
        
        if staged:
            cmd.append("--staged")
        
        if target:
            cmd.append(target)
        
        try:
            result = subprocess.run(
                cmd,
                cwd=cwd,
                capture_output=True,
                text=True
            )
            
            return result.stdout or "No changes"
        except Exception as e:
            return f"Error: {str(e)}"
