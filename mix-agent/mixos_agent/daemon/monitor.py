"""Daemon monitoring for the AI agent."""

import os
import sys
import time
import signal
import threading
import psutil
from typing import Optional, Dict, Any, Callable
from dataclasses import dataclass, field
import json


@dataclass
class DaemonStatus:
    """Daemon status information."""
    pid: int
    status: str  # running, stopped, error
    uptime: float
    cpu_percent: float
    memory_mb: float
    threads: int
    requests_total: int = 0
    requests_ok: int = 0
    requests_error: int = 0
    last_request_time: Optional[float] = None


class DaemonMonitor:
    """Monitors the agent daemon."""
    
    def __init__(
        self,
        pid_file: str = None,
        status_file: str = None,
        check_interval: int = 5,
    ):
        self.pid_file = pid_file or "/var/run/mixos-agent.pid"
        self.status_file = status_file or "/var/run/mixos-agent.status"
        self.check_interval = check_interval
        
        self.start_time = time.time()
        self.requests_total = 0
        self.requests_ok = 0
        self.requests_error = 0
        self.last_request_time: Optional[float] = None
        
        self._running = False
        self._monitor_thread: Optional[threading.Thread] = None
        self._callbacks: Dict[str, Callable] = {}
    
    def start(self):
        """Start the monitor."""
        self._running = True
        self._write_pid()
        
        self._monitor_thread = threading.Thread(target=self._monitor_loop, daemon=True)
        self._monitor_thread.start()
    
    def stop(self):
        """Stop the monitor."""
        self._running = False
        
        if self._monitor_thread:
            self._monitor_thread.join(timeout=5)
        
        self._cleanup()
    
    def _write_pid(self):
        """Write PID file."""
        try:
            os.makedirs(os.path.dirname(self.pid_file), exist_ok=True)
            with open(self.pid_file, "w") as f:
                f.write(str(os.getpid()))
        except Exception:
            pass
    
    def _cleanup(self):
        """Cleanup PID and status files."""
        for path in [self.pid_file, self.status_file]:
            try:
                if os.path.exists(path):
                    os.remove(path)
            except Exception:
                pass
    
    def _monitor_loop(self):
        """Main monitoring loop."""
        while self._running:
            try:
                status = self.get_status()
                self._write_status(status)
                
                # Check for issues
                self._check_health(status)
                
            except Exception as e:
                self._trigger_callback("error", str(e))
            
            time.sleep(self.check_interval)
    
    def _write_status(self, status: DaemonStatus):
        """Write status file."""
        try:
            os.makedirs(os.path.dirname(self.status_file), exist_ok=True)
            with open(self.status_file, "w") as f:
                json.dump({
                    "pid": status.pid,
                    "status": status.status,
                    "uptime": status.uptime,
                    "cpu_percent": status.cpu_percent,
                    "memory_mb": status.memory_mb,
                    "threads": status.threads,
                    "requests_total": status.requests_total,
                    "requests_ok": status.requests_ok,
                    "requests_error": status.requests_error,
                    "last_request_time": status.last_request_time,
                    "timestamp": time.time(),
                }, f)
        except Exception:
            pass
    
    def _check_health(self, status: DaemonStatus):
        """Check daemon health and trigger callbacks."""
        # High memory usage
        if status.memory_mb > 1024:  # > 1GB
            self._trigger_callback("high_memory", status.memory_mb)
        
        # High CPU usage
        if status.cpu_percent > 90:
            self._trigger_callback("high_cpu", status.cpu_percent)
        
        # High error rate
        if status.requests_total > 100:
            error_rate = status.requests_error / status.requests_total
            if error_rate > 0.1:  # > 10% errors
                self._trigger_callback("high_error_rate", error_rate)
    
    def _trigger_callback(self, event: str, data: Any = None):
        """Trigger a callback."""
        if event in self._callbacks:
            try:
                self._callbacks[event](data)
            except Exception:
                pass
    
    def on(self, event: str, callback: Callable):
        """Register a callback for an event."""
        self._callbacks[event] = callback
    
    def get_status(self) -> DaemonStatus:
        """Get current daemon status."""
        pid = os.getpid()
        
        try:
            process = psutil.Process(pid)
            
            return DaemonStatus(
                pid=pid,
                status="running",
                uptime=time.time() - self.start_time,
                cpu_percent=process.cpu_percent(),
                memory_mb=process.memory_info().rss / (1024 * 1024),
                threads=process.num_threads(),
                requests_total=self.requests_total,
                requests_ok=self.requests_ok,
                requests_error=self.requests_error,
                last_request_time=self.last_request_time,
            )
        except Exception:
            return DaemonStatus(
                pid=pid,
                status="error",
                uptime=time.time() - self.start_time,
                cpu_percent=0,
                memory_mb=0,
                threads=0,
            )
    
    def record_request(self, success: bool = True):
        """Record a request."""
        self.requests_total += 1
        if success:
            self.requests_ok += 1
        else:
            self.requests_error += 1
        self.last_request_time = time.time()
    
    @staticmethod
    def read_status(status_file: str = None) -> Optional[Dict[str, Any]]:
        """Read status from file (for external monitoring)."""
        status_file = status_file or "/var/run/mixos-agent.status"
        
        try:
            with open(status_file) as f:
                return json.load(f)
        except Exception:
            return None
    
    @staticmethod
    def is_running(pid_file: str = None) -> bool:
        """Check if daemon is running."""
        pid_file = pid_file or "/var/run/mixos-agent.pid"
        
        try:
            with open(pid_file) as f:
                pid = int(f.read().strip())
            
            # Check if process exists
            os.kill(pid, 0)
            return True
        except Exception:
            return False


