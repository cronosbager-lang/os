"""HTTP tools for the AI agent."""

import subprocess
import os
import json
from typing import Optional, List, Dict
from dataclasses import dataclass, field
import urllib.request
import urllib.error
import urllib.parse

from ..base import Tool, ToolParameter, SafetyLevel


@dataclass
class HttpRequestTool(Tool):
    """Make HTTP requests."""
    
    name: str = "http_request"
    description: str = "Make an HTTP request"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="url",
            type="string",
            description="URL to request",
            required=True
        ),
        ToolParameter(
            name="method",
            type="string",
            description="HTTP method (GET, POST, PUT, DELETE, PATCH)",
            required=False,
            default="GET"
        ),
        ToolParameter(
            name="headers",
            type="string",
            description="Headers as JSON string",
            required=False
        ),
        ToolParameter(
            name="data",
            type="string",
            description="Request body",
            required=False
        ),
        ToolParameter(
            name="timeout",
            type="integer",
            description="Timeout in seconds",
            required=False,
            default=30
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.SAFE
    
    def execute(
        self,
        url: str,
        method: str = "GET",
        headers: str = None,
        data: str = None,
        timeout: int = 30
    ) -> str:
        """Make an HTTP request."""
        try:
            # Parse headers
            header_dict = {}
            if headers:
                try:
                    header_dict = json.loads(headers)
                except json.JSONDecodeError:
                    pass
            
            # Prepare request
            req = urllib.request.Request(url, method=method.upper())
            
            for key, value in header_dict.items():
                req.add_header(key, value)
            
            # Add data if provided
            if data:
                req.data = data.encode('utf-8')
                if 'Content-Type' not in header_dict:
                    req.add_header('Content-Type', 'application/json')
            
            # Make request
            with urllib.request.urlopen(req, timeout=timeout) as response:
                status = response.status
                response_headers = dict(response.headers)
                body = response.read().decode('utf-8')
                
                # Try to format JSON response
                try:
                    body = json.dumps(json.loads(body), indent=2)
                except:
                    pass
                
                return f"Status: {status}\nHeaders: {json.dumps(response_headers, indent=2)}\n\nBody:\n{body}"
        
        except urllib.error.HTTPError as e:
            return f"HTTP Error {e.code}: {e.reason}\n{e.read().decode('utf-8', errors='ignore')}"
        except urllib.error.URLError as e:
            return f"URL Error: {e.reason}"
        except Exception as e:
            return f"Error: {str(e)}"


@dataclass
class HttpGetTool(Tool):
    """Make HTTP GET request."""
    
    name: str = "http_get"
    description: str = "Make an HTTP GET request"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="url",
            type="string",
            description="URL to request",
            required=True
        ),
        ToolParameter(
            name="headers",
            type="string",
            description="Headers as JSON string",
            required=False
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.SAFE
    
    def execute(self, url: str, headers: str = None) -> str:
        """Make GET request."""
        tool = HttpRequestTool()
        return tool.execute(url=url, method="GET", headers=headers)


@dataclass
class HttpPostTool(Tool):
    """Make HTTP POST request."""
    
    name: str = "http_post"
    description: str = "Make an HTTP POST request"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="url",
            type="string",
            description="URL to request",
            required=True
        ),
        ToolParameter(
            name="data",
            type="string",
            description="Request body (JSON)",
            required=True
        ),
        ToolParameter(
            name="headers",
            type="string",
            description="Headers as JSON string",
            required=False
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.SAFE
    
    def execute(self, url: str, data: str, headers: str = None) -> str:
        """Make POST request."""
        tool = HttpRequestTool()
        return tool.execute(url=url, method="POST", data=data, headers=headers)


@dataclass
class DownloadFileTool(Tool):
    """Download a file from URL."""
    
    name: str = "download_file"
    description: str = "Download a file from a URL"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="url",
            type="string",
            description="URL to download from",
            required=True
        ),
        ToolParameter(
            name="path",
            type="string",
            description="Destination path",
            required=True
        ),
        ToolParameter(
            name="overwrite",
            type="boolean",
            description="Overwrite if exists",
            required=False,
            default=False
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.REQUIRES_CONFIRMATION
    
    def execute(self, url: str, path: str, overwrite: bool = False) -> str:
        """Download a file."""
        try:
            # Check if file exists
            if os.path.exists(path) and not overwrite:
                return f"Error: File already exists at {path}"
            
            # Create directory if needed
            os.makedirs(os.path.dirname(os.path.abspath(path)), exist_ok=True)
            
            # Download using curl for better progress handling
            result = subprocess.run(
                ["curl", "-L", "-o", path, url],
                capture_output=True,
                text=True,
                timeout=300
            )
            
            if result.returncode != 0:
                return f"Error downloading: {result.stderr}"
            
            # Get file size
            size = os.path.getsize(path)
            return f"Downloaded {url} to {path} ({size} bytes)"
        
        except subprocess.TimeoutExpired:
            return "Error: Download timed out"
        except Exception as e:
            return f"Error: {str(e)}"


@dataclass
class CurlTool(Tool):
    """Execute curl command."""
    
    name: str = "curl"
    description: str = "Execute a curl command"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="url",
            type="string",
            description="URL to request",
            required=True
        ),
        ToolParameter(
            name="options",
            type="string",
            description="Additional curl options",
            required=False
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.SAFE
    
    def execute(self, url: str, options: str = "") -> str:
        """Execute curl."""
        cmd = ["curl", "-s"]
        
        if options:
            cmd.extend(options.split())
        
        cmd.append(url)
        
        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=60
            )
            
            return result.stdout or result.stderr
        except subprocess.TimeoutExpired:
            return "Error: Request timed out"
        except Exception as e:
            return f"Error: {str(e)}"


@dataclass
class PingTool(Tool):
    """Ping a host."""
    
    name: str = "ping"
    description: str = "Ping a host to check connectivity"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="host",
            type="string",
            description="Host to ping",
            required=True
        ),
        ToolParameter(
            name="count",
            type="integer",
            description="Number of pings",
            required=False,
            default=4
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.SAFE
    
    def execute(self, host: str, count: int = 4) -> str:
        """Ping a host."""
        try:
            result = subprocess.run(
                ["ping", "-c", str(count), host],
                capture_output=True,
                text=True,
                timeout=30
            )
            
            return result.stdout or result.stderr
        except subprocess.TimeoutExpired:
            return f"Error: Ping to {host} timed out"
        except Exception as e:
            return f"Error: {str(e)}"


@dataclass
class CheckPortTool(Tool):
    """Check if a port is open."""
    
    name: str = "check_port"
    description: str = "Check if a port is open on a host"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="host",
            type="string",
            description="Host to check",
            required=True
        ),
        ToolParameter(
            name="port",
            type="integer",
            description="Port number",
            required=True
        ),
        ToolParameter(
            name="timeout",
            type="integer",
            description="Timeout in seconds",
            required=False,
            default=5
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.SAFE
    
    def execute(self, host: str, port: int, timeout: int = 5) -> str:
        """Check port."""
        try:
            result = subprocess.run(
                ["nc", "-zv", "-w", str(timeout), host, str(port)],
                capture_output=True,
                text=True,
                timeout=timeout + 5
            )
            
            if result.returncode == 0:
                return f"Port {port} on {host} is OPEN"
            else:
                return f"Port {port} on {host} is CLOSED"
        except subprocess.TimeoutExpired:
            return f"Port {port} on {host} - connection timed out"
        except Exception as e:
            return f"Error: {str(e)}"
