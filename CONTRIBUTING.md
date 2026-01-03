# Contributing to Boxed

First off, thank you for considering contributing to Boxed! We appreciate your help in making this the best sovereign execution engine for AI agents.

## 🛠️ Development Setup

Please refer to the [README.md](README.md#🏗️-local-development) for the full setup guide (Go, Rust, Docker).

### Key Commands

```bash
# Build everything
make build

# Run tests
make test
```

## 📜 Code of Conduct

By participating in this project, you agree to abide by our code of conduct (Standard Open Source behavior).

## 🌿 Branching Strategy

- `main`: Production-ready code.
- `feat/*`: For new features.
- `fix/*`: For bug fixes.

## 🧪 Testing Policy

We take stability seriously. 
- **Go**: All API changes must include integration tests in `tests/integration/`.
- **Rust**: Use unit tests in `agent/src/` where appropriate.
- **SDK**: Verified via `sdk/typescript/test/`.

## 📮 Pull Request Process

1. Fork the repo and create your branch from `main`.
2. Ensure your code passes all tests (`make test`).
3. Update documentation if you've added new features or changed APIs.
4. Submit your PR and wait for a maintainer review.

## 💎 Anti-Bloat Policy

Boxed is designed to be **lightweight and fast**.
- **Agent**: Keep the binary size under 5MB if possible. Avoid heavy dependencies.
- **Control Plane**: Use Go's standard library where possible.

Questions? Reach out via GitHub Issues!
