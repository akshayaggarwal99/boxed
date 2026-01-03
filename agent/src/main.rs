//! Boxed Agent - The lightweight binary that runs inside sandboxes.
//!
//! The agent is responsible for:
//! - Executing commands from the Control Plane via JSON-RPC 2.0
//! - Streaming stdout/stderr in real-time
//! - Watching for artifacts (files in /output) and streaming them back
//!
//! # Architecture
//!
//! The agent is designed to never panic. If user code crashes, the agent
//! handles the error gracefully and remains alive for subsequent commands.

use anyhow::Result;
use tracing::{error, info};
use tracing_subscriber::EnvFilter;

mod executor;
mod fs_watcher;
mod rpc;

#[tokio::main]
async fn main() -> Result<()> {
    // Initialize structured JSON logging
    tracing_subscriber::fmt()
        .with_env_filter(EnvFilter::from_default_env().add_directive("boxed_agent=info".parse()?))
        .json()
        .init();

    info!(version = env!("CARGO_PKG_VERSION"), "🗳️ Boxed Agent starting");

    // The agent communicates over stdin/stdout for maximum compatibility
    // Docker: Attaches via exec
    // Firecracker: Connects via vsock, forwarded to stdin/stdout
    if let Err(e) = run_agent().await {
        error!(error = %e, "Agent encountered fatal error");
        std::process::exit(1);
    }

    info!("Boxed Agent stopped");
    Ok(())
}

async fn run_agent() -> Result<()> {
    // Initialize RPC listener
    let stdin = tokio::io::stdin();
    let stdout = tokio::io::stdout();
    let mut rpc = rpc::RpcHandler::new(stdin, stdout);

    // Initialize executor
    let mut executor = executor::Executor::new();

    // Initialize FS watcher
    let (_watcher, mut artifact_rx) = fs_watcher::FsWatcher::new("/output").await?;
    
    // Channel for events (Stdout, Stderr, Exit, Artifact, Error)
    let (event_tx, mut event_rx) = tokio::sync::mpsc::channel::<rpc::StreamEvent>(100);

    info!("Ready to accept commands");

    loop {
        tokio::select! {
            // Read next request (handles EOF)
            request_res = rpc.read_request() => {
                let request = match request_res {
                    Ok(Some(req)) => req,
                    Ok(None) => {
                        info!("EOF received, shutting down");
                        break;
                    }
                    Err(e) => {
                        error!("Failed to read request: {}", e);
                        continue;
                    }
                };

                match request.method.as_str() {
                    "exec" => {
                        let params: rpc::ExecParams = serde_json::from_value(request.params.clone())?;
                        let config = executor::ExecConfig {
                            cmd: params.cmd,
                            args: params.args,
                            env: params.env,
                            cwd: "/workspace".to_string(),
                        };
                        
                        if let Some(id) = request.id {
                            rpc.send_response(rpc::Response::success(id, serde_json::Value::Null)).await?;
                        }

                        // Start execution and spawn monitoring task
                        match executor.exec(config, false).await {
                            Ok(mut output_rx) => {
                                let tx = event_tx.clone();
                                tokio::spawn(async move {
                                    while let Some(output) = output_rx.recv().await {
                                        match output {
                                            executor::ProcessOutput::Stdout(line) => {
                                                let _ = tx.send(rpc::StreamEvent::Stdout { chunk: line + "\n" }).await;
                                            }
                                            executor::ProcessOutput::Stderr(line) => {
                                                let _ = tx.send(rpc::StreamEvent::Stderr { chunk: line + "\n" }).await;
                                            }
                                            executor::ProcessOutput::Error(e) => {
                                                let _ = tx.send(rpc::StreamEvent::Error { message: e }).await;
                                            }
                                            _ => {}
                                        }
                                    }
                                    // Note: In this simple implementation, we don't handle wait_for_completion 
                                    // inside the monitoring task because it needs &mut self.
                                    // We will improve this in the next iteration.
                                    let _ = tx.send(rpc::StreamEvent::Exit { code: 0 }).await;
                                });
                            }
                            Err(e) => {
                                let _ = event_tx.send(rpc::StreamEvent::Error { message: e.to_string() }).await;
                            }
                        }
                    }
                    "repl.start" => {
                        let params: rpc::ReplStartParams = serde_json::from_value(request.params.clone())?;
                        let config = executor::ExecConfig {
                            cmd: params.cmd,
                            args: params.args,
                            env: params.env,
                            cwd: "/workspace".to_string(),
                        };

                        if let Some(id) = request.id {
                            rpc.send_response(rpc::Response::success(id, serde_json::Value::Null)).await?;
                        }

                        match executor.exec(config, true).await {
                            Ok(mut output_rx) => {
                                let tx = event_tx.clone();
                                tokio::spawn(async move {
                                    while let Some(output) = output_rx.recv().await {
                                        match output {
                                            executor::ProcessOutput::Stdout(line) => {
                                                let _ = tx.send(rpc::StreamEvent::Stdout { chunk: line + "\n" }).await;
                                            }
                                            executor::ProcessOutput::Stderr(line) => {
                                                let _ = tx.send(rpc::StreamEvent::Stderr { chunk: line + "\n" }).await;
                                            }
                                            executor::ProcessOutput::Error(e) => {
                                                let _ = tx.send(rpc::StreamEvent::Error { message: e }).await;
                                            }
                                            _ => {}
                                        }
                                    }
                                    let _ = tx.send(rpc::StreamEvent::Exit { code: 0 }).await;
                                });
                            }
                            Err(e) => {
                                let _ = event_tx.send(rpc::StreamEvent::Error { message: e.to_string() }).await;
                            }
                        }
                    }
                    "repl.input" => {
                        let params: rpc::ReplInputParams = serde_json::from_value(request.params.clone())?;
                        match executor.write_stdin(&params.data).await {
                            Ok(_) => {
                                if let Some(id) = request.id {
                                    rpc.send_response(rpc::Response::success(id, serde_json::Value::Null)).await?;
                                }
                            }
                            Err(e) => {
                                if let Some(id) = request.id {
                                    rpc.send_response(rpc::Response::error(id, rpc::INVALID_PARAMS, &e.to_string())).await?;
                                }
                            }
                        }
                    }
                    _ => {
                        if let Some(id) = request.id {
                            rpc.send_response(rpc::Response::error(id, rpc::METHOD_NOT_FOUND, "Method not found")).await?;
                        }
                    }
                }
            }
            // Process events
            event = event_rx.recv() => {
                if let Some(e) = event {
                    rpc.send_event(e).await?;
                }
            }
            // Process artifacts
            artifact = artifact_rx.recv() => {
                if let Some(a) = artifact {
                    rpc.send_event(rpc::StreamEvent::Artifact {
                        path: a.path,
                        mime: a.mime,
                        data_base64: a.data_base64
                    }).await?;
                }
            }
        }
    }
    
    Ok(())
}
