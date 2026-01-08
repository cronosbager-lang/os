"""Audit logging for the AI agent."""

from typing import Dict, Any, Optional, List
from dataclasses import dataclass, field
from enum import Enum
import json
import os
import time
from datetime import datetime
import hashlib


class AuditEventType(Enum):
    """Types of audit events."""
    # Session events
    SESSION_START = "session.start"
    SESSION_END = "session.end"
    
    # Message events
    MESSAGE_USER = "message.user"
    MESSAGE_ASSISTANT = "message.assistant"
    
    # Tool events
    TOOL_CALL = "tool.call"
    TOOL_RESULT = "tool.result"
    TOOL_ERROR = "tool.error"
    
    # Permission events
    PERMISSION_CHECK = "permission.check"
    PERMISSION_DENIED = "permission.denied"
    PERMISSION_GRANTED = "permission.granted"
    
    # Security events
    SECURITY_BLOCKED = "security.blocked"
    SECURITY_WARNING = "security.warning"
    
    # System events
    SYSTEM_ERROR = "system.error"
    SYSTEM_CONFIG = "system.config"


@dataclass
class AuditEvent:
    """Represents an audit event."""
    id: str
    timestamp: float
    event_type: AuditEventType
    session_id: Optional[str]
    data: Dict[str, Any]
    
    def to_dict(self) -> Dict[str, Any]:
        return {
            "id": self.id,
            "timestamp": self.timestamp,
            "event_type": self.event_type.value,
            "session_id": self.session_id,
            "data": self.data,
        }
    
    def to_json(self) -> str:
        return json.dumps(self.to_dict())
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "AuditEvent":
        return cls(
            id=data["id"],
            timestamp=data["timestamp"],
            event_type=AuditEventType(data["event_type"]),
            session_id=data.get("session_id"),
            data=data.get("data", {}),
        )


