//! Filesystem watcher for artifact detection.
//!
//! This module monitors the /output directory for new files and streams them
//! back to the Control Plane as base64-encoded artifacts.

use anyhow::{Context, Result};
use base64::Engine;
use notify::{Config, Event, EventKind, RecommendedWatcher, RecursiveMode, Watcher};
use std::path::{Path, PathBuf};
use tokio::fs;
use tokio::sync::mpsc;
use tracing::{debug, error, info, warn};

/// An artifact detected in the watched directory.
#[derive(Debug, Clone)]
pub struct Artifact {
    /// Path relative to the watched directory
    pub path: String,
    /// MIME type of the file
    pub mime: String,
    /// Base64-encoded file contents
    pub data_base64: String,
}

/// Maximum file size to stream inline (files larger than this should use upload)
const MAX_INLINE_SIZE: u64 = 10 * 1024 * 1024; // 10 MB

/// Filesystem watcher for artifact detection.
pub struct FsWatcher {
    /// The directory being watched
    watch_dir: PathBuf,
    /// The underlying file watcher
    _watcher: RecommendedWatcher,
}

impl FsWatcher {
    /// Create a new filesystem watcher for the given directory.
    ///
    /// Returns a receiver channel that will emit detected artifacts.
    pub async fn new(watch_dir: impl AsRef<Path>) -> Result<(Self, mpsc::Receiver<Artifact>)> {
        let watch_dir = watch_dir.as_ref().to_path_buf();

        // Create the output directory if it doesn't exist
        fs::create_dir_all(&watch_dir)
            .await
            .context("Failed to create watch directory")?;

        let (artifact_tx, artifact_rx) = mpsc::channel(100);
        let (event_tx, mut event_rx) = mpsc::channel(100);

        // Create the file watcher
        let tx = event_tx.clone();
        let watcher = RecommendedWatcher::new(
            move |res: Result<Event, notify::Error>| {
                if let Ok(event) = res {
                    let _ = tx.blocking_send(event);
                }
            },
            Config::default(),
        )?;

        // Process file events in a background task
        let artifact_tx_clone = artifact_tx.clone();
        let watch_dir_clone = watch_dir.clone();
        tokio::spawn(async move {
            while let Some(event) = event_rx.recv().await {
                if let Err(e) =
                    process_event(event, &watch_dir_clone, &artifact_tx_clone).await
                {
                    error!(error = %e, "Failed to process file event");
                }
            }
        });

        let mut fs_watcher = Self {
            watch_dir,
            _watcher: watcher,
        };

        // Start watching the directory
        fs_watcher.start_watching()?;

        info!(dir = %fs_watcher.watch_dir.display(), "Filesystem watcher started");

        Ok((fs_watcher, artifact_rx))
    }

    /// Start watching the output directory.
    fn start_watching(&mut self) -> Result<()> {
        self._watcher
            .watch(&self.watch_dir, RecursiveMode::Recursive)
            .context("Failed to watch directory")?;
        Ok(())
    }
}

/// Process a filesystem event and potentially emit an artifact.
async fn process_event(
    event: Event,
    watch_dir: &Path,
    artifact_tx: &mpsc::Sender<Artifact>,
) -> Result<()> {
    // We only care about file creation and modification
    match event.kind {
        EventKind::Create(_) | EventKind::Modify(_) => {}
        _ => return Ok(()),
    }

    for path in event.paths {
        // Skip directories
        if path.is_dir() {
            continue;
        }

        // Skip hidden files
        if path
            .file_name()
            .map(|n| n.to_string_lossy().starts_with('.'))
            .unwrap_or(false)
        {
            continue;
        }

        debug!(path = %path.display(), "File event detected");

        // Read and encode the file
        match read_artifact(&path, watch_dir).await {
            Ok(Some(artifact)) => {
                info!(
                    path = %artifact.path,
                    mime = %artifact.mime,
                    size = artifact.data_base64.len(),
                    "Artifact detected"
                );
                if artifact_tx.send(artifact).await.is_err() {
                    warn!("Artifact receiver dropped");
                }
            }
            Ok(None) => {
                // File too large or unreadable
            }
            Err(e) => {
                warn!(path = %path.display(), error = %e, "Failed to read artifact");
            }
        }
    }

    Ok(())
}

/// Read a file and convert it to an artifact.
async fn read_artifact(path: &Path, watch_dir: &Path) -> Result<Option<Artifact>> {
    // Get file metadata
    let metadata = fs::metadata(path).await?;

    // Skip files that are too large for inline streaming
    if metadata.len() > MAX_INLINE_SIZE {
        warn!(
            path = %path.display(),
            size = metadata.len(),
            "File too large for inline streaming"
        );
        return Ok(None);
    }

    // Read file contents
    let data = fs::read(path).await?;

    // Detect MIME type
    let mime = mime_guess::from_path(path)
        .first_or_octet_stream()
        .to_string();

    // Create relative path
    let relative_path = path
        .strip_prefix(watch_dir)
        .unwrap_or(path)
        .to_string_lossy()
        .to_string();

    // Base64 encode
    let data_base64 = base64::engine::general_purpose::STANDARD.encode(&data);

    Ok(Some(Artifact {
        path: relative_path,
        mime,
        data_base64,
    }))
}

#[cfg(test)]
mod tests {
    use super::*;
    use tempfile::tempdir;

    #[tokio::test]
    async fn test_mime_detection() {
        let path = Path::new("/tmp/test.png");
        let mime = mime_guess::from_path(path)
            .first_or_octet_stream()
            .to_string();
        assert_eq!(mime, "image/png");
    }

    #[tokio::test]
    async fn test_watcher_creation() {
        let dir = tempdir().unwrap();
        let result = FsWatcher::new(dir.path()).await;
        assert!(result.is_ok());
    }
}
