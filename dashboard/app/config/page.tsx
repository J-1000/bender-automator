'use client';

import { useEffect, useState } from 'react';

interface ProviderConfig {
  enabled: boolean;
  base_url?: string;
  api_key?: string;
  model: string;
  vision_model?: string;
  timeout_seconds: number;
}

interface Config {
  llm: {
    default_provider: string;
    providers: Record<string, ProviderConfig>;
  };
  clipboard: {
    enabled: boolean;
    min_length: number;
    debounce_ms: number;
    auto_summarize: boolean;
    notification: boolean;
  };
  auto_file: {
    enabled: boolean;
    watch_dirs: string[];
    destination_root: string;
    use_llm_classification: boolean;
    auto_move: boolean;
    auto_rename: boolean;
    settle_delay_ms: number;
  };
  git: {
    enabled: boolean;
    commit_format: string;
    include_scope: boolean;
    include_body: boolean;
    max_subject_length: number;
  };
  screenshots: {
    enabled: boolean;
    watch_dir: string;
    destination: string;
    rename: boolean;
    use_vision: boolean;
    vision_provider: string;
    settle_delay_ms: number;
  };
  queue: {
    max_concurrent: number;
    default_timeout_seconds: number;
    max_retries: number;
  };
  notifications: {
    enabled: boolean;
    sound: boolean;
    show_previews: boolean;
  };
}

