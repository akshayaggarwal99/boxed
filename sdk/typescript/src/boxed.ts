import axios, { AxiosInstance } from 'axios';

export interface BoxedOptions {
    baseUrl: string;
}

export interface CreateSessionOptions {
    template: string;
    timeoutMs?: number;
    metadata?: Record<string, string>;
}

export interface RunOptions {
    code: string;
    language?: string;
}

export interface Artifact {
    type: string;
    path: string;
    url: string; // Data URI for MVP
}

export interface ExecutionResult {
    stdout: string;
    stderr: string;
    artifacts: Artifact[];
    exitCode: number;
}

export class Session {
    constructor(private readonly client: AxiosInstance, public readonly id: string) { }

    /**
     * Executes code within the sandbox session.
     * @param codeOrOptions The code string or options object.
     */
    async run(codeOrOptions: string | RunOptions): Promise<ExecutionResult> {
        const options = typeof codeOrOptions === 'string' ? { code: codeOrOptions } : codeOrOptions;
        const language = options.language || 'python';

        try {
            const response = await this.client.post<{
                stdout: string;
                stderr: string;
                exit_code?: number;
                artifacts?: Array<{ mime: string; path: string; data_base64: string }>;
            }>(`/v1/sandbox/${this.id}/exec`, {
                code: options.code,
                language: language,
            });

            const data = response.data;
            const artifacts: Artifact[] = (data.artifacts || []).map(a => ({
                type: a.mime,
                path: a.path,
                url: `data:${a.mime};base64,${a.data_base64}`
            }));

            // Exit code might be missing if process failed to report it? 
            // Our API returns *int.
            const exitCode = data.exit_code !== undefined ? data.exit_code : -1;

            return {
                stdout: data.stdout || '',
                stderr: data.stderr || '',
                artifacts: artifacts,
                exitCode: exitCode,
            };
        } catch (error: any) {
            // Enhance error if possible
            if (axios.isAxiosError(error) && error.response) {
                throw new Error(`Execution failed: ${error.response.status} ${JSON.stringify(error.response.data)}`);
            }
            throw error;
        }
    }

    /**
     * Closes the session and releases resources (stops the sandbox).
     */
    async close(): Promise<void> {
        try {
            await this.client.delete(`/v1/sandbox/${this.id}`);
        } catch (error: any) {
            if (axios.isAxiosError(error) && error.response && error.response.status === 404) {
                // Already closed
                return;
            }
            throw error;
        }
    }

    /**
     * Lists files in the sandbox.
     * @param path Directory path (default: "/")
     */
    async listFiles(path: string = '/'): Promise<FileEntry[]> {
        const response = await this.client.get<{ files: FileEntry[] }>(`/v1/sandbox/${this.id}/files`, {
            params: { path }
        });
        return response.data.files || [];
    }

    /**
     * Uploads a file to the sandbox.
     * @param options Upload options
     */
    async uploadFile(file: Buffer | Blob, path?: string): Promise<{ status: string; path: string }> {
        const formData = new FormData();

        // If path looks like a file path (starts after /), extract it.
        // The server appends file.Filename to path.
        // So we should pass the directory in 'path' and the filename in the blob field.
        let destDir = '/workspace';
        let filename = 'file';

        if (path) {
            if (path.endsWith('/')) {
                destDir = path;
            } else {
                const parts = path.split('/');
                filename = parts.pop() || 'file';
                destDir = parts.join('/') || '/';
            }
        }

        formData.append('file', new Blob([file as any]), filename);
        formData.append('path', destDir);

        const response = await this.client.post(`/v1/sandbox/${this.id}/files`, formData, {
            headers: { 'Content-Type': 'multipart/form-data' }
        });
        return response.data;
    }

    /**
     * Downloads a file from the sandbox.
     * @param path Path to file
     */
    async downloadFile(path: string): Promise<ArrayBuffer> {
        const response = await this.client.get(`/v1/sandbox/${this.id}/files/content`, {
            params: { path },
            responseType: 'arraybuffer'
        });
        return response.data;
    }

    /**
     * Starts an interactive session (REPL) using WebSockets.
     * @param language The language shell to start (default: "bash")
     */
    async interact(language: string = 'bash'): Promise<Interaction> {
        const WebSocket = require('ws');
        const baseUrl = this.client.defaults.baseURL || 'http://localhost:8080';
        const wsUrl = baseUrl.replace(/^http/, 'ws') + `/v1/sandbox/${this.id}/interact?lang=${language}`;

        const ws = new WebSocket(wsUrl);
        return new Interaction(ws);
    }
}

export class Interaction {
    private onOutputHandlers: Array<(data: { type: 'stdout' | 'stderr' | 'exit' | 'error', chunk?: string, code?: number, message?: string }) => void> = [];

    constructor(private readonly ws: any) {
        ws.on('message', (data: Buffer) => {
            try {
                const event = JSON.parse(data.toString());
                if (event.method) {
                    const type = event.method;
                    const params = event.params || {};
                    this.onOutputHandlers.forEach(h => h({
                        type: type as any,
                        chunk: params.chunk,
                        code: params.code,
                        message: params.message
                    }));
                }
            } catch (e) {
                // Not a JSON-RPC event? ignore or pass as raw stdout chunk?
                // Our API implementation wraps most things.
            }
        });
    }

    onOutput(handler: (data: { type: 'stdout' | 'stderr' | 'exit' | 'error', chunk?: string, code?: number, message?: string }) => void) {
        this.onOutputHandlers.push(handler);
    }

    async write(data: string): Promise<void> {
        return new Promise((resolve, reject) => {
            this.ws.send(data, (err: any) => {
                if (err) reject(err);
                else resolve();
            });
        });
    }

    async close(): Promise<void> {
        this.ws.close();
    }
}

export interface FileEntry {
    name: string;
    path: string;
    size: number;
    mode: number;
    is_dir: boolean;
    last_modified: string;
}

export class Boxed {
    private readonly client: AxiosInstance;

    constructor(options: BoxedOptions) {
        this.client = axios.create({
            baseURL: options.baseUrl,
            headers: {
                'Content-Type': 'application/json',
            }
        });
    }

    /**
     * Creates a new sandbox session.
     */
    async createSession(options: CreateSessionOptions): Promise<Session> {
        const timeoutSec = options.timeoutMs ? Math.ceil(options.timeoutMs / 1000) : 300;

        try {
            const response = await this.client.post<{ sandbox_id: string; status: string }>('/v1/sandbox', {
                template: options.template,
                timeout: timeoutSec,
                metadata: options.metadata,
            });

            return new Session(this.client, response.data.sandbox_id);
        } catch (error: any) {
            if (axios.isAxiosError(error) && error.response) {
                throw new Error(`Failed to create session: ${error.response.status} ${JSON.stringify(error.response.data)}`);
            }
            throw error;
        }
    }

    /**
     * Lists all active sandbox sessions.
     */
    async listSessions(): Promise<Array<{ id: string; state: string; created_at: string; driver_type: string }>> {
        try {
            const response = await this.client.get<{ sandboxes: Array<any> }>('/v1/sandbox');
            return response.data.sandboxes || [];
        } catch (error: any) {
            if (axios.isAxiosError(error) && error.response) {
                throw new Error(`Failed to list sessions: ${error.response.status} ${JSON.stringify(error.response.data)}`);
            }
            throw error;
        }
    }
}
