import { stop } from './stop.js';
import { start } from './start.js';

export async function restart(): Promise<void> {
  await stop();
  await start();
}
