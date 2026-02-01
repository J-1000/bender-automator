import { existsSync, mkdirSync, writeFileSync, unlinkSync, readFileSync } from 'fs';
import { execSync } from 'child_process';
import chalk from 'chalk';
import ora from 'ora';

const DAEMON_PATH = '/usr/local/bin/benderd';
const LAUNCHAGENT_DIR = `${process.env.HOME}/Library/LaunchAgents`;
const LAUNCHAGENT_PATH = `${LAUNCHAGENT_DIR}/com.bender.daemon.plist`;
const CONFIG_DIR = `${process.env.HOME}/.config/bender`;
const CONFIG_PATH = `${CONFIG_DIR}/config.yaml`;
const LOG_DIR = '/usr/local/var/log/bender';

const LAUNCHAGENT_PLIST = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.bender.daemon</string>
    <key>ProgramArguments</key>
    <array>
        <string>${DAEMON_PATH}</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>${LOG_DIR}/stdout.log</string>
    <key>StandardErrorPath</key>
    <string>${LOG_DIR}/stderr.log</string>
    <key>EnvironmentVariables</key>
    <dict>
        <key>BENDER_CONFIG</key>
        <string>${CONFIG_PATH}</string>
    </dict>
</dict>
</plist>`;

export async function installAgent(): Promise<void> {
  const spinner = ora('Installing LaunchAgent...').start();

  try {
    // Check if daemon binary exists
    if (!existsSync(DAEMON_PATH)) {
      spinner.fail(`Daemon binary not found at ${DAEMON_PATH}`);
      console.log(chalk.yellow('Please install the daemon binary first'));
      return;
    }

    // Create directories
    mkdirSync(LAUNCHAGENT_DIR, { recursive: true });
    mkdirSync(CONFIG_DIR, { recursive: true });
    mkdirSync(LOG_DIR, { recursive: true });

    // Write LaunchAgent plist
    writeFileSync(LAUNCHAGENT_PATH, LAUNCHAGENT_PLIST);

    // Copy default config if not exists
    if (!existsSync(CONFIG_PATH)) {
      // Would copy from package or create default
      spinner.info('Remember to create config at ' + CONFIG_PATH);
    }

    spinner.succeed('LaunchAgent installed');
    console.log(chalk.gray('Start with: bender start'));
  } catch (err) {
    spinner.fail(`Failed to install: ${err}`);
  }
}

export async function uninstallAgent(): Promise<void> {
  const spinner = ora('Uninstalling LaunchAgent...').start();

  try {
    // Stop if running
    if (existsSync(LAUNCHAGENT_PATH)) {
      try {
        execSync(`launchctl unload ${LAUNCHAGENT_PATH}`, { stdio: 'ignore' });
      } catch {
        // Ignore errors if not loaded
      }
      unlinkSync(LAUNCHAGENT_PATH);
    }

    spinner.succeed('LaunchAgent uninstalled');
  } catch (err) {
    spinner.fail(`Failed to uninstall: ${err}`);
  }
}

export async function installHooks(): Promise<void> {
  const spinner = ora('Installing git hooks...').start();

  try {
    // Check if we're in a git repo
    try {
      execSync('git rev-parse --git-dir', { stdio: 'ignore' });
    } catch {
      spinner.fail('Not a git repository');
      return;
    }

    const hooksDir = execSync('git rev-parse --git-dir', { encoding: 'utf-8' }).trim() + '/hooks';
    mkdirSync(hooksDir, { recursive: true });

    // Install prepare-commit-msg hook
    const hookPath = `${hooksDir}/prepare-commit-msg`;
    const hookContent = `#!/bin/bash
# Bender git hook - generate commit message
COMMIT_MSG_FILE=$1
COMMIT_SOURCE=$2

# Only generate for new commits (not merges, amends, etc.)
if [ -z "$COMMIT_SOURCE" ]; then
    GENERATED=$(echo '{"jsonrpc":"2.0","method":"git.generate_commit","id":1}' | nc -U /tmp/bender.sock 2>/dev/null | jq -r '.result.message // empty')
    if [ -n "$GENERATED" ]; then
        echo "$GENERATED" > "$COMMIT_MSG_FILE"
    fi
fi
`;

    writeFileSync(hookPath, hookContent);
    execSync(`chmod +x ${hookPath}`);

    spinner.succeed('Git hooks installed');
    console.log(chalk.gray('Hook: prepare-commit-msg'));
  } catch (err) {
    spinner.fail(`Failed to install hooks: ${err}`);
  }
}

export async function uninstallHooks(): Promise<void> {
  const spinner = ora('Removing git hooks...').start();

  try {
    // Check if we're in a git repo
    try {
      execSync('git rev-parse --git-dir', { stdio: 'ignore' });
    } catch {
      spinner.fail('Not a git repository');
      return;
    }

    const hooksDir = execSync('git rev-parse --git-dir', { encoding: 'utf-8' }).trim() + '/hooks';
    const hookPath = `${hooksDir}/prepare-commit-msg`;

    if (existsSync(hookPath)) {
      // Check if it's our hook
      const content = readFileSync(hookPath, 'utf-8');
      if (content.includes('Bender git hook')) {
        unlinkSync(hookPath);
        spinner.succeed('Git hooks removed');
      } else {
        spinner.warn('Hook exists but was not installed by Bender');
      }
    } else {
      spinner.info('No Bender hooks installed');
    }
  } catch (err) {
    spinner.fail(`Failed to remove hooks: ${err}`);
  }
}
