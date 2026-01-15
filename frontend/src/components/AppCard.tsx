import { Link } from 'react-router-dom';
import type { App } from '@/lib/types';
import StatusBadge from './StatusBadge';

interface AppCardProps {
  app: App;
}

export default function AppCard({ app }: AppCardProps) {
  return (
    <Link to={`/apps/${app.id}`}>
      <div className="bg-[var(--surface)] rounded-lg border border-[var(--border-subtle)] p-6 hover:border-[var(--border-strong)] transition-colors cursor-pointer">
        <div className="flex items-start justify-between">
          <div className="flex-1">
            <h3 className="text-xl font-semibold text-[var(--text-primary)] mb-2">{app.name}</h3>
            <p className="text-sm text-[var(--text-secondary)] mb-3">
              <span className="font-medium">Repository:</span> {app.repo_url}
            </p>
            <p className="text-sm text-[var(--text-secondary)] mb-3">
              <span className="font-medium">Branch:</span> {app.branch}
            </p>
            {app.url && (
              <a
                href={app.url}
                target="_blank"
                rel="noopener noreferrer"
                onClick={(e) => e.stopPropagation()}
                className="text-[var(--info)] hover:text-[var(--primary)] text-sm transition-colors"
              >
                {app.url} →
              </a>
            )}
          </div>
          <StatusBadge status={app.status} />
        </div>
        {app.deployment && (
          <div className="mt-4 pt-4 border-t border-[var(--border-subtle)]">
            <p className="text-xs text-[var(--text-muted)]">
              Deployment: <span className="font-medium text-[var(--text-secondary)]">{app.deployment.state}</span>
            </p>
          </div>
        )}
      </div>
    </Link>
  );
}


