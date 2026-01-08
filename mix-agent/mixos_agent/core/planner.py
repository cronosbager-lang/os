"""Planner module - creates execution plans for tasks."""

from typing import List, Dict, Any
from dataclasses import dataclass, field
from loguru import logger

from .brain import Brain
from .executor import ExecutionStep


@dataclass
class ExecutionPlan:
    """Represents an execution plan for a task."""
    task: str
    steps: List[ExecutionStep] = field(default_factory=list)
    estimated_time: int = 0  # seconds


class Planner:
    """Creates execution plans for tasks."""
    
    def __init__(self, brain: Brain):
        self.brain = brain
        
        # Predefined task templates
        self.templates = {
            "setup_python": self._plan_setup_python,
            "setup_go": self._plan_setup_go,
            "setup_node": self._plan_setup_node,
            "setup_rust": self._plan_setup_rust,
            "install_docker": self._plan_install_docker,
        }
    
    def create_plan(self, task: str) -> ExecutionPlan:
        """Create an execution plan for a task."""
        logger.info(f"Creating plan for task: {task}")
        
        # Classify intent
        intent = self.brain.classify_intent(task)
        
        # Check for template match
        task_lower = task.lower()
        for template_name, template_func in self.templates.items():
            if template_name.replace("_", " ") in task_lower:
                return template_func(task)
        
        # Generate plan using AI
        return self._generate_plan(task, intent)
    
    def _generate_plan(self, task: str, intent: Dict[str, Any]) -> ExecutionPlan:
        """Generate a plan using AI reasoning."""
        plan = ExecutionPlan(task=task)
        
        # Simple intent-based planning
        if intent["intent"] == "install":
            package = intent["entities"].get("package", "")
            if package:
                plan.steps.append(ExecutionStep(
                    name=f"Install {package}",
                    action="install_package",
                    tool="install_package",
                    params={"package_name": package},
                    requires_confirmation=True,
                ))
        
        elif intent["intent"] == "setup":
            # Generic setup - check for language keywords
            if "python" in task.lower():
                return self._plan_setup_python(task)
            elif "go" in task.lower() or "golang" in task.lower():
                return self._plan_setup_go(task)
            elif "node" in task.lower() or "javascript" in task.lower():
                return self._plan_setup_node(task)
        
        return plan
    
    def _plan_setup_python(self, task: str) -> ExecutionPlan:
        """Create plan for Python environment setup."""
        plan = ExecutionPlan(task=task, estimated_time=120)
        
        plan.steps = [
            ExecutionStep(
                name="Check Python installation",
                action="check_command",
                tool="execute_command",
                params={"command": "python3 --version"},
            ),
            ExecutionStep(
                name="Install Python if needed",
                action="install_package",
                tool="install_package",
                params={"package_name": "python"},
                requires_confirmation=True,
            ),
            ExecutionStep(
                name="Install pip",
                action="install_package",
                tool="install_package",
                params={"package_name": "python-pip"},
                requires_confirmation=True,
            ),
            ExecutionStep(
                name="Create virtual environment",
                action="execute_command",
                tool="execute_command",
                params={"command": "python3 -m venv venv"},
            ),
            ExecutionStep(
                name="Upgrade pip",
                action="execute_command",
                tool="execute_command",
                params={"command": "venv/bin/pip install --upgrade pip"},
            ),
        ]
        
        return plan
    
    def _plan_setup_go(self, task: str) -> ExecutionPlan:
        """Create plan for Go environment setup."""
        plan = ExecutionPlan(task=task, estimated_time=60)
        
        plan.steps = [
            ExecutionStep(
                name="Install Go",
                action="install_package",
                tool="install_package",
                params={"package_name": "go"},
                requires_confirmation=True,
            ),
            ExecutionStep(
                name="Setup GOPATH",
                action="execute_command",
                tool="execute_command",
                params={"command": "mkdir -p ~/go/{bin,src,pkg}"},
            ),
            ExecutionStep(
                name="Initialize Go module",
                action="execute_command",
                tool="execute_command",
                params={"command": "go mod init"},
            ),
        ]
        
        return plan
    
    def _plan_setup_node(self, task: str) -> ExecutionPlan:
        """Create plan for Node.js environment setup."""
        plan = ExecutionPlan(task=task, estimated_time=90)
        
        plan.steps = [
            ExecutionStep(
                name="Install Node.js",
                action="install_package",
                tool="install_package",
                params={"package_name": "nodejs"},
                requires_confirmation=True,
            ),
            ExecutionStep(
                name="Install npm",
                action="install_package",
                tool="install_package",
                params={"package_name": "npm"},
                requires_confirmation=True,
            ),
            ExecutionStep(
                name="Initialize npm project",
                action="execute_command",
                tool="execute_command",
                params={"command": "npm init -y"},
            ),
        ]
        
        return plan
    
    def _plan_setup_rust(self, task: str) -> ExecutionPlan:
        """Create plan for Rust environment setup."""
        plan = ExecutionPlan(task=task, estimated_time=180)
        
        plan.steps = [
            ExecutionStep(
                name="Install Rust",
                action="install_package",
                tool="install_package",
                params={"package_name": "rust"},
                requires_confirmation=True,
            ),
            ExecutionStep(
                name="Initialize Cargo project",
                action="execute_command",
                tool="execute_command",
                params={"command": "cargo init"},
            ),
        ]
        
        return plan
    
    def _plan_install_docker(self, task: str) -> ExecutionPlan:
        """Create plan for Docker installation."""
        plan = ExecutionPlan(task=task, estimated_time=120)
        
        plan.steps = [
            ExecutionStep(
                name="Install Docker",
                action="install_package",
                tool="install_package",
                params={"package_name": "docker"},
                requires_confirmation=True,
            ),
            ExecutionStep(
                name="Start Docker service",
                action="execute_command",
                tool="execute_command",
                params={"command": "systemctl start docker"},
            ),
            ExecutionStep(
                name="Enable Docker on boot",
                action="execute_command",
                tool="execute_command",
                params={"command": "systemctl enable docker"},
            ),
            ExecutionStep(
                name="Add user to docker group",
                action="execute_command",
                tool="execute_command",
                params={"command": "usermod -aG docker $USER"},
            ),
        ]
        
        return plan
