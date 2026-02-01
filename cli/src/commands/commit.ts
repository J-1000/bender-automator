import { execSync } from 'child_process';
import chalk from 'chalk';
import ora from 'ora';
import inquirer from 'inquirer';
import { client } from '../lib/client.js';

interface CommitOptions {
  auto?: boolean;
}

interface CommitResult {
  message: string;
  subject: string;
  body?: string;
}

export async function commit(options: CommitOptions): Promise<void> {
  const spinner = ora('Analyzing staged changes...').start();

  try {
    // Check if we're in a git repo
    try {
      execSync('git rev-parse --git-dir', { stdio: 'ignore' });
    } catch {
      spinner.fail('Not a git repository');
      return;
    }

    // Check for staged changes
    const staged = execSync('git diff --cached --name-only', { encoding: 'utf-8' }).trim();
    if (!staged) {
      spinner.fail('No staged changes');
      console.log(chalk.gray('Stage changes with: git add <files>'));
      return;
    }

    // Get the diff
    const diff = execSync('git diff --cached', { encoding: 'utf-8' });
    const files = staged.split('\n').filter(Boolean);

    const result = await client.call<CommitResult>('git.generate_commit', {
      diff,
      files,
    });

    spinner.stop();
    console.log();
    console.log(chalk.bold('Generated commit message:'));
    console.log('─'.repeat(60));
    console.log(result.message);
    console.log('─'.repeat(60));
    console.log();

    if (options.auto) {
      // Auto-commit without prompting
      await doCommit(result.message);
      return;
    }

    const { action } = await inquirer.prompt([
      {
        type: 'list',
        name: 'action',
        message: 'What would you like to do?',
        choices: [
          { name: '[e]dit', value: 'edit' },
          { name: '[a]ccept', value: 'accept' },
          { name: '[r]egenerate', value: 'regenerate' },
          { name: '[c]ancel', value: 'cancel' },
        ],
      },
    ]);

    switch (action) {
      case 'accept':
        await doCommit(result.message);
        break;
      case 'edit':
        const { editedMessage } = await inquirer.prompt([
          {
            type: 'editor',
            name: 'editedMessage',
            message: 'Edit commit message:',
            default: result.message,
          },
        ]);
        if (editedMessage.trim()) {
          await doCommit(editedMessage.trim());
        } else {
          console.log(chalk.yellow('Commit cancelled (empty message)'));
        }
        break;
      case 'regenerate':
        console.log(chalk.gray('Regenerating...'));
        await commit(options);
        break;
      case 'cancel':
        console.log(chalk.gray('Commit cancelled'));
        break;
    }
  } catch (err) {
    spinner.fail(`Failed to generate commit message: ${err}`);
  }
}

async function doCommit(message: string): Promise<void> {
  try {
    execSync(`git commit -m "${message.replace(/"/g, '\\"')}"`, {
      encoding: 'utf-8',
      stdio: 'pipe',
    });

    // Get the commit hash
    const hash = execSync('git rev-parse --short HEAD', { encoding: 'utf-8' }).trim();
    console.log(chalk.green(`Committed: ${hash}`));
  } catch (err: any) {
    console.log(chalk.red('Commit failed:'));
    console.log(err.stderr || err.message);
  }
}
