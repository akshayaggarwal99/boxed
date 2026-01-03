import requests
import os
import json
import base64
import time
from typing import List, Optional, Dict, Any
from dataclasses import dataclass

@dataclass
class Artifact:
    name: str
    content_base64: str
    
    @property
    def url(self) -> str:
        # Simple data URL representation
        ext = self.name.split('.')[-1] if '.' in self.name else 'bin'
        mime = 'application/octet-stream'
        if ext == 'png': mime = 'image/png'
        elif ext == 'jpg' or ext == 'jpeg': mime = 'image/jpeg'
        elif ext == 'pdf': mime = 'application/pdf'
        return f"data:{mime};base64,{self.content_base64}"

@dataclass
class ExecutionResult:
    stdout: str
    stderr: str
    exit_code: int
    artifacts: List[Artifact]

class Session:
    def __init__(self, client: 'Boxed', session_id: str):
        self.client = client
        self.id = session_id
        self.base_url = f"{client.base_url}/v1/sandbox/{session_id}"

    def run(self, code: str, language: str = "python", timeout_ms: int = 30000) -> ExecutionResult:
        resp = requests.post(
            f"{self.base_url}/exec",
            headers=self.client._headers(),
            json={
                "code": code,
                "language": language,
                "timeout_ms": timeout_ms
            }
        )
        resp.raise_for_status()
        data = resp.json()
        
        # Parse artifacts from response if present
        artifacts = []
        for a in (data.get("artifacts") or []):
            artifacts.append(Artifact(
                name=a["path"].split('/')[-1],
                content_base64=a["data_base64"]
            ))
        
        return ExecutionResult(
            stdout=data.get("stdout", ""),
            stderr=data.get("stderr", ""),
            exit_code=data.get("exit_code", 0),
            artifacts=artifacts
        )

    def list_files(self, path: str = "/") -> List[Dict[str, Any]]:
        resp = requests.get(
            f"{self.base_url}/files",
            params={"path": path},
            headers=self.client._headers()
        )
        resp.raise_for_status()
        return resp.json().get("files", [])

    def upload_file(self, content: bytes, path: str) -> Dict[str, Any]:
        # Split path into directory and filename to match TS SDK behavior
        if path.endswith('/'):
            dest_dir = path
            filename = 'file'
        else:
            filename = os.path.basename(path)
            dest_dir = os.path.dirname(path) or '/'

        files = {'file': (filename, content)}
        data = {'path': dest_dir}
        resp = requests.post(
            f"{self.base_url}/files",
            files=files,
            data=data,
            headers=self.client._headers()
        )
        resp.raise_for_status()
        return resp.json()

    def download_file(self, path: str) -> bytes:
        resp = requests.get(
            f"{self.base_url}/files/content",
            params={"path": path},
            headers=self.client._headers()
        )
        resp.raise_for_status()
        return resp.content

    def interact(self, language: str = "bash") -> 'Interaction':
        from websocket import create_connection
        ws_url = self.client.base_url.replace("http", "ws") + f"/v1/sandbox/{self.id}/interact?lang={language}"
        
        # Attach API key to query if present (websockets often use query for auth)
        if self.client.api_key:
            ws_url += f"&api_key={self.client.api_key}"
            
        ws = create_connection(ws_url)
        return Interaction(ws)

    def close(self):
        resp = requests.delete(self.base_url, headers=self.client._headers())
        resp.raise_for_status()

class Interaction:
    def __init__(self, ws):
        self.ws = ws

    def write(self, data: str):
        self.ws.send(data)

    def read_event(self, timeout: float = 5.0) -> Optional[Dict[str, Any]]:
        self.ws.settimeout(timeout)
        try:
            message = self.ws.recv()
            return json.loads(message)
        except Exception:
            return None

    def close(self):
        self.ws.close()

class Boxed:
    def __init__(self, base_url: str = "http://localhost:8080", api_key: Optional[str] = None):
        self.base_url = base_url.rstrip('/')
        self.api_key = api_key

    def _headers(self) -> Dict[str, str]:
        headers = {}
        if self.api_key:
            headers["X-Boxed-API-Key"] = self.api_key
        return headers

    def create_session(self, template: str = "python:3.10-slim", timeout_ms: int = 30000) -> Session:
        resp = requests.post(
            f"{self.base_url}/v1/sandbox",
            headers=self._headers(),
            json={
                "template": template,
                "timeout": timeout_ms // 1000
            }
        )
        resp.raise_for_status()
        session_id = resp.json()["sandbox_id"]
        return Session(self, session_id)

    def list_sessions(self) -> List[str]:
        resp = requests.get(f"{self.base_url}/v1/sandbox", headers=self._headers())
        resp.raise_for_status()
        sandboxes = resp.json().get("sandboxes") or []
        return [s["id"] for s in sandboxes]
