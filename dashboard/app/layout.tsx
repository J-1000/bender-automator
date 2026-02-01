import type { Metadata } from 'next';
import './globals.css';

export const metadata: Metadata = {
  title: 'Bender Dashboard',
  description: 'Local AI Workflow Automator',
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body className="bg-gray-50 dark:bg-gray-900 min-h-screen">
        <nav className="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700">
          <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
            <div className="flex justify-between h-16">
              <div className="flex items-center">
                <span className="text-xl font-bold text-gray-900 dark:text-white">
                  Bender
                </span>
              </div>
              <div className="flex items-center space-x-4">
                <a
                  href="/"
                  className="text-gray-600 dark:text-gray-300 hover:text-gray-900 dark:hover:text-white"
                >
                  Home
                </a>
                <a
                  href="/tasks"
                  className="text-gray-600 dark:text-gray-300 hover:text-gray-900 dark:hover:text-white"
                >
                  Tasks
                </a>
                <a
                  href="/config"
                  className="text-gray-600 dark:text-gray-300 hover:text-gray-900 dark:hover:text-white"
                >
                  Config
                </a>
                <a
                  href="/logs"
                  className="text-gray-600 dark:text-gray-300 hover:text-gray-900 dark:hover:text-white"
                >
                  Logs
                </a>
              </div>
            </div>
          </div>
        </nav>
        <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
          {children}
        </main>
      </body>
    </html>
  );
}
