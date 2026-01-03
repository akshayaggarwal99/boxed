//! JSON-RPC 2.0 protocol handler.
//!
//! This module implements the communication protocol between the Control Plane
//! and the Agent, using JSON-RPC 2.0 over raw streams.

use anyhow::{Context, Result};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use tokio::io::{AsyncBufReadExt, AsyncWriteExt, BufReader, BufWriter};

/// JSON-RPC 2.0 request structure.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Request {
    pub jsonrpc: String,
    pub method: String,
    #[serde(default)]
    pub params: serde_json::Value,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub id: Option<serde_json::Value>,
}

impl Request {

    /// Create a notification (no response expected).
    pub fn notification(method: &str, params: serde_json::Value) -> Self {
        Self {
            jsonrpc: "2.0".to_string(),
            method: method.to_string(),
            params,
            id: None,
        }
    }
}

/// JSON-RPC 2.0 response structure.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Response {
    pub jsonrpc: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub result: Option<serde_json::Value>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub error: Option<RpcError>,
    pub id: serde_json::Value,
}

impl Response {
    /// Create a success response.
    pub fn success(id: serde_json::Value, result: serde_json::Value) -> Self {
        Self {
            jsonrpc: "2.0".to_string(),
            result: Some(result),
            error: None,
            id,
        }
    }

    /// Create an error response.
    pub fn error(id: serde_json::Value, code: i32, message: &str) -> Self {
        Self {
            jsonrpc: "2.0".to_string(),
            result: None,
            error: Some(RpcError {
                code,
                message: message.to_string(),
                data: None,
            }),
            id,
        }
    }
}

/// JSON-RPC 2.0 error object.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RpcError {
    pub code: i32,
    pub message: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub data: Option<serde_json::Value>,
}

// Standard JSON-RPC 2.0 error codes
pub const INVALID_PARAMS: i32 = -32602;
pub const METHOD_NOT_FOUND: i32 = -32601;

/// Streaming event from Agent to Control Plane.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "method", content = "params")]
pub enum StreamEvent {
    /// Standard output chunk
    #[serde(rename = "stdout")]
    Stdout { chunk: String },
    
    /// Standard error chunk
    #[serde(rename = "stderr")]
    Stderr { chunk: String },
    
    /// Process exited
    #[serde(rename = "exit")]
    Exit { code: i32 },
    
    /// Artifact detected
    #[serde(rename = "artifact")]
    Artifact {
        path: String,
        mime: String,
        data_base64: String,
    },
    
    /// Error occurred
    #[serde(rename = "error")]
    Error { message: String },
}

/// Parameters for the "exec" method.
#[derive(Debug, Clone, Deserialize)]
pub struct ExecParams {
    pub cmd: String,
    #[serde(default)]
    pub args: Vec<String>,
    #[serde(default)]
    pub env: HashMap<String, String>,
}

/// Parameters for the "repl.start" method.
#[derive(Debug, Clone, Deserialize)]
pub struct ReplStartParams {
    pub cmd: String,
    #[serde(default)]
    pub args: Vec<String>,
    #[serde(default)]
    pub env: HashMap<String, String>,
}

/// Parameters for the "repl.input" method.
#[derive(Debug, Clone, Deserialize)]
pub struct ReplInputParams {
    pub data: String,
}

/// RPC handler that processes incoming requests.
pub struct RpcHandler<R, W> {
    reader: BufReader<R>,
    writer: BufWriter<W>,
}

impl<R, W> RpcHandler<R, W>
where
    R: tokio::io::AsyncRead + Unpin,
    W: tokio::io::AsyncWrite + Unpin,
{
    /// Create a new RPC handler with the given reader and writer.
    pub fn new(reader: R, writer: W) -> Self {
        Self {
            reader: BufReader::new(reader),
            writer: BufWriter::new(writer),
        }
    }

    /// Read the next request from the stream.
    pub async fn read_request(&mut self) -> Result<Option<Request>> {
        let mut line = String::new();
        let bytes_read = self
            .reader
            .read_line(&mut line)
            .await
            .context("Failed to read from stream")?;

        if bytes_read == 0 {
            return Ok(None); // EOF
        }

        let request: Request =
            serde_json::from_str(&line).context("Failed to parse JSON-RPC request")?;

        Ok(Some(request))
    }

    /// Send a response to the stream.
    pub async fn send_response(&mut self, response: Response) -> Result<()> {
        let json = serde_json::to_string(&response)?;
        self.writer.write_all(json.as_bytes()).await?;
        self.writer.write_all(b"\n").await?;
        self.writer.flush().await?;
        Ok(())
    }

    /// Send a streaming event (notification) to the stream.
    pub async fn send_event(&mut self, event: StreamEvent) -> Result<()> {
        let notification = match &event {
            StreamEvent::Stdout { chunk } => {
                Request::notification("stdout", serde_json::json!({ "chunk": chunk }))
            }
            StreamEvent::Stderr { chunk } => {
                Request::notification("stderr", serde_json::json!({ "chunk": chunk }))
            }
            StreamEvent::Exit { code } => {
                Request::notification("exit", serde_json::json!({ "code": code }))
            }
            StreamEvent::Artifact {
                path,
                mime,
                data_base64,
            } => Request::notification(
                "artifact",
                serde_json::json!({
                    "path": path,
                    "mime": mime,
                    "data_base64": data_base64
                }),
            ),
            StreamEvent::Error { message } => {
                Request::notification("error", serde_json::json!({ "message": message }))
            }
        };

        let json = serde_json::to_string(&notification)?;
        self.writer.write_all(json.as_bytes()).await?;
        self.writer.write_all(b"\n").await?;
        self.writer.flush().await?;
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_request_serialization() {
        let request = Request {
            jsonrpc: "2.0".to_string(),
            method: "exec".to_string(),
            params: serde_json::json!({
                "cmd": "python3",
                "args": ["-c", "print('hello')"]
            }),
            id: Some(serde_json::json!(1)),
        };

        let json = serde_json::to_string(&request).unwrap();
        assert!(json.contains("\"jsonrpc\":\"2.0\""));
        assert!(json.contains("\"method\":\"exec\""));
    }

    #[test]
    fn test_response_success() {
        let response = Response::success(
            serde_json::json!(1),
            serde_json::json!({"status": "ok"}),
        );

        assert!(response.result.is_some());
        assert!(response.error.is_none());
    }

    #[test]
    fn test_response_error() {
        let response = Response::error(
            serde_json::json!(1),
            METHOD_NOT_FOUND,
            "Method not found",
        );

        assert!(response.result.is_none());
        assert!(response.error.is_some());
        assert_eq!(response.error.unwrap().code, METHOD_NOT_FOUND);
    }
}