class AuditLogger:
    """Logs audit events for the AI agent."""
    
    def __init__(
        self,
        log_path: str = None,
        max_file_size: int = 10 * 1024 * 1024,  # 10MB
        max_files: int = 10,
        console_output: bool = False,
    ):
        self.log_path = log_path or os.path.expanduser("~/.mixos/agent/audit")
        self.max_file_size = max_file_size
        self.max_files = max_files
        self.console_output = console_output
        
        self.current_session_id: Optional[str] = None
        self._event_count = 0
        
        # Create log directory
        os.makedirs(self.log_path, exist_ok=True)
    
    def _generate_event_id(self) -> str:
        """Generate a unique event ID."""
        self._event_count += 1
        data = f"{time.time()}-{self._event_count}".encode()
        return hashlib.sha256(data).hexdigest()[:16]
    
    def _get_current_log_file(self) -> str:
        """Get the current log file path."""
        date_str = datetime.now().strftime("%Y-%m-%d")
        return os.path.join(self.log_path, f"audit-{date_str}.jsonl")
    
    def _rotate_logs(self):
        """Rotate log files if needed."""
        current_file = self._get_current_log_file()
        
        if os.path.exists(current_file):
            size = os.path.getsize(current_file)
            if size >= self.max_file_size:
                # Rename current file with timestamp
                timestamp = datetime.now().strftime("%H%M%S")
                new_name = current_file.replace(".jsonl", f"-{timestamp}.jsonl")
                os.rename(current_file, new_name)
        
        # Remove old files
        files = sorted([
            f for f in os.listdir(self.log_path)
            if f.startswith("audit-") and f.endswith(".jsonl")
        ])
        
        while len(files) > self.max_files:
            oldest = files.pop(0)
            os.remove(os.path.join(self.log_path, oldest))
    
    def log(
        self,
        event_type: AuditEventType,
        data: Dict[str, Any] = None,
        session_id: str = None,
    ) -> AuditEvent:
        """Log an audit event."""
        event = AuditEvent(
            id=self._generate_event_id(),
            timestamp=time.time(),
            event_type=event_type,
            session_id=session_id or self.current_session_id,
            data=data or {},
        )
        
        # Write to file
        self._rotate_logs()
        log_file = self._get_current_log_file()
        
        with open(log_file, "a") as f:
            f.write(event.to_json() + "\n")
        
        # Console output
        if self.console_output:
            print(f"[AUDIT] {event.event_type.value}: {event.data}")
        
        return event
    
    def log_session_start(self, session_id: str, metadata: Dict[str, Any] = None):
        """Log session start."""
        self.current_session_id = session_id
        return self.log(
            AuditEventType.SESSION_START,
            {"metadata": metadata or {}},
            session_id,
        )
    
    def log_session_end(self, session_id: str = None):
        """Log session end."""
        event = self.log(
            AuditEventType.SESSION_END,
            {},
            session_id or self.current_session_id,
        )
        self.current_session_id = None
        return event
    
    def log_user_message(self, message: str, session_id: str = None):
        """Log user message."""
        return self.log(
            AuditEventType.MESSAGE_USER,
            {"message": message[:1000]},  # Truncate long messages
            session_id,
        )
    
    def log_assistant_message(self, message: str, session_id: str = None):
        """Log assistant message."""
        return self.log(
            AuditEventType.MESSAGE_ASSISTANT,
            {"message": message[:1000]},
            session_id,
        )
    
    def log_tool_call(
        self,
        tool_name: str,
        parameters: Dict[str, Any],
        session_id: str = None,
    ):
        """Log tool call."""
        # Sanitize sensitive parameters
        safe_params = self._sanitize_params(parameters)
        
        return self.log(
            AuditEventType.TOOL_CALL,
            {"tool": tool_name, "parameters": safe_params},
            session_id,
        )
    
    def log_tool_result(
        self,
        tool_name: str,
        result: str,
        success: bool,
        duration: float = None,
        session_id: str = None,
    ):
        """Log tool result."""
        return self.log(
            AuditEventType.TOOL_RESULT,
            {
                "tool": tool_name,
                "success": success,
                "result": result[:500],  # Truncate
                "duration": duration,
            },
            session_id,
        )
    
    def log_tool_error(
        self,
        tool_name: str,
        error: str,
        session_id: str = None,
    ):
        """Log tool error."""
        return self.log(
            AuditEventType.TOOL_ERROR,
            {"tool": tool_name, "error": error},
            session_id,
        )
    
    def log_permission_denied(
        self,
        permission: str,
        tool_name: str = None,
        session_id: str = None,
    ):
        """Log permission denied."""
        return self.log(
            AuditEventType.PERMISSION_DENIED,
            {"permission": permission, "tool": tool_name},
            session_id,
        )
    
    def log_security_blocked(
        self,
        reason: str,
        command: str = None,
        session_id: str = None,
    ):
        """Log security block."""
        return self.log(
            AuditEventType.SECURITY_BLOCKED,
            {"reason": reason, "command": command[:200] if command else None},
            session_id,
        )
    
    def _sanitize_params(self, params: Dict[str, Any]) -> Dict[str, Any]:
        """Remove sensitive information from parameters."""
        sensitive_keys = {
            "password", "secret", "token", "key", "api_key",
            "auth", "credential", "private",
        }
        
        sanitized = {}
        for key, value in params.items():
            key_lower = key.lower()
            if any(s in key_lower for s in sensitive_keys):
                sanitized[key] = "[REDACTED]"
            elif isinstance(value, dict):
                sanitized[key] = self._sanitize_params(value)
            else:
                sanitized[key] = value
        
        return sanitized
    
    def query(
        self,
        event_type: AuditEventType = None,
        session_id: str = None,
        start_time: float = None,
        end_time: float = None,
        limit: int = 100,
    ) -> List[AuditEvent]:
        """Query audit events."""
        events = []
        
        # Read all log files
        for filename in sorted(os.listdir(self.log_path), reverse=True):
            if not filename.startswith("audit-") or not filename.endswith(".jsonl"):
                continue
            
            filepath = os.path.join(self.log_path, filename)
            
            with open(filepath) as f:
                for line in f:
                    if not line.strip():
                        continue
                    
                    try:
                        event = AuditEvent.from_dict(json.loads(line))
                        
                        # Apply filters
                        if event_type and event.event_type != event_type:
                            continue
                        if session_id and event.session_id != session_id:
                            continue
                        if start_time and event.timestamp < start_time:
                            continue
                        if end_time and event.timestamp > end_time:
                            continue
                        
                        events.append(event)
                        
                        if len(events) >= limit:
                            return events
                    except Exception:
                        continue
        
        return events
    
    def get_session_events(self, session_id: str) -> List[AuditEvent]:
        """Get all events for a session."""
        return self.query(session_id=session_id, limit=1000)
    
    def get_recent_events(self, count: int = 50) -> List[AuditEvent]:
        """Get recent events."""
        return self.query(limit=count)


