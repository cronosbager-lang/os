"""Daemon monitoring for the AI agent."""

import os
import sys
import time
import signal
import threading
import asyncio
import psutil
from typing import Optional, Dict, Any, Callable, List
from dataclasses import dataclass, field
from datetime import datetime, timedelta
from collections import deque
import json
from loguru import logger


@dataclass
class RequestMetrics:
    """Metrics for a single request."""
    timestamp: float
    duration: float
    success: bool
    endpoint: str = ""
    tokens: int = 0


@dataclass
class DaemonStatus:
    """Daemon status information."""
    pid: int
    status: str  # running, stopped, error
    uptime: float
    cpu_percent: float
    memory_mb: float
    memory_percent: float
    threads: int
    open_files: int = 0
    connections: int = 0
    requests_total: int = 0
    requests_ok: int = 0
    requests_error: int = 0
    requests_per_minute: float = 0.0
    average_response_time: float = 0.0
    tokens_total: int = 0
    last_request_time: Optional[float] = None
    last_error: Optional[str] = None
    last_error_time: Optional[float] = None


@dataclass
class SystemMetrics:
    """System-wide metrics."""
    cpu_percent: float
    memory_percent: float
    memory_available_mb: float
    disk_percent: float
    disk_free_gb: float
    load_average: List[float]
    network_bytes_sent: int = 0
    network_bytes_recv: int = 0


