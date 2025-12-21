import { Link } from 'react-router-dom';
import type { App } from '@/lib/types';

interface AppCardProps {
  app: App;
}

export default function AppCard({ app }: AppCardProps) {
  const getStatusColor = (status: string) => {
    switch (status.toLowerCase()) {
      case 'healthy':
      case 'running':
        return 'bg-green-100 text-green-800';
      case 'pending':
      case 'building':
        return 'bg-yellow-100 text-yellow-800';
      case 'failed':
        return 'bg-red-100 text-red-800';
      default:
        return 'bg-gray-100 text-gray-800';
    }
  };

  return (
    <Link to={`/apps/${app.id}`}>
      <div className="bg-white rounded-lg shadow-md p-6 hover:shadow-lg transition-shadow cursor-pointer">
        <div className="flex items-start justify-between">
          <div className="flex-1">
            <h3 className="text-xl font-semibold text-gray-900 mb-2">{app.name}</h3>
            <p className="text-sm text-gray-600 mb-3">
              <span className="font-medium">Repository:</span> {app.repo_url}
            </p>
            <p className="text-sm text-gray-600 mb-3">
              <span className="font-medium">Branch:</span> {app.branch}
            </p>
            {app.url && (
              <a
                href={app.url}
                target="_blank"
                rel="noopener noreferrer"
                onClick={(e) => e.stopPropagation()}
                className="text-blue-600 hover:text-blue-800 text-sm"
              >
                {app.url} →
              </a>
            )}
          </div>
          <span className={`px-3 py-1 rounded-full text-xs font-medium ${getStatusColor(app.status)}`}>
            {app.status}
          </span>
        </div>
        {app.deployment && (
          <div className="mt-4 pt-4 border-t border-gray-200">
            <p className="text-xs text-gray-500">
              Deployment: <span className="font-medium">{app.deployment.state}</span>
            </p>
          </div>
        )}
      </div>
    </Link>
  );
}

