import { NextResponse } from 'next/server';
import { callDaemon } from '../../../lib/daemon';

export async function GET() {
  try {
    const config = await callDaemon('config.get');
    return NextResponse.json(config);
  } catch {
    return NextResponse.json(
      { error: 'Cannot fetch config' },
      { status: 503 }
    );
  }
}

export async function PUT(request: Request) {
  try {
    const body = await request.json();
    await callDaemon('config.set', body);
    return NextResponse.json({ status: 'ok' });
  } catch {
    return NextResponse.json(
      { error: 'Cannot update config' },
      { status: 503 }
    );
  }
}

export async function POST(request: Request) {
  try {
    const { action } = await request.json();
    if (action === 'reload') {
      await callDaemon('config.reload');
      return NextResponse.json({ status: 'reloaded' });
    }
    return NextResponse.json({ error: 'Unknown action' }, { status: 400 });
  } catch {
    return NextResponse.json(
      { error: 'Cannot perform action' },
      { status: 503 }
    );
  }
}
