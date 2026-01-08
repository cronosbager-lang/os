"""Main entry point for MIXOS Agent."""

import typer
from loguru import logger
import sys

from .config.loader import load_config
from .daemon.service import AgentDaemon

app = typer.Typer(help="MIXOS AI Agent")


@app.command()
def start(
    config_path: str = typer.Option(
        "/etc/mixos/agent.toml",
        "--config", "-c",
        help="Path to configuration file"
    ),
    foreground: bool = typer.Option(
        False,
        "--foreground", "-f",
        help="Run in foreground (don't daemonize)"
    ),
    debug: bool = typer.Option(
        False,
        "--debug", "-d",
        help="Enable debug logging"
    ),
):
    """Start the MIXOS AI Agent."""
    # Configure logging
    logger.remove()
    log_level = "DEBUG" if debug else "INFO"
    logger.add(sys.stderr, level=log_level)
    logger.add("/var/log/mixos/agent.log", rotation="100 MB", level=log_level)
    
    logger.info("Starting MIXOS AI Agent...")
    
    try:
        config = load_config(config_path)
        daemon = AgentDaemon(config)
        daemon.run(foreground=foreground)
    except Exception as e:
        logger.error(f"Failed to start agent: {e}")
        raise typer.Exit(1)


@app.command()
def stop():
    """Stop the MIXOS AI Agent."""
    import os
    import signal
    
    pid_file = "/var/run/mixos-agent.pid"
    
    if not os.path.exists(pid_file):
        typer.echo("Agent is not running")
        raise typer.Exit(1)
    
    with open(pid_file) as f:
        pid = int(f.read().strip())
    
    try:
        os.kill(pid, signal.SIGTERM)
        typer.echo("Agent stopped")
    except ProcessLookupError:
        os.remove(pid_file)
        typer.echo("Agent was not running (stale pid file removed)")


@app.command()
def status():
    """Show agent status."""
    import os
    import psutil
    
    pid_file = "/var/run/mixos-agent.pid"
    
    if not os.path.exists(pid_file):
        typer.echo("Agent is not running")
        raise typer.Exit(1)
    
    with open(pid_file) as f:
        pid = int(f.read().strip())
    
    try:
        proc = psutil.Process(pid)
        typer.echo(f"Agent is running (PID: {pid})")
        typer.echo(f"  Memory: {proc.memory_info().rss / 1024 / 1024:.1f} MB")
        typer.echo(f"  CPU: {proc.cpu_percent():.1f}%")
    except psutil.NoSuchProcess:
        typer.echo("Agent is not running (stale pid file)")
        os.remove(pid_file)
        raise typer.Exit(1)


def main():
    """Main entry point."""
    app()


if __name__ == "__main__":
    main()
