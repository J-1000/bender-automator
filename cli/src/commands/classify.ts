import { existsSync, statSync } from 'fs';
import { resolve } from 'path';
import chalk from 'chalk';
import ora from 'ora';
import { client } from '../lib/client.js';

interface ClassifyResult {
  category: string;
  destination: string;
  confidence: number;
}

export async function classify(file: string): Promise<void> {
  const spinner = ora('Classifying file...').start();

  try {
    const filePath = resolve(file);

    if (!existsSync(filePath)) {
      spinner.fail(`File not found: ${file}`);
      return;
    }

    const stats = statSync(filePath);
    if (stats.isDirectory()) {
      spinner.fail('Cannot classify directories');
      return;
    }

    const result = await client.call<ClassifyResult>('file.classify', {
      path: filePath,
    });

    spinner.stop();
    console.log(chalk.bold('File Classification'));
    console.log('─'.repeat(40));
    console.log(`${chalk.gray('File:')}        ${file}`);
    console.log(`${chalk.gray('Category:')}    ${result.category}`);
    console.log(`${chalk.gray('Suggested:')}   ${result.destination}`);
    if (result.confidence) {
      console.log(`${chalk.gray('Confidence:')} ${Math.round(result.confidence * 100)}%`);
    }
    console.log('─'.repeat(40));
    console.log(chalk.gray('To move: mv "' + filePath + '" "' + result.destination + '"'));
  } catch (err) {
    spinner.fail(`Failed to classify: ${err}`);
  }
}
