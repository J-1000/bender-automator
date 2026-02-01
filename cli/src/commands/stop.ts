import { spawn } from 'child_process';
import { existsSync } from 'fs';
import chalk from 'chalk';
import ora from 'ora';
import { client, DaemonStatus } from '../lib/client.js';

const LAUNCHAGENT_PATH = `${process.env.HOME}/Library/LaunchAgents/com.bender.daemon.plist`;

export async function stop(): Promise<void> {
  const spinner = ora('Stopping daemon...').start();

  try {
    // Check if running
    if (!(await client.isRunning())) {
      spinner.succeed('Daemon is not running');
      return;
    }

    // Get PID
    const status = await client.call<DaemonStatus>('status.get');

    // Try launchctl first if agent is installed
    if (existsSync(LAUNCHAGENT_PATH)) {
      const proc = spawn('launchctl', ['unload', LAUNCHAGENT_PATH]);
      await new Promise<void>((resolve, reject) => {
        proc.on('close', (code) => {
          if (code === 0) resolve();
          else reject(new Error(`launchctl exited with code ${code}`));
        });
      });
    } else if (status.pid) {
      // Send SIGTERM directly
      process.kill(status.pid, 'SIGTERM');
    }

    // Wait for daemon to stop
    let attempts = 0;
    while (attempts < 20) {
      await new Promise((resolve) => setTimeout(resolve, 250));
      if (!(await client.isRunning())) {
        spinner.succeed('Daemon stopped');
        return;
      }
      attempts++;
    }

    spinner.fail('Daemon did not stop gracefully');
    console.log(chalk.yellow(`Try: kill ${status.pid}`));
  } catch (err) {
    spinner.fail(`Failed to stop daemon: ${err}`);
  }
}
