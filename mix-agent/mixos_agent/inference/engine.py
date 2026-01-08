"""Inference engine using llama.cpp."""

from typing import List, Dict, Optional
from pathlib import Path
from loguru import logger

from ..config.loader import ModelConfig, InferenceConfig


class InferenceEngine:
    """Inference engine using llama-cpp-python."""
    
    def __init__(self, model_config: ModelConfig, inference_config: InferenceConfig):
        self.model_config = model_config
        self.inference_config = inference_config
        self.model = None
        
        self._load_model()
    
    def _load_model(self):
        """Load the LLM model."""
        model_path = Path(self.model_config.path)
        
        if not model_path.exists():
            logger.warning(f"Model not found at {model_path}, using mock mode")
            self.model = None
            return
        
        try:
            from llama_cpp import Llama
            
            logger.info(f"Loading model from {model_path}")
            self.model = Llama(
                model_path=str(model_path),
                n_ctx=self.model_config.context_length,
                n_threads=self.inference_config.threads,
                n_batch=self.inference_config.batch_size,
                n_gpu_layers=self.inference_config.gpu_layers,
                verbose=False,
            )
            logger.info("Model loaded successfully")
        except ImportError:
            logger.warning("llama-cpp-python not installed, using mock mode")
            self.model = None
        except Exception as e:
            logger.error(f"Failed to load model: {e}")
            self.model = None
    
    def generate(
        self,
        messages: List[Dict[str, str]],
        max_tokens: int = 512,
        stop: Optional[List[str]] = None,
    ) -> str:
        """Generate a response from the model."""
        if self.model is None:
            return self._mock_generate(messages)
        
        try:
            response = self.model.create_chat_completion(
                messages=messages,
                max_tokens=max_tokens,
                temperature=self.model_config.temperature,
                top_p=self.model_config.top_p,
                top_k=self.model_config.top_k,
                repeat_penalty=self.model_config.repeat_penalty,
                stop=stop,
            )
            
            return response["choices"][0]["message"]["content"]
        except Exception as e:
            logger.error(f"Generation failed: {e}")
            return f"I encountered an error: {e}"
    
    def _mock_generate(self, messages: List[Dict[str, str]]) -> str:
        """Generate a mock response when model is not available."""
        last_message = messages[-1]["content"] if messages else ""
        
        # Simple keyword-based responses for testing
        if "install" in last_message.lower():
            return "I'll help you install that package. Let me run the installation command."
        elif "status" in last_message.lower():
            return "The system is running normally. All services are operational."
        elif "help" in last_message.lower():
            return """I can help you with:
- Installing packages: `mix install <package>`
- System status: `mix status`
- Docker management: `mix container list`
- Development setup: `mix dev setup`

What would you like to do?"""
        else:
            return "I understand. How can I assist you with your MIXOS system?"
