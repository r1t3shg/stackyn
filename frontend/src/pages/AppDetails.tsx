import { useEffect, useState } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { appsApi } from '@/lib/api';
import type { App, Deployment } from '@/lib/types';
import StatusBadge from '@/components/StatusBadge';
import DeploymentCard from '@/components/DeploymentCard';

export default function AppDetailsPage() {
  const params = useParams<{ id: string }>();
  const navigate = useNavigate();
  const appId = params.id!;

  const [app, setApp] = useState<App | null>(null);
  const [deployments, setDeployments] = useState<Deployment[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [redeploying, setRedeploying] = useState(false);

  useEffect(() => {
    if (appId) {
      loadApp();
      loadDeployments();
    }
  }, [appId]);

  const loadApp = async () => {
    try {
      setError(null);
      const data = await appsApi.getById(appId);
      setApp(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load app');
      console.error('Error loading app:', err);
    } finally {
      setLoading(false);
    }
  };

  const loadDeployments = async () => {
    try {
      const data = await appsApi.getDeployments(appId);
      setDeployments(data);
    } catch (err) {
      console.error('Error loading deployments:', err);
    }
  };

  const handleRedeploy = async () => {
    if (!confirm('Are you sure you want to redeploy this app?')) {
      return;
    }

    setRedeploying(true);
    try {
      await appsApi.redeploy(appId);
      // Reload app and deployments after redeploy
      await loadApp();
      await loadDeployments();
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to redeploy app');
      console.error('Error redeploying app:', err);
    } finally {
      setRedeploying(false);
    }
  };

  const handleDelete = async () => {
    if (!confirm('Are you sure you want to delete this app? This action cannot be undone.')) {
      return;
    }

    try {
      await appsApi.delete(appId);
      navigate('/');
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to delete app');
      console.error('Error deleting app:', err);
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-center">
          <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
          <p className="mt-4 text-gray-600">Loading app...</p>
        </div>
      </div>
    );
  }

  if (error || !app) {
    return (
      <div className="min-h-screen bg-gray-50">
        <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
          <Link to="/" className="text-blue-600 hover:text-blue-800 mb-6 inline-block">
            ← Back to Apps
          </Link>
          <div className="bg-red-50 border border-red-200 rounded-lg p-6">
            <p className="text-red-800">{error || 'App not found'}</p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <Link to="/" className="text-blue-600 hover:text-blue-800 mb-6 inline-block">
          ← Back to Apps
        </Link>

        <div className="bg-white rounded-lg shadow-md p-8 mb-6">
          <div className="flex items-start justify-between mb-6">
            <div>
              <h1 className="text-3xl font-bold text-gray-900 mb-2">{app.name}</h1>
              <StatusBadge status={app.status} />
            </div>
            <div className="flex space-x-3">
              <button
                onClick={handleRedeploy}
                disabled={redeploying}
                className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white font-medium rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {redeploying ? 'Redeploying...' : 'Redeploy'}
              </button>
              <button
                onClick={handleDelete}
                className="px-4 py-2 bg-red-600 hover:bg-red-700 text-white font-medium rounded-lg transition-colors"
              >
                Delete
              </button>
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
            <div>
              <h3 className="text-sm font-medium text-gray-500 mb-1">Repository URL</h3>
              <p className="text-gray-900">{app.repo_url}</p>
            </div>
            <div>
              <h3 className="text-sm font-medium text-gray-500 mb-1">Branch</h3>
              <p className="text-gray-900">{app.branch}</p>
            </div>
            <div>
              <h3 className="text-sm font-medium text-gray-500 mb-1">App URL</h3>
              {app.url ? (
                <a
                  href={app.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-600 hover:text-blue-800"
                >
                  {app.url}
                </a>
              ) : (
                <p className="text-gray-500">Not deployed yet</p>
              )}
            </div>
            <div>
              <h3 className="text-sm font-medium text-gray-500 mb-1">Created</h3>
              <p className="text-gray-900">
                {new Date(app.created_at).toLocaleString()}
              </p>
            </div>
          </div>

          {app.deployment && (
            <div className="border-t border-gray-200 pt-6">
              <h3 className="text-lg font-semibold text-gray-900 mb-3">Latest Deployment</h3>
              <div className="bg-gray-50 rounded-lg p-4">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <p className="text-sm text-gray-500">Status</p>
                    <StatusBadge status={app.deployment.state} />
                  </div>
                  {app.deployment.last_deployed_at && (
                    <div>
                      <p className="text-sm text-gray-500">Last Deployed</p>
                      <p className="text-gray-900">
                        {new Date(app.deployment.last_deployed_at).toLocaleString()}
                      </p>
                    </div>
                  )}
                </div>
              </div>
            </div>
          )}
        </div>

        <div>
          <h2 className="text-2xl font-bold text-gray-900 mb-4">Deployments</h2>
          {deployments.length === 0 ? (
            <div className="bg-white rounded-lg shadow-md p-8 text-center">
              <p className="text-gray-600">No deployments found</p>
            </div>
          ) : (
            <div className="space-y-4">
              {deployments.map((deployment) => (
                <DeploymentCard key={deployment.id} deployment={deployment} appId={appId} />
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

