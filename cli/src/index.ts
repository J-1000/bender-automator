#!/usr/bin/env node

import { Command } from 'commander';

const program = new Command();

program
  .name('bender')
  .description('Local AI Workflow Automator for macOS')
  .version('0.1.0');

// Daemon management
program
  .command('start')
  .description('Start the Bender daemon')
  .action(async () => {
    const { start } = await import('./commands/start.js');
    await start();
  });

program
  .command('stop')
  .description('Stop the Bender daemon')
  .action(async () => {
    const { stop } = await import('./commands/stop.js');
    await stop();
  });

program
  .command('restart')
  .description('Restart the Bender daemon')
  .action(async () => {
    const { restart } = await import('./commands/restart.js');
    await restart();
  });

program
  .command('status')
  .description('Show daemon status')
  .action(async () => {
    const { status } = await import('./commands/status.js');
    await status();
  });

// Configuration
program
  .command('config')
  .description('Manage configuration')
  .argument('[action]', 'get, set, or validate')
  .argument('[key]', 'configuration key')
  .argument('[value]', 'configuration value')
  .action(async (action, key, value) => {
    const { config } = await import('./commands/config.js');
    await config(action, key, value);
  });

// Ad-hoc tasks
program
  .command('summarize')
  .description('Summarize clipboard or provided text')
  .argument('[text]', 'text to summarize')
  .action(async (text) => {
    const { summarize } = await import('./commands/summarize.js');
    await summarize(text);
  });

program
  .command('classify')
  .description('Classify a file and suggest location')
  .argument('<file>', 'file to classify')
  .action(async (file) => {
    const { classify } = await import('./commands/classify.js');
    await classify(file);
  });

program
  .command('rename')
  .description('Generate intelligent names for files')
  .argument('<files...>', 'files to rename')
  .action(async (files) => {
    const { rename } = await import('./commands/rename.js');
    await rename(files);
  });

program
  .command('commit')
  .description('Generate git commit message')
  .option('--auto', 'automatically commit with generated message')
  .action(async (options) => {
    const { commit } = await import('./commands/commit.js');
    await commit(options);
  });

program
  .command('screenshot')
  .description('Tag a screenshot with AI-detected metadata')
  .argument('<file>', 'screenshot image to tag')
  .action(async (file) => {
    const { screenshot } = await import('./commands/screenshot.js');
    await screenshot(file);
  });

// Installation
const install = program.command('install').description('Install components');

install
  .command('hooks')
  .description('Install git hooks in current repo')
  .action(async () => {
    const { installHooks } = await import('./commands/install.js');
    await installHooks();
  });

install
  .command('agent')
  .description('Install LaunchAgent')
  .action(async () => {
    const { installAgent } = await import('./commands/install.js');
    await installAgent();
  });

// Uninstall
const uninstall = program.command('uninstall').description('Uninstall components');

uninstall
  .command('hooks')
  .description('Remove git hooks')
  .action(async () => {
    const { uninstallHooks } = await import('./commands/install.js');
    await uninstallHooks();
  });

uninstall
  .command('agent')
  .description('Remove LaunchAgent')
  .action(async () => {
    const { uninstallAgent } = await import('./commands/install.js');
    await uninstallAgent();
  });

// Keychain
const kc = program.command('keychain').description('Manage API keys in macOS Keychain');

kc
  .command('set')
  .description('Store an API key')
  .argument('<provider>', 'provider name (e.g. openai, anthropic)')
  .action(async (provider) => {
    const { keychainSet } = await import('./commands/keychain.js');
    await keychainSet(provider);
  });

kc
  .command('get')
  .description('Show stored API key (masked)')
  .argument('<provider>', 'provider name')
  .action(async (provider) => {
    const { keychainGet } = await import('./commands/keychain.js');
    await keychainGet(provider);
  });

kc
  .command('delete')
  .description('Remove a stored API key')
  .argument('<provider>', 'provider name')
  .action(async (provider) => {
    const { keychainDelete } = await import('./commands/keychain.js');
    await keychainDelete(provider);
  });

kc
  .command('list')
  .description('List stored API keys')
  .action(async () => {
    const { keychainList } = await import('./commands/keychain.js');
    await keychainList();
  });

// Logs
program
  .command('logs')
  .description('View daemon logs')
  .option('-f, --follow', 'follow log output')
  .option('--level <level>', 'filter by log level')
  .action(async (options) => {
    const { logs } = await import('./commands/logs.js');
    await logs(options);
  });

program.parse();
