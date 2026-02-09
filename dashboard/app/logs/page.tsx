'use client';

import { useEffect, useState, useCallback } from 'react';

interface LogEntry {
  time: string;
  level: string;
  message: string;
}

const LEVELS = ['', 'DEBUG', 'INFO', 'WARN', 'ERROR'] as const;

const LEVEL_COLORS: Record<string, string> = {
  DEBUG: 'text-gray-500',
  INFO: 'text-blue-600 dark:text-blue-400',
  WARN: 'text-yellow-600 dark:text-yellow-400',
  ERROR: 'text-red-600 dark:text-red-400',
};

export default function LogsPage() {
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [level, setLevel] = useState('');
  const [search, setSearch] = useState('');
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchLogs = useCallback(async () => {
    try {
      const params = new URLSearchParams({ limit: '200' });
      if (level) params.set('level', level);
      const res = await fetch(`/api/logs?${params}`);
      if (res.ok) {
        setLogs(await res.json());
        setError(null);
      } else {
        setError('Cannot fetch logs');
      }
    } catch {
      setError('Cannot connect to daemon');
    }
  }, [level]);

  useEffect(() => {
    fetchLogs();
    if (!autoRefresh) return;
    const interval = setInterval(fetchLogs, 3000);
    return () => clearInterval(interval);
  }, [fetchLogs, autoRefresh]);

  const filtered = search
    ? logs.filter((l) => l.message.toLowerCase().includes(search.toLowerCase()))
    : logs;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold text-gray-900 dark:text-white">Logs</h1>
        <p className="mt-2 text-gray-600 dark:text-gray-400">
          Real-time daemon log viewer
        </p>
      </div>

      {/* Controls */}
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-4 flex flex-wrap items-center gap-4">
        <div className="flex items-center gap-2">
          <label className="text-sm text-gray-700 dark:text-gray-300">Level:</label>
          <select
            value={level}
            onChange={(e) => setLevel(e.target.value)}
            className="input"
          >
            {LEVELS.map((l) => (
              <option key={l} value={l}>
                {l || 'All'}
              </option>
            ))}
          </select>
        </div>

        <div className="flex-1">
          <input
            type="text"
            placeholder="Search logs..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="input w-full"
          />
        </div>

        <button
          onClick={() => setAutoRefresh(!autoRefresh)}
          className={`px-3 py-1.5 rounded-lg text-sm transition ${
            autoRefresh
              ? 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400'
              : 'bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300'
          }`}
        >
          {autoRefresh ? 'Auto-refresh ON' : 'Auto-refresh OFF'}
        </button>

        <button
          onClick={fetchLogs}
          className="px-3 py-1.5 bg-blue-600 text-white rounded-lg text-sm hover:bg-blue-700 transition"
        >
          Refresh
        </button>
      </div>

      {/* Log entries */}
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden">
        {error ? (
          <div className="p-6 text-red-600 dark:text-red-400">{error}</div>
        ) : filtered.length === 0 ? (
          <div className="p-6 text-gray-500">No log entries</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b dark:border-gray-700 bg-gray-50 dark:bg-gray-900">
                  <th className="text-left px-4 py-3 text-gray-500 dark:text-gray-400 font-medium w-48">
                    Time
                  </th>
                  <th className="text-left px-4 py-3 text-gray-500 dark:text-gray-400 font-medium w-20">
                    Level
                  </th>
                  <th className="text-left px-4 py-3 text-gray-500 dark:text-gray-400 font-medium">
                    Message
                  </th>
                </tr>
              </thead>
              <tbody className="font-mono text-xs">
                {filtered.map((entry, i) => (
                  <tr
                    key={i}
                    className="border-b dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-900/50"
                  >
                    <td className="px-4 py-2 text-gray-500 dark:text-gray-400 whitespace-nowrap">
                      {new Date(entry.time).toLocaleString()}
                    </td>
                    <td className={`px-4 py-2 font-semibold ${LEVEL_COLORS[entry.level] || ''}`}>
                      {entry.level}
                    </td>
                    <td className="px-4 py-2 text-gray-900 dark:text-gray-100 break-all">
                      {entry.message}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      <p className="text-sm text-gray-500 dark:text-gray-400">
        Showing {filtered.length} of {logs.length} entries
      </p>
    </div>
  );
}
