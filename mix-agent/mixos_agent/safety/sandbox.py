"""Sandbox execution environment for the AI agent."""

import subprocess
import os
import tempfile
import shutil
from typing import Optional, List, Dict, Any
from dataclasses import dataclass
import json


@dataclass
class SandboxConfig:
    """Configuration for sandbox execution."""
    enabled: bool = True
    timeout: int = 60  # seconds
    max_memory: int = 512  # MB
    max_processes: int = 10
    network_enabled: bool = False
    filesystem_readonly: bool = True
    allowed_paths: List[str] = None
    blocked_commands: List[str] = None
    
    def __post_init__(self):
        if self.allowed_paths is None:
            self.allowed_paths = ["/tmp", "/var/tmp"]
        if self.blocked_commands is None:
            self.blocked_commands = [
                "rm -rf /",
                "dd if=/dev/zero",
                "mkfs",
                "fdisk",
                "shutdown",
                "reboot",
                "halt",
                "poweroff",
            ]


@dataclass
class ExecutionResult:
    """Result of sandbox execution."""
    success: bool
    stdout: str
    stderr: str
    exit_code: int
    duration: float
    error: Optional[str] = None


class Sandbox:
    """Sandbox for safe command execution."""
    
    def __init__(self, config: SandboxConfig = None):
        self.config = config or SandboxConfig()
        self.temp_dir: Optional[str] = None
    
    def __enter__(self):
        """Create sandbox environment."""
        if self.config.enabled:
            self.temp_dir = tempfile.mkdtemp(prefix="mixos_sandbox_")
        return self
    
    def __exit__(self, exc_type, exc_val, exc_tb):
        """Cleanup sandbox environment."""
        if self.temp_dir and os.path.exists(self.temp_dir):
            shutil.rmtree(self.temp_dir, ignore_errors=True)
    
    def is_command_allowed(self, command: str) -> tuple[bool, str]:
        """Check if a command is allowed."""
        command_lower = command.lower()
        
        # Check blocked commands
        for blocked in self.config.blocked_commands:
            if blocked.lower() in command_lower:
                return False, f"Command contains blocked pattern: {blocked}"
        
        # Check for dangerous patterns
        dangerous_patterns = [
            ("rm -rf /", "Recursive deletion of root"),
            (":(){ :|:& };:", "Fork bomb"),
            ("> /dev/sd", "Direct disk write"),
            ("chmod 777 /", "Dangerous permission change"),
            ("curl | sh", "Piped remote execution"),
            ("wget | sh", "Piped remote execution"),
        ]
        
        for pattern, reason in dangerous_patterns:
            if pattern in command_lower:
                return False, f"Dangerous pattern detected: {reason}"
        
        return True, ""
    
    def execute(
        self,
        command: str,
        cwd: str = None,
        env: Dict[str, str] = None,
        input_data: str = None,
    ) -> ExecutionResult:
        """Execute a command in the sandbox."""
        import time
        
        # Check if command is allowed
        allowed, reason = self.is_command_allowed(command)
        if not allowed:
            return ExecutionResult(
                success=False,
                stdout="",
                stderr="",
                exit_code=-1,
                duration=0,
                error=f"Command blocked: {reason}",
            )
        
        # Prepare environment
        exec_env = os.environ.copy()
        if env:
            exec_env.update(env)
        
        # Set working directory
        if cwd is None:
            cwd = self.temp_dir or "/tmp"
        
        # Build command with resource limits
        if self.config.enabled:
            # Use timeout and ulimit for basic sandboxing
            wrapped_command = self._wrap_command(command)
        else:
            wrapped_command = command
        
        start_time = time.time()
        
        try:
            result = subprocess.run(
                wrapped_command,
                shell=True,
                cwd=cwd,
                env=exec_env,
                capture_output=True,
                text=True,
                timeout=self.config.timeout,
                input=input_data,
            )
            
            duration = time.time() - start_time
            
            return ExecutionResult(
                success=result.returncode == 0,
                stdout=result.stdout,
                stderr=result.stderr,
                exit_code=result.returncode,
                duration=duration,
            )
        
        except subprocess.TimeoutExpired:
            return ExecutionResult(
                success=False,
                stdout="",
                stderr="",
                exit_code=-1,
                duration=self.config.timeout,
                error=f"Command timed out after {self.config.timeout} seconds",
            )
        
        except Exception as e:
            return ExecutionResult(
                success=False,
                stdout="",
                stderr="",
                exit_code=-1,
                duration=time.time() - start_time,
                error=str(e),
            )
    
    def _wrap_command(self, command: str) -> str:
        """Wrap command with resource limits."""
        # Basic resource limiting using ulimit
        limits = []
        
        # Memory limit (in KB)
        if self.config.max_memory:
            limits.append(f"ulimit -v {self.config.max_memory * 1024}")
        
        # Process limit
        if self.config.max_processes:
            limits.append(f"ulimit -u {self.config.max_processes}")
        
        # File size limit (100MB)
        limits.append("ulimit -f 102400")
        
        # CPU time limit
        limits.append(f"ulimit -t {self.config.timeout}")
        
        if limits:
            return f"({'; '.join(limits)}; {command})"
        
        return command
    
    def execute_script(
        self,
        script: str,
        interpreter: str = "bash",
        **kwargs,
    ) -> ExecutionResult:
        """Execute a script in the sandbox."""
        # Write script to temp file
        script_path = os.path.join(
            self.temp_dir or "/tmp",
            "sandbox_script.sh"
        )
        
        with open(script_path, "w") as f:
            f.write(script)
        
        os.chmod(script_path, 0o755)
        
        return self.execute(f"{interpreter} {script_path}", **kwargs)


