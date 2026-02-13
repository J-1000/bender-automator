import { existsSync } from 'fs';
import { resolve } from 'path';
import chalk from 'chalk';
import ora from 'ora';
import { client } from '../lib/client.js';

interface PipelineStatusResult {
  auto_file: {
    enabled: boolean;
    auto_move: boolean;
    auto_rename: boolean;
    settle_delay_ms: number;
    watch_dirs: string[];
  };
  screenshot: {
    enabled: boolean;
    use_vision: boolean;
    rename: boolean;
    settle_delay_ms: number;
    watch_dir: string;
    destination: string;
  };
}

interface AutoFileResult {
  original_path: string;
  final_path: string;
  category: string;
  new_name?: string;
  steps: { name: string; status: string; detail?: string }[];
}

interface ScreenshotPipelineResult {
  original_path: string;
  final_path: string;
  app: string;
  description: string;
  tags: string[];
  steps: { name: string; status: string; detail?: string }[];
}

export async function pipelineStatus(): Promise<void> {
  const spinner = ora('Fetching pipeline status...').start();

  try {
    const result = await client.call<PipelineStatusResult>('pipeline.status');
    spinner.stop();

    console.log(chalk.bold('Pipeline Status'));
    console.log('═'.repeat(50));

    // Auto-file
    console.log();
    console.log(chalk.bold('Auto-File Pipeline'));
    console.log('─'.repeat(50));
    console.log(`${chalk.gray('Enabled:')}       ${result.auto_file.enabled ? chalk.green('yes') : chalk.red('no')}`);
    console.log(`${chalk.gray('Auto Move:')}     ${result.auto_file.auto_move ? chalk.green('yes') : chalk.red('no')}`);
    console.log(`${chalk.gray('Auto Rename:')}   ${result.auto_file.auto_rename ? chalk.green('yes') : chalk.red('no')}`);
    console.log(`${chalk.gray('Settle Delay:')}  ${result.auto_file.settle_delay_ms}ms`);
    console.log(`${chalk.gray('Watch Dirs:')}    ${result.auto_file.watch_dirs.join(', ')}`);

    // Screenshot
    console.log();
    console.log(chalk.bold('Screenshot Pipeline'));
    console.log('─'.repeat(50));
    console.log(`${chalk.gray('Enabled:')}       ${result.screenshot.enabled ? chalk.green('yes') : chalk.red('no')}`);
    console.log(`${chalk.gray('Vision:')}        ${result.screenshot.use_vision ? chalk.green('yes') : chalk.red('no')}`);
    console.log(`${chalk.gray('Rename:')}        ${result.screenshot.rename ? chalk.green('yes') : chalk.red('no')}`);
    console.log(`${chalk.gray('Settle Delay:')}  ${result.screenshot.settle_delay_ms}ms`);
    console.log(`${chalk.gray('Watch Dir:')}     ${result.screenshot.watch_dir}`);
    console.log(`${chalk.gray('Destination:')}   ${result.screenshot.destination}`);
  } catch (err) {
    spinner.fail(`Failed to fetch pipeline status: ${err}`);
  }
}

export async function pipelineRun(type: string, file: string): Promise<void> {
  const filePath = resolve(file);

  if (!existsSync(filePath)) {
    console.error(chalk.red(`File not found: ${file}`));
    return;
  }

  const method = type === 'auto-file' ? 'pipeline.auto_file' : 'pipeline.screenshot';
  const label = type === 'auto-file' ? 'Auto-file pipeline' : 'Screenshot pipeline';
  const spinner = ora(`Running ${label}...`).start();

  try {
    if (type === 'auto-file') {
      const result = await client.call<AutoFileResult>(method, { path: filePath });
      spinner.stop();

      console.log(chalk.bold(label));
      console.log('─'.repeat(50));
      console.log(`${chalk.gray('Original:')}  ${result.original_path}`);
      console.log(`${chalk.gray('Final:')}     ${result.final_path}`);
      console.log(`${chalk.gray('Category:')}  ${result.category}`);
      if (result.new_name) {
        console.log(`${chalk.gray('Renamed:')}   ${result.new_name}`);
      }
      console.log();
      console.log(chalk.gray('Steps:'));
      for (const step of result.steps) {
        const icon = step.status === 'ok' ? chalk.green('✓') : chalk.red('✗');
        console.log(`  ${icon} ${step.name}${step.detail ? ` — ${step.detail}` : ''}`);
      }
    } else {
      const result = await client.call<ScreenshotPipelineResult>(method, { path: filePath });
      spinner.stop();

      console.log(chalk.bold(label));
      console.log('─'.repeat(50));
      console.log(`${chalk.gray('Original:')}     ${result.original_path}`);
      console.log(`${chalk.gray('Final:')}        ${result.final_path}`);
      console.log(`${chalk.gray('App:')}          ${result.app}`);
      console.log(`${chalk.gray('Description:')}  ${result.description}`);
      if (result.tags.length > 0) {
        console.log(`${chalk.gray('Tags:')}         ${result.tags.join(', ')}`);
      }
      console.log();
      console.log(chalk.gray('Steps:'));
      for (const step of result.steps) {
        const icon = step.status === 'ok' ? chalk.green('✓') : chalk.red('✗');
        console.log(`  ${icon} ${step.name}${step.detail ? ` — ${step.detail}` : ''}`);
      }
    }
  } catch (err) {
    spinner.fail(`Pipeline failed: ${err}`);
  }
}
