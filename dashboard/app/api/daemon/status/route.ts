import { NextResponse } from 'next/server';
import { callDaemon } from '../../../../lib/daemon';

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
