import chalk from 'chalk';
import ora from 'ora';
import { client } from '../lib/client.js';

interface UndoResult {
  undone: number;
  task_id: string;
}

export async function undo(taskId: string): Promise<void> {
  const spinner = ora('Undoing file operations...').start();

  try {
    const result = await client.call<UndoResult>('undo', {
      task_id: taskId,
    });

    if (result.undone === 0) {
      spinner.warn('No operations to undo for this task');
    } else {
      spinner.succeed(`Undid ${result.undone} operation${result.undone > 1 ? 's' : ''} for task ${taskId}`);
    }
  } catch (err) {
    spinner.fail(`Failed to undo: ${err}`);
  }
}
