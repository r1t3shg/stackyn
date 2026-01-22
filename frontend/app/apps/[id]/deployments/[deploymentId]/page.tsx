'use client';

import { useEffect, useState, useCallback } from 'react';
import { useParams } from 'next/navigation';
import Link from 'next/link';
import { deploymentsApi } from '@/lib/api';
import type { Deployment, DeploymentLogs } from '@/lib/types';
import { extractString } from '@/lib/types';
import StatusBadge from '@/components/StatusBadge';
import LogsViewer from '@/components/LogsViewer';

export default function DeploymentDetailsPage() {
  const params = useParams();
  const deploymentId = params.deploymentId as string;
  const appId = params.id as string;

  const [deployment, setDeployment] = useState<Deployment | null>(null);
  const [logs, setLogs] = useState<DeploymentLogs | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadDeployment = useCallback(async () => {
    try {
      setError(null);
      const data = await deploymentsApi.getById(deploymentId);
      setDeployment(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load deployment');
      console.error('Error loading deployment:', err);
    } finally {
      setLoading(false);
    }
  }, [deploymentId]);

  const loadLogs = useCallback(async () => {
    try {
      const data = await deploymentsApi.getLogs(deploymentId);
      setLogs(data);
    } catch (err) {
      console.error('Error loading logs:', err);
    }
  }, [deploymentId]);

  useEffect(() => {
    if (deploymentId) {
      loadDeployment();
      loadLogs();
    }
  }, [deploymentId, loadDeployment, loadLogs]);

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-center">
          <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
          <p className="mt-4 text-gray-600">Loading deployment...</p>
        </div>
      </div>
    );
  }

  if (error || !deployment) {
    return (
      <div className="min-h-screen bg-gray-50">
        <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
          <Link
            href={`/apps/${appId}`}
            className="text-blue-600 hover:text-blue-800 mb-6 inline-block"
          >
            ← Back to App
          </Link>
          <div className="bg-red-50 border border-red-200 rounded-lg p-6">
            <p className="text-red-800">{error || 'Deployment not found'}</p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <Link
          href={`/apps/${appId}`}
          className="text-blue-600 hover:text-blue-800 mb-6 inline-block"
        >
          ← Back to App
        </Link>

        <div className="bg-white rounded-lg shadow-md p-8 mb-6">
          <div className="flex items-start justify-between mb-6">
            <div>
              <h1 className="text-3xl font-bold text-gray-900 mb-2">
                Deployment #{deployment.id}
              </h1>
              <StatusBadge status={deployment.status} />
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
            <div>
              <h3 className="text-sm font-medium text-gray-500 mb-1">App ID</h3>
              <p className="text-gray-900">{deployment.app_id}</p>
            </div>
            <div>
              <h3 className="text-sm font-medium text-gray-500 mb-1">Status</h3>
              <StatusBadge status={deployment.status} />
            </div>
            {extractString(deployment.subdomain) && (
              <div>
                <h3 className="text-sm font-medium text-gray-500 mb-1">Subdomain</h3>
                <p className="text-gray-900">{extractString(deployment.subdomain)}</p>
              </div>
            )}
            {extractString(deployment.image_name) && (
              <div>
                <h3 className="text-sm font-medium text-gray-500 mb-1">Image Name</h3>
                <p className="text-gray-900 font-mono text-sm">{extractString(deployment.image_name)}</p>
              </div>
            )}
            {extractString(deployment.container_id) && (
              <div>
                <h3 className="text-sm font-medium text-gray-500 mb-1">Container ID</h3>
                <p className="text-gray-900 font-mono text-sm">{extractString(deployment.container_id)}</p>
              </div>
            )}
            <div>
              <h3 className="text-sm font-medium text-gray-500 mb-1">Created</h3>
              <p className="text-gray-900">
                {new Date(deployment.created_at).toLocaleString()}
              </p>
            </div>

            <div>
              <h3 className="text-sm font-medium text-gray-500 mb-1">Updated</h3>
              <p className="text-gray-900">
                {new Date(deployment.updated_at).toLocaleString()}
              </p>
            </div>
          </div>

          {logs?.error_message && deployment.status !== 'stopped' && (
            <div className="mt-6 p-4 bg-red-50 border border-red-200 rounded-lg">
              <h3 className="text-sm font-medium text-red-800 mb-2">Error Message</h3>
              <p className="text-red-800">{extractString(logs.error_message)}</p>
            </div>
          )}
        </div>

        <div className="space-y-6">
          {logs && extractString(logs.build_log) && (
            <div>
              <h2 className="text-xl font-bold text-gray-900 mb-4">Build Logs</h2>
              <LogsViewer logs={extractString(logs.build_log)} title="Build Logs" />
            </div>
          )}

          {logs && !extractString(logs.build_log) && (
            <div className="bg-white rounded-lg shadow-md p-8 text-center">
              <p className="text-gray-600">No build logs available yet</p>
            </div>
          )}

          {logs && extractString(logs.runtime_log) && (
            <div>
              <h2 className="text-xl font-bold text-gray-900 mb-4">Runtime Logs</h2>
              <LogsViewer logs={extractString(logs.runtime_log)} title="Runtime Logs" />
            </div>
          )}

          {logs && !extractString(logs.runtime_log) && (
            <div className="bg-white rounded-lg shadow-md p-8 text-center">
              <p className="text-gray-600">No runtime logs available yet</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}



