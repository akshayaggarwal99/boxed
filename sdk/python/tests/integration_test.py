import os
import time
from boxed_sdk import Boxed

def test_integration():
    api_key = os.getenv("BOXED_API_KEY", "")
    client = Boxed(base_url="http://localhost:8080", api_key=api_key)

    print("--- 1. Listing Sessions ---")
    sessions = client.list_sessions()
    print(f"Active sessions: {len(sessions)}")

    print("\n--- 2. Creating Session ---")
    session = client.create_session(template="python:3.10-slim")
    print(f"Created session: {session.id}")

    print("\n--- 3. Running Code ---")
    result = session.run("print('hello from python sdk')", language="python")
    print(f"Stdout: {result.stdout.strip()}")
    print(f"Stderr: {result.stderr.strip()}")
    assert "hello from python sdk" in result.stdout

    print("\n--- 4. Filesystem: Upload ---")
    content = b"test file content"
    session.upload_file(content, "/workspace/test.txt")
    print("Uploaded test.txt")

    print("\n--- 5. Filesystem: List ---")
    files = session.list_files("/workspace")
    file_names = [f["name"] for f in files]
    print(f"Files in /workspace: {file_names}")
    assert "test.txt" in file_names

    print("\n--- 6. Filesystem: Download ---")
    downloaded = session.download_file("/workspace/test.txt")
    print(f"Downloaded content: {downloaded.decode()}")
    assert downloaded == content

    print("\n--- 7. Interactive (REPL) ---")
    interaction = session.interact(language="bash")
    interaction.write("echo 'boxed-interact-test'\n")
    
    found = False
    start_time = time.time()
    while time.time() - start_time < 5:
        event = interaction.read_event()
        if event and event.get("method") == "stdout":
            chunk = event.get("params", {}).get("chunk", "")
            if "boxed-interact-test" in chunk:
                print(f"Received interactive output: {chunk.strip()}")
                found = True
                break
    
    interaction.close()
    assert found, "Did not receive expected interactive output"

    print("\n--- 8. Closing Session ---")
    session.close()
    print("Session closed")

if __name__ == "__main__":
    try:
        test_integration()
        print("\n✅ Integration Test Passed!")
    except Exception as e:
        print(f"\n❌ Integration Test Failed: {e}")
        import traceback
        traceback.print_exc()
        exit(1)
