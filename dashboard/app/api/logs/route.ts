import { NextResponse } from 'next/server';
import { callDaemon } from '../../../lib/daemon';

export async function GET(request: Request) {
  const { searchParams } = new URL(request.url);
  const limit = parseInt(searchParams.get('limit') || '100');
  const level = searchParams.get('level') || '';

  try {
    const logs = await callDaemon('logs.get', { limit, level });
    return NextResponse.json(logs || []);
  } catch {
    return NextResponse.json([], { status: 503 });
  }
}
