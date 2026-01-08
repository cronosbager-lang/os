#!/usr/bin/env python3
"""Integration tests for MIXOS AI Agent."""

import os
import sys
import json
import time
import unittest
import subprocess
import tempfile
from pathlib import Path

# Add mix-agent to path
ROOT_DIR = Path(__file__).parent.parent.parent
sys.path.insert(0, str(ROOT_DIR / "mix-agent"))

try:
    from mixos_agent.config.loader import AgentConfig, load_config
    from mixos_agent.core.agent import Agent
    from mixos_agent.tools.registry import ToolRegistry
    from mixos_agent.safety.validator import SafetyValidator
    HAS_AGENT = True
except ImportError:
    HAS_AGENT = False


class TestAgentConfig(unittest.TestCase):
    """Test agent configuration loading."""
    
    def test_default_config(self):
        """Test default configuration values."""
        if not HAS_AGENT:
            self.skipTest("mix-agent not available")
        
        config = AgentConfig()
        self.assertEqual(config.name, "Mix Agent")
        self.assertEqual(config.version, "1.0.0")
        self.assertTrue(config.enabled)
        self.assertEqual(config.model.context_length, 4096)
        self.assertEqual(config.inference.threads, 4)
    
    def test_load_config_file(self):
        """Test loading configuration from file."""
        if not HAS_AGENT:
            self.skipTest("mix-agent not available")
        
        # Create temporary config file
        with tempfile.NamedTemporaryFile(mode='w', suffix='.toml', delete=False) as f:
            f.write("""
[agent]
name = "Test Agent"
version = "2.0.0"

[model]
context_length = 2048
temperature = 0.5

[inference]
threads = 8
""")
            config_path = f.name
        
        try:
            config = load_config(config_path)
            self.assertEqual(config.name, "Test Agent")
            self.assertEqual(config.version, "2.0.0")
            self.assertEqual(config.model.context_length, 2048)
            self.assertEqual(config.model.temperature, 0.5)
            self.assertEqual(config.inference.threads, 8)
        finally:
            os.unlink(config_path)


class TestToolRegistry(unittest.TestCase):
    """Test tool registry functionality."""
    
    def test_default_tools_registered(self):
        """Test that default tools are registered."""
        if not HAS_AGENT:
            self.skipTest("mix-agent not available")
        
        registry = ToolRegistry()
        tools = registry.list_tools()
        
        self.assertGreater(len(tools), 0)
        
        # Check for essential tools
        tool_names = registry.get_tool_names()
        self.assertIn("install_package", tool_names)
        self.assertIn("execute_command", tool_names)
        self.assertIn("read_file", tool_names)
    
    def test_get_tool(self):
        """Test getting a specific tool."""
        if not HAS_AGENT:
            self.skipTest("mix-agent not available")
        
        registry = ToolRegistry()
        
        tool = registry.get_tool("read_file")
        self.assertIsNotNone(tool)
        self.assertEqual(tool.name, "read_file")
        
        # Non-existent tool
        tool = registry.get_tool("nonexistent_tool")
        self.assertIsNone(tool)


class TestSafetyValidator(unittest.TestCase):
    """Test safety validation."""
    
    def setUp(self):
        if not HAS_AGENT:
            self.skipTest("mix-agent not available")
        
        from mixos_agent.config.loader import SafetyConfig
        self.config = SafetyConfig()
        self.validator = SafetyValidator(self.config)
    
    def test_safe_commands(self):
        """Test that safe commands are allowed."""
        safe_commands = [
            "ls -la",
            "cat /etc/os-release",
            "echo hello",
            "pwd",
            "whoami",
        ]
        
        for cmd in safe_commands:
            self.assertTrue(
                self.validator.is_command_safe(cmd),
                f"Command should be safe: {cmd}"
            )
    
    def test_dangerous_commands_blocked(self):
        """Test that dangerous commands are blocked."""
        dangerous_commands = [
            "rm -rf /",
            "rm -rf /*",
            "dd if=/dev/zero of=/dev/sda",
            ":(){ :|:& };:",  # Fork bomb
        ]
        
        for cmd in dangerous_commands:
            self.assertFalse(
                self.validator.is_command_safe(cmd),
                f"Command should be blocked: {cmd}"
            )
    
    def test_forbidden_commands(self):
        """Test forbidden command list."""
        for forbidden in self.config.forbidden_commands:
            self.assertFalse(
                self.validator.is_safe(forbidden),
                f"Forbidden command should be blocked: {forbidden}"
            )


class TestAgentIntegration(unittest.TestCase):
    """Integration tests for the full agent."""
    
    def setUp(self):
        if not HAS_AGENT:
            self.skipTest("mix-agent not available")
        
        self.config = AgentConfig()
        # Use mock mode (no model file)
        self.config.model.path = "/nonexistent/model.gguf"
    
    def test_agent_initialization(self):
        """Test agent initializes correctly."""
        agent = Agent(self.config)
        
        self.assertIsNotNone(agent.inference)
        self.assertIsNotNone(agent.tools)
        self.assertIsNotNone(agent.memory)
        self.assertIsNotNone(agent.safety)
    
    def test_agent_status(self):
        """Test agent status reporting."""
        agent = Agent(self.config)
        status = agent.get_status()
        
        self.assertEqual(status["status"], "running")
        self.assertIn("uptime", status)
        self.assertIn("memory_usage", status)
    
    def test_agent_chat_mock(self):
        """Test chat in mock mode."""
        agent = Agent(self.config)
        
        response = agent.chat("help")
        self.assertIsInstance(response, str)
        self.assertGreater(len(response), 0)


class TestToolExecution(unittest.TestCase):
    """Test tool execution."""
    
    def setUp(self):
        if not HAS_AGENT:
            self.skipTest("mix-agent not available")
    
    def test_read_file_tool(self):
        """Test read_file tool."""
        registry = ToolRegistry()
        tool = registry.get_tool("read_file")
        
        # Create temp file
        with tempfile.NamedTemporaryFile(mode='w', delete=False) as f:
            f.write("test content")
            temp_path = f.name
        
        try:
            result = tool.execute(path=temp_path)
            self.assertEqual(result, "test content")
        finally:
            os.unlink(temp_path)
    
    def test_create_directory_tool(self):
        """Test create_directory tool."""
        registry = ToolRegistry()
        tool = registry.get_tool("create_directory")
        
        with tempfile.TemporaryDirectory() as tmpdir:
            new_dir = os.path.join(tmpdir, "test_dir")
            result = tool.execute(path=new_dir)
            
            self.assertTrue(os.path.isdir(new_dir))
            self.assertIn("Created", result)
    
    def test_execute_command_tool(self):
        """Test execute_command tool."""
        registry = ToolRegistry()
        tool = registry.get_tool("execute_command")
        
        result = tool.execute(command="echo hello")
        self.assertIn("hello", result)


def run_tests():
    """Run all tests."""
    loader = unittest.TestLoader()
    suite = unittest.TestSuite()
    
    # Add test classes
    suite.addTests(loader.loadTestsFromTestCase(TestAgentConfig))
    suite.addTests(loader.loadTestsFromTestCase(TestToolRegistry))
    suite.addTests(loader.loadTestsFromTestCase(TestSafetyValidator))
    suite.addTests(loader.loadTestsFromTestCase(TestAgentIntegration))
    suite.addTests(loader.loadTestsFromTestCase(TestToolExecution))
    
    # Run tests
    runner = unittest.TextTestRunner(verbosity=2)
    result = runner.run(suite)
    
    return 0 if result.wasSuccessful() else 1


if __name__ == "__main__":
    sys.exit(run_tests())
