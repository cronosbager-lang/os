"""Project management tools for the AI agent."""

import subprocess
import os
import json
from typing import Optional, List, Dict
from dataclasses import dataclass, field

from ..base import Tool, ToolParameter, SafetyLevel


# Project templates
PROJECT_TEMPLATES = {
    "python-fastapi": {
        "files": {
            "app/__init__.py": "",
            "app/main.py": '''from fastapi import FastAPI

app = FastAPI()

@app.get("/")
def read_root():
    return {"message": "Hello World"}

@app.get("/health")
def health():
    return {"status": "ok"}
''',
            "requirements.txt": '''fastapi>=0.109.0
uvicorn[standard]>=0.27.0
pydantic>=2.5.0
''',
            "README.md": "# FastAPI Project\n\nRun with: `uvicorn app.main:app --reload`\n",
            ".gitignore": "__pycache__/\n*.pyc\nvenv/\n.env\n",
        }
    },
    "python-flask": {
        "files": {
            "app/__init__.py": "",
            "app/main.py": '''from flask import Flask, jsonify

app = Flask(__name__)

@app.route("/")
def index():
    return jsonify({"message": "Hello World"})

if __name__ == "__main__":
    app.run(debug=True)
''',
            "requirements.txt": "flask>=3.0.0\n",
            "README.md": "# Flask Project\n\nRun with: `python app/main.py`\n",
            ".gitignore": "__pycache__/\n*.pyc\nvenv/\n.env\n",
        }
    },
    "node-express": {
        "files": {
            "src/index.js": '''const express = require('express');
const app = express();
const port = process.env.PORT || 3000;

app.use(express.json());

app.get('/', (req, res) => {
    res.json({ message: 'Hello World' });
});

app.listen(port, () => {
    console.log(`Server running on port ${port}`);
});
''',
            "package.json": json.dumps({
                "name": "express-app",
                "version": "1.0.0",
                "main": "src/index.js",
                "scripts": {
                    "start": "node src/index.js",
                    "dev": "nodemon src/index.js"
                },
                "dependencies": {
                    "express": "^4.18.0"
                },
                "devDependencies": {
                    "nodemon": "^3.0.0"
                }
            }, indent=2),
            "README.md": "# Express Project\n\nRun with: `npm start`\n",
            ".gitignore": "node_modules/\n.env\n",
        }
    },
    "go-api": {
        "files": {
            "main.go": '''package main

import (
    "encoding/json"
    "log"
    "net/http"
)

func main() {
    http.HandleFunc("/", handleRoot)
    http.HandleFunc("/health", handleHealth)
    
    log.Println("Server starting on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
    json.NewEncoder(w).Encode(map[string]string{"message": "Hello World"})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
    json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
''',
            "go.mod": "module app\n\ngo 1.21\n",
            "README.md": "# Go API\n\nRun with: `go run main.go`\n",
            ".gitignore": "*.exe\n*.test\n",
        }
    },
}


