'use client';

import { useEffect, useState } from 'react';

interface DaemonStatus {
  running: boolean;
  version: string;
  uptime: string;
  pid: number;
}

interface Stats {
  tasksToday: number;
  filesOrganized: number;
  commitsGenerated: number;
}

export default function Home() {
  const [status, setStatus] = useState<DaemonStatus | null>(null);
  const [stats, setStats] = useState<Stats>({
    tasksToday: 0,
    filesOrganized: 0,
    commitsGenerated: 0,
  });
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchStatus();
    const interval = setInterval(fetchStatus, 5000);
    return () => clearInterval(interval);
  }, []);

  async function fetchStatus() {
    try {
      const res = await fetch('/api/daemon/status');
      if (res.ok) {
        const data = await res.json();
        setStatus(data);
        setError(null);
      } else {
        setStatus(null);
        setError('Daemon not running');
      }
    } catch {
      setStatus(null);
      setError('Cannot connect to dashboard API');
    }
  }

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-3xl font-bold text-gray-900 dark:text-white">
          Dashboard
        </h1>
        <p className="mt-2 text-gray-600 dark:text-gray-400">
          Monitor and control your Bender daemon
        </p>
      </div>

      {/* Status Card */}
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
        <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
          Daemon Status
        </h2>
        {error ? (
          <div className="flex items-center space-x-2">
            <span className="h-3 w-3 rounded-full bg-red-500"></span>
            <span className="text-red-600 dark:text-red-400">{error}</span>
          </div>
        ) : status ? (
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div>
              <p className="text-sm text-gray-500 dark:text-gray-400">Status</p>
              <div className="flex items-center space-x-2 mt-1">
                <span className="h-3 w-3 rounded-full bg-green-500"></span>
                <span className="font-medium text-gray-900 dark:text-white">
                  Running
                </span>
              </div>
            </div>
            <div>
              <p className="text-sm text-gray-500 dark:text-gray-400">Version</p>
              <p className="font-medium text-gray-900 dark:text-white mt-1">
                {status.version}
              </p>
            </div>
            <div>
              <p className="text-sm text-gray-500 dark:text-gray-400">Uptime</p>
              <p className="font-medium text-gray-900 dark:text-white mt-1">
                {status.uptime}
              </p>
            </div>
            <div>
              <p className="text-sm text-gray-500 dark:text-gray-400">PID</p>
              <p className="font-medium text-gray-900 dark:text-white mt-1">
                {status.pid}
              </p>
            </div>
          </div>
        ) : (
          <p className="text-gray-500">Loading...</p>
        )}
      </div>

      {/* Quick Stats */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
          <p className="text-sm text-gray-500 dark:text-gray-400">Tasks Today</p>
          <p className="text-3xl font-bold text-gray-900 dark:text-white mt-2">
            {stats.tasksToday}
          </p>
        </div>
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
          <p className="text-sm text-gray-500 dark:text-gray-400">
            Files Organized
          </p>
          <p className="text-3xl font-bold text-gray-900 dark:text-white mt-2">
            {stats.filesOrganized}
          </p>
        </div>
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
          <p className="text-sm text-gray-500 dark:text-gray-400">
            Commits Generated
          </p>
          <p className="text-3xl font-bold text-gray-900 dark:text-white mt-2">
            {stats.commitsGenerated}
          </p>
        </div>
      </div>

      {/* Quick Actions */}
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
        <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
          Quick Actions
        </h2>
        <div className="flex flex-wrap gap-4">
          <button className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition">
            Summarize Clipboard
          </button>
          <button className="px-4 py-2 bg-gray-200 dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg hover:bg-gray-300 dark:hover:bg-gray-600 transition">
            Generate Commit
          </button>
          <button className="px-4 py-2 bg-gray-200 dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg hover:bg-gray-300 dark:hover:bg-gray-600 transition">
            View Logs
          </button>
        </div>
      </div>
    </div>
  );
}