class DaemonController:
    """Controls the agent daemon."""
    
    def __init__(
        self,
        pid_file: str = None,
        log_file: str = None,
    ):
        self.pid_file = pid_file or "/var/run/mixos-agent.pid"
        self.log_file = log_file or "/var/log/mixos-agent.log"
    
    def start(self, foreground: bool = False):
        """Start the daemon."""
        if self.is_running():
            return False, "Daemon is already running"
        
        if foreground:
            # Run in foreground
            return True, "Running in foreground"
        
        # Daemonize
        try:
            pid = os.fork()
            if pid > 0:
                # Parent process
                return True, f"Daemon started with PID {pid}"
        except OSError as e:
            return False, f"Fork failed: {e}"
        
        # Child process
        os.setsid()
        os.umask(0)
        
        # Second fork
        try:
            pid = os.fork()
            if pid > 0:
                sys.exit(0)
        except OSError:
            sys.exit(1)
        
        # Redirect standard file descriptors
        sys.stdout.flush()
        sys.stderr.flush()
        
        with open("/dev/null", "r") as devnull:
            os.dup2(devnull.fileno(), sys.stdin.fileno())
        
        with open(self.log_file, "a+") as log:
            os.dup2(log.fileno(), sys.stdout.fileno())
            os.dup2(log.fileno(), sys.stderr.fileno())
        
        # Write PID file
        with open(self.pid_file, "w") as f:
            f.write(str(os.getpid()))
        
        return True, "Daemon started"
    
    def stop(self, timeout: int = 10) -> tuple[bool, str]:
        """Stop the daemon."""
        if not self.is_running():
            return False, "Daemon is not running"
        
        try:
            with open(self.pid_file) as f:
                pid = int(f.read().strip())
            
            # Send SIGTERM
            os.kill(pid, signal.SIGTERM)
            
            # Wait for process to stop
            for _ in range(timeout):
                try:
                    os.kill(pid, 0)
                    time.sleep(1)
                except OSError:
                    # Process stopped
                    self._cleanup()
                    return True, "Daemon stopped"
            
            # Force kill
            os.kill(pid, signal.SIGKILL)
            self._cleanup()
            return True, "Daemon killed"
        
        except Exception as e:
            return False, f"Failed to stop daemon: {e}"
    
    def restart(self) -> tuple[bool, str]:
        """Restart the daemon."""
        if self.is_running():
            success, msg = self.stop()
            if not success:
                return False, f"Failed to stop: {msg}"
            time.sleep(1)
        
        return self.start()
    
    def status(self) -> Dict[str, Any]:
        """Get daemon status."""
        if not self.is_running():
            return {"status": "stopped"}
        
        status = DaemonMonitor.read_status()
        if status:
            return status
        
        return {"status": "running", "details": "Status file not available"}
    
    def is_running(self) -> bool:
        """Check if daemon is running."""
        return DaemonMonitor.is_running(self.pid_file)
    
    def _cleanup(self):
        """Cleanup PID file."""
        try:
            if os.path.exists(self.pid_file):
                os.remove(self.pid_file)
        except Exception:
            pass


class HealthChecker:
    """Performs health checks on the daemon."""
    
    def __init__(self, agent):
        self.agent = agent
        self.checks: Dict[str, Callable] = {}
        
        # Register default checks
        self.register("model", self._check_model)
        self.register("memory", self._check_memory)
        self.register("tools", self._check_tools)
    
    def register(self, name: str, check: Callable):
        """Register a health check."""
        self.checks[name] = check
    
    def run_all(self) -> Dict[str, Any]:
        """Run all health checks."""
        results = {}
        
        for name, check in self.checks.items():
            try:
                result = check()
                results[name] = {"status": "ok" if result else "fail"}
            except Exception as e:
                results[name] = {"status": "error", "error": str(e)}
        
        # Overall status
        all_ok = all(r.get("status") == "ok" for r in results.values())
        results["overall"] = "healthy" if all_ok else "unhealthy"
        
        return results
    
    def _check_model(self) -> bool:
        """Check if model is loaded."""
        return self.agent.model is not None
    
    def _check_memory(self) -> bool:
        """Check memory usage."""
        process = psutil.Process()
        memory_mb = process.memory_info().rss / (1024 * 1024)
        return memory_mb < 2048  # Less than 2GB
    
    def _check_tools(self) -> bool:
        """Check if tools are available."""
        return len(self.agent.get_tools()) > 0
