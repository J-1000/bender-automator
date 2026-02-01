import { readFileSync, writeFileSync, existsSync } from 'fs';
import { spawn } from 'child_process';
import chalk from 'chalk';
import YAML from 'yaml';

const CONFIG_PATH = `${process.env.HOME}/.config/bender/config.yaml`;

export async function config(
  action?: string,
  key?: string,
  value?: string
): Promise<void> {
  if (!action) {
    // Open config in editor
    await openInEditor();
    return;
  }

  switch (action) {
    case 'get':
      await getConfig(key);
      break;
    case 'set':
      await setConfig(key, value);
      break;
    case 'validate':
      await validateConfig();
      break;
    default:
      console.log(chalk.red(`Unknown action: ${action}`));
      console.log('Usage: bender config [get|set|validate] [key] [value]');
  }
}

async function openInEditor(): Promise<void> {
  if (!existsSync(CONFIG_PATH)) {
    console.log(chalk.yellow('Config file not found. Creating default...'));
    // Create would happen via install command
  }

  const editor = process.env.EDITOR || 'nano';
  const proc = spawn(editor, [CONFIG_PATH], {
    stdio: 'inherit',
  });

  return new Promise((resolve) => {
    proc.on('close', () => resolve());
  });
}

async function getConfig(key?: string): Promise<void> {
  if (!existsSync(CONFIG_PATH)) {
    console.log(chalk.red('Config file not found'));
    return;
  }

  const content = readFileSync(CONFIG_PATH, 'utf-8');
  const cfg = YAML.parse(content);

  if (!key) {
    console.log(YAML.stringify(cfg));
    return;
  }

  const value = getNestedValue(cfg, key);
  if (value === undefined) {
    console.log(chalk.red(`Key not found: ${key}`));
  } else if (typeof value === 'object') {
    console.log(YAML.stringify(value));
  } else {
    console.log(value);
  }
}

async function setConfig(key?: string, value?: string): Promise<void> {
  if (!key || value === undefined) {
    console.log(chalk.red('Usage: bender config set <key> <value>'));
    return;
  }

  if (!existsSync(CONFIG_PATH)) {
    console.log(chalk.red('Config file not found'));
    return;
  }

  const content = readFileSync(CONFIG_PATH, 'utf-8');
  const cfg = YAML.parse(content);

  // Parse value as JSON if possible, otherwise use as string
  let parsedValue: unknown;
  try {
    parsedValue = JSON.parse(value);
  } catch {
    parsedValue = value;
  }

  setNestedValue(cfg, key, parsedValue);
  writeFileSync(CONFIG_PATH, YAML.stringify(cfg));
  console.log(chalk.green(`Set ${key} = ${value}`));
}

async function validateConfig(): Promise<void> {
  if (!existsSync(CONFIG_PATH)) {
    console.log(chalk.red('Config file not found'));
    return;
  }

  try {
    const content = readFileSync(CONFIG_PATH, 'utf-8');
    YAML.parse(content);
    console.log(chalk.green('Config is valid YAML'));
  } catch (err) {
    console.log(chalk.red('Config validation failed:'));
    console.log(err);
  }
}

function getNestedValue(obj: Record<string, unknown>, path: string): unknown {
  const keys = path.split('.');
  let current: unknown = obj;
  for (const key of keys) {
    if (current === null || typeof current !== 'object') {
      return undefined;
    }
    current = (current as Record<string, unknown>)[key];
  }
  return current;
}

function setNestedValue(
  obj: Record<string, unknown>,
  path: string,
  value: unknown
): void {
  const keys = path.split('.');
  let current = obj;
  for (let i = 0; i < keys.length - 1; i++) {
    const key = keys[i];
    if (!(key in current) || typeof current[key] !== 'object') {
      current[key] = {};
    }
    current = current[key] as Record<string, unknown>;
  }
  current[keys[keys.length - 1]] = value;
}