@dataclass
class CreateProjectTool(Tool):
    """Create a new project from template."""
    
    name: str = "create_project"
    description: str = "Create a new project from a template"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="name",
            type="string",
            description="Project name",
            required=True
        ),
        ToolParameter(
            name="template",
            type="string",
            description="Template: python-fastapi, python-flask, node-express, go-api",
            required=True
        ),
        ToolParameter(
            name="path",
            type="string",
            description="Parent directory for the project",
            required=False
        ),
        ToolParameter(
            name="git_init",
            type="boolean",
            description="Initialize git repository",
            required=False,
            default=True
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.SAFE
    
    def execute(
        self,
        name: str,
        template: str,
        path: str = None,
        git_init: bool = True
    ) -> str:
        """Create a new project."""
        if template not in PROJECT_TEMPLATES:
            available = ", ".join(PROJECT_TEMPLATES.keys())
            return f"Unknown template '{template}'. Available: {available}"
        
        try:
            # Determine project path
            if path:
                project_path = os.path.join(path, name)
            else:
                project_path = os.path.join(os.getcwd(), name)
            
            # Create project directory
            os.makedirs(project_path, exist_ok=True)
            
            # Create files from template
            template_data = PROJECT_TEMPLATES[template]
            for file_path, content in template_data["files"].items():
                full_path = os.path.join(project_path, file_path)
                os.makedirs(os.path.dirname(full_path), exist_ok=True)
                with open(full_path, "w") as f:
                    f.write(content)
            
            results = [f"Created project '{name}' from template '{template}'"]
            
            # Initialize git
            if git_init:
                subprocess.run(
                    ["git", "init"],
                    cwd=project_path,
                    capture_output=True
                )
                results.append("Initialized git repository")
            
            results.append(f"Location: {project_path}")
            
            return "\n".join(results)
        except Exception as e:
            return f"Error creating project: {str(e)}"


@dataclass
class DetectProjectTool(Tool):
    """Detect project type and configuration."""
    
    name: str = "detect_project"
    description: str = "Detect the type and configuration of a project"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="path",
            type="string",
            description="Project path",
            required=True
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.SAFE
    
    def execute(self, path: str) -> str:
        """Detect project type."""
        if not os.path.isdir(path):
            return f"Error: '{path}' is not a directory"
        
        info = {
            "path": os.path.abspath(path),
            "languages": [],
            "frameworks": [],
            "package_managers": [],
            "has_docker": False,
            "has_git": False,
            "has_tests": False,
        }
        
        files = os.listdir(path)
        
        # Detect languages and frameworks
        if "requirements.txt" in files or "pyproject.toml" in files:
            info["languages"].append("Python")
            info["package_managers"].append("pip")
            
            # Check for frameworks
            req_file = os.path.join(path, "requirements.txt")
            if os.path.exists(req_file):
                with open(req_file) as f:
                    content = f.read().lower()
                    if "fastapi" in content:
                        info["frameworks"].append("FastAPI")
                    if "flask" in content:
                        info["frameworks"].append("Flask")
                    if "django" in content:
                        info["frameworks"].append("Django")
        
        if "package.json" in files:
            info["languages"].append("JavaScript/TypeScript")
            info["package_managers"].append("npm")
            
            pkg_file = os.path.join(path, "package.json")
            with open(pkg_file) as f:
                pkg = json.load(f)
                deps = {**pkg.get("dependencies", {}), **pkg.get("devDependencies", {})}
                if "express" in deps:
                    info["frameworks"].append("Express")
                if "react" in deps:
                    info["frameworks"].append("React")
                if "vue" in deps:
                    info["frameworks"].append("Vue")
                if "next" in deps:
                    info["frameworks"].append("Next.js")
        
        if "go.mod" in files:
            info["languages"].append("Go")
            info["package_managers"].append("go mod")
        
        if "Cargo.toml" in files:
            info["languages"].append("Rust")
            info["package_managers"].append("cargo")
        
        if "Gemfile" in files:
            info["languages"].append("Ruby")
            info["package_managers"].append("bundler")
            if "rails" in open(os.path.join(path, "Gemfile")).read().lower():
                info["frameworks"].append("Rails")
        
        # Check for Docker
        if "Dockerfile" in files or "docker-compose.yml" in files:
            info["has_docker"] = True
        
        # Check for git
        if ".git" in files:
            info["has_git"] = True
        
        # Check for tests
        if "tests" in files or "test" in files or "spec" in files:
            info["has_tests"] = True
        
        # Format output
        output = [f"Project: {info['path']}"]
        output.append(f"Languages: {', '.join(info['languages']) or 'Unknown'}")
        output.append(f"Frameworks: {', '.join(info['frameworks']) or 'None detected'}")
        output.append(f"Package Managers: {', '.join(info['package_managers']) or 'None'}")
        output.append(f"Docker: {'Yes' if info['has_docker'] else 'No'}")
        output.append(f"Git: {'Yes' if info['has_git'] else 'No'}")
        output.append(f"Tests: {'Yes' if info['has_tests'] else 'No'}")
        
        return "\n".join(output)


@dataclass
class SetupDevEnvTool(Tool):
    """Setup complete development environment."""
    
    name: str = "setup_dev_env"
    description: str = "Setup a complete development environment for a project"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="path",
            type="string",
            description="Project path",
            required=True
        ),
        ToolParameter(
            name="install_deps",
            type="boolean",
            description="Install dependencies",
            required=False,
            default=True
        ),
        ToolParameter(
            name="setup_docker",
            type="boolean",
            description="Setup Docker if not present",
            required=False,
            default=False
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.SAFE
    
    def execute(
        self,
        path: str,
        install_deps: bool = True,
        setup_docker: bool = False
    ) -> str:
        """Setup development environment."""
        results = []
        
        try:
            # Detect project type
            files = os.listdir(path)
            
            # Python setup
            if "requirements.txt" in files or "pyproject.toml" in files:
                venv_path = os.path.join(path, "venv")
                if not os.path.exists(venv_path):
                    subprocess.run(
                        ["python3", "-m", "venv", venv_path],
                        capture_output=True
                    )
                    results.append("Created Python virtualenv")
                
                if install_deps:
                    pip = os.path.join(venv_path, "bin", "pip")
                    if "requirements.txt" in files:
                        subprocess.run(
                            [pip, "install", "-r", "requirements.txt"],
                            cwd=path,
                            capture_output=True,
                            timeout=300
                        )
                        results.append("Installed Python dependencies")
            
            # Node.js setup
            if "package.json" in files:
                if install_deps and "node_modules" not in files:
                    pm = "yarn" if "yarn.lock" in files else "npm"
                    subprocess.run(
                        [pm, "install"],
                        cwd=path,
                        capture_output=True,
                        timeout=300
                    )
                    results.append(f"Installed Node.js dependencies ({pm})")
            
            # Go setup
            if "go.mod" in files:
                if install_deps:
                    subprocess.run(
                        ["go", "mod", "download"],
                        cwd=path,
                        capture_output=True,
                        timeout=120
                    )
                    results.append("Downloaded Go dependencies")
            
            # Rust setup
            if "Cargo.toml" in files:
                if install_deps:
                    subprocess.run(
                        ["cargo", "fetch"],
                        cwd=path,
                        capture_output=True,
                        timeout=120
                    )
                    results.append("Fetched Rust dependencies")
            
            # Docker setup
            if setup_docker and "Dockerfile" not in files:
                # Generate basic Dockerfile based on project type
                dockerfile = self._generate_dockerfile(path, files)
                if dockerfile:
                    with open(os.path.join(path, "Dockerfile"), "w") as f:
                        f.write(dockerfile)
                    results.append("Generated Dockerfile")
            
            return "\n".join(results) or "Environment already set up"
        except Exception as e:
            return f"Error: {str(e)}"
    
    def _generate_dockerfile(self, path: str, files: List[str]) -> Optional[str]:
        """Generate a Dockerfile based on project type."""
        if "requirements.txt" in files:
            return '''FROM python:3.11-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
CMD ["python", "main.py"]
'''
        elif "package.json" in files:
            return '''FROM node:20-slim
WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production
COPY . .
CMD ["node", "index.js"]
'''
        elif "go.mod" in files:
            return '''FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o main .

FROM alpine:latest
COPY --from=builder /app/main /main
CMD ["/main"]
'''
        return None


@dataclass
class GenerateDockerfileTool(Tool):
    """Generate a Dockerfile for a project."""
    
    name: str = "generate_dockerfile"
    description: str = "Generate a Dockerfile for a project"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="path",
            type="string",
            description="Project path",
            required=True
        ),
        ToolParameter(
            name="language",
            type="string",
            description="Language: python, node, go, rust",
            required=False
        ),
        ToolParameter(
            name="multi_stage",
            type="boolean",
            description="Use multi-stage build",
            required=False,
            default=True
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.SAFE
    
    def execute(
        self,
        path: str,
        language: str = None,
        multi_stage: bool = True
    ) -> str:
        """Generate Dockerfile."""
        # Auto-detect language if not specified
        if not language:
            files = os.listdir(path)
            if "requirements.txt" in files or "pyproject.toml" in files:
                language = "python"
            elif "package.json" in files:
                language = "node"
            elif "go.mod" in files:
                language = "go"
            elif "Cargo.toml" in files:
                language = "rust"
            else:
                return "Could not detect project language"
        
        dockerfiles = {
            "python": '''FROM python:3.11-slim AS builder
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

FROM python:3.11-slim
WORKDIR /app
COPY --from=builder /usr/local/lib/python3.11/site-packages /usr/local/lib/python3.11/site-packages
COPY . .
EXPOSE 8000
CMD ["python", "-m", "uvicorn", "app.main:app", "--host", "0.0.0.0"]
''',
            "node": '''FROM node:20-slim AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci

FROM node:20-slim
WORKDIR /app
COPY --from=builder /app/node_modules ./node_modules
COPY . .
EXPOSE 3000
CMD ["node", "src/index.js"]
''',
            "go": '''FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o main .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/main .
EXPOSE 8080
CMD ["./main"]
''',
            "rust": '''FROM rust:1.75-slim AS builder
WORKDIR /app
COPY Cargo.toml Cargo.lock ./
RUN mkdir src && echo 'fn main() {}' > src/main.rs
RUN cargo build --release
RUN rm -rf src
COPY . .
RUN cargo build --release

FROM debian:bookworm-slim
COPY --from=builder /app/target/release/app /usr/local/bin/
CMD ["app"]
''',
        }
        
        if language not in dockerfiles:
            return f"Unsupported language: {language}"
        
        dockerfile_content = dockerfiles[language]
        
        # Write Dockerfile
        dockerfile_path = os.path.join(path, "Dockerfile")
        with open(dockerfile_path, "w") as f:
            f.write(dockerfile_content)
        
        # Also create .dockerignore
        dockerignore = '''__pycache__/
*.pyc
node_modules/
target/
.git/
.env
*.log
'''
        with open(os.path.join(path, ".dockerignore"), "w") as f:
            f.write(dockerignore)
        
        return f"Generated Dockerfile for {language} project at {dockerfile_path}"
