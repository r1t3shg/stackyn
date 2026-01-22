'use client';

import { useEffect, useState, useCallback } from 'react';
import { useParams, useRouter } from 'next/navigation';
import Link from 'next/link';
import { appsApi } from '@/lib/api';
import type { App, Deployment, EnvVar } from '@/lib/types';
import { extractString } from '@/lib/types';
import StatusBadge from '@/components/StatusBadge';
import DeploymentCard from '@/components/DeploymentCard';

export default function AppDetailsPage() {
  const params = useParams();
  const router = useRouter();
  const appId = params.id as string;

  const [app, setApp] = useState<App | null>(null);
  const [deployments, setDeployments] = useState<Deployment[]>([]);
  const [envVars, setEnvVars] = useState<EnvVar[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [redeploying, setRedeploying] = useState(false);
  const [showAddEnvVar, setShowAddEnvVar] = useState(false);
  const [newEnvKey, setNewEnvKey] = useState('');
  const [newEnvValue, setNewEnvValue] = useState('');
  const [addingEnvVar, setAddingEnvVar] = useState(false);
  const [envVarsError, setEnvVarsError] = useState<string | null>(null);
  const [loadingEnvVars, setLoadingEnvVars] = useState(false);

  const loadApp = useCallback(async () => {
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
  }, [appId]);

  const loadDeployments = useCallback(async () => {
    try {
      const data = await appsApi.getDeployments(appId);
      setDeployments(data);
    } catch (err) {
      console.error('Error loading deployments:', err);
    }
  }, [appId]);

  const loadEnvVars = useCallback(async () => {
    setLoadingEnvVars(true);
    setEnvVarsError(null);
    try {
      const data = await appsApi.getEnvVars(appId);
      // Ensure data is always an array, never null or undefined
      setEnvVars(Array.isArray(data) ? data : []);
      console.log('Environment variables loaded:', data);
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to load environment variables';
      console.error('Error loading environment variables:', err);
      setEnvVarsError(errorMessage);
      // Still show the section even if there's an error - set to empty array
      setEnvVars([]);
    } finally {
      setLoadingEnvVars(false);
    }
  }, [appId]);

  useEffect(() => {
    if (appId) {
      console.log('Loading app details for ID:', appId);
      loadApp();
      loadDeployments();
      loadEnvVars();
    }
  }, [appId, loadApp, loadDeployments, loadEnvVars]);

  // Poll for status updates when app is building or deploying
  useEffect(() => {
    if (!appId || !app) {
      return;
    }

    // Only poll if app is in a transitional state (not failed or stopped)
    const isTransitioning = (app.status === 'pending' || 
                            app.status === 'building' || 
                            app.status === 'deploying') &&
                           app.status !== 'failed' &&
                           app.status !== 'stopped' &&
                           (app.deployment && (app.deployment.state === 'pending' || 
                                               app.deployment.state === 'building' || 
                                               app.deployment.state === 'deploying'));

    if (!isTransitioning) {
      return;
    }

    // Poll every 2 seconds while building/deploying
    const interval = setInterval(() => {
      loadApp();
      loadDeployments();
    }, 2000);

    return () => clearInterval(interval);
  }, [appId, app, loadApp, loadDeployments]);

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
      router.push('/');
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to delete app');
      console.error('Error deleting app:', err);
    }
  };

  const handleAddEnvVar = async () => {
    if (!newEnvKey.trim()) {
      alert('Key is required');
      return;
    }

    setAddingEnvVar(true);
    try {
      await appsApi.createEnvVar(appId, { key: newEnvKey.trim(), value: newEnvValue });
      setNewEnvKey('');
      setNewEnvValue('');
      setShowAddEnvVar(false);
      await loadEnvVars();
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to add environment variable');
      console.error('Error adding environment variable:', err);
    } finally {
      setAddingEnvVar(false);
    }
  };

  const handleDeleteEnvVar = async (key: string) => {
    if (!confirm(`Are you sure you want to delete the environment variable "${key}"?`)) {
      return;
    }

    try {
      await appsApi.deleteEnvVar(appId, key);
      await loadEnvVars();
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to delete environment variable');
      console.error('Error deleting environment variable:', err);
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
          <Link href="/" className="text-blue-600 hover:text-blue-800 mb-6 inline-block">
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
        <Link href="/" className="text-blue-600 hover:text-blue-800 mb-6 inline-block">
          ← Back to Apps
        </Link>

        <div className="bg-white rounded-lg shadow-md p-8 mb-6">
          {/* Build Error Banner - Show prominently at the top */}
          {(app.status === 'failed' || deployments.some(d => d.status === 'failed' && extractString(d.error_message))) && (
            <div className="mb-6 p-4 bg-red-50 border-l-4 border-red-500 rounded-r-lg">
              <div className="flex items-start">
                <div className="flex-shrink-0">
                  <svg className="h-5 w-5 text-red-400" viewBox="0 0 20 20" fill="currentColor">
                    <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
                  </svg>
                </div>
                <div className="ml-3 flex-1">
                  <h3 className="text-sm font-medium text-red-800 mb-2">
                    Build Failed
                  </h3>
                  {(() => {
                    const failedDeployment = deployments.find(d => d.status === 'failed' && extractString(d.error_message));
                    const errorMsg = failedDeployment ? extractString(failedDeployment.error_message) : 'Build process failed. Please check the deployment logs for details.';
                    return (
                      <div className="text-sm text-red-700">
                        <p className="font-mono whitespace-pre-wrap break-words">{errorMsg}</p>
                        {failedDeployment && (
                          <Link
                            href={`/apps/${appId}/deployments/${failedDeployment.id}`}
                            className="mt-2 inline-block text-sm font-medium text-red-800 hover:text-red-900 underline"
                          >
                            View deployment details →
                          </Link>
                        )}
                      </div>
                    );
                  })()}
                </div>
              </div>
            </div>
          )}

          <div className="flex items-start justify-between mb-6">
            <div className="flex-1">
              <h1 className="text-3xl font-bold text-gray-900 mb-3">{app.name}</h1>
              <div className="flex items-center gap-3 mb-2">
                <StatusBadge status={app.status} />
                {(app.status === 'building' || app.status === 'deploying' || app.status === 'pending') && (
                  <span className="text-sm text-gray-600">Updating...</span>
                )}
              </div>
              {app.status === 'running' && app.url && (
                <div className="mt-2">
                  <a
                    href={app.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex items-center gap-2 text-blue-600 hover:text-blue-800 font-medium"
                  >
                    <span>Visit App</span>
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                    </svg>
                  </a>
                </div>
              )}
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

        {/* Environment Variables Section */}
        <div className="bg-white rounded-lg shadow-md p-8 mb-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-2xl font-bold text-gray-900">Environment Variables</h2>
            <button
              onClick={() => setShowAddEnvVar(!showAddEnvVar)}
              className="px-4 py-2 bg-green-600 hover:bg-green-700 text-white font-medium rounded-lg transition-colors"
            >
              {showAddEnvVar ? 'Cancel' : '+ Add Variable'}
            </button>
          </div>

          {envVarsError && (
            <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4 mb-4">
              <p className="text-yellow-800 text-sm">
                <strong>Warning:</strong> {envVarsError}
              </p>
            </div>
          )}

          {!loadingEnvVars && showAddEnvVar && (
            <div className="bg-gray-50 rounded-lg p-4 mb-4">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Key
                  </label>
                  <input
                    type="text"
                    value={newEnvKey}
                    onChange={(e) => setNewEnvKey(e.target.value)}
                    placeholder="e.g., DATABASE_URL"
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Value
                  </label>
                  <input
                    type="text"
                    value={newEnvValue}
                    onChange={(e) => setNewEnvValue(e.target.value)}
                    placeholder="e.g., postgres://..."
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                </div>
              </div>
              <button
                onClick={handleAddEnvVar}
                disabled={addingEnvVar || !newEnvKey.trim()}
                className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white font-medium rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {addingEnvVar ? 'Adding...' : 'Add Variable'}
              </button>
            </div>
          )}

          {loadingEnvVars ? (
            <div className="text-center py-8 text-gray-500">
              <div className="inline-block animate-spin rounded-full h-6 w-6 border-b-2 border-blue-600 mb-2"></div>
              <p>Loading environment variables...</p>
            </div>
          ) : !envVars || envVars.length === 0 ? (
            <div className="text-center py-8 text-gray-500">
              <p>No environment variables configured</p>
              <p className="text-sm mt-2">Add environment variables that will be injected into your running container</p>
            </div>
          ) : (
            <div className="space-y-2">
              {envVars.map((envVar) => (
                <div
                  key={envVar.id}
                  className="flex items-center justify-between bg-gray-50 rounded-lg p-4 hover:bg-gray-100 transition-colors"
                >
                  <div className="flex-1">
                    <div className="flex items-center space-x-4">
                      <span className="font-mono font-semibold text-gray-900">{envVar.key}</span>
                      <span className="text-gray-400">=</span>
                      <span className="font-mono text-gray-600">{envVar.value}</span>
                    </div>
                  </div>
                  <button
                    onClick={() => handleDeleteEnvVar(envVar.key)}
                    className="ml-4 px-3 py-1 text-sm bg-red-600 hover:bg-red-700 text-white rounded transition-colors"
                  >
                    Delete
                  </button>
                </div>
              ))}
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


