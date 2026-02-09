import { existsSync, statSync } from 'fs';
import { resolve, extname } from 'path';
import chalk from 'chalk';
import ora from 'ora';
import { client } from '../lib/client.js';

const IMAGE_EXTENSIONS = new Set(['.png', '.jpg', '.jpeg', '.gif', '.webp']);

interface ScreenshotResult {
  app: string;
  description: string;
  tags: string[];
  suggested_name: string;
}

export async function screenshot(file: string): Promise<void> {
  const spinner = ora('Tagging screenshot...').start();

  try {
    const filePath = resolve(file);

    if (!existsSync(filePath)) {
      spinner.fail(`File not found: ${file}`);
      return;
    }

    const stats = statSync(filePath);
    if (stats.isDirectory()) {
      spinner.fail('Cannot tag directories');
      return;
    }

    const ext = extname(filePath).toLowerCase();
    if (!IMAGE_EXTENSIONS.has(ext)) {
      spinner.fail(`Not an image file (supported: ${[...IMAGE_EXTENSIONS].join(', ')})`);
      return;
    }

    const result = await client.call<ScreenshotResult>('screenshot.tag', {
      path: filePath,
    });

    spinner.stop();
    console.log(chalk.bold('Screenshot Tags'));
    console.log('─'.repeat(40));
    console.log(`${chalk.gray('File:')}        ${file}`);
    console.log(`${chalk.gray('App:')}         ${result.app}`);
    console.log(`${chalk.gray('Description:')} ${result.description}`);
    console.log(`${chalk.gray('Tags:')}        ${result.tags.join(', ')}`);
    console.log(`${chalk.gray('Suggested:')}   ${result.suggested_name}`);
    console.log('─'.repeat(40));
  } catch (err) {
    spinner.fail(`Failed to tag screenshot: ${err}`);
  }
}
