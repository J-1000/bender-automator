interface StatusBadgeProps {
  status: 'running' | 'stopped' | 'error';
  label?: string;
}

export function StatusBadge({ status, label }: StatusBadgeProps) {
  const colors = {
    running: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200',
    stopped: 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200',
    error: 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200',
  };

  const icons = {
    running: '●',
    stopped: '○',
    error: '!',
  };

  return (
    <span
      className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${colors[status]}`}
    >
      <span className="mr-1">{icons[status]}</span>
      {label || status}
    </span>
  );
}
