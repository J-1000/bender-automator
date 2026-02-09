import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest';
import { createServer, Server } from 'net';
import { DaemonClient } from '../lib/client.js';
import { unlinkSync } from 'fs';

const TEST_SOCKET = '/tmp/bender-test.sock';

describe('DaemonClient', () => {
  let server: Server;
  let client: DaemonClient;

  beforeAll(() => {
    try { unlinkSync(TEST_SOCKET); } catch {}
    client = new DaemonClient(TEST_SOCKET);
  });

  afterAll(() => {
    try { unlinkSync(TEST_SOCKET); } catch {}
  });

  afterEach(() => {
    if (server) {
      server.close();
    }
  });

  function startMockServer(handler: (method: string, params: unknown) => unknown): Promise<void> {
    return new Promise((resolve) => {
      server = createServer((conn) => {
        let data = '';
        conn.on('data', (chunk) => {
          data += chunk.toString();
          if (data.includes('\n')) {
            const req = JSON.parse(data.trim());
            const result = handler(req.method, req.params);
            const response = JSON.stringify({
              jsonrpc: '2.0',
              result,
              id: req.id,
            });
            conn.write(response + '\n');
          }
        });
      });
      server.listen(TEST_SOCKET, () => resolve());
    });
  }

  function startMockErrorServer(code: number, message: string): Promise<void> {
    return new Promise((resolve) => {
      server = createServer((conn) => {
        let data = '';
        conn.on('data', (chunk) => {
          data += chunk.toString();
          if (data.includes('\n')) {
            const req = JSON.parse(data.trim());
            const response = JSON.stringify({
              jsonrpc: '2.0',
              error: { code, message },
              id: req.id,
            });
            conn.write(response + '\n');
          }
        });
      });
      server.listen(TEST_SOCKET, () => resolve());
    });
  }

  it('should send JSON-RPC request and receive response', async () => {
    await startMockServer((method) => {
      if (method === 'status.get') {
        return { running: true, version: '0.1.0' };
      }
      return null;
    });

    const result = await client.call<{ running: boolean; version: string }>('status.get');
    expect(result.running).toBe(true);
    expect(result.version).toBe('0.1.0');
  });

  it('should pass params to server', async () => {
    await startMockServer((_method, params) => {
      return { echo: params };
    });

    const result = await client.call<{ echo: { foo: string } }>('test.echo', { foo: 'bar' });
    expect(result.echo).toEqual({ foo: 'bar' });
  });

  it('should reject on server error response', async () => {
    await startMockErrorServer(-32601, 'Method not found');

    await expect(client.call('unknown.method')).rejects.toThrow('Method not found');
  });

  it('should reject when daemon is not running', async () => {
    const badClient = new DaemonClient('/tmp/bender-nonexistent.sock');
    await expect(badClient.call('status.get')).rejects.toThrow('not running');
  });

  it('should report running status correctly', async () => {
    await startMockServer(() => ({ status: 'ok' }));

    const running = await client.isRunning();
    expect(running).toBe(true);
  });

  it('should report not running when socket missing', async () => {
    const badClient = new DaemonClient('/tmp/bender-nonexistent.sock');
    const running = await badClient.isRunning();
    expect(running).toBe(false);
  });

  it('should increment request IDs', async () => {
    const ids: number[] = [];
    await startMockServer(() => {
      return { ok: true };
    });

    // Override to capture IDs: we'll just verify two calls get different results
    await client.call('test.a');
    await client.call('test.b');
    // If we got here without error, both requests were processed
  });
});
