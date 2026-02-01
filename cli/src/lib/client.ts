import { createConnection, Socket } from 'net';

const SOCKET_PATH = '/tmp/bender.sock';

interface JsonRpcRequest {
  jsonrpc: '2.0';
  method: string;
  params?: unknown;
  id: number;
}

interface JsonRpcResponse {
  jsonrpc: '2.0';
  result?: unknown;
  error?: {
    code: number;
    message: string;
    data?: unknown;
  };
  id: number;
}

export class DaemonClient {
  private socketPath: string;
  private requestId = 0;

  constructor(socketPath = SOCKET_PATH) {
    this.socketPath = socketPath;
  }

  async call<T>(method: string, params?: unknown): Promise<T> {
    return new Promise((resolve, reject) => {
      const socket = createConnection(this.socketPath);
      let data = '';

      socket.on('connect', () => {
        const request: JsonRpcRequest = {
          jsonrpc: '2.0',
          method,
          params,
          id: ++this.requestId,
        };
        socket.write(JSON.stringify(request) + '\n');
      });

      socket.on('data', (chunk) => {
        data += chunk.toString();
        if (data.includes('\n')) {
          socket.end();
        }
      });

      socket.on('end', () => {
        try {
          const response: JsonRpcResponse = JSON.parse(data.trim());
          if (response.error) {
            reject(new Error(response.error.message));
          } else {
            resolve(response.result as T);
          }
        } catch (err) {
          reject(new Error(`Failed to parse response: ${data}`));
        }
      });

      socket.on('error', (err) => {
        if ((err as NodeJS.ErrnoException).code === 'ENOENT') {
          reject(new Error('Daemon is not running. Start it with: bender start'));
        } else if ((err as NodeJS.ErrnoException).code === 'ECONNREFUSED') {
          reject(new Error('Cannot connect to daemon. Is it running?'));
        } else {
          reject(err);
        }
      });

      socket.setTimeout(10000, () => {
        socket.destroy();
        reject(new Error('Request timed out'));
      });
    });
  }

  async isRunning(): Promise<boolean> {
    try {
      await this.call('status.get');
      return true;
    } catch {
      return false;
    }
  }
}

export interface DaemonStatus {
  running: boolean;
  version: string;
  uptime: string;
  started_at: string;
  pid: number;
  go_version: string;
}

export interface HealthCheck {
  status: string;
  checks: Record<string, string>;
  timestamp: string;
}

export interface Task {
  id: string;
  type: string;
  priority: number;
  status: string;
  created_at: string;
  started_at?: string;
  finished_at?: string;
}

export const client = new DaemonClient();
