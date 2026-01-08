"""Safety validator for commands and actions."""

import re
from typing import List
from loguru import logger

from ..config.loader import SafetyConfig


class SafetyValidator:
    """Validates commands and actions for safety."""
    
    # Dangerous patterns that should never be executed
    DANGEROUS_PATTERNS = [
        r"rm\s+-rf\s+/\s*$",
        r"rm\s+-rf\s+/\*",
        r"dd\s+if=/dev/zero\s+of=/dev/sd",
        r"mkfs\s+/dev/sd[a-z]$",
        r":\(\)\{\s*:\|:\s*&\s*\};:",  # Fork bomb
        r"chmod\s+-R\s+777\s+/",
        r"chown\s+-R\s+.*\s+/\s*$",
        r">\s*/dev/sd[a-z]",
        r"curl.*\|\s*sh",
        r"wget.*\|\s*sh",
    ]
    
    def __init__(self, config: SafetyConfig):
        self.config = config
        self.forbidden_commands = config.forbidden_commands
        self.require_confirmation = config.require_confirmation
        
        # Compile dangerous patterns
        self._dangerous_re = [re.compile(p) for p in self.DANGEROUS_PATTERNS]
    
    def is_safe(self, action: str) -> bool:
        """Check if an action is safe to execute."""
        # Check against forbidden commands
        for forbidden in self.forbidden_commands:
            if forbidden in action:
                logger.warning(f"Blocked forbidden command: {action}")
                return False
        
        return True
    
    def is_command_safe(self, command: str) -> bool:
        """Check if a shell command is safe to execute."""
        # Check against dangerous patterns
        for pattern in self._dangerous_re:
            if pattern.search(command):
                logger.warning(f"Blocked dangerous command: {command}")
                return False
        
        # Check against forbidden commands
        for forbidden in self.forbidden_commands:
            if forbidden in command:
                logger.warning(f"Blocked forbidden command: {command}")
                return False
        
        return True
    
    def is_tool_allowed(self, tool_name: str) -> bool:
        """Check if a tool is allowed to be used."""
        # All tools are allowed by default
        # Could be extended with a blacklist
        return True
    
    def requires_confirmation(self, action: str) -> bool:
        """Check if an action requires user confirmation."""
        for pattern in self.require_confirmation:
            if pattern in action:
                return True
        return False
    
    def sanitize_command(self, command: str) -> str:
        """Sanitize a command by removing potentially dangerous parts."""
        # Remove shell redirections to system files
        command = re.sub(r">\s*/etc/", "> /tmp/", command)
        command = re.sub(r">\s*/dev/", "> /tmp/", command)
        
        return command