class DaemonMonitor:
    """Monitors the agent daemon."""
    
    def __init__(
        self,
        pid_file: str = None,
        status_file: str = None,
        check_interval: int = 5,
        metrics_window: int = 300,  # 5 minutes
    ):
        self.pid_file = pid_file or "/var/run/mixos-agent.pid"
        self.status_file = status_file or "/var/run/mixos-agent.status"
        self.check_interval = check_interval
        self.metrics_window = metrics_window
        
        self.start_time = time.time()
        self.requests_total = 0
        self.requests_ok = 0
        self.requests_error = 0
        self.tokens_total = 0
        self.last_request_time: Optional[float] = None
        self.last_error: Optional[str] = None
        self.last_error_time: Optional[float] = None
        
        # Recent metrics for rate calculations
        self._recent_requests: deque = deque(maxlen=1000)
        self._response_times: deque = deque(maxlen=1000)
        
        self._running = False
        self._monitor_thread: Optional[threading.Thread] = None
        self._callbacks: Dict[str, List[Callable]] = {}
        self._alerts: Dict[str, float] = {}  # alert_type -> last_triggered_time
        self._alert_cooldown = 60  # seconds between same alerts
    
    def start(self):
        """Start the monitor."""
        self._running = True
        self._write_pid()
        
        self._monitor_thread = threading.Thread(target=self._monitor_loop, daemon=True)
        self._monitor_thread.start()
        logger.info("Daemon monitor started")
    
    def stop(self):
        """Stop the monitor."""
        self._running = False
        
        if self._monitor_thread:
            self._monitor_thread.join(timeout=5)
        
        self._cleanup()
        logger.info("Daemon monitor stopped")
    
    def _write_pid(self):
        """Write PID file."""
        try:
            os.makedirs(os.path.dirname(self.pid_file), exist_ok=True)
            with open(self.pid_file, "w") as f:
                f.write(str(os.getpid()))
        except Exception as e:
            logger.warning(f"Failed to write PID file: {e}")
    
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
                self._check_health(status)
                self._cleanup_old_metrics()
            except Exception as e:
                logger.error(f"Monitor loop error: {e}")
                self._trigger_callback("error", {"error": str(e)})
            
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
                    "memory_percent": status.memory_percent,
                    "threads": status.threads,
                    "open_files": status.open_files,
                    "connections": status.connections,
                    "requests_total": status.requests_total,
                    "requests_ok": status.requests_ok,
                    "requests_error": status.requests_error,
                    "requests_per_minute": status.requests_per_minute,
                    "average_response_time": status.average_response_time,
                    "tokens_total": status.tokens_total,
                    "last_request_time": status.last_request_time,
                    "last_error": status.last_error,
                    "last_error_time": status.last_error_time,
                    "timestamp": time.time(),
                }, f, indent=2)
        except Exception as e:
            logger.warning(f"Failed to write status file: {e}")
    
    def _check_health(self, status: DaemonStatus):
        """Check daemon health and trigger callbacks."""
        now = time.time()
        
        # High memory usage (> 2GB or > 80%)
        if status.memory_mb > 2048 or status.memory_percent > 80:
            self._trigger_alert("high_memory", {
                "memory_mb": status.memory_mb,
                "memory_percent": status.memory_percent,
            })
        
        # High CPU usage (> 90%)
        if status.cpu_percent > 90:
            self._trigger_alert("high_cpu", {"cpu_percent": status.cpu_percent})
        
        # High error rate (> 10%)
        if status.requests_total > 100:
            error_rate = status.requests_error / status.requests_total
            if error_rate > 0.1:
                self._trigger_alert("high_error_rate", {"error_rate": error_rate})
        
        # Slow response time (> 5 seconds average)
        if status.average_response_time > 5.0:
            self._trigger_alert("slow_response", {
                "average_response_time": status.average_response_time,
            })
        
        # No requests for a long time (> 5 minutes) - might indicate issues
        if status.last_request_time and (now - status.last_request_time) > 300:
            self._trigger_alert("idle", {
                "idle_seconds": now - status.last_request_time,
            })
    
    def _trigger_alert(self, alert_type: str, data: Dict[str, Any]):
        """Trigger an alert with cooldown."""
        now = time.time()
        last_triggered = self._alerts.get(alert_type, 0)
        
        if now - last_triggered >= self._alert_cooldown:
            self._alerts[alert_type] = now
            self._trigger_callback(alert_type, data)
            logger.warning(f"Alert triggered: {alert_type} - {data}")
    
    def _trigger_callback(self, event: str, data: Any = None):
        """Trigger callbacks for an event."""
        if event in self._callbacks:
            for callback in self._callbacks[event]:
                try:
                    callback(data)
                except Exception as e:
                    logger.error(f"Callback error for {event}: {e}")
    
    def _cleanup_old_metrics(self):
        """Remove metrics older than the window."""
        cutoff = time.time() - self.metrics_window
        
        while self._recent_requests and self._recent_requests[0].timestamp < cutoff:
            self._recent_requests.popleft()
    
    def on(self, event: str, callback: Callable):
        """Register a callback for an event."""
        if event not in self._callbacks:
            self._callbacks[event] = []
        self._callbacks[event].append(callback)
    
    def off(self, event: str, callback: Callable = None):
        """Unregister a callback."""
        if event in self._callbacks:
            if callback:
                self._callbacks[event] = [c for c in self._callbacks[event] if c != callback]
            else:
                del self._callbacks[event]
    
    def get_status(self) -> DaemonStatus:
        """Get current daemon status."""
        pid = os.getpid()
        
        try:
            process = psutil.Process(pid)
            mem_info = process.memory_info()
            
            # Calculate requests per minute
            now = time.time()
            recent_count = sum(
                1 for r in self._recent_requests
                if now - r.timestamp < 60
            )
            
            # Calculate average response time
            if self._response_times:
                avg_response = sum(self._response_times) / len(self._response_times)
            else:
                avg_response = 0.0
            
            return DaemonStatus(
                pid=pid,
                status="running",
                uptime=now - self.start_time,
                cpu_percent=process.cpu_percent(),
                memory_mb=mem_info.rss / (1024 * 1024),
                memory_percent=process.memory_percent(),
                threads=process.num_threads(),
                open_files=len(process.open_files()),
                connections=len(process.connections()),
                requests_total=self.requests_total,
                requests_ok=self.requests_ok,
                requests_error=self.requests_error,
                requests_per_minute=recent_count,
                average_response_time=avg_response,
                tokens_total=self.tokens_total,
                last_request_time=self.last_request_time,
                last_error=self.last_error,
                last_error_time=self.last_error_time,
            )
        except Exception as e:
            logger.error(f"Failed to get status: {e}")
            return DaemonStatus(
                pid=pid,
                status="error",
                uptime=time.time() - self.start_time,
                cpu_percent=0,
                memory_mb=0,
                memory_percent=0,
                threads=0,
            )
    
    def get_system_metrics(self) -> SystemMetrics:
        """Get system-wide metrics."""
        try:
            mem = psutil.virtual_memory()
            disk = psutil.disk_usage("/")
            net = psutil.net_io_counters()
            
            return SystemMetrics(
                cpu_percent=psutil.cpu_percent(),
                memory_percent=mem.percent,
                memory_available_mb=mem.available / (1024 * 1024),
                disk_percent=disk.percent,
                disk_free_gb=disk.free / (1024 * 1024 * 1024),
                load_average=list(os.getloadavg()) if hasattr(os, 'getloadavg') else [0, 0, 0],
                network_bytes_sent=net.bytes_sent,
                network_bytes_recv=net.bytes_recv,
            )
        except Exception as e:
            logger.error(f"Failed to get system metrics: {e}")
            return SystemMetrics(
                cpu_percent=0,
                memory_percent=0,
                memory_available_mb=0,
                disk_percent=0,
                disk_free_gb=0,
                load_average=[0, 0, 0],
            )
    
    def record_request(
        self,
        success: bool = True,
        duration: float = 0.0,
        endpoint: str = "",
        tokens: int = 0,
        error: str = None,
    ):
        """Record a request."""
        now = time.time()
        
        self.requests_total += 1
        if success:
            self.requests_ok += 1
        else:
            self.requests_error += 1
            if error:
                self.last_error = error
                self.last_error_time = now
        
        self.last_request_time = now
        self.tokens_total += tokens
        
        # Store for rate calculations
        self._recent_requests.append(RequestMetrics(
            timestamp=now,
            duration=duration,
            success=success,
            endpoint=endpoint,
            tokens=tokens,
        ))
        
        if duration > 0:
            self._response_times.append(duration)
    
    def get_metrics_summary(self) -> Dict[str, Any]:
        """Get a summary of recent metrics."""
        now = time.time()
        
        # Last minute
        last_minute = [r for r in self._recent_requests if now - r.timestamp < 60]
        # Last 5 minutes
        last_5min = [r for r in self._recent_requests if now - r.timestamp < 300]
        
        def calc_stats(requests: List[RequestMetrics]) -> Dict[str, Any]:
            if not requests:
                return {"count": 0, "success_rate": 0, "avg_duration": 0, "tokens": 0}
            
            success_count = sum(1 for r in requests if r.success)
            durations = [r.duration for r in requests if r.duration > 0]
            
            return {
                "count": len(requests),
                "success_rate": success_count / len(requests) if requests else 0,
                "avg_duration": sum(durations) / len(durations) if durations else 0,
                "tokens": sum(r.tokens for r in requests),
            }
        
        return {
            "last_minute": calc_stats(last_minute),
            "last_5_minutes": calc_stats(last_5min),
            "total": {
                "requests": self.requests_total,
                "success": self.requests_ok,
                "errors": self.requests_error,
                "tokens": self.tokens_total,
            },
        }
    
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
