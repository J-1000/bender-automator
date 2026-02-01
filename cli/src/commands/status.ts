import chalk from 'chalk';
import { client, DaemonStatus, HealthCheck } from '../lib/client.js';

export async function status(): Promise<void> {
  try {
    const [daemonStatus, health] = await Promise.all([
      client.call<DaemonStatus>('status.get'),
      client.call<HealthCheck>('status.health'),
    ]);

    console.log(chalk.bold('Bender Daemon Status'));
    console.log('─'.repeat(40));
    console.log(`${chalk.gray('Status:')}    ${chalk.green('● Running')}`);
    console.log(`${chalk.gray('Version:')}   ${daemonStatus.version}`);
    console.log(`${chalk.gray('Uptime:')}    ${daemonStatus.uptime}`);
    console.log(`${chalk.gray('PID:')}       ${daemonStatus.pid}`);
    console.log(`${chalk.gray('Go:')}        ${daemonStatus.go_version}`);
    console.log();
    console.log(chalk.bold('Health Checks'));
    console.log('─'.repeat(40));
    for (const [check, status] of Object.entries(health.checks)) {
      const icon = status === 'ok' ? chalk.green('✓') : chalk.red('✗');
      console.log(`${icon} ${check}: ${status}`);
    }
  } catch (err) {
    console.log(chalk.bold('Bender Daemon Status'));
    console.log('─'.repeat(40));
    console.log(`${chalk.gray('Status:')}    ${chalk.red('● Stopped')}`);
    console.log();
    console.log(chalk.yellow('Start the daemon with: bender start'));
  }
}
