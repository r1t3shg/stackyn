interface LogsViewerProps {
  logs: string | null | undefined;
  title?: string;
}

export default function LogsViewer({ logs, title = 'Logs' }: LogsViewerProps) {
  if (!logs) {
    return (
      <div className="bg-[var(--surface)] border border-[var(--border-subtle)] rounded-lg p-4">
        <p className="text-sm text-[var(--text-muted)]">No logs available</p>
      </div>
    );
  }

  return (
    <div className="bg-[var(--terminal-bg)] text-[var(--text-primary)] rounded-lg p-4 font-mono text-sm overflow-x-auto border border-[var(--border-subtle)]">
      <div className="mb-2 text-[var(--text-muted)] text-xs font-semibold uppercase">{title}</div>
      <pre className="whitespace-pre-wrap break-words">{logs}</pre>
    </div>
  );
}


