import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest';
import { createServer, Server } from 'net';
import { DaemonClient } from '../lib/client.js';
import { unlinkSync } from 'fs';

const TEST_SOCKET = '/tmp/bender-cmd-test.sock';

describe('CLI commands - RPC integration', () => {
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

  function startMockServer(handler: (method: string, params: any) => unknown): Promise<void> {
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

  // screenshot.tag tests
  describe('screenshot.tag', () => {
    it('should call screenshot.tag with path param', async () => {
      let capturedMethod = '';
      let capturedParams: any = null;

      await startMockServer((method, params) => {
        capturedMethod = method;
        capturedParams = params;
        return {
          app: 'Safari',
          description: 'A browser window',
          tags: ['browser', 'web'],
          suggested_name: 'safari-browser-window.png',
        };
      });

      const result = await client.call<any>('screenshot.tag', { path: '/tmp/test.png' });

      expect(capturedMethod).toBe('screenshot.tag');
      expect(capturedParams.path).toBe('/tmp/test.png');
      expect(result.app).toBe('Safari');
      expect(result.tags).toEqual(['browser', 'web']);
      expect(result.suggested_name).toBe('safari-browser-window.png');
    });
  });

  // undo tests
  describe('undo', () => {
    it('should call undo with task_id param', async () => {
      let capturedMethod = '';
      let capturedParams: any = null;

      await startMockServer((method, params) => {
        capturedMethod = method;
        capturedParams = params;
        return { undone: 2, task_id: 'abc-123' };
      });

      const result = await client.call<any>('undo', { task_id: 'abc-123' });

      expect(capturedMethod).toBe('undo');
      expect(capturedParams.task_id).toBe('abc-123');
      expect(result.undone).toBe(2);
    });

    it('should return zero when no operations to undo', async () => {
      await startMockServer(() => {
        return { undone: 0, task_id: 'no-ops' };
      });

      const result = await client.call<any>('undo', { task_id: 'no-ops' });
      expect(result.undone).toBe(0);
    });
  });

  // task.history tests
  describe('task.history', () => {
    it('should call task.history with limit param', async () => {
      let capturedParams: any = null;

      await startMockServer((_method, params) => {
        capturedParams = params;
        return [
          { id: 'task-1', type: 'file.classify', status: 'completed', created_at: '2026-02-09T10:00:00Z' },
          { id: 'task-2', type: 'git.commit', status: 'pending', created_at: '2026-02-09T10:01:00Z' },
        ];
      });

      const result = await client.call<any[]>('task.history', { limit: 10 });

      expect(capturedParams.limit).toBe(10);
      expect(result).toHaveLength(2);
      expect(result[0].type).toBe('file.classify');
      expect(result[1].status).toBe('pending');
    });

    it('should return empty array when no tasks', async () => {
      await startMockServer(() => []);

      const result = await client.call<any[]>('task.history', { limit: 20 });
      expect(result).toEqual([]);
    });
  });
});

// screenshot image extension validation (unit test, no server needed)
describe('screenshot image validation', () => {
  const IMAGE_EXTENSIONS = new Set(['.png', '.jpg', '.jpeg', '.gif', '.webp']);

  it('should accept valid image extensions', () => {
    for (const ext of ['.png', '.jpg', '.jpeg', '.gif', '.webp']) {
      expect(IMAGE_EXTENSIONS.has(ext)).toBe(true);
    }
  });

  it('should reject non-image extensions', () => {
    for (const ext of ['.txt', '.pdf', '.mp4', '.zip', '.html']) {
      expect(IMAGE_EXTENSIONS.has(ext)).toBe(false);
    }
  });
});
