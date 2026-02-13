import { NextResponse } from 'next/server';
import { callDaemon } from '../../../lib/daemon';

export async function GET() {
  try {
    const status = await callDaemon('pipeline.status');
    return NextResponse.json(status);
  } catch {
    return NextResponse.json(
      { error: 'Cannot fetch pipeline status' },
      { status: 503 }
    );
  }
}
