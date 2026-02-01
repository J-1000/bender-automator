import { existsSync, renameSync } from 'fs';
import { resolve, dirname, basename } from 'path';
import chalk from 'chalk';
import ora from 'ora';
import inquirer from 'inquirer';
import { client } from '../lib/client.js';

interface RenameResult {
  original: string;
  suggested: string;
  reason: string;
}

export async function rename(files: string[]): Promise<void> {
  for (const file of files) {
    await renameFile(file);
  }
}

async function renameFile(file: string): Promise<void> {
  const spinner = ora(`Analyzing ${file}...`).start();

  try {
    const filePath = resolve(file);

    if (!existsSync(filePath)) {
      spinner.fail(`File not found: ${file}`);
      return;
    }

    const result = await client.call<RenameResult>('file.rename', {
      path: filePath,
    });

    spinner.stop();
    console.log();
    console.log(chalk.bold('Suggested Rename'));
    console.log('─'.repeat(50));
    console.log(`${chalk.gray('Current:')}   ${basename(filePath)}`);
    console.log(`${chalk.gray('Suggested:')} ${chalk.green(result.suggested)}`);
    if (result.reason) {
      console.log(`${chalk.gray('Reason:')}    ${result.reason}`);
    }
    console.log('─'.repeat(50));

    const { action } = await inquirer.prompt([
      {
        type: 'list',
        name: 'action',
        message: 'What would you like to do?',
        choices: [
          { name: 'Accept and rename', value: 'accept' },
          { name: 'Edit name', value: 'edit' },
          { name: 'Skip', value: 'skip' },
        ],
      },
    ]);

    if (action === 'skip') {
      console.log(chalk.gray('Skipped'));
      return;
    }

    let newName = result.suggested;
    if (action === 'edit') {
      const { editedName } = await inquirer.prompt([
        {
          type: 'input',
          name: 'editedName',
          message: 'New name:',
          default: result.suggested,
        },
      ]);
      newName = editedName;
    }

    const newPath = resolve(dirname(filePath), newName);
    if (existsSync(newPath)) {
      console.log(chalk.red(`File already exists: ${newName}`));
      return;
    }

    renameSync(filePath, newPath);
    console.log(chalk.green(`Renamed to: ${newName}`));
  } catch (err) {
    spinner.fail(`Failed to rename: ${err}`);
  }
}
