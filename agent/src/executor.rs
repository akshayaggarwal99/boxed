//! Process execution and supervision.
//!
//! This module handles spawning user code as child processes, capturing their
//! output, and managing their lifecycle.

use anyhow::{Context, Result};
use std::collections::HashMap;
use std::process::Stdio;
use tokio::io::{AsyncBufReadExt, BufReader};
use tokio::process::{Child, Command};
use tokio::sync::mpsc;
use tracing::{debug, error, info, warn};

/// Output event from a running process.
#[derive(Debug, Clone)]
pub enum ProcessOutput {
    /// A line from stdout
    Stdout(String),
    /// A line from stderr  
    Stderr(String),
    /// Process exited with the given code
    Exit(i32),
    /// Error occurred during execution
    Error(String),
}

/// Configuration for process execution.
#[derive(Debug, Clone)]
pub struct ExecConfig {
    /// The command to run (e.g., "python3")
    pub cmd: String,
    /// Arguments to pass to the command
    pub args: Vec<String>,
    /// Environment variables to set
    pub env: HashMap<String, String>,
    /// Working directory
    pub cwd: String,
}

impl Default for ExecConfig {
    fn default() -> Self {
        Self {
            cmd: String::new(),
            args: Vec::new(),
            env: HashMap::new(),
            cwd: "/workspace".to_string(),
        }
    }
}

/// Process executor that manages child processes.
pub struct Executor {
    /// Currently running process, if any
    current: Option<Child>,
    /// Handle to child's stdin
    stdin: Option<tokio::process::ChildStdin>,
}

impl Executor {
    /// Create a new Executor.
    pub fn new() -> Self {
        Self { 
            current: None,
            stdin: None,
        }
    }

    /// Execute a command and stream its output.
    ///
    /// Returns a channel that receives output events until the process completes.
    pub async fn exec(&mut self, config: ExecConfig, pipe_stdin: bool) -> Result<mpsc::Receiver<ProcessOutput>> {
        info!(cmd = %config.cmd, args = ?config.args, "Spawning process");

        let (tx, rx) = mpsc::channel(100);

        // Build the command
        let mut cmd = Command::new(&config.cmd);
        cmd.args(&config.args)
            .current_dir(&config.cwd)
            .stdin(if pipe_stdin { Stdio::piped() } else { Stdio::null() })
            .stdout(Stdio::piped())
            .stderr(Stdio::piped())
            .kill_on_drop(true);

        // Set environment variables
        for (key, value) in &config.env {
            cmd.env(key, value);
        }

        // Spawn the process
        let mut child = cmd.spawn().context("Failed to spawn process")?;

        let stdout = child.stdout.take().expect("stdout piped");
        let stderr = child.stderr.take().expect("stderr piped");
        
        // If stdin is piped, take it and store it
        if pipe_stdin {
             let stdin = child.stdin.take().expect("stdin piped");
             self.stdin = Some(stdin);
        }

        self.current = Some(child);

        // Spawn tasks to read stdout and stderr
        let tx_stdout = tx.clone();
        tokio::spawn(async move {
            let reader = BufReader::new(stdout);
            let mut lines = reader.lines();
            while let Ok(Some(line)) = lines.next_line().await {
                if tx_stdout.send(ProcessOutput::Stdout(line)).await.is_err() {
                    break;
                }
            }
        });

        let tx_stderr = tx.clone();
        tokio::spawn(async move {
            let reader = BufReader::new(stderr);
            let mut lines = reader.lines();
            while let Ok(Some(line)) = lines.next_line().await {
                if tx_stderr.send(ProcessOutput::Stderr(line)).await.is_err() {
                    break;
                }
            }
        });

        Ok(rx)
    }

    /// Write to the stdin of the current process.
    pub async fn write_stdin(&mut self, data: &str) -> Result<()> {
        use tokio::io::AsyncWriteExt;
        if let Some(stdin) = self.stdin.as_mut() {
            stdin.write_all(data.as_bytes()).await.context("Failed to write to stdin")?;
            stdin.flush().await.context("Failed to flush stdin")?;
            Ok(())
        } else {
            anyhow::bail!("Process has no persistent stdin")
        }
    }

    /// Wait for the current process to complete.
    pub async fn wait_for_completion(&mut self) -> Option<ProcessOutput> {
        self.stdin = None; // Close stdin to allow process to exit if waiting for it
        if let Some(mut child) = self.current.take() {
            match child.wait().await {
                Ok(status) => {
                    let code = status.code().unwrap_or(-1);
                    debug!(exit_code = code, "Process completed");
                    Some(ProcessOutput::Exit(code))
                }
                Err(e) => {
                    error!(error = %e, "Failed to wait for process");
                    Some(ProcessOutput::Error(e.to_string()))
                }
            }
        } else {
            None
        }
    }

}

impl Default for Executor {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_exec_echo() {
        let mut executor = Executor::new();
        let config = ExecConfig {
            cmd: "echo".to_string(),
            args: vec!["hello".to_string()],
            ..Default::default()
        };

        let mut rx = executor.exec(config).await.unwrap();
        
        // Should receive stdout
        if let Some(ProcessOutput::Stdout(line)) = rx.recv().await {
            assert_eq!(line, "hello");
        }
    }
}
