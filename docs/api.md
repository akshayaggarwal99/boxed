# 📖 Boxed API Reference (v0.2.0)

This document provides a detailed reference for the Boxed REST API. All requests should be made to the base URL (default: `http://localhost:8080/v1`).

## 🔐 Security & Authentication

Boxed uses a **Bring Your Own Key (BYOK)** model. Since you run your own Boxed instance, you define the secret key at startup.

### Setting the Key
Set the `BOXED_API_KEY` environment variable before starting the server:

```bash
# 1. Generate a secure secret
export BOXED_API_KEY=$(openssl rand -hex 32)
echo "Your API Key is: $BOXED_API_KEY"

# 2. Start Boxed
./bin/boxed serve
```

### Authentication
Once configured, all requests must include the key via the `X-Boxed-API-Key` header or the `api_key` query parameter.

**Required Headers:**
- `X-Boxed-API-Key`: The secret you defined at startup.
- `Content-Type`: `application/json` (for POST requests).

---

## 🏗️ Sandbox Management

### Create Sandbox
`POST /sandbox`

Creates a new ephemeral sandbox.

**Request Body (JSON):**
| Field | Type | Description |
| :--- | :--- | :--- |
| `template` | string | Docker image (e.g., `python:3.10-slim`). Required. |
| `timeout` | int | Hard TTL in seconds (max 1800). Default: 300. |
| `context` | array | List of files to pre-inject: `[{ "path": "...", "content_base64": "..." }]`. |

**Example (curl):**
```bash
curl -X POST http://localhost:8080/v1/sandbox \
  -H "X-Boxed-API-Key: your-secret-key" \
  -H "Content-Type: application/json" \
  -d '{
    "template": "python:3.10-slim",
    "timeout": 600,
    "context": [
      { "path": "config.yaml", "content_base64": "YXBpX2tleTogMTIzNDU=" }
    ]
  }'
```

---

### List Sandboxes
`GET /sandbox`

Returns all active sandboxes.

**Example (curl):**
```bash
curl http://localhost:8080/v1/sandbox
```

---

### Delete Sandbox
`DELETE /sandbox/:id`

Gracefully stops and removes a sandbox.

---

## ⚡ Execution

### Execute Code
`POST /sandbox/:id/exec`

Runs code inside the sandbox and returns standard output/error.

**Request Body (JSON):**
| Field | Type | Description |
| :--- | :--- | :--- |
| `code` | string | The code to execute. |
| `language` | string | Only `python` is currently supported in standard templates. |

**Example (SDK):**
```typescript
const result = await session.run('print("Hello World")');
console.log(result.stdout);
```

---

## 📂 Filesystem API

### List Files
`GET /sandbox/:id/files?path=/root`

Lists files and directories at a specific path.

---

### Upload File
`POST /sandbox/:id/files`

Uploads a file via `multipart/form-data`.

**Form Fields:**
- `file`: The file data.
- `path`: The target directory in the sandbox (e.g., `/workspace`).

---

### Download File
`GET /sandbox/:id/files/content?path=/file.txt`

Streams the raw content of a specific file.

---

## �️ Interactive Sessions (Sticky Sessions)

Boxed support stateful, interactive sessions via WebSockets. This allows for persistent shells or long-running execution where you can send input in real-time.

### Interact
`GET /sandbox/:id/interact?lang=bash` (WebSocket)

Upgrades the connection to a WebSocket for real-time interaction.

**Protocol:** JSON-RPC 2.0 Notifications.

| Method | Params | Description |
| :--- | :--- | :--- |
| `stdout` | `{ chunk: string }` | Received when the shell writes to stdout. |
| `stderr` | `{ chunk: string }` | Received when the shell writes to stderr. |
| `repl.input` | `{ data: string }` | Send this to the sandbox to provide stdin. |
| `exit` | `{ code: int }` | Received when the interactive process terminates. |

**Example (TypeScript SDK):**
```typescript
const interaction = await session.interact('python');
interaction.onOutput((evt) => {
  if (evt.type === 'stdout') process.stdout.write(evt.chunk);
});
await interaction.write("print('hello from REPL')\n");
```

**Example (CLI):**
```bash
boxed repl <sandbox-id> --lang python
```

---

## 🛠️ ROADMAP: Network Policy (Airlock)
Coming soon: Granular egress/ingress control for sandboxed processes.
