import { useState, useEffect } from 'react';
import Layout from '../components/Layout';
import { adminAppsApi } from '../lib/api';
import type { App } from '../lib/types';

export default function Apps() {
  const [apps, setApps] = useState<App[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [page, setPage] = useState(0);
  const [total, setTotal] = useState(0);
  const limit = 50;

  useEffect(() => {
    loadApps();
  }, [page]);

  const loadApps = async () => {
    setLoading(true);
    setError('');
    try {
      const response = await adminAppsApi.list(limit, page * limit);
      setApps(response.apps);
      setTotal(response.total);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load apps');
    } finally {
      setLoading(false);
    }
  };

  const handleAction = async (action: 'stop' | 'start' | 'redeploy', appId: string) => {
    try {
      if (action === 'stop') {
        await adminAppsApi.stop(appId);
      } else if (action === 'start') {
        await adminAppsApi.start(appId);
      } else if (action === 'redeploy') {
        await adminAppsApi.redeploy(appId);
      }
      loadApps();
    } catch (err) {
      alert(err instanceof Error ? err.message : `Failed to ${action} app`);
    }
  };

  const handleDelete = async (appId: string, appName: string) => {
    if (!confirm(`Are you sure you want to delete app "${appName}"? This will permanently delete the app and all its deployments. This action cannot be undone.`)) {
      return;
    }

    try {
      await adminAppsApi.delete(appId);
      // Reload apps after deletion
      await loadApps();
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to delete app');
    }
  };

  const getStatusColor = (status: string) => {
    switch (status?.toLowerCase()) {
      case 'running':
        return 'bg-green-100 text-green-800';
      case 'pending':
      case 'building':
        return 'bg-yellow-100 text-yellow-800';
      case 'failed':
      case 'stopped':
        return 'bg-red-100 text-red-800';
      default:
        return 'bg-gray-100 text-gray-800';
    }
  };

  const totalPages = Math.ceil(total / limit);

  return (
    <Layout>
      <div className="space-y-6">
        <div className="flex justify-between items-center">
          <div>
            <h1 className="text-3xl font-bold text-gray-900">Apps</h1>
            <p className="mt-2 text-sm text-gray-600">
              Monitor and manage deployed applications
            </p>
          </div>
        </div>

        <div className="bg-white shadow rounded-lg">
          {error && (
            <div className="p-4 bg-red-50 border-b border-red-200">
              <p className="text-sm text-red-800">{error}</p>
            </div>
          )}

          {loading ? (
            <div className="p-8 text-center text-gray-500">Loading...</div>
          ) : (
            <>
              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-gray-200">
                  <thead className="bg-gray-50">
                    <tr>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                        Name
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                        Status
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                        URL
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                        Repository
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                        Deployments
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                        Created
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                        Actions
                      </th>
                    </tr>
                  </thead>
                  <tbody className="bg-white divide-y divide-gray-200">
                    {apps.map((app) => (
                      <tr key={app.id}>
                        <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                          {app.name}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap">
                          <span
                            className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${getStatusColor(
                              app.status
                            )}`}
                          >
                            {app.status || 'Unknown'}
                          </span>
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                          {app.url ? (
                            <a
                              href={app.url}
                              target="_blank"
                              rel="noopener noreferrer"
                              className="text-indigo-600 hover:text-indigo-900"
                            >
                              {app.url}
                            </a>
                          ) : (
                            '-'
                          )}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                          <div className="max-w-xs truncate" title={app.repo_url}>
                            {app.repo_url}
                          </div>
                          {app.branch && (
                            <div className="text-xs text-gray-400">Branch: {app.branch}</div>
                          )}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                          {app.deployment_count || 0}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                          {new Date(app.created_at).toLocaleDateString()}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm font-medium space-x-2">
                          <button
                            onClick={() => handleAction('stop', app.id)}
                            className="text-red-600 hover:text-red-900"
                          >
                            Stop
                          </button>
                          <button
                            onClick={() => handleAction('start', app.id)}
                            className="text-green-600 hover:text-green-900"
                          >
                            Start
                          </button>
                          <button
                            onClick={() => handleAction('redeploy', app.id)}
                            className="text-indigo-600 hover:text-indigo-900"
                          >
                            Redeploy
                          </button>
                          <button
                            onClick={() => handleDelete(app.id, app.name)}
                            className="text-red-600 hover:text-red-900 font-semibold"
                          >
                            Delete
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>

              {totalPages > 1 && (
                <div className="px-6 py-4 border-t border-gray-200 flex items-center justify-between">
                  <div className="text-sm text-gray-700">
                    Showing {page * limit + 1} to {Math.min((page + 1) * limit, total)} of {total} apps
                  </div>
                  <div className="flex space-x-2">
                    <button
                      onClick={() => setPage(Math.max(0, page - 1))}
                      disabled={page === 0}
                      className="px-4 py-2 border border-gray-300 rounded-md text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      Previous
                    </button>
                    <button
                      onClick={() => setPage(Math.min(totalPages - 1, page + 1))}
                      disabled={page >= totalPages - 1}
                      className="px-4 py-2 border border-gray-300 rounded-md text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      Next
                    </button>
                  </div>
                </div>
              )}
            </>
          )}
        </div>
      </div>
    </Layout>
  );
}

