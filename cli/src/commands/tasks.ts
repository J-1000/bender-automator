import chalk from 'chalk';
import ora from 'ora';
import { client, Task } from '../lib/client.js';

interface TasksOptions {
  limit?: string;
  status?: string;
}

function statusColor(status: string): string {
  switch (status) {
    case 'completed': return chalk.green(status);
    case 'running':   return chalk.yellow(status);
    case 'failed':    return chalk.red(status);
    case 'pending':   return chalk.gray(status);
    default:          return status;
  }
}

function truncateId(id: string, len = 8): string {
  return id.length > len ? id.slice(0, len) : id;
}

function formatTime(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleString();
}

export async function tasks(options: TasksOptions): Promise<void> {
  const spinner = ora('Fetching tasks...').start();

  try {
    const limit = options.limit ? parseInt(options.limit, 10) : 20;
    let result = await client.call<Task[]>('task.history', { limit });

    if (options.status) {
      result = result.filter(t => t.status === options.status);
    }

    spinner.stop();

    if (result.length === 0) {
      console.log(chalk.gray('No tasks found'));
      return;
    }

    // Header
    const cols = { id: 10, type: 22, status: 12, created: 20 };
    console.log(
      chalk.bold(
        'ID'.padEnd(cols.id) +
        'Type'.padEnd(cols.type) +
        'Status'.padEnd(cols.status) +
        'Created'
      )
    );
    console.log('─'.repeat(cols.id + cols.type + cols.status + cols.created));

    for (const t of result) {
      console.log(
        chalk.cyan(truncateId(t.id).padEnd(cols.id)) +
        t.type.padEnd(cols.type) +
        statusColor(t.status).padEnd(cols.status + 10) + // extra for ANSI codes
        chalk.gray(formatTime(t.created_at))
      );
    }

    console.log(chalk.gray(`\n${result.length} task${result.length !== 1 ? 's' : ''}`));
  } catch (err) {
    spinner.fail(`Failed to fetch tasks: ${err}`);
  }
}
