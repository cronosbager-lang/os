"""Agent daemon service."""

import os
import sys
import signal
from pathlib import Path
from loguru import logger
import uvicorn

from ..config.loader import AgentConfig
from ..core.agent import Agent
from ..api.server import create_app


class AgentDaemon:
    """Daemon service for MIXOS Agent."""
    
    def __init__(self, config: AgentConfig):
        self.config = config
        self.pid_file = Path("/var/run/mixos-agent.pid")
        self.agent = None
        self.app = None
    
    def run(self, foreground: bool = False):
        """Run the agent daemon."""
        if not foreground:
            self._daemonize()
        
        # Write PID file
        self._write_pid()
        
        # Setup signal handlers
        signal.signal(signal.SIGTERM, self._handle_signal)
        signal.signal(signal.SIGINT, self._handle_signal)
        
        try:
            # Initialize agent
            logger.info("Initializing agent...")
            self.agent = Agent(self.config)
            
            # Create API app
            logger.info("Creating API server...")
            self.app = create_app(self.config, self.agent)
            
            # Run server
            logger.info(f"Starting API server on {self.config.api.host}:{self.config.api.port}")
            uvicorn.run(
                self.app,
                host=self.config.api.host,
                port=self.config.api.port,
                log_level="info" if self.config.logging.level == "info" else "debug",
            )
        except Exception as e:
            logger.error(f"Agent failed: {e}")
            raise
        finally:
            self._cleanup()
    
    def _daemonize(self):
        """Daemonize the process."""
        # First fork
        try:
            pid = os.fork()
            if pid > 0:
                sys.exit(0)
        except OSError as e:
            logger.error(f"Fork #1 failed: {e}")
            sys.exit(1)
        
        # Decouple from parent environment
        os.chdir("/")
        os.setsid()
        os.umask(0)
        
        # Second fork
        try:
            pid = os.fork()
            if pid > 0:
                sys.exit(0)
        except OSError as e:
            logger.error(f"Fork #2 failed: {e}")
            sys.exit(1)
        
        # Redirect standard file descriptors
        sys.stdout.flush()
        sys.stderr.flush()
        
        with open("/dev/null", "r") as devnull:
            os.dup2(devnull.fileno(), sys.stdin.fileno())
    
    def _write_pid(self):
        """Write PID file."""
        self.pid_file.parent.mkdir(parents=True, exist_ok=True)
        self.pid_file.write_text(str(os.getpid()))
    
    def _cleanup(self):
        """Cleanup on exit."""
        if self.pid_file.exists():
            self.pid_file.unlink()
    
    def _handle_signal(self, signum, frame):
        """Handle termination signals."""
        logger.info(f"Received signal {signum}, shutting down...")
        self._cleanup()
        sys.exit(0)
