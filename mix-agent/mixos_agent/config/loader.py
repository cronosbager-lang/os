"""Configuration loader for MIXOS Agent."""

from dataclasses import dataclass, field
from pathlib import Path
from typing import List, Optional
import toml


@dataclass
class ModelConfig:
    path: str = "/opt/mixos/ai/model/mix-small-1.1b-q4.gguf"
    context_length: int = 4096
    temperature: float = 0.7
    top_p: float = 0.95
    top_k: int = 40
    repeat_penalty: float = 1.1


@dataclass
class InferenceConfig:
    threads: int = 4
    batch_size: int = 512
    gpu_layers: int = 0


@dataclass
class MemoryConfig:
    max_conversation_history: int = 50
    enable_long_term_memory: bool = True
    memory_db_path: str = "/var/lib/mixos/agent/memory.db"


@dataclass
class SafetyConfig:
    require_confirmation: List[str] = field(default_factory=lambda: [
        "install_package", "remove_package", "system_update"
    ])
    forbidden_commands: List[str] = field(default_factory=lambda: [
        "rm -rf /", "dd if=/dev/zero"
    ])
    max_execution_time: int = 300
    enable_sandboxing: bool = True


@dataclass
class APIConfig:
    host: str = "127.0.0.1"
    port: int = 8765
    enable_cors: bool = True


@dataclass
class LoggingConfig:
    level: str = "info"
    path: str = "/var/log/mixos/agent.log"
    max_size: str = "100MB"
    rotation: str = "daily"


@dataclass
class AgentConfig:
    name: str = "Mix Agent"
    version: str = "1.0.0"
    enabled: bool = True
    auto_start: bool = True
    
    model: ModelConfig = field(default_factory=ModelConfig)
    inference: InferenceConfig = field(default_factory=InferenceConfig)
    memory: MemoryConfig = field(default_factory=MemoryConfig)
    safety: SafetyConfig = field(default_factory=SafetyConfig)
    api: APIConfig = field(default_factory=APIConfig)
    logging: LoggingConfig = field(default_factory=LoggingConfig)


def load_config(config_path: str) -> AgentConfig:
    """Load configuration from TOML file."""
    path = Path(config_path)
    
    if not path.exists():
        # Return default config if file doesn't exist
        return AgentConfig()
    
    with open(path) as f:
        data = toml.load(f)
    
    config = AgentConfig()
    
    # Load agent section
    if "agent" in data:
        agent = data["agent"]
        config.name = agent.get("name", config.name)
        config.version = agent.get("version", config.version)
        config.enabled = agent.get("enabled", config.enabled)
        config.auto_start = agent.get("auto_start", config.auto_start)
    
    # Load model section
    if "model" in data:
        model = data["model"]
        config.model = ModelConfig(
            path=model.get("path", config.model.path),
            context_length=model.get("context_length", config.model.context_length),
            temperature=model.get("temperature", config.model.temperature),
            top_p=model.get("top_p", config.model.top_p),
            top_k=model.get("top_k", config.model.top_k),
            repeat_penalty=model.get("repeat_penalty", config.model.repeat_penalty),
        )
    
    # Load inference section
    if "inference" in data:
        inference = data["inference"]
        config.inference = InferenceConfig(
            threads=inference.get("threads", config.inference.threads),
            batch_size=inference.get("batch_size", config.inference.batch_size),
            gpu_layers=inference.get("gpu_layers", config.inference.gpu_layers),
        )
    
    # Load memory section
    if "memory" in data:
        memory = data["memory"]
        config.memory = MemoryConfig(
            max_conversation_history=memory.get("max_conversation_history", 
                                                config.memory.max_conversation_history),
            enable_long_term_memory=memory.get("enable_long_term_memory",
                                               config.memory.enable_long_term_memory),
            memory_db_path=memory.get("memory_db_path", config.memory.memory_db_path),
        )
    
    # Load safety section
    if "safety" in data:
        safety = data["safety"]
        config.safety = SafetyConfig(
            require_confirmation=safety.get("require_confirmation",
                                           config.safety.require_confirmation),
            forbidden_commands=safety.get("forbidden_commands",
                                         config.safety.forbidden_commands),
            max_execution_time=safety.get("max_execution_time",
                                         config.safety.max_execution_time),
            enable_sandboxing=safety.get("enable_sandboxing",
                                        config.safety.enable_sandboxing),
        )
    
    # Load API section
    if "api" in data:
        api = data["api"]
        config.api = APIConfig(
            host=api.get("host", config.api.host),
            port=api.get("port", config.api.port),
            enable_cors=api.get("enable_cors", config.api.enable_cors),
        )
    
    # Load logging section
    if "logging" in data:
        logging_cfg = data["logging"]
        config.logging = LoggingConfig(
            level=logging_cfg.get("level", config.logging.level),
            path=logging_cfg.get("path", config.logging.path),
            max_size=logging_cfg.get("max_size", config.logging.max_size),
            rotation=logging_cfg.get("rotation", config.logging.rotation),
        )
    
    return config
