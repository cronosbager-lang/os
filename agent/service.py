"""MixOS Agent Service - IPC Integration Layer"""

import asyncio
import json
import logging
import sys
from pathlib import Path

# Add parent paths for imports
sys.path.insert(0, str(Path(__file__).parent.parent / "mix-agent"))

from ipc import IPCClient

logging.basicConfig(
    level=logging.INFO,
    format='[%(asctime)s] [agent] [%(levelname)s] %(message)s',
    datefmt='%Y-%m-%d %H:%M:%S'
)
logger = logging.getLogger(__name__)


class AgentService:
    """MixOS Agent Service with IPC integration"""

    def __init__(self):
        self.ipc = IPCClient("agent")
        self.agent = None  # Will be initialized with actual agent

    async def start(self):
        """Start the agent service"""
        logger.info("Starting MixOS Agent Service...")

        # Connect to IPC broker
        if not await self.ipc.connect():
            logger.error("Failed to connect to IPC broker")
            return False

        # Register handlers
        self.ipc.register_handler("chat", self.handle_chat)
        self.ipc.register_handler("execute", self.handle_execute)
        self.ipc.register_handler("status", self.handle_status)
        self.ipc.register_handler("tools", self.handle_tools)

        # Subscribe to events
        await self.ipc.subscribe(["system.*", "package.*", "build.*"])

        # Register event handlers
        self.ipc.register_event_handler("system.shutdown", self.handle_shutdown_event)

        logger.info("Agent Service ready")
        
        # Send ready event
        await self.ipc.send_event("agent.ready", {"version": "0.1.0"})

        return True

    async def handle_chat(self, payload: bytes) -> dict:
        """Handle chat request"""
        try:
            data = json.loads(payload.decode('utf-8'))
            prompt = data.get("prompt", "")
            context = data.get("context", {})

            logger.info(f"Chat request: {prompt[:50]}...")

            # TODO: Integrate with actual agent brain
            # For now, return a placeholder response
            response = {
                "success": True,
                "response": f"Received: {prompt}",
                "actions": [],
            }

            return response
        except Exception as e:
            logger.error(f"Chat error: {e}")
            return {"success": False, "error": str(e)}

    async def handle_execute(self, payload: bytes) -> dict:
        """Handle execute request"""
        try:
            data = json.loads(payload.decode('utf-8'))
            action = data.get("action", "")
            params = data.get("params", {})

            logger.info(f"Execute request: {action}")

            # Execute action based on type
            if action == "install_package":
                # Call package manager via IPC
                from ipc.client import call_pkgmgr
                result = await call_pkgmgr(
                    self.ipc,
                    "install",
                    params.get("packages", [])
                )
                return {"success": True, "result": result}

            elif action == "build_package":
                from ipc.client import call_builder
                result = await call_builder(
                    self.ipc,
                    params.get("package_name"),
                    params.get("source_path"),
                    params.get("output_path"),
                    params.get("env", {})
                )
                return {"success": True, "result": result}

            elif action == "resolve_deps":
                from ipc.client import call_resolver
                result = await call_resolver(
                    self.ipc,
                    params.get("packages", []),
                    params.get("include_optional", False)
                )
                return {"success": True, "result": result}

            else:
                return {"success": False, "error": f"Unknown action: {action}"}

        except Exception as e:
            logger.error(f"Execute error: {e}")
            return {"success": False, "error": str(e)}

    async def handle_status(self, payload: bytes) -> dict:
        """Handle status request"""
        return {
            "success": True,
            "status": "running",
            "version": "0.1.0",
            "capabilities": [
                "chat",
                "execute",
                "package_management",
                "build",
                "dependency_resolution",
            ],
        }

    async def handle_tools(self, payload: bytes) -> dict:
        """Handle tools listing request"""
        # TODO: Get actual tools from agent
        tools = [
            {"name": "install_package", "description": "Install a package"},
            {"name": "remove_package", "description": "Remove a package"},
            {"name": "build_package", "description": "Build a package from source"},
            {"name": "resolve_deps", "description": "Resolve package dependencies"},
            {"name": "cache_get", "description": "Get value from cache"},
            {"name": "cache_put", "description": "Put value in cache"},
        ]
        return {"success": True, "tools": tools}

    async def handle_shutdown_event(self, payload: bytes):
        """Handle system shutdown event"""
        logger.info("Received shutdown event")
        await self.stop()

    async def stop(self):
        """Stop the agent service"""
        logger.info("Stopping Agent Service...")
        await self.ipc.send_event("agent.stopping", {})
        await self.ipc.close()

    async def run(self):
        """Run the agent service"""
        if not await self.start():
            return

        # Keep running until stopped
        try:
            while True:
                await asyncio.sleep(1)
        except asyncio.CancelledError:
            pass
        finally:
            await self.stop()


async def main():
    service = AgentService()
    await service.run()


if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        logger.info("Interrupted")
