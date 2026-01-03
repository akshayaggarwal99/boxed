# Boxed Python SDK

Sovereign code execution for AI agents, in Python.

## Installation

```bash
pip install -e .
```

## Usage

```python
from boxed_sdk import Boxed

client = Boxed(base_url="http://localhost:8080", api_key="your-secret-key")

# 1. Create session
session = client.create_session(template="python:3.10-slim")

# 2. Run code
result = session.run("print('hello from python sdk')")
print(result.stdout)

# 3. Handle artifacts
for artifact in result.artifacts:
    print(f"Generated: {artifact.name}")
    # print(artifact.url) # Data URL

# 4. Cleanup
session.close()
```
