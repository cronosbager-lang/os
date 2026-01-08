"""Main Agent class that orchestrates all components."""

from typing import Optional, Dict, Any, List
from loguru import logger
import time

from ..config.loader import AgentConfig
from ..inference.engine import InferenceEngine
from ..tools.registry import ToolRegistry
from ..memory.manager import MemoryManager
from ..safety.validator import SafetyValidator
from .brain import Brain
from .executor import Executor
from .planner import Planner


class Agent:
    """Main MIXOS AI Agent."""
    
    def __init__(self, config: AgentConfig):
        self.config = config
        self.start_time = time.time()
        self.request_count = 0
        
        # Initialize components
        logger.info("Initializing inference engine...")
        self.inference = InferenceEngine(config.model, config.inference)
        
        logger.info("Initializing tool registry...")
        self.tools = ToolRegistry()
        
        logger.info("Initializing memory manager...")
        self.memory = MemoryManager(config.memory)
        
        logger.info("Initializing safety validator...")
        self.safety = SafetyValidator(config.safety)
        
        logger.info("Initializing brain...")
        self.brain = Brain(self.inference, self.tools, self.memory)
        
        logger.info("Initializing executor...")
        self.executor = Executor(self.tools, self.safety)
        
        logger.info("Initializing planner...")
        self.planner = Planner(self.brain)
        
        logger.info("Agent initialized successfully")
    
    def chat(self, message: str, session_id: Optional[str] = None) -> str:
        """Process a chat message and return response."""
        self.request_count += 1
        
        # Get conversation context
        context = self.memory.get_context(session_id)
        
        # Add user message to context
        context.append({"role": "user", "content": message})
        
        # Generate response
        response = self.brain.think(context)
        
        # Check if response contains tool calls
        if self.brain.has_tool_call(response):
            tool_result = self._execute_tool_call(response)
            # Generate final response with tool result
            context.append({"role": "assistant", "content": response})
            context.append({"role": "tool", "content": tool_result})
            response = self.brain.think(context)
        
        # Save to memory
        self.memory.add_message(session_id, "user", message)
        self.memory.add_message(session_id, "assistant", response)
        
        return response
    
    def execute_task(self, task: str, auto_confirm: bool = False) -> Dict[str, Any]:
        """Execute an autonomous task."""
        self.request_count += 1
        
        logger.info(f"Executing task: {task}")
        
        # Create execution plan
        plan = self.planner.create_plan(task)
        
        result = {
            "task_id": f"task_{int(time.time())}",
            "status": "in_progress",
            "message": "",
            "steps": [],
        }
        
        # Execute each step
        for step in plan.steps:
            step_result = {
                "name": step.name,
                "status": "pending",
                "output": "",
            }
            
            # Check safety
            if not auto_confirm and step.requires_confirmation:
                if not self.safety.is_safe(step.action):
                    step_result["status"] = "skipped"
                    step_result["output"] = "Requires confirmation"
                    result["steps"].append(step_result)
                    continue
            
            # Execute step
            try:
                output = self.executor.execute(step)
                step_result["status"] = "done"
                step_result["output"] = output
            except Exception as e:
                step_result["status"] = "failed"
                step_result["output"] = str(e)
                result["status"] = "failed"
                result["message"] = f"Failed at step: {step.name}"
                result["steps"].append(step_result)
                return result
            
            result["steps"].append(step_result)
        
        result["status"] = "completed"
        result["message"] = "Task completed successfully"
        
        return result
    
    def _execute_tool_call(self, response: str) -> str:
        """Extract and execute tool call from response."""
        tool_name, params = self.brain.parse_tool_call(response)
        
        if not self.safety.is_tool_allowed(tool_name):
            return f"Tool '{tool_name}' is not allowed"
        
        try:
            result = self.executor.execute_tool(tool_name, params)
            return result
        except Exception as e:
            return f"Tool execution failed: {e}"
    
    def get_status(self) -> Dict[str, Any]:
        """Get agent status."""
        uptime = time.time() - self.start_time
        hours = int(uptime // 3600)
        minutes = int((uptime % 3600) // 60)
        
        return {
            "status": "running",
            "model": self.config.model.path.split("/")[-1],
            "uptime": f"{hours}h {minutes}m",
            "memory_usage": self._get_memory_usage(),
            "requests_ok": self.request_count,
        }
    
    def _get_memory_usage(self) -> str:
        """Get current memory usage."""
        import psutil
        process = psutil.Process()
        mem = process.memory_info().rss / 1024 / 1024
        return f"{mem:.1f} MB"
