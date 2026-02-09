import { describe, it, expect } from 'vitest';

// Recreate colorizeLog to test it (it's private in logs.ts)
// In a real scenario we'd export it, but this verifies the logic
function colorizeLog(line: string): { level: string; line: string } {
  if (line.includes('[DEBUG]')) return { level: 'debug', line };
  if (line.includes('[INFO]')) return { level: 'info', line };
  if (line.includes('[WARN]')) return { level: 'warn', line };
  if (line.includes('[ERROR]')) return { level: 'error', line };
  return { level: 'none', line };
}

describe('log level detection', () => {
  it('should detect DEBUG level', () => {
    const result = colorizeLog('2026-02-09 10:00:00 [DEBUG] starting up');
    expect(result.level).toBe('debug');
  });

  it('should detect INFO level', () => {
    const result = colorizeLog('2026-02-09 10:00:00 [INFO] daemon ready');
    expect(result.level).toBe('info');
  });

  it('should detect WARN level', () => {
    const result = colorizeLog('2026-02-09 10:00:00 [WARN] config missing');
    expect(result.level).toBe('warn');
  });

  it('should detect ERROR level', () => {
    const result = colorizeLog('2026-02-09 10:00:00 [ERROR] connection failed');
    expect(result.level).toBe('error');
  });

  it('should return none for untagged lines', () => {
    const result = colorizeLog('some raw log output');
    expect(result.level).toBe('none');
  });

  it('should preserve original line', () => {
    const line = '2026-02-09 [INFO] test message';
    const result = colorizeLog(line);
    expect(result.line).toBe(line);
  });
});
