import { createConnection } from 'net';
import { NextResponse } from 'next/server';

const SOCKET_PATH = '/tmp/bender.sock';

async function callDaemon(method: string, params?: unknown): Promise<unknown> {
  return new Promise((resolve, reject) => {
    const socket = createConnection(SOCKET_PATH);
    let data = '';

    socket.on('connect', () => {
      const request = {
        jsonrpc: '2.0',
        method,
        params,
        id: 1,
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
        const response = JSON.parse(data.trim());
        if (response.error) {
          reject(new Error(response.error.message));
        } else {
          resolve(response.result);
        }
      } catch (err) {
        reject(err);
      }
    });

    socket.on('error', reject);
    socket.setTimeout(5000, () => {
      socket.destroy();
      reject(new Error('Timeout'));
    });
  });
}

export async function GET() {
  try {
    const status = await callDaemon('status.get');
    return NextResponse.json(status);
  } catch {
    return NextResponse.json(
      { error: 'Daemon not running' },
      { status: 503 }
    );
  }
}
