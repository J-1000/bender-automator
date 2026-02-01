import { spawn } from 'child_process';
import { existsSync } from 'fs';
import chalk from 'chalk';

const LOG_DIR = '/usr/local/var/log/bender';
const STDOUT_LOG = `${LOG_DIR}/stdout.log`;
const STDERR_LOG = `${LOG_DIR}/stderr.log`;

interface LogOptions {
  follow?: boolean;
  level?: string;
}

export async function logs(options: LogOptions): Promise<void> {
  const logFile = STDOUT_LOG;

  if (!existsSync(logFile)) {
    console.log(chalk.yellow('No log file found'));
    console.log(chalk.gray(`Expected at: ${logFile}`));
    return;
  }

  const args = options.follow ? ['-f', logFile] : [logFile];

  if (options.level) {
    // Filter by level using grep
    const level = options.level.toUpperCase();
    const tailProc = spawn('tail', args);
    const grepProc = spawn('grep', ['--line-buffered', level]);

    tailProc.stdout.pipe(grepProc.stdin);

    grepProc.stdout.on('data', (data) => {
      const line = data.toString();
      console.log(colorizeLog(line));
    });

    grepProc.on('close', () => {
      tailProc.kill();
    });

    process.on('SIGINT', () => {
      tailProc.kill();
      grepProc.kill();
    });
  } else {
    const proc = spawn('tail', args, { stdio: 'pipe' });

    proc.stdout.on('data', (data) => {
      const lines = data.toString().split('\n');
      for (const line of lines) {
        if (line.trim()) {
          console.log(colorizeLog(line));
        }
      }
    });

    proc.stderr.pipe(process.stderr);

    process.on('SIGINT', () => {
      proc.kill();
    });

    if (!options.follow) {
      await new Promise<void>((resolve) => {
        proc.on('close', resolve);
      });
    }
  }
}

function colorizeLog(line: string): string {
  if (line.includes('[DEBUG]')) {
    return chalk.gray(line);
  }
  if (line.includes('[INFO]')) {
    return chalk.white(line);
  }
  if (line.includes('[WARN]')) {
    return chalk.yellow(line);
  }
  if (line.includes('[ERROR]')) {
    return chalk.red(line);
  }
  return line;
}
