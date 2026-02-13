'use client';

import { useEffect, useState } from 'react';

interface PipelineStatus {
  auto_file: {
    enabled: boolean;
    auto_move: boolean;
    auto_rename: boolean;
    settle_delay_ms: number;
    watch_dirs: string[];
  };
  screenshot: {
    enabled: boolean;
    use_vision: boolean;
    rename: boolean;
    settle_delay_ms: number;
    watch_dir: string;
    destination: string;
  };
}

interface Task {
  id: string;
  type: string;
  status: string;
  payload?: string;
  result?: string;
  error?: string;
  created_at: string;
  finished_at?: string;
}

export default function PipelinesPage() {
  const [status, setStatus] = useState<PipelineStatus | null>(null);
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(true);
  const [undoing, setUndoing] = useState<string | null>(null);

  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchData, 5000);
    return () => clearInterval(interval);
  }, []);

  async function fetchData() {
    try {
      const [statusRes, tasksRes] = await Promise.all([
        fetch('/api/pipelines'),
        fetch('/api/tasks'),
      ]);
      if (statusRes.ok) setStatus(await statusRes.json());
      if (tasksRes.ok) {
        const allTasks: Task[] = await tasksRes.json();
        setTasks(allTasks.filter((t) => t.type.startsWith('pipeline.')));
      }
    } catch (err) {
      console.error('Failed to fetch pipeline data:', err);
    } finally {
      setLoading(false);
    }
  }

  async function handleUndo(taskId: string) {
    setUndoing(taskId);
    try {
      const res = await fetch('/api/tasks', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ action: 'undo', task_id: taskId }),
      });
      if (res.ok) {
        const data = await res.json();
        alert(`Undid ${data.undone} operation(s)`);
      }
    } catch {
      alert('Undo failed');
    } finally {
      setUndoing(null);
    }
  }

  function getStatusColor(s: string) {
    switch (s) {
      case 'completed': return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200';
      case 'running': return 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200';
      case 'pending': return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200';
      case 'failed': return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200';
      default: return 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200';
    }
  }

  if (loading) {
    return <p className="text-gray-500">Loading pipelines...</p>;
  }

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-3xl font-bold text-gray-900 dark:text-white">Pipelines</h1>
        <p className="mt-2 text-gray-600 dark:text-gray-400">
          Automated file processing workflows
        </p>
      </div>

      {/* Status cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Auto-file card */}
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6 space-y-3">
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white">Auto-File</h2>
            <span className={`px-2 py-0.5 text-xs font-semibold rounded-full ${
              status?.auto_file.enabled
                ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
                : 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400'
            }`}>
              {status?.auto_file.enabled ? 'Active' : 'Inactive'}
            </span>
          </div>
          {status && (
            <div className="text-sm space-y-1 text-gray-600 dark:text-gray-400">
              <p>Auto Move: {status.auto_file.auto_move ? 'Yes' : 'No'}</p>
              <p>Auto Rename: {status.auto_file.auto_rename ? 'Yes' : 'No'}</p>
              <p>Settle Delay: {status.auto_file.settle_delay_ms}ms</p>
              <p>Watch: {status.auto_file.watch_dirs.join(', ')}</p>
            </div>
          )}
        </div>

        {/* Screenshot card */}
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6 space-y-3">
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white">Screenshot</h2>
            <span className={`px-2 py-0.5 text-xs font-semibold rounded-full ${
              status?.screenshot.enabled
                ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
                : 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400'
            }`}>
              {status?.screenshot.enabled ? 'Active' : 'Inactive'}
            </span>
          </div>
          {status && (
            <div className="text-sm space-y-1 text-gray-600 dark:text-gray-400">
              <p>Vision: {status.screenshot.use_vision ? 'Yes' : 'No'}</p>
              <p>Rename: {status.screenshot.rename ? 'Yes' : 'No'}</p>
              <p>Settle Delay: {status.screenshot.settle_delay_ms}ms</p>
              <p>Watch: {status.screenshot.watch_dir}</p>
              <p>Destination: {status.screenshot.destination}</p>
            </div>
          )}
        </div>
      </div>

      {/* Recent pipeline activity */}
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-white">Recent Activity</h2>
        </div>
        {tasks.length === 0 ? (
          <div className="p-8 text-center text-gray-500">No pipeline tasks yet</div>
        ) : (
          <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
            <thead className="bg-gray-50 dark:bg-gray-900">
              <tr>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">ID</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Type</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Status</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Created</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
              {tasks.map((t) => (
                <tr key={t.id}>
                  <td className="px-4 py-3 text-sm font-mono text-gray-900 dark:text-white">{t.id.slice(0, 8)}</td>
                  <td className="px-4 py-3 text-sm text-gray-900 dark:text-white">{t.type}</td>
                  <td className="px-4 py-3">
                    <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${getStatusColor(t.status)}`}>
                      {t.status}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">
                    {new Date(t.created_at).toLocaleString()}
                  </td>
                  <td className="px-4 py-3">
                    {t.status === 'completed' && (
                      <button
                        onClick={() => handleUndo(t.id)}
                        disabled={undoing === t.id}
                        className="text-sm px-3 py-1 bg-yellow-500 text-white rounded hover:bg-yellow-600 transition disabled:opacity-50"
                      >
                        {undoing === t.id ? 'Undoing...' : 'Undo'}
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
