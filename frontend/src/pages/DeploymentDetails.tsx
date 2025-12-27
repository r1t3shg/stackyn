import { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { deploymentsApi, appsApi } from '@/lib/api';
import type { Deployment, DeploymentLogs, App } from '@/lib/types';
import { extractString } from '@/lib/types';
import StatusBadge from '@/components/StatusBadge';
import LogsViewer from '@/components/LogsViewer';

export default function DeploymentDetailsPage() {
  const params = useParams<{ id: string; deploymentId: string }>();
  const deploymentId = params.deploymentId!;
  const appId = params.id!;

  const [deployment, setDeployment] = useState<Deployment | null>(null);
  const [app, setApp] = useState<App | null>(null);
  const [logs, setLogs] = useState<DeploymentLogs | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [autoRefresh, setAutoRefresh] = useState(false);

  useEffect(() => {
    if (deploymentId && appId) {
      loadDeployment();
      loadApp();
      loadLogs();
    }
  }, [deploymentId, appId]);

  useEffect(() => {
    // Auto-refresh logs every 5 seconds if deployment is building or pending
    if (deployment && (deployment.status === 'building' || deployment.status === 'pending')) {
      const interval = setInterval(() => {
        if (autoRefresh) {
          loadLogs();
          loadDeployment();
        }
      }, 5000);
      return () => clearInterval(interval);
    }
  }, [deployment?.status, autoRefresh]);

  const loadDeployment = async () => {
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
  };

  const loadApp = async () => {
    try {
      const data = await appsApi.getById(appId);
      setApp(data);
    } catch (err) {
      console.error('Error loading app:', err);
    }
  };

  const loadLogs = async () => {
    try {
      const data = await deploymentsApi.getLogs(deploymentId);
      setLogs(data);
      // Auto-enable refresh if deployment is in progress
      if (deployment?.status === 'building' || deployment?.status === 'pending') {
        setAutoRefresh(true);
      }
    } catch (err) {
      console.error('Error loading logs:', err);
    }
  };

  const getStatusLabel = (status: string): string => {
    switch (status) {
      case 'running':
        return 'Healthy';
      case 'building':
      case 'pending':
        return 'Deploying';
      case 'failed':
        return 'Failed';
      case 'stopped':
        return 'Stopped';
      default:
        return status;
    }
  };

  const getStatusSummary = (): string => {
    if (!deployment || !app) return '';
    
    switch (deployment.status) {
      case 'running':
        return `Your app "${app.name}" is live and running. The deployment completed successfully and is serving traffic.`;
      case 'building':
        return `Your app "${app.name}" is currently being built and deployed. This usually takes a few minutes.`;
      case 'pending':
        return `Your app "${app.name}" is queued for deployment. It will start building shortly.`;
      case 'failed':
        return `The deployment for "${app.name}" failed. Check the error message and logs below to identify the issue.`;
      case 'stopped':
        return `The deployment for "${app.name}" has been stopped. It is no longer serving traffic.`;
      default:
        return `Deployment status: ${deployment.status}`;
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-[var(--app-bg)] flex items-center justify-center">
        <div className="text-center">
          <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-[var(--primary)]"></div>
          <p className="mt-4 text-[var(--text-secondary)]">Loading deployment...</p>
        </div>
      </div>
    );
  }

  if (error || !deployment) {
    return (
      <div className="min-h-screen bg-[var(--app-bg)]">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
          <Link
            to={`/apps/${appId}`}
            className="text-[var(--info)] hover:text-[var(--primary)] mb-6 inline-block transition-colors"
          >
            ← Back to App
          </Link>
          <div className="bg-[var(--error)]/10 border border-[var(--error)] rounded-lg p-6">
            <p className="text-[var(--error)]">{error || 'Deployment not found'}</p>
          </div>
        </div>
      </div>
    );
  }

  const statusLabel = getStatusLabel(deployment.status);
  const statusSummary = getStatusSummary();

  return (
    <div className="min-h-screen bg-[var(--app-bg)]">
      <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <Link
          to={`/apps/${appId}`}
          className="text-[var(--info)] hover:text-[var(--primary)] mb-6 inline-block transition-colors"
        >
          ← Back to App
        </Link>

        {/* Status Header Section */}
        <div className="bg-[var(--surface)] rounded-lg border border-[var(--border-subtle)] p-6 mb-6">
          <div className="flex items-start justify-between mb-4">
            <div className="flex-1">
              <div className="flex items-center gap-4 mb-3">
                <h1 className="text-3xl font-bold text-[var(--text-primary)]">
                  {app?.name || 'Deployment'}
                </h1>
                <StatusBadge status={statusLabel} />
              </div>
              <p className="text-[var(--text-secondary)] text-lg mb-4">
                {statusSummary}
              </p>
            </div>
          </div>
        </div>

        {/* Deployment Logs Section */}
        <div className="bg-[var(--surface)] rounded-lg border border-[var(--border-subtle)] p-6 mb-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-xl font-bold text-[var(--text-primary)]">Deployment Logs</h2>
            {(deployment.status === 'building' || deployment.status === 'pending') && (
              <div className="flex items-center gap-2">
                <label className="flex items-center gap-2 text-sm text-[var(--text-secondary)] cursor-pointer">
                  <input
                    type="checkbox"
                    checked={autoRefresh}
                    onChange={(e) => setAutoRefresh(e.target.checked)}
                    className="rounded"
                  />
                  Auto-refresh
                </label>
              </div>
            )}
          </div>
          
          {logs && extractString(logs.runtime_log) ? (
            <LogsViewer logs={extractString(logs.runtime_log)} />
          ) : (
            <div className="bg-[var(--surface)] rounded-lg border border-[var(--border-subtle)] p-8 text-center">
              <p className="text-[var(--text-secondary)]">
                {deployment.status === 'pending' || deployment.status === 'building'
                  ? 'Runtime logs will appear here once the deployment is running...'
                  : 'No runtime logs available yet'}
              </p>
            </div>
          )}
        </div>

        {/* Error Message Section */}
        {logs?.error_message && extractString(logs.error_message) && (
          <div className="bg-[var(--error)]/10 border border-[var(--error)] rounded-lg p-6 mb-6">
            <h3 className="text-lg font-semibold text-[var(--error)] mb-3">Deployment Failed</h3>
            <p className="text-[var(--error)] mb-4 whitespace-pre-wrap">
              {extractString(logs.error_message)}
            </p>
            <div className="bg-[var(--surface)] rounded-lg p-4 border border-[var(--border-subtle)]">
              <h4 className="text-sm font-semibold text-[var(--text-primary)] mb-2">What to do next:</h4>
              <ul className="list-disc list-inside text-sm text-[var(--text-secondary)] space-y-1">
                <li>Check the error message above for specific details</li>
                <li>Verify your Dockerfile is in the repository root</li>
                <li>Ensure your build commands are correct</li>
                <li>Check that all dependencies are properly specified</li>
                <li>Fix the issues and redeploy the app</li>
              </ul>
            </div>
          </div>
        )}

        {/* Contextual Guidance Section */}
        <div className="bg-[var(--surface)] rounded-lg border border-[var(--border-subtle)] p-6">
          <h3 className="text-lg font-semibold text-[var(--text-primary)] mb-4">Understanding Your Deployment</h3>
          
          {deployment.status === 'running' && (
            <div className="space-y-4">
              <div>
                <h4 className="text-sm font-semibold text-[var(--text-primary)] mb-2">Zero-Downtime Redeploys</h4>
                <p className="text-sm text-[var(--text-secondary)]">
                  When you redeploy, Stackyn builds your new version in the background. Once ready, it seamlessly replaces the old container with the new one, ensuring your app stays online throughout the process. There's no downtime during redeployments.
                </p>
              </div>
              <div>
                <h4 className="text-sm font-semibold text-[var(--text-primary)] mb-2">Automatic Restarts</h4>
                <p className="text-sm text-[var(--text-secondary)]">
                  If your app crashes or encounters an error, Stackyn automatically restarts the container. The restart count above shows how many times this has happened. If you see frequent restarts, check your logs for errors.
                </p>
              </div>
            </div>
          )}

          {(deployment.status === 'building' || deployment.status === 'pending') && (
            <div>
              <h4 className="text-sm font-semibold text-[var(--text-primary)] mb-2">What's Happening Now</h4>
              <p className="text-sm text-[var(--text-secondary)] mb-3">
                Your deployment is in progress. Stackyn is:
              </p>
              <ul className="list-disc list-inside text-sm text-[var(--text-secondary)] space-y-1 ml-2">
                <li>Cloning your repository from Git</li>
                <li>Building your Docker image</li>
                <li>Starting your container</li>
                <li>Verifying your app is running</li>
              </ul>
              <p className="text-sm text-[var(--text-secondary)] mt-3">
                This typically takes 2-5 minutes. You can watch the progress in the logs above. Once complete, your app will be live and accessible.
              </p>
            </div>
          )}

          {deployment.status === 'failed' && (
            <div>
              <h4 className="text-sm font-semibold text-[var(--text-primary)] mb-2">What Went Wrong</h4>
              <p className="text-sm text-[var(--text-secondary)]">
                The deployment failed during the build or startup process. Common causes include missing Dockerfiles, build errors, or runtime crashes. Review the error message and logs above to identify the specific issue. Once fixed, you can redeploy from the app details page.
              </p>
            </div>
          )}

          <div className="mt-6 pt-6 border-t border-[var(--border-subtle)]">
            <h4 className="text-sm font-semibold text-[var(--text-primary)] mb-2">Need Help?</h4>
            <p className="text-sm text-[var(--text-secondary)]">
              If you're stuck, check the logs for detailed error messages. Most deployment issues are related to Dockerfile configuration, missing dependencies, or application startup errors. Fix these issues in your code and redeploy.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