class AuditReport:
    """Generate audit reports."""
    
    def __init__(self, logger: AuditLogger):
        self.logger = logger
    
    def generate_session_report(self, session_id: str) -> Dict[str, Any]:
        """Generate a report for a session."""
        events = self.logger.get_session_events(session_id)
        
        if not events:
            return {"error": "No events found for session"}
        
        report = {
            "session_id": session_id,
            "start_time": None,
            "end_time": None,
            "duration": None,
            "message_count": 0,
            "tool_calls": 0,
            "tool_errors": 0,
            "security_blocks": 0,
            "tools_used": {},
        }
        
        for event in events:
            if event.event_type == AuditEventType.SESSION_START:
                report["start_time"] = event.timestamp
            elif event.event_type == AuditEventType.SESSION_END:
                report["end_time"] = event.timestamp
            elif event.event_type == AuditEventType.MESSAGE_USER:
                report["message_count"] += 1
            elif event.event_type == AuditEventType.TOOL_CALL:
                report["tool_calls"] += 1
                tool = event.data.get("tool", "unknown")
                report["tools_used"][tool] = report["tools_used"].get(tool, 0) + 1
            elif event.event_type == AuditEventType.TOOL_ERROR:
                report["tool_errors"] += 1
            elif event.event_type == AuditEventType.SECURITY_BLOCKED:
                report["security_blocks"] += 1
        
        if report["start_time"] and report["end_time"]:
            report["duration"] = report["end_time"] - report["start_time"]
        
        return report
    
    def generate_daily_report(self, date: str = None) -> Dict[str, Any]:
        """Generate a daily report."""
        if date is None:
            date = datetime.now().strftime("%Y-%m-%d")
        
        # Parse date
        dt = datetime.strptime(date, "%Y-%m-%d")
        start_time = dt.timestamp()
        end_time = start_time + 86400
        
        events = self.logger.query(
            start_time=start_time,
            end_time=end_time,
            limit=10000,
        )
        
        report = {
            "date": date,
            "total_events": len(events),
            "sessions": set(),
            "tool_calls": 0,
            "errors": 0,
            "security_blocks": 0,
            "event_types": {},
        }
        
        for event in events:
            if event.session_id:
                report["sessions"].add(event.session_id)
            
            event_type = event.event_type.value
            report["event_types"][event_type] = report["event_types"].get(event_type, 0) + 1
            
            if event.event_type == AuditEventType.TOOL_CALL:
                report["tool_calls"] += 1
            elif event.event_type in (AuditEventType.TOOL_ERROR, AuditEventType.SYSTEM_ERROR):
                report["errors"] += 1
            elif event.event_type == AuditEventType.SECURITY_BLOCKED:
                report["security_blocks"] += 1
        
        report["sessions"] = len(report["sessions"])
        
        return report
