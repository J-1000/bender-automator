import { execSync } from 'child_process';
import chalk from 'chalk';
import ora from 'ora';
import { client } from '../lib/client.js';

interface SummarizeResult {
  summary: string;
}

export async function summarize(text?: string): Promise<void> {
  const spinner = ora('Summarizing...').start();

  try {
    // Get text from clipboard if not provided
    if (!text) {
      text = execSync('pbpaste', { encoding: 'utf-8' });
    }

    if (!text || text.trim().length === 0) {
      spinner.fail('No text to summarize');
      return;
    }

    const result = await client.call<SummarizeResult>('clipboard.summarize', {
      content: text,
    });

    spinner.stop();
    console.log(chalk.bold('Summary:'));
    console.log('─'.repeat(40));
    console.log(result.summary);
    console.log('─'.repeat(40));
  } catch (err) {
    spinner.fail(`Failed to summarize: ${err}`);
  }
}
