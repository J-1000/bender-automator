import { spawn } from 'child_process';
import { existsSync } from 'fs';
import chalk from 'chalk';
import ora from 'ora';
import { client } from '../lib/client.js';

const DAEMON_PATH = '/usr/local/bin/benderd';
const LAUNCHAGENT_PATH = `${process.env.HOME}/Library/LaunchAgents/com.bender.daemon.plist`;

export async function start(): Promise<void> {
  const spinner = ora('Starting daemon...').start();

  try {
    // Check if already running
    if (await client.isRunning()) {
      spinner.succeed('Daemon is already running');
      return;
    }

    // Check if LaunchAgent is installed
    if (existsSync(LAUNCHAGENT_PATH)) {
      // Use launchctl to start
      const proc = spawn('launchctl', ['load', LAUNCHAGENT_PATH]);
      await new Promise<void>((resolve, reject) => {
        proc.on('close', (code) => {
          if (code === 0) resolve();
          else reject(new Error(`launchctl exited with code ${code}`));
        });
      });
    } else if (existsSync(DAEMON_PATH)) {
      // Start directly in background
      const proc = spawn(DAEMON_PATH, [], {
        detached: true,
        stdio: 'ignore',
      });
      proc.unref();
    } else {
      spinner.fail('Daemon not installed');
      console.log(chalk.yellow('Install the daemon with: bender install agent'));
      return;
    }

    // Wait for daemon to be ready
    let attempts = 0;
    while (attempts < 20) {
      await new Promise((resolve) => setTimeout(resolve, 250));
      if (await client.isRunning()) {
        spinner.succeed('Daemon started');
        return;
      }
      attempts++;
    }

    spinner.fail('Daemon failed to start');
  } catch (err) {
    spinner.fail(`Failed to start daemon: ${err}`);
  }
}
