interface StatusBadgeProps {
  status: string;
}

export default function StatusBadge({ status }: StatusBadgeProps) {
  const getStatusColor = (status: string) => {
    switch (status.toLowerCase()) {
      case 'healthy':
      case 'running':
        return 'bg-[var(--primary-muted)] text-[var(--success)] border-[var(--success)]';
      case 'pending':
      case 'building':
      case 'deploying':
        return 'bg-[var(--warning)]/10 text-[var(--warning)] border-[var(--warning)]';
      case 'failed':
        return 'bg-[var(--error)]/10 text-[var(--error)] border-[var(--error)]';
      case 'stopped':
      case 'expired':
        return 'bg-[var(--surface)] text-[var(--text-muted)] border-[var(--border-subtle)]';
      default:
        return 'bg-[var(--surface)] text-[var(--text-secondary)] border-[var(--border-subtle)]';
    }
  };

  // Show "expired" for stopped deployments
  const displayStatus = status.toLowerCase() === 'stopped' ? 'expired' : status;

  return (
    <span className={`px-3 py-1 rounded-full text-sm font-medium border ${getStatusColor(status)}`}>
      {displayStatus}
    </span>
  );
}


