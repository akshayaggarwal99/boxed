<p align="center">
  <img src="logo.svg" alt="Boxed Logo" width="320">
</p>

# Boxed

**The Sovereign Code Execution Engine for AI Agents.**

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev)
[![Rust](https://img.shields.io/badge/Rust-1.75+-DEA584?logo=rust)](https://www.rust-lang.org)
[![TypeScript](https://img.shields.io/badge/TypeScript-SDK-3178C6?logo=typescript)](https://www.typescriptlang.org/)
[![Python](https://img.shields.io/badge/Python-SDK-3776AB?logo=python)](https://www.python.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

---

## The Story 📖

Building an AI Agent that writes code? You have a problem.

*   Run it locally? 🚨 **Security Risk.** One `rm -rf /` and your laptop is gone.
*   Run it in cloud? 💸 **Expensive.** AWS instances for every user?
*   Use SaaS sandbox? 🐌 **Vendor Lock-in.** High latency and data privacy concerns.

**Meet Boxed.** The open-source, sovereign engine that gives your Agents a safe place to play. It provides a unified API to spawn ephemeral sandboxes, execute arbitrary code, and retrieve results instantly.

---

## ✨ Features

- **🔒 Secure by Default** — Defense-in-depth isolation (Docker now, Firecracker planned).
- **🛡️ API Authentication** — Hardened endpoints with API Key support.
- **⚡ Sub-second Startup** — Ephemeral environments ready in milliseconds.
- **📁 First-class Artifacts** — Auto-magic handling of generated files (images, PDFs, datasets).
- **🔌 Polyglot SDKs** — First-class support for TypeScript and Python.
- **🌐 Network Control** — Strict egress filtering to keep your network safe.

---

## 🚀 Getting Started

### 📋 Prerequisites

To run Boxed locally, you'll need:
- **Go 1.22+** (for the Control Plane)
- **Rust 1.75+** (for the Agent)
- **Docker Desktop** (running and accessible)
- **Standard Images**: Ensure you have a base image like `python:3.10-slim` pulled:
  ```bash
  docker pull python:3.10-slim
  ```
> [!NOTE]
> **First Run**: The first sandbox creation may take a few seconds while Docker pulls the required images. Subsequent runs are near-instant.

---

### 🏗️ Local Development

We provide a `Makefile` to simplify the build process.

```bash
# 1. Clone the repository
git clone https://github.com/akshayaggarwal99/boxed.git
cd boxed

# 2. Build everything (Agent + CLI)
make build

# 3. Start the Control Plane with Auth
export BOXED_API_KEY="super-secret-key"
./bin/boxed serve --api-key $BOXED_API_KEY

# Cleanup build artifacts
make clean
```

### 🔐 Security & Auth

Boxed supports API Key authentication. You can set the key via the `--api-key` flag or `BOXED_API_KEY` environment variable.

All CLI commands and SDKs must provide this key:
```bash
./bin/boxed list --api-key $BOXED_API_KEY
```

---

### 💻 CLI Usage

```bash
# Run interactive REPL (Sticky Session)
./bin/boxed repl <sandbox-id> --lang python
```

---

### 🔌 SDKs

#### TypeScript
```bash
# Local install
npm install ./sdk/typescript
```

#### Python
```bash
# Local install
pip install -e ./sdk/python
```

---

### 3. Execute Code (Python Example)
```python
from boxed_sdk import Boxed

client = Boxed(base_url="http://localhost:8080", api_key="super-secret-key")

# Create a secure session
session = client.create_session(template="python:3.10-slim")

# Run unsafe code
result = session.run("print('hello from boxed')")
print(result.stdout)

# Cleanup
session.close()
```

---

## 🛠️ Architecture

Boxed uses a **Control Plane vs Data Plane** architecture.

![Architecture Diagram](architecture.svg)

*   **Control Plane (Go)**: High-performance REST API with Auth middleware.
*   **Agent (Rust)**: Lightweight (~5MB) binary injected into every sandbox to manage lifecycle and streaming.

---

## 🗺️ Roadmap

- [x] **Phase 1: Enterprise Edition** (Docker Backend, SDK)
- [x] **Phase 1.5: Sticky Sessions** (REPL Mode, WebSocket Proxy)
- [x] **Phase 1.6: Security Hardening** (Auth, CSRF Protection)
- [ ] **Phase 2: SaaS Edition** (Firecracker MicroVMs)
- [ ] **Phase 4**: Public Tunneling (`*.boxed.run`)

---

## 🤝 Contributing

Contributions are welcome! Please read our [Contributing Guide](CONTRIBUTING.md).

## 📄 License

MIT License — do whatever you want with it.
