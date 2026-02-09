import { describe, it, expect, vi, beforeEach } from 'vitest';
import { existsSync, readFileSync, writeFileSync } from 'fs';
import { tmpdir } from 'os';
import { join } from 'path';
import { mkdtempSync, rmSync } from 'fs';

// Test the nested value helpers by recreating them
// (they're private in config.ts but the logic is worth testing)

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

describe('config nested value helpers', () => {
  it('should get top-level values', () => {
    const obj = { llm: { default_provider: 'ollama' } };
    expect(getNestedValue(obj, 'llm')).toEqual({ default_provider: 'ollama' });
  });

  it('should get deeply nested values', () => {
    const obj = { llm: { providers: { openai: { enabled: true } } } };
    expect(getNestedValue(obj, 'llm.providers.openai.enabled')).toBe(true);
  });

  it('should return undefined for missing keys', () => {
    const obj = { llm: {} };
    expect(getNestedValue(obj, 'llm.providers.openai')).toBeUndefined();
  });

  it('should return undefined for null intermediate', () => {
    const obj = { llm: null };
    expect(getNestedValue(obj, 'llm.providers')).toBeUndefined();
  });

  it('should set top-level values', () => {
    const obj: Record<string, unknown> = {};
    setNestedValue(obj, 'foo', 'bar');
    expect(obj.foo).toBe('bar');
  });

  it('should set deeply nested values, creating intermediates', () => {
    const obj: Record<string, unknown> = {};
    setNestedValue(obj, 'llm.providers.openai.enabled', true);
    expect((obj as any).llm.providers.openai.enabled).toBe(true);
  });

  it('should overwrite existing values', () => {
    const obj: Record<string, unknown> = { llm: { default_provider: 'ollama' } };
    setNestedValue(obj, 'llm.default_provider', 'openai');
    expect((obj as any).llm.default_provider).toBe('openai');
  });

  it('should handle numeric string values', () => {
    const obj: Record<string, unknown> = {};
    setNestedValue(obj, 'queue.max_concurrent', 4);
    expect((obj as any).queue.max_concurrent).toBe(4);
  });
});
