import { NextResponse } from 'next/server';
import { callDaemon } from '../../../lib/daemon';

export async function GET() {
  try {
    const tasks = await callDaemon('task.queue');
    return NextResponse.json(tasks || []);
  } catch {
    return NextResponse.json([]);
  }
}

export async function POST(request: Request) {
  try {
    const body = await request.json();

    if (body.action === 'undo') {
      const result = await callDaemon('undo', { task_id: body.task_id });
      return NextResponse.json(result);
    }

    if (body.action === 'cancel') {
      const result = await callDaemon('task.cancel', { id: body.task_id });
      return NextResponse.json(result);
    }

    return NextResponse.json({ error: 'Unknown action' }, { status: 400 });
  } catch {
    return NextResponse.json(
      { error: 'Action failed' },
      { status: 503 }
    );
  }
}
