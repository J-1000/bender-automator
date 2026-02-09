import chalk from 'chalk';
import { createInterface } from 'readline';
import { client } from '../lib/client.js';

export async function keychainSet(provider: string): Promise<void> {
  const secret = await readSecret(`Enter API key for ${provider}: `);
  if (!secret) {
    console.log(chalk.red('No key provided'));
    return;
  }

  try {
    const result = await client.call<{ status: string; account: string }>(
      'keychain.set',
      { account: provider, secret }
    );
    console.log(chalk.green(`Stored API key for ${result.account} in macOS Keychain`));
    console.log(
      chalk.dim(
        `Set api_key: "keychain:${provider}" in your config to use it`
      )
    );
  } catch (err) {
    console.log(chalk.red(`Failed to store key: ${(err as Error).message}`));
  }
}

export async function keychainGet(provider: string): Promise<void> {
  try {
    const result = await client.call<{ account: string; preview: string }>(
      'keychain.get',
      { account: provider }
    );
    console.log(`${chalk.bold(result.account)}: ${result.preview}`);
  } catch (err) {
    console.log(chalk.red((err as Error).message));
  }
}

export async function keychainDelete(provider: string): Promise<void> {
  try {
    const result = await client.call<{ status: string; account: string }>(
      'keychain.delete',
      { account: provider }
    );
    console.log(chalk.green(`Deleted API key for ${result.account} from macOS Keychain`));
  } catch (err) {
    console.log(chalk.red((err as Error).message));
  }
}

export async function keychainList(): Promise<void> {
  const providers = ['openai', 'anthropic'];
  console.log(chalk.bold('Keychain entries:'));
  for (const p of providers) {
    try {
      const result = await client.call<{ account: string; preview: string }>(
        'keychain.get',
        { account: p }
      );
      console.log(`  ${chalk.green('+')} ${result.account}: ${result.preview}`);
    } catch {
      console.log(`  ${chalk.dim('-')} ${p}: ${chalk.dim('not set')}`);
    }
  }
}

function readSecret(prompt: string): Promise<string> {
  return new Promise((resolve) => {
    const rl = createInterface({
      input: process.stdin,
      output: process.stdout,
    });
    // Disable echo for secret input
    if (process.stdin.isTTY) {
      process.stdout.write(prompt);
      const stdin = process.stdin;
      stdin.setRawMode(true);
      stdin.resume();
      let secret = '';
      const onData = (ch: Buffer) => {
        const c = ch.toString();
        if (c === '\n' || c === '\r') {
          stdin.setRawMode(false);
          stdin.pause();
          stdin.removeListener('data', onData);
          rl.close();
          process.stdout.write('\n');
          resolve(secret);
        } else if (c === '\u0003') {
          // Ctrl+C
          stdin.setRawMode(false);
          process.exit(0);
        } else if (c === '\u007f') {
          // Backspace
          secret = secret.slice(0, -1);
        } else {
          secret += c;
          process.stdout.write('*');
        }
      };
      stdin.on('data', onData);
    } else {
      rl.question(prompt, (answer) => {
        rl.close();
        resolve(answer);
      });
    }
  });
}
