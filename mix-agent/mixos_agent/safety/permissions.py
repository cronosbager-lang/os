"""Permission management for the AI agent."""

from typing import List, Dict, Any, Optional, Set
from dataclasses import dataclass, field
from enum import Enum
import json
import os


class Permission(Enum):
    """Available permissions."""
    # File operations
    FILE_READ = "file:read"
    FILE_WRITE = "file:write"
    FILE_DELETE = "file:delete"
    FILE_EXECUTE = "file:execute"
    
    # System operations
    SYSTEM_INFO = "system:info"
    SYSTEM_PROCESS = "system:process"
    SYSTEM_SERVICE = "system:service"
    SYSTEM_PACKAGE = "system:package"
    
    # Network operations
    NETWORK_HTTP = "network:http"
    NETWORK_SSH = "network:ssh"
    NETWORK_SOCKET = "network:socket"
    
    # Docker operations
    DOCKER_READ = "docker:read"
    DOCKER_WRITE = "docker:write"
    DOCKER_EXEC = "docker:exec"
    
    # VM operations
    VM_READ = "vm:read"
    VM_WRITE = "vm:write"
    VM_CONTROL = "vm:control"
    
    # Git operations
    GIT_READ = "git:read"
    GIT_WRITE = "git:write"
    GIT_PUSH = "git:push"
    
    # Dangerous operations
    DANGEROUS_EXECUTE = "dangerous:execute"
    DANGEROUS_DELETE = "dangerous:delete"
    DANGEROUS_SYSTEM = "dangerous:system"


@dataclass
class PermissionSet:
    """A set of permissions."""
    permissions: Set[Permission] = field(default_factory=set)
    
    def has(self, permission: Permission) -> bool:
        """Check if permission is granted."""
        return permission in self.permissions
    
    def grant(self, permission: Permission):
        """Grant a permission."""
        self.permissions.add(permission)
    
    def revoke(self, permission: Permission):
        """Revoke a permission."""
        self.permissions.discard(permission)
    
    def grant_all(self, permissions: List[Permission]):
        """Grant multiple permissions."""
        self.permissions.update(permissions)
    
    def to_list(self) -> List[str]:
        """Convert to list of permission strings."""
        return [p.value for p in self.permissions]
    
    @classmethod
    def from_list(cls, permissions: List[str]) -> "PermissionSet":
        """Create from list of permission strings."""
        perm_set = cls()
        for p in permissions:
            try:
                perm_set.grant(Permission(p))
            except ValueError:
                pass
        return perm_set


# Predefined permission profiles
PERMISSION_PROFILES = {
    "minimal": PermissionSet(permissions={
        Permission.FILE_READ,
        Permission.SYSTEM_INFO,
    }),
    "standard": PermissionSet(permissions={
        Permission.FILE_READ,
        Permission.FILE_WRITE,
        Permission.SYSTEM_INFO,
        Permission.SYSTEM_PROCESS,
        Permission.NETWORK_HTTP,
        Permission.GIT_READ,
        Permission.GIT_WRITE,
        Permission.DOCKER_READ,
    }),
    "developer": PermissionSet(permissions={
        Permission.FILE_READ,
        Permission.FILE_WRITE,
        Permission.FILE_EXECUTE,
        Permission.SYSTEM_INFO,
        Permission.SYSTEM_PROCESS,
        Permission.SYSTEM_PACKAGE,
        Permission.NETWORK_HTTP,
        Permission.NETWORK_SSH,
        Permission.GIT_READ,
        Permission.GIT_WRITE,
        Permission.GIT_PUSH,
        Permission.DOCKER_READ,
        Permission.DOCKER_WRITE,
        Permission.DOCKER_EXEC,
        Permission.VM_READ,
        Permission.VM_WRITE,
    }),
    "admin": PermissionSet(permissions=set(Permission)),
}


