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
      <div className="bg-white rounded-lg shadow-md p-6 hover:shadow-lg transition-shadow cursor-pointer">
        <div className="flex items-start justify-between mb-4">
          <div>
            <h3 className="text-lg font-semibold text-gray-900 mb-2">
              Deployment #{deployment.id}
            </h3>
            <p className="text-sm text-gray-600">
              Created: {formatDate(deployment.created_at)}
            </p>
          </div>
          <StatusBadge status={deployment.status} />
        </div>
        {subdomain && (
          <p className="text-sm text-gray-600 mb-2">
            <span className="font-medium">Subdomain:</span> {subdomain}
          </p>
        )}
        {errorMessage && (
          <div className="mt-3 p-3 bg-red-50 border border-red-200 rounded">
            <p className="text-sm text-red-800">{errorMessage}</p>
          </div>
        )}
      </div>
    </Link>
  );
}

