import { Link } from 'react-router-dom';
import type { Deployment } from '@/lib/types';
import { extractString } from '@/lib/types';
import StatusBadge from './StatusBadge';

interface DeploymentCardProps {
  deployment: Deployment;
  appId: string | number;
}

export default function DeploymentCard({ deployment, appId }: DeploymentCardProps) {
  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString();
  };

  const subdomain = extractString(deployment.subdomain);
  const errorMessage = extractString(deployment.error_message);

  return (
    <Link to={`/apps/${appId}/deployments/${deployment.id}`}>
      <div className="bg-[var(--surface)] rounded-lg border border-[var(--border-subtle)] p-6 hover:border-[var(--border-strong)] transition-colors cursor-pointer">
        <div className="flex items-start justify-between mb-4">
          <div>
            <h3 className="text-lg font-semibold text-[var(--text-primary)] mb-2">
              Deployment #{deployment.id}
            </h3>
            <p className="text-sm text-[var(--text-secondary)]">
              Created: {formatDate(deployment.created_at)}
            </p>
          </div>
          <StatusBadge status={deployment.status} />
        </div>
        {subdomain && (
          <p className="text-sm text-[var(--text-secondary)] mb-2">
            <span className="font-medium">Subdomain:</span> {subdomain}
          </p>
        )}
        {errorMessage && deployment.status !== 'stopped' && (
          <div className="mt-3 p-3 bg-[var(--error)]/10 border border-[var(--error)] rounded">
            <p className="text-sm text-[var(--error)]">{errorMessage}</p>
          </div>
        )}
      </div>
    </Link>
  );
}