class PermissionManager:
    """Manages permissions for the AI agent."""
    
    def __init__(
        self,
        config_path: str = None,
        default_profile: str = "standard",
    ):
        self.config_path = config_path or os.path.expanduser(
            "~/.mixos/agent/permissions.json"
        )
        self.permissions = PermissionSet()
        self.pending_requests: List[Dict[str, Any]] = []
        
        # Load or set default permissions
        if os.path.exists(self.config_path):
            self.load()
        else:
            self.set_profile(default_profile)
    
    def set_profile(self, profile_name: str):
        """Set permissions from a profile."""
        if profile_name in PERMISSION_PROFILES:
            self.permissions = PermissionSet(
                permissions=PERMISSION_PROFILES[profile_name].permissions.copy()
            )
    
    def check(self, permission: Permission) -> bool:
        """Check if a permission is granted."""
        return self.permissions.has(permission)
    
    def check_all(self, permissions: List[Permission]) -> bool:
        """Check if all permissions are granted."""
        return all(self.permissions.has(p) for p in permissions)
    
    def check_any(self, permissions: List[Permission]) -> bool:
        """Check if any permission is granted."""
        return any(self.permissions.has(p) for p in permissions)
    
    def grant(self, permission: Permission):
        """Grant a permission."""
        self.permissions.grant(permission)
        self.save()
    
    def revoke(self, permission: Permission):
        """Revoke a permission."""
        self.permissions.revoke(permission)
        self.save()
    
    def request_permission(
        self,
        permission: Permission,
        reason: str,
        tool_name: str = None,
    ) -> Dict[str, Any]:
        """Request a permission (for user approval)."""
        request = {
            "permission": permission.value,
            "reason": reason,
            "tool_name": tool_name,
            "status": "pending",
        }
        self.pending_requests.append(request)
        return request
    
    def approve_request(self, index: int) -> bool:
        """Approve a pending permission request."""
        if 0 <= index < len(self.pending_requests):
            request = self.pending_requests[index]
            permission = Permission(request["permission"])
            self.grant(permission)
            request["status"] = "approved"
            return True
        return False
    
    def deny_request(self, index: int) -> bool:
        """Deny a pending permission request."""
        if 0 <= index < len(self.pending_requests):
            self.pending_requests[index]["status"] = "denied"
            return True
        return False
    
    def get_pending_requests(self) -> List[Dict[str, Any]]:
        """Get pending permission requests."""
        return [r for r in self.pending_requests if r["status"] == "pending"]
    
    def save(self):
        """Save permissions to disk."""
        os.makedirs(os.path.dirname(self.config_path), exist_ok=True)
        
        data = {
            "permissions": self.permissions.to_list(),
        }
        
        with open(self.config_path, "w") as f:
            json.dump(data, f, indent=2)
    
    def load(self):
        """Load permissions from disk."""
        if not os.path.exists(self.config_path):
            return
        
        with open(self.config_path) as f:
            data = json.load(f)
        
        self.permissions = PermissionSet.from_list(data.get("permissions", []))


class PermissionChecker:
    """Decorator and utility for checking permissions."""
    
    def __init__(self, manager: PermissionManager):
        self.manager = manager
    
    def require(self, *permissions: Permission):
        """Decorator to require permissions for a function."""
        def decorator(func):
            def wrapper(*args, **kwargs):
                for perm in permissions:
                    if not self.manager.check(perm):
                        raise PermissionError(
                            f"Permission denied: {perm.value}"
                        )
                return func(*args, **kwargs)
            return wrapper
        return decorator
    
    def check_tool(self, tool_name: str) -> List[Permission]:
        """Get required permissions for a tool."""
        # Map tool names to required permissions
        tool_permissions = {
            # File tools
            "read_file": [Permission.FILE_READ],
            "write_file": [Permission.FILE_WRITE],
            "delete_file": [Permission.FILE_DELETE],
            "execute_command": [Permission.FILE_EXECUTE],
            
            # System tools
            "system_info": [Permission.SYSTEM_INFO],
            "list_processes": [Permission.SYSTEM_PROCESS],
            "kill_process": [Permission.SYSTEM_PROCESS],
            "install_package": [Permission.SYSTEM_PACKAGE],
            
            # Network tools
            "http_request": [Permission.NETWORK_HTTP],
            "ssh_command": [Permission.NETWORK_SSH],
            
            # Docker tools
            "docker_list": [Permission.DOCKER_READ],
            "docker_run": [Permission.DOCKER_WRITE],
            "docker_exec": [Permission.DOCKER_EXEC],
            
            # Git tools
            "git_status": [Permission.GIT_READ],
            "git_commit": [Permission.GIT_WRITE],
            "git_push": [Permission.GIT_PUSH],
            
            # VM tools
            "vm_list": [Permission.VM_READ],
            "vm_create": [Permission.VM_WRITE],
            "vm_start": [Permission.VM_CONTROL],
            "vm_stop": [Permission.VM_CONTROL],
        }
        
        return tool_permissions.get(tool_name, [])
    
    def can_use_tool(self, tool_name: str) -> tuple[bool, List[str]]:
        """Check if a tool can be used."""
        required = self.check_tool(tool_name)
        missing = [p.value for p in required if not self.manager.check(p)]
        return len(missing) == 0, missing