export default function ConfigPage() {
  const [config, setConfig] = useState<Config | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    fetchConfig();
  }, []);

  async function fetchConfig() {
    try {
      const res = await fetch('/api/config');
      if (res.ok) {
        setConfig(await res.json());
        setError(null);
      } else {
        setError('Cannot load configuration');
      }
    } catch {
      setError('Cannot connect to daemon');
    }
  }

  async function handleSave() {
    if (!config) return;
    setSaving(true);
    setSaved(false);
    try {
      const res = await fetch('/api/config', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(config),
      });
      if (res.ok) {
        setSaved(true);
        setTimeout(() => setSaved(false), 3000);
      }
    } catch {
      setError('Failed to save');
    } finally {
      setSaving(false);
    }
  }

  async function handleReload() {
    try {
      await fetch('/api/config', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ action: 'reload' }),
      });
      await fetchConfig();
    } catch {
      setError('Failed to reload');
    }
  }

  function updateConfig(path: string, value: unknown) {
    if (!config) return;
    const updated = { ...config };
    const keys = path.split('.');
    let obj: Record<string, unknown> = updated as unknown as Record<string, unknown>;
    for (let i = 0; i < keys.length - 1; i++) {
      obj[keys[i]] = { ...(obj[keys[i]] as Record<string, unknown>) };
      obj = obj[keys[i]] as Record<string, unknown>;
    }
    obj[keys[keys.length - 1]] = value;
    setConfig(updated as unknown as Config);
  }

  if (error && !config) {
    return (
      <div className="space-y-4">
        <h1 className="text-3xl font-bold text-gray-900 dark:text-white">Configuration</h1>
        <div className="bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400 p-4 rounded-lg">
          {error}
        </div>
      </div>
    );
  }

  if (!config) {
    return <p className="text-gray-500">Loading configuration...</p>;
  }

  return (
    <div className="space-y-8">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold text-gray-900 dark:text-white">Configuration</h1>
          <p className="mt-2 text-gray-600 dark:text-gray-400">Manage daemon settings</p>
        </div>
        <div className="flex space-x-3">
          <button
            onClick={handleReload}
            className="px-4 py-2 bg-gray-200 dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg hover:bg-gray-300 dark:hover:bg-gray-600 transition"
          >
            Reload
          </button>
          <button
            onClick={handleSave}
            disabled={saving}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition disabled:opacity-50"
          >
            {saving ? 'Saving...' : saved ? 'Saved!' : 'Save'}
          </button>
        </div>
      </div>

      {/* LLM Providers */}
      <Section title="LLM Providers">
        <Field label="Default Provider">
          <select
            value={config.llm.default_provider}
            onChange={(e) => updateConfig('llm.default_provider', e.target.value)}
            className="input"
          >
            <option value="ollama">Ollama</option>
            <option value="openai">OpenAI</option>
            <option value="anthropic">Anthropic</option>
          </select>
        </Field>
        {Object.entries(config.llm.providers).map(([name, provider]) => (
          <div key={name} className="border dark:border-gray-700 rounded-lg p-4 space-y-3">
            <div className="flex items-center justify-between">
              <h4 className="font-medium text-gray-900 dark:text-white capitalize">{name}</h4>
              <Toggle
                checked={provider.enabled}
                onChange={(v) => updateConfig(`llm.providers.${name}.enabled`, v)}
              />
            </div>
            {provider.enabled && (
              <>
                <Field label="Model">
                  <input
                    type="text"
                    value={provider.model}
                    onChange={(e) => updateConfig(`llm.providers.${name}.model`, e.target.value)}
                    className="input"
                  />
                </Field>
                <Field label="Timeout (seconds)">
                  <input
                    type="number"
                    value={provider.timeout_seconds}
                    onChange={(e) => updateConfig(`llm.providers.${name}.timeout_seconds`, parseInt(e.target.value))}
                    className="input"
                  />
                </Field>
              </>
            )}
          </div>
        ))}
      </Section>

      {/* Clipboard */}
      <Section title="Clipboard Monitoring">
        <div className="flex items-center justify-between">
          <span className="text-gray-700 dark:text-gray-300">Enabled</span>
          <Toggle
            checked={config.clipboard.enabled}
            onChange={(v) => updateConfig('clipboard.enabled', v)}
          />
        </div>
        <Field label="Min Length (chars)">
          <input
            type="number"
            value={config.clipboard.min_length}
            onChange={(e) => updateConfig('clipboard.min_length', parseInt(e.target.value))}
            className="input"
          />
        </Field>
        <div className="flex items-center justify-between">
          <span className="text-gray-700 dark:text-gray-300">Auto Summarize</span>
          <Toggle
            checked={config.clipboard.auto_summarize}
            onChange={(v) => updateConfig('clipboard.auto_summarize', v)}
          />
        </div>
        <div className="flex items-center justify-between">
          <span className="text-gray-700 dark:text-gray-300">Notifications</span>
          <Toggle
            checked={config.clipboard.notification}
            onChange={(v) => updateConfig('clipboard.notification', v)}
          />
        </div>
      </Section>

      {/* Auto File */}
      <Section title="Auto File Organization">
        <div className="flex items-center justify-between">
          <span className="text-gray-700 dark:text-gray-300">Enabled</span>
          <Toggle
            checked={config.auto_file.enabled}
            onChange={(v) => updateConfig('auto_file.enabled', v)}
          />
        </div>
        <Field label="Destination Root">
          <input
            type="text"
            value={config.auto_file.destination_root}
            onChange={(e) => updateConfig('auto_file.destination_root', e.target.value)}
            className="input"
          />
        </Field>
        <div className="flex items-center justify-between">
          <span className="text-gray-700 dark:text-gray-300">Use LLM Classification</span>
          <Toggle
            checked={config.auto_file.use_llm_classification}
            onChange={(v) => updateConfig('auto_file.use_llm_classification', v)}
          />
        </div>
        <div className="flex items-center justify-between">
          <span className="text-gray-700 dark:text-gray-300">Auto Move</span>
          <Toggle
            checked={config.auto_file.auto_move}
            onChange={(v) => updateConfig('auto_file.auto_move', v)}
          />
        </div>
        <div className="flex items-center justify-between">
          <span className="text-gray-700 dark:text-gray-300">Auto Rename</span>
          <Toggle
            checked={config.auto_file.auto_rename}
            onChange={(v) => updateConfig('auto_file.auto_rename', v)}
          />
        </div>
        <Field label="Settle Delay (ms)">
          <input
            type="number"
            value={config.auto_file.settle_delay_ms}
            onChange={(e) => updateConfig('auto_file.settle_delay_ms', parseInt(e.target.value))}
            className="input"
          />
        </Field>
      </Section>

      {/* Screenshots */}
      <Section title="Screenshots">
        <div className="flex items-center justify-between">
          <span className="text-gray-700 dark:text-gray-300">Enabled</span>
          <Toggle
            checked={config.screenshots.enabled}
            onChange={(v) => updateConfig('screenshots.enabled', v)}
          />
        </div>
        <Field label="Watch Directory">
          <input
            type="text"
            value={config.screenshots.watch_dir}
            onChange={(e) => updateConfig('screenshots.watch_dir', e.target.value)}
            className="input"
          />
        </Field>
        <Field label="Destination">
          <input
            type="text"
            value={config.screenshots.destination}
            onChange={(e) => updateConfig('screenshots.destination', e.target.value)}
            className="input"
          />
        </Field>
        <div className="flex items-center justify-between">
          <span className="text-gray-700 dark:text-gray-300">Use Vision</span>
          <Toggle
            checked={config.screenshots.use_vision}
            onChange={(v) => updateConfig('screenshots.use_vision', v)}
          />
        </div>
        <div className="flex items-center justify-between">
          <span className="text-gray-700 dark:text-gray-300">Rename</span>
          <Toggle
            checked={config.screenshots.rename}
            onChange={(v) => updateConfig('screenshots.rename', v)}
          />
        </div>
        <Field label="Settle Delay (ms)">
          <input
            type="number"
            value={config.screenshots.settle_delay_ms}
            onChange={(e) => updateConfig('screenshots.settle_delay_ms', parseInt(e.target.value))}
            className="input"
          />
        </Field>
      </Section>

      {/* Git */}
      <Section title="Git Integration">
        <div className="flex items-center justify-between">
          <span className="text-gray-700 dark:text-gray-300">Enabled</span>
          <Toggle
            checked={config.git.enabled}
            onChange={(v) => updateConfig('git.enabled', v)}
          />
        </div>
        <Field label="Commit Format">
          <select
            value={config.git.commit_format}
            onChange={(e) => updateConfig('git.commit_format', e.target.value)}
            className="input"
          >
            <option value="conventional">Conventional</option>
            <option value="simple">Simple</option>
            <option value="detailed">Detailed</option>
          </select>
        </Field>
        <Field label="Max Subject Length">
          <input
            type="number"
            value={config.git.max_subject_length}
            onChange={(e) => updateConfig('git.max_subject_length', parseInt(e.target.value))}
            className="input"
          />
        </Field>
        <div className="flex items-center justify-between">
          <span className="text-gray-700 dark:text-gray-300">Include Scope</span>
          <Toggle
            checked={config.git.include_scope}
            onChange={(v) => updateConfig('git.include_scope', v)}
          />
        </div>
        <div className="flex items-center justify-between">
          <span className="text-gray-700 dark:text-gray-300">Include Body</span>
          <Toggle
            checked={config.git.include_body}
            onChange={(v) => updateConfig('git.include_body', v)}
          />
        </div>
      </Section>

      {/* Queue */}
      <Section title="Task Queue">
        <Field label="Max Concurrent Tasks">
          <input
            type="number"
            value={config.queue.max_concurrent}
            onChange={(e) => updateConfig('queue.max_concurrent', parseInt(e.target.value))}
            className="input"
          />
        </Field>
        <Field label="Timeout (seconds)">
          <input
            type="number"
            value={config.queue.default_timeout_seconds}
            onChange={(e) => updateConfig('queue.default_timeout_seconds', parseInt(e.target.value))}
            className="input"
          />
        </Field>
        <Field label="Max Retries">
          <input
            type="number"
            value={config.queue.max_retries}
            onChange={(e) => updateConfig('queue.max_retries', parseInt(e.target.value))}
            className="input"
          />
        </Field>
      </Section>

      {/* Notifications */}
      <Section title="Notifications">
        <div className="flex items-center justify-between">
          <span className="text-gray-700 dark:text-gray-300">Enabled</span>
          <Toggle
            checked={config.notifications.enabled}
            onChange={(v) => updateConfig('notifications.enabled', v)}
          />
        </div>
        <div className="flex items-center justify-between">
          <span className="text-gray-700 dark:text-gray-300">Sound</span>
          <Toggle
            checked={config.notifications.sound}
            onChange={(v) => updateConfig('notifications.sound', v)}
          />
        </div>
        <div className="flex items-center justify-between">
          <span className="text-gray-700 dark:text-gray-300">Show Previews</span>
          <Toggle
            checked={config.notifications.show_previews}
            onChange={(v) => updateConfig('notifications.show_previews', v)}
          />
        </div>
      </Section>
    </div>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6 space-y-4">
      <h2 className="text-lg font-semibold text-gray-900 dark:text-white">{title}</h2>
      {children}
    </div>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between">
      <label className="text-sm text-gray-700 dark:text-gray-300">{label}</label>
      {children}
    </div>
  );
}

function Toggle({ checked, onChange }: { checked: boolean; onChange: (v: boolean) => void }) {
  return (
    <button
      onClick={() => onChange(!checked)}
      className={`relative w-11 h-6 rounded-full transition-colors ${
        checked ? 'bg-blue-600' : 'bg-gray-300 dark:bg-gray-600'
      }`}
    >
      <span
        className={`absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full transition-transform ${
          checked ? 'translate-x-5' : ''
        }`}
      />
    </button>
  );
}
