import { Boxed } from '../src';
import assert from 'assert';

async function main() {
    console.log("📦 Boxed SDK Integration Test");

    // 1. Connect (assuming server running on 8080)
    const client = new Boxed({ baseUrl: 'http://localhost:8080' });
    console.log("✅ Client initialized");

    // 2. Create Session
    console.log("Creating session...");
    const session = await client.createSession({
        template: 'python:3.10-slim',
        timeoutMs: 10000
    });
    console.log(`✅ Session created: ${session.id}`);

    // 3. Run Code
    console.log("Running code...");
    const code = `
print("Hello SDK")
import sys
print("Stderr Output", file=sys.stderr)
`;

    const result = await session.run(code);
    console.log("Run finished:", result);

    assert.ok(result.stdout.includes("Hello SDK"), "Stdout should contain Hello SDK");
    assert.ok(result.stderr.includes("Stderr Output"), "Stderr should contain Stderr Output");
    assert.strictEqual(result.exitCode, 0, "Exit code should be 0");
    console.log("✅ Execution verified");

    // 4. File System
    console.log("Testing File System...");

    // Upload
    const uploadContent = Buffer.from("SDK Upload Test");
    await session.uploadFile(uploadContent as any, "/workspace/sdk_upload.txt");
    console.log("✅ File uploaded");

    // List
    const files = await session.listFiles("/workspace");
    const found = files.find(f => f.name === "sdk_upload.txt");
    assert.ok(found, "Uploaded file found in listing");
    console.log("✅ File listed");

    // Download
    const downloaded = await session.downloadFile("/workspace/sdk_upload.txt");
    const content = Buffer.from(downloaded).toString();
    assert.strictEqual(content, "SDK Upload Test", "Downloaded content matches");
    console.log("✅ File downloaded");

    // 5. Close
    console.log("Closing session...");
    await session.close();
    console.log("✅ Session closed");
}

main().catch(err => {
    console.error("❌ Test Failed:", err);
    process.exit(1);
});
