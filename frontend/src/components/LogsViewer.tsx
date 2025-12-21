interface LogsViewerProps {
  logs: string | null | undefined;
  title?: string;
}

export default function LogsViewer({ logs, title = 'Logs' }: LogsViewerProps) {
  if (!logs) {
    return (
      <div className="bg-gray-50 border border-gray-200 rounded-lg p-4">
        <p className="text-sm text-gray-500">No logs available</p>
      </div>
    );
  }

  return (
    <div className="bg-gray-900 text-gray-100 rounded-lg p-4 font-mono text-sm overflow-x-auto">
      <div className="mb-2 text-gray-400 text-xs font-semibold uppercase">{title}</div>
      <pre className="whitespace-pre-wrap break-words">{logs}</pre>
    </div>
  );
}