class DockerSandbox(Sandbox):
    """Docker-based sandbox for stronger isolation."""
    
    def __init__(
        self,
        config: SandboxConfig = None,
        image: str = "alpine:latest",
    ):
        super().__init__(config)
        self.image = image
        self.container_id: Optional[str] = None
    
    def __enter__(self):
        """Create Docker container."""
        super().__enter__()
        
        if not self._docker_available():
            return self
        
        # Create container
        cmd = [
            "docker", "run", "-d",
            "--rm",
            f"--memory={self.config.max_memory}m",
            f"--pids-limit={self.config.max_processes}",
        ]
        
        if not self.config.network_enabled:
            cmd.append("--network=none")
        
        if self.config.filesystem_readonly:
            cmd.append("--read-only")
        
        # Mount temp directory
        if self.temp_dir:
            cmd.extend(["-v", f"{self.temp_dir}:/workspace"])
        
        cmd.extend([self.image, "sleep", "infinity"])
        
        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
            )
            if result.returncode == 0:
                self.container_id = result.stdout.strip()
        except Exception:
            pass
        
        return self
    
    def __exit__(self, exc_type, exc_val, exc_tb):
        """Cleanup Docker container."""
        if self.container_id:
            try:
                subprocess.run(
                    ["docker", "kill", self.container_id],
                    capture_output=True,
                )
            except Exception:
                pass
        
        super().__exit__(exc_type, exc_val, exc_tb)
    
    def _docker_available(self) -> bool:
        """Check if Docker is available."""
        try:
            result = subprocess.run(
                ["docker", "info"],
                capture_output=True,
            )
            return result.returncode == 0
        except Exception:
            return False
    
    def execute(
        self,
        command: str,
        cwd: str = None,
        env: Dict[str, str] = None,
        input_data: str = None,
    ) -> ExecutionResult:
        """Execute command in Docker container."""
        if not self.container_id:
            # Fall back to regular sandbox
            return super().execute(command, cwd, env, input_data)
        
        import time
        
        # Check if command is allowed
        allowed, reason = self.is_command_allowed(command)
        if not allowed:
            return ExecutionResult(
                success=False,
                stdout="",
                stderr="",
                exit_code=-1,
                duration=0,
                error=f"Command blocked: {reason}",
            )
        
        # Build docker exec command
        docker_cmd = ["docker", "exec"]
        
        if cwd:
            docker_cmd.extend(["-w", cwd])
        
        if env:
            for key, value in env.items():
                docker_cmd.extend(["-e", f"{key}={value}"])
        
        docker_cmd.extend([self.container_id, "sh", "-c", command])
        
        start_time = time.time()
        
        try:
            result = subprocess.run(
                docker_cmd,
                capture_output=True,
                text=True,
                timeout=self.config.timeout,
                input=input_data,
            )
            
            duration = time.time() - start_time
            
            return ExecutionResult(
                success=result.returncode == 0,
                stdout=result.stdout,
                stderr=result.stderr,
                exit_code=result.returncode,
                duration=duration,
            )
        
        except subprocess.TimeoutExpired:
            return ExecutionResult(
                success=False,
                stdout="",
                stderr="",
                exit_code=-1,
                duration=self.config.timeout,
                error=f"Command timed out after {self.config.timeout} seconds",
            )
        
        except Exception as e:
            return ExecutionResult(
                success=False,
                stdout="",
                stderr="",
                exit_code=-1,
                duration=time.time() - start_time,
                error=str(e),
            )
