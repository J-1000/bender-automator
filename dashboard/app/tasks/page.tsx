'use client';

import { useEffect, useState } from 'react';

interface Task {
  id: string;
  type: string;
  status: string;
  payload?: string;
  result?: string;
  error?: string;
  created_at: string;
  started_at?: string;
  finished_at?: string;
}

export default function TasksPage() {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(true);
  const [selected, setSelected] = useState<Task | null>(null);
  const [undoing, setUndoing] = useState(false);

  useEffect(() => {
    fetchTasks();
    const interval = setInterval(fetchTasks, 5000);
    return () => clearInterval(interval);
  }, []);

  async function fetchTasks() {
    try {
      const res = await fetch('/api/tasks');
      if (res.ok) {
        const data = await res.json();
        setTasks(data);
      }
    } catch (err) {
      console.error('Failed to fetch tasks:', err);
    } finally {
      setLoading(false);
    }
  }

  async function handleUndo(taskId: string) {
    setUndoing(true);
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
      setUndoing(false);
    }
  }

  function getStatusColor(status: string) {
    switch (status) {
      case 'completed':
        return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200';
      case 'running':
        return 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200';
      case 'pending':
        return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200';
      case 'failed':
        return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200';
      default:
        return 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200';
    }
  }

  function formatDate(date: string) {
    return new Date(date).toLocaleString();
  }

  function isFileOperation(type: string) {
    return type === 'file.classify' || type === 'file.rename';
  }

  function formatJSON(str: string | undefined) {
    if (!str) return null;
    try {
      return JSON.stringify(JSON.parse(str), null, 2);
    } catch {
      return str;
    }
  }

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-3xl font-bold text-gray-900 dark:text-white">Tasks</h1>
        <p className="mt-2 text-gray-600 dark:text-gray-400">
          View and manage task queue
        </p>
      </div>

      <div className="flex gap-6">
        {/* Task list */}
        <div className="flex-1 bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden">
          {loading ? (
            <div className="p-8 text-center text-gray-500">Loading...</div>
          ) : tasks.length === 0 ? (
            <div className="p-8 text-center text-gray-500">No tasks yet</div>
          ) : (
            <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
              <thead className="bg-gray-50 dark:bg-gray-900">
                <tr>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">
                    ID
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">
                    Type
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">
                    Status
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">
                    Created
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                {tasks.map((task) => (
                  <tr
                    key={task.id}
                    onClick={() => setSelected(task)}
                    className={`cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-700/50 transition ${
                      selected?.id === task.id ? 'bg-blue-50 dark:bg-blue-900/20' : ''
                    }`}
                  >
                    <td className="px-4 py-3 text-sm font-mono text-gray-900 dark:text-white">
                      {task.id.slice(0, 8)}
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-900 dark:text-white">
                      {task.type}
                    </td>
                    <td className="px-4 py-3">
                      <span
                        className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${getStatusColor(
                          task.status
                        )}`}
                      >
                        {task.status}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">
                      {formatDate(task.created_at)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>

        {/* Task detail panel */}
        {selected && (
          <div className="w-96 bg-white dark:bg-gray-800 rounded-lg shadow p-6 space-y-4 self-start">
            <div className="flex justify-between items-start">
              <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
                Task Detail
              </h3>
              <button
                onClick={() => setSelected(null)}
                className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-200"
              >
                &times;
              </button>
            </div>

            <div className="space-y-3 text-sm">
              <div>
                <span className="text-gray-500 dark:text-gray-400">ID:</span>
                <span className="ml-2 font-mono text-gray-900 dark:text-white">
                  {selected.id}
                </span>
              </div>
              <div>
                <span className="text-gray-500 dark:text-gray-400">Type:</span>
                <span className="ml-2 text-gray-900 dark:text-white">{selected.type}</span>
              </div>
              <div>
                <span className="text-gray-500 dark:text-gray-400">Status:</span>
                <span
                  className={`ml-2 px-2 text-xs font-semibold rounded-full ${getStatusColor(
                    selected.status
                  )}`}
                >
                  {selected.status}
                </span>
              </div>
              <div>
                <span className="text-gray-500 dark:text-gray-400">Created:</span>
                <span className="ml-2 text-gray-900 dark:text-white">
                  {formatDate(selected.created_at)}
                </span>
              </div>
              {selected.finished_at && (
                <div>
                  <span className="text-gray-500 dark:text-gray-400">Finished:</span>
                  <span className="ml-2 text-gray-900 dark:text-white">
                    {formatDate(selected.finished_at)}
                  </span>
                </div>
              )}

              {selected.payload && (
                <div>
                  <p className="text-gray-500 dark:text-gray-400 mb-1">Payload:</p>
                  <pre className="bg-gray-100 dark:bg-gray-900 rounded p-2 text-xs overflow-auto max-h-40">
                    {formatJSON(selected.payload)}
                  </pre>
                </div>
              )}

              {selected.result && (
                <div>
                  <p className="text-gray-500 dark:text-gray-400 mb-1">Result:</p>
                  <pre className="bg-gray-100 dark:bg-gray-900 rounded p-2 text-xs overflow-auto max-h-40">
                    {formatJSON(selected.result)}
                  </pre>
                </div>
              )}

              {selected.error && (
                <div>
                  <p className="text-gray-500 dark:text-gray-400 mb-1">Error:</p>
                  <p className="text-red-600 dark:text-red-400 text-xs">{selected.error}</p>
                </div>
              )}

              {isFileOperation(selected.type) && selected.status === 'completed' && (
                <button
                  onClick={() => handleUndo(selected.id)}
                  disabled={undoing}
                  className="w-full mt-2 px-4 py-2 bg-yellow-500 text-white rounded-lg hover:bg-yellow-600 transition disabled:opacity-50 text-sm"
                >
                  {undoing ? 'Undoing...' : 'Undo Operation'}
                </button>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
