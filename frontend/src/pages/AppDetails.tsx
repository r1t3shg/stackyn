import { useEffect, useState } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { appsApi, deploymentsApi } from '@/lib/api';
import type { App, Deployment, EnvVar, DeploymentLogs } from '@/lib/types';
import StatusBadge from '@/components/StatusBadge';
import LogsViewer from '@/components/LogsViewer';
import { extractString } from '@/lib/types';

type Tab = 'overview' | 'deployments' | 'logs' | 'metrics' | 'settings';

export default function AppDetailsPage() {
  const params = useParams<{ id: string }>();
  const navigate = useNavigate();
  const appId = params.id!;

  const [app, setApp] = useState<App | null>(null);
  const [deployments, setDeployments] = useState<Deployment[]>([]);
  const [envVars, setEnvVars] = useState<EnvVar[]>([]);
  const [logs, setLogs] = useState<DeploymentLogs | null>(null);
  const [activeTab, setActiveTab] = useState<Tab>('overview');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [showAddEnvVar, setShowAddEnvVar] = useState(false);
  const [newEnvKey, setNewEnvKey] = useState('');
  const [newEnvValue, setNewEnvValue] = useState('');
  const [addingEnvVar, setAddingEnvVar] = useState(false);
  const [envVarsError, setEnvVarsError] = useState<string | null>(null);
  const [loadingEnvVars, setLoadingEnvVars] = useState(false);
  const [autoRefresh, setAutoRefresh] = useState(false);

  useEffect(() => {
    if (appId) {
      loadApp();
      loadDeployments();
      loadEnvVars();
    }
  }, [appId]);

  useEffect(() => {
    if (activeTab === 'logs' && app?.deployment?.active_deployment_id) {
      loadLogs();
      // Auto-refresh logs every 5 seconds
      const interval = setInterval(() => {
        if (autoRefresh) {
          loadLogs();
        }
      }, 5000);
      return () => clearInterval(interval);
    }
  }, [activeTab, app?.deployment?.active_deployment_id, autoRefresh]);

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
      setDeployments(Array.isArray(data) ? data : []);
    } catch (err) {
      console.error('Error loading deployments:', err);
      setDeployments([]);
    }
  };

  const loadEnvVars = async () => {
    setLoadingEnvVars(true);
    setEnvVarsError(null);
    try {
      const data = await appsApi.getEnvVars(appId);
      setEnvVars(Array.isArray(data) ? data : []);
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to load environment variables';
      setEnvVarsError(errorMessage);
      setEnvVars([]);
    } finally {
      setLoadingEnvVars(false);
    }
  };

  const loadLogs = async () => {
    if (!app?.deployment?.active_deployment_id) return;
    try {
      const deploymentId = app.deployment.active_deployment_id.replace('dep_', '');
      const data = await deploymentsApi.getLogs(deploymentId);
      setLogs(data);
    } catch (err) {
      console.error('Error loading logs:', err);
    }
  };

  const handleRedeploy = async () => {
    if (!confirm('Are you sure you want to redeploy this app? This will trigger a new build.')) {
      return;
    }
    setActionLoading('redeploy');
    try {
      await appsApi.redeploy(appId);
      await loadApp();
      await loadDeployments();
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to redeploy app');
      console.error('Error redeploying app:', err);
    } finally {
      setActionLoading(null);
    }
  };

  const handleDelete = async () => {
    if (!confirm('Are you sure you want to delete this app? This action cannot be undone and will remove all deployments and data.')) {
      return;
    }
    setActionLoading('delete');
    try {
      await appsApi.delete(appId);
      navigate('/apps');
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to delete app');
      console.error('Error deleting app:', err);
      setActionLoading(null);
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

  const formatDate = (dateString: string | null | undefined) => {
    if (!dateString) return 'Never';
    try {
      const date = new Date(dateString);
      return date.toLocaleString();
    } catch {
      return 'Invalid date';
    }
  };

  const formatRelativeTime = (dateString: string | null | undefined) => {
    if (!dateString) return 'Never';
    try {
      const date = new Date(dateString);
      const now = new Date();
      const diffMs = now.getTime() - date.getTime();
      const diffMins = Math.floor(diffMs / 60000);
      const diffHours = Math.floor(diffMs / 3600000);
      const diffDays = Math.floor(diffMs / 86400000);

      if (diffMins < 1) return 'Just now';
      if (diffMins < 60) return `${diffMins}m ago`;
      if (diffHours < 24) return `${diffHours}h ago`;
      if (diffDays < 7) return `${diffDays}d ago`;
      return date.toLocaleDateString();
    } catch {
      return 'Invalid date';
    }
  };

  const calculateUptime = (createdAt: string | null | undefined) => {
    if (!createdAt) return 'N/A';
    try {
      const start = new Date(createdAt);
      const now = new Date();
      const diffMs = now.getTime() - start.getTime();
      const diffDays = Math.floor(diffMs / 86400000);
      const diffHours = Math.floor((diffMs % 86400000) / 3600000);
      const diffMins = Math.floor((diffMs % 3600000) / 60000);

      if (diffDays > 0) return `${diffDays}d ${diffHours}h`;
      if (diffHours > 0) return `${diffHours}h ${diffMins}m`;
      return `${diffMins}m`;
    } catch {
      return 'N/A';
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-[var(--app-bg)] flex items-center justify-center">
        <div className="text-center">
          <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-[var(--primary)]"></div>
          <p className="mt-4 text-[var(--text-secondary)]">Loading app...</p>
        </div>
      </div>
    );
  }

  if (error || !app) {
    return (
      <div className="min-h-screen bg-[var(--app-bg)]">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
          <Link to="/apps" className="text-[var(--info)] hover:text-[var(--primary)] mb-6 inline-block transition-colors">
            ← Back to Apps
          </Link>
          <div className="bg-[var(--error)]/10 border border-[var(--error)] rounded-lg p-6">
            <p className="text-[var(--error)]">{error || 'App not found'}</p>
          </div>
        </div>
      </div>
    );
  }

  const activeDeployment = deployments.find(d => d.id.toString() === app.deployment?.active_deployment_id?.replace('dep_', ''));

  return (
    <div className="min-h-screen bg-[var(--app-bg)]">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <Link to="/apps" className="text-[var(--info)] hover:text-[var(--primary)] mb-6 inline-block transition-colors">
          ← Back to Apps
        </Link>

        {/* Header Section */}
        <div className="bg-[var(--surface)] rounded-lg border border-[var(--border-subtle)] p-6 mb-6">
          <div className="flex items-start justify-between mb-4">
            <div className="flex-1">
              <h1 className="text-3xl font-bold text-[var(--text-primary)] mb-3">{app.name}</h1>
              <div className="flex items-center gap-4 flex-wrap">
                <StatusBadge status={app.status || 'unknown'} />
                {app.url && (
                  <a
                    href={app.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-[var(--info)] hover:text-[var(--primary)] text-sm transition-colors flex items-center gap-1"
                  >
                    {app.url}
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                    </svg>
                  </a>
                )}
              </div>
            </div>
            <div className="flex gap-2">
              <button
                onClick={handleRedeploy}
                disabled={actionLoading !== null}
                className="px-4 py-2 bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-[var(--app-bg)] font-medium rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                title="Redeploy app"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                </svg>
                {actionLoading === 'redeploy' ? 'Redeploying...' : 'Redeploy'}
              </button>
              <button
                onClick={handleDelete}
                disabled={actionLoading !== null}
                className="px-4 py-2 bg-[var(--error)] hover:bg-[var(--error)]/80 text-white font-medium rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                title="Delete app"
              >
                {actionLoading === 'delete' ? 'Deleting...' : 'Delete'}
              </button>
            </div>
          </div>

          {/* Runtime Information */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 pt-4 border-t border-[var(--border-subtle)]">
            <div>
              <div className="text-xs text-[var(--text-muted)] mb-1">RAM</div>
              <div className="text-lg font-semibold text-[var(--text-primary)]">
                {app.deployment?.resource_limits?.memory_mb || '—'} MB
              </div>
            </div>
            <div>
              <div className="text-xs text-[var(--text-muted)] mb-1">CPU</div>
              <div className="text-lg font-semibold text-[var(--text-primary)]">
                {app.deployment?.resource_limits?.cpu || '—'} vCPU
              </div>
            </div>
            <div>
              <div className="text-xs text-[var(--text-muted)] mb-1">Container Status</div>
              <div className="text-lg font-semibold text-[var(--text-primary)]">
                {app.status === 'running' || app.status === 'healthy' ? 'Running' : app.status || 'Unknown'}
              </div>
            </div>
            <div>
              <div className="text-xs text-[var(--text-muted)] mb-1">Uptime</div>
              <div className="text-lg font-semibold text-[var(--text-primary)]">
                {calculateUptime(app.deployment?.last_deployed_at || app.created_at)}
              </div>
            </div>
            <div>
              <div className="text-xs text-[var(--text-muted)] mb-1">Last Deployed</div>
              <div className="text-sm text-[var(--text-secondary)]">
                {formatRelativeTime(app.deployment?.last_deployed_at || app.updated_at)}
              </div>
            </div>
            <div>
              <div className="text-xs text-[var(--text-muted)] mb-1">Deployment</div>
              <div className="text-sm text-[var(--text-secondary)] font-mono">
                {activeDeployment ? `#${activeDeployment.id}` : '—'}
              </div>
            </div>
          </div>
        </div>

        {/* Tabs */}
        <div className="mb-6">
          <div className="border-b border-[var(--border-subtle)]">
            <nav className="flex space-x-8 overflow-x-auto">
              {(['overview', 'deployments', 'logs', 'metrics', 'settings'] as Tab[]).map((tab) => (
                <button
                  key={tab}
                  onClick={() => setActiveTab(tab)}
                  className={`py-4 px-1 border-b-2 font-medium text-sm transition-colors whitespace-nowrap ${
                    activeTab === tab
                      ? 'border-[var(--primary)] text-[var(--primary)]'
                      : 'border-transparent text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:border-[var(--border-subtle)]'
                  }`}
                >
                  {tab.charAt(0).toUpperCase() + tab.slice(1)}
                </button>
              ))}
            </nav>
          </div>
        </div>

        {/* Tab Content */}
        <div>
          {/* Overview Tab */}
          {activeTab === 'overview' && (
            <div className="space-y-6">
              <div className="bg-[var(--surface)] rounded-lg border border-[var(--border-subtle)] p-6">
                <h2 className="text-xl font-semibold text-[var(--text-primary)] mb-4">App Information</h2>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                  <div>
                    <div className="text-sm text-[var(--text-muted)] mb-1">Repository URL</div>
                    <div className="text-[var(--text-primary)] font-mono text-sm">{app.repo_url}</div>
                  </div>
                  <div>
                    <div className="text-sm text-[var(--text-muted)] mb-1">Branch</div>
                    <div className="text-[var(--text-primary)]">{app.branch}</div>
                  </div>
                  <div>
                    <div className="text-sm text-[var(--text-muted)] mb-1">Created</div>
                    <div className="text-[var(--text-primary)]">{formatDate(app.created_at)}</div>
                  </div>
                  <div>
                    <div className="text-sm text-[var(--text-muted)] mb-1">Last Updated</div>
                    <div className="text-[var(--text-primary)]">{formatDate(app.updated_at)}</div>
                  </div>
                </div>
              </div>

              {app.deployment && (
                <div className="bg-[var(--surface)] rounded-lg border border-[var(--border-subtle)] p-6">
                  <h2 className="text-xl font-semibold text-[var(--text-primary)] mb-4">Resource Limits</h2>
                  <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    {app.deployment.resource_limits && (
                      <>
                        <div className="bg-[var(--elevated)] rounded-lg p-4 border border-[var(--border-subtle)]">
                          <div className="text-xs text-[var(--text-muted)] mb-1">Memory</div>
                          <div className="text-2xl font-bold text-[var(--text-primary)]">
                            {app.deployment.resource_limits.memory_mb} MB
                          </div>
                        </div>
                        <div className="bg-[var(--elevated)] rounded-lg p-4 border border-[var(--border-subtle)]">
                          <div className="text-xs text-[var(--text-muted)] mb-1">CPU</div>
                          <div className="text-2xl font-bold text-[var(--text-primary)]">
                            {app.deployment.resource_limits.cpu} vCPU
                          </div>
                        </div>
                        <div className="bg-[var(--elevated)] rounded-lg p-4 border border-[var(--border-subtle)]">
                          <div className="text-xs text-[var(--text-muted)] mb-1">Disk</div>
                          <div className="text-2xl font-bold text-[var(--text-primary)]">
                            {app.deployment.resource_limits.disk_gb} GB
                          </div>
                        </div>
                      </>
                    )}
                  </div>
                </div>
              )}

              <div className="bg-[var(--primary-muted)]/20 rounded-lg border border-[var(--primary-muted)] p-4">
                <div className="flex items-start gap-3">
                  <svg className="w-5 h-5 text-[var(--primary)] mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                  <div>
                    <div className="font-medium text-[var(--text-primary)] mb-1">How auto-deploy works</div>
                    <div className="text-sm text-[var(--text-secondary)]">
                      Stackyn automatically builds and deploys your app when you push to the configured branch ({app.branch}). 
                      No manual steps required - just push your code and Stackyn handles the rest.
                    </div>
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* Deployments Tab */}
          {activeTab === 'deployments' && (
            <div className="space-y-4">
              <div className="bg-[var(--surface)] rounded-lg border border-[var(--border-subtle)] p-6">
                <h2 className="text-xl font-semibold text-[var(--text-primary)] mb-4">Deployment History</h2>
                {deployments.length === 0 ? (
                  <div className="text-center py-8 text-[var(--text-muted)]">
                    <p>No deployments found</p>
                  </div>
                ) : (
                  <div className="space-y-3">
                    {deployments.map((deployment) => {
                      const isActive = deployment.id.toString() === app.deployment?.active_deployment_id?.replace('dep_', '');
                      const isSuccess = deployment.status === 'running';
                      const isFailed = deployment.status === 'failed';
                      
                      return (
                        <Link
                          key={deployment.id}
                          to={`/apps/${appId}/deployments/${deployment.id}`}
                          className="block"
                        >
                          <div className={`bg-[var(--elevated)] rounded-lg border p-4 hover:border-[var(--border-strong)] transition-colors ${
                            isActive ? 'border-[var(--primary)]' : 'border-[var(--border-subtle)]'
                          }`}>
                            <div className="flex items-center justify-between">
                              <div className="flex items-center gap-4">
                                <div>
                                  <div className="font-semibold text-[var(--text-primary)]">
                                    Deployment #{deployment.id}
                                    {isActive && (
                                      <span className="ml-2 px-2 py-0.5 text-xs bg-[var(--primary-muted)] text-[var(--primary)] rounded">
                                        Active
                                      </span>
                                    )}
                                  </div>
                                  <div className="text-sm text-[var(--text-secondary)] mt-1">
                                    {formatDate(deployment.created_at)}
                                  </div>
                                </div>
                                <StatusBadge status={deployment.status} />
                              </div>
                              <div className="flex items-center gap-2">
                                {isSuccess && (
                                  <svg className="w-5 h-5 text-[var(--success)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                                  </svg>
                                )}
                                {isFailed && (
                                  <svg className="w-5 h-5 text-[var(--error)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                                  </svg>
                                )}
                                {isActive && !isFailed && (
                                  <button
                                    onClick={(e) => {
                                      e.preventDefault();
                                      e.stopPropagation();
                                      // Rollback functionality - for now just show alert
                                      alert('Rollback functionality will be available soon. This will redeploy the previous successful deployment.');
                                    }}
                                    className="px-3 py-1 text-sm bg-[var(--surface)] hover:bg-[var(--elevated)] text-[var(--text-primary)] border border-[var(--border-subtle)] rounded transition-colors"
                                  >
                                    Rollback
                                  </button>
                                )}
                              </div>
                            </div>
                            {extractString(deployment.error_message) && (
                              <div className="mt-3 p-3 bg-[var(--error)]/10 border border-[var(--error)] rounded">
                                <p className="text-sm text-[var(--error)]">{extractString(deployment.error_message)}</p>
                              </div>
                            )}
                          </div>
                        </Link>
                      );
                    })}
                  </div>
                )}
              </div>
            </div>
          )}

          {/* Logs Tab */}
          {activeTab === 'logs' && (
            <div className="space-y-4">
              <div className="bg-[var(--surface)] rounded-lg border border-[var(--border-subtle)] p-6">
                <div className="flex items-center justify-between mb-4">
                  <h2 className="text-xl font-semibold text-[var(--text-primary)]">Application Logs</h2>
                  <div className="flex items-center gap-3">
                    <label className="flex items-center gap-2 text-sm text-[var(--text-secondary)]">
                      <input
                        type="checkbox"
                        checked={autoRefresh}
                        onChange={(e) => setAutoRefresh(e.target.checked)}
                        className="rounded"
                      />
                      Auto-refresh
                    </label>
                    <button
                      onClick={loadLogs}
                      className="px-3 py-1 text-sm bg-[var(--surface)] hover:bg-[var(--elevated)] text-[var(--text-primary)] border border-[var(--border-subtle)] rounded transition-colors"
                    >
                      Refresh
                    </button>
                  </div>
                </div>
                {logs ? (
                  <div className="space-y-4">
                    {logs.build_log && (
                      <LogsViewer logs={logs.build_log} title="Build Logs" />
                    )}
                    {logs.runtime_log && (
                      <LogsViewer logs={logs.runtime_log} title="Runtime Logs" />
                    )}
                    {!logs.build_log && !logs.runtime_log && (
                      <div className="text-center py-8 text-[var(--text-muted)]">
                        <p>No logs available yet</p>
                      </div>
                    )}
                  </div>
                ) : (
                  <div className="text-center py-8 text-[var(--text-muted)]">
                    <p>Loading logs...</p>
                  </div>
                )}
              </div>
            </div>
          )}

          {/* Metrics Tab */}
          {activeTab === 'metrics' && (
            <div className="space-y-6">
              {app.deployment?.usage_stats ? (
                <>
                  <div className="bg-[var(--surface)] rounded-lg border border-[var(--border-subtle)] p-6">
                    <h2 className="text-xl font-semibold text-[var(--text-primary)] mb-4">Resource Usage</h2>
                    <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                      <div className="bg-[var(--elevated)] rounded-lg p-4 border border-[var(--border-subtle)]">
                        <div className="flex items-center justify-between mb-2">
                          <div className="text-sm text-[var(--text-muted)]">Memory Usage</div>
                          <div className="text-sm font-semibold text-[var(--text-primary)]">
                            {app.deployment.usage_stats.memory_usage_percent.toFixed(1)}%
                          </div>
                        </div>
                        <div className="w-full bg-[var(--surface)] rounded-full h-3 mb-2">
                          <div
                            className={`h-3 rounded-full transition-all ${
                              app.deployment.usage_stats.memory_usage_percent > 90
                                ? 'bg-[var(--error)]'
                                : app.deployment.usage_stats.memory_usage_percent > 70
                                ? 'bg-[var(--warning)]'
                                : 'bg-[var(--success)]'
                            }`}
                            style={{
                              width: `${Math.min(app.deployment.usage_stats.memory_usage_percent, 100)}%`,
                            }}
                          ></div>
                        </div>
                        <div className="text-xs text-[var(--text-secondary)]">
                          {app.deployment.usage_stats.memory_usage_mb} MB / {app.deployment.resource_limits?.memory_mb || 0} MB
                        </div>
                      </div>

                      <div className="bg-[var(--elevated)] rounded-lg p-4 border border-[var(--border-subtle)]">
                        <div className="text-sm text-[var(--text-muted)] mb-1">CPU Load</div>
                        <div className="text-2xl font-bold text-[var(--text-primary)]">
                          {app.deployment.resource_limits?.cpu || 0} vCPU
                        </div>
                        <div className="text-xs text-[var(--text-muted)] mt-1">Allocated</div>
                      </div>

                      <div className="bg-[var(--elevated)] rounded-lg p-4 border border-[var(--border-subtle)]">
                        <div className="text-sm text-[var(--text-muted)] mb-1">Restart Count</div>
                        <div className="text-2xl font-bold text-[var(--text-primary)]">
                          {app.deployment.usage_stats.restart_count}
                        </div>
                        <div className="text-xs text-[var(--text-muted)] mt-1">
                          {app.deployment.usage_stats.restart_count === 0
                            ? 'No restarts'
                            : app.deployment.usage_stats.restart_count === 1
                            ? 'Restarted once'
                            : `Restarted ${app.deployment.usage_stats.restart_count} times`}
                        </div>
                      </div>
                    </div>
                  </div>

                  <div className="bg-[var(--surface)] rounded-lg border border-[var(--border-subtle)] p-6">
                    <h2 className="text-xl font-semibold text-[var(--text-primary)] mb-4">Disk Usage</h2>
                    <div className="bg-[var(--elevated)] rounded-lg p-4 border border-[var(--border-subtle)]">
                      <div className="flex items-center justify-between mb-2">
                        <div className="text-sm text-[var(--text-muted)]">Disk Usage</div>
                        <div className="text-sm font-semibold text-[var(--text-primary)]">
                          {app.deployment.usage_stats.disk_usage_percent.toFixed(1)}%
                        </div>
                      </div>
                      <div className="w-full bg-[var(--surface)] rounded-full h-3 mb-2">
                        <div
                          className={`h-3 rounded-full transition-all ${
                            app.deployment.usage_stats.disk_usage_percent > 90
                              ? 'bg-[var(--error)]'
                              : app.deployment.usage_stats.disk_usage_percent > 70
                              ? 'bg-[var(--warning)]'
                              : 'bg-[var(--success)]'
                          }`}
                          style={{
                            width: `${Math.min(app.deployment.usage_stats.disk_usage_percent, 100)}%`,
                          }}
                        ></div>
                      </div>
                      <div className="text-xs text-[var(--text-secondary)]">
                        {app.deployment.usage_stats.disk_usage_gb.toFixed(2)} GB / {app.deployment.resource_limits?.disk_gb || 0} GB
                      </div>
                    </div>
                  </div>
                </>
              ) : (
                <div className="bg-[var(--surface)] rounded-lg border border-[var(--border-subtle)] p-6 text-center">
                  <p className="text-[var(--text-muted)]">No metrics available yet. Metrics will appear after the app is deployed.</p>
                </div>
              )}
            </div>
          )}

          {/* Settings Tab */}
          {activeTab === 'settings' && (
            <div className="space-y-6">
              {/* Environment Variables */}
              <div className="bg-[var(--surface)] rounded-lg border border-[var(--border-subtle)] p-6">
                <div className="flex items-center justify-between mb-4">
                  <div>
                    <h2 className="text-xl font-semibold text-[var(--text-primary)]">Environment Variables</h2>
                    <p className="text-sm text-[var(--text-muted)] mt-1">Configure environment variables for your app</p>
                  </div>
                  <button
                    onClick={() => setShowAddEnvVar(!showAddEnvVar)}
                    className="px-4 py-2 bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-[var(--app-bg)] font-medium rounded-lg transition-colors"
                  >
                    {showAddEnvVar ? 'Cancel' : '+ Add Variable'}
                  </button>
                </div>

                {envVarsError && (
                  <div className="bg-[var(--warning)]/10 border border-[var(--warning)] rounded-lg p-4 mb-4">
                    <p className="text-[var(--warning)] text-sm">{envVarsError}</p>
                  </div>
                )}

                {showAddEnvVar && (
                  <div className="bg-[var(--elevated)] rounded-lg p-4 mb-4 border border-[var(--border-subtle)]">
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
                      <div>
                        <label className="block text-sm font-medium text-[var(--text-secondary)] mb-1">Key</label>
                        <input
                          type="text"
                          value={newEnvKey}
                          onChange={(e) => setNewEnvKey(e.target.value)}
                          placeholder="e.g., DATABASE_URL"
                          className="w-full px-3 py-2 bg-[var(--surface)] border border-[var(--border-subtle)] rounded-lg focus:outline-none focus:ring-2 focus:ring-[var(--focus-border)] text-[var(--text-primary)]"
                        />
                      </div>
                      <div>
                        <label className="block text-sm font-medium text-[var(--text-secondary)] mb-1">Value</label>
                        <input
                          type="text"
                          value={newEnvValue}
                          onChange={(e) => setNewEnvValue(e.target.value)}
                          placeholder="e.g., postgres://..."
                          className="w-full px-3 py-2 bg-[var(--surface)] border border-[var(--border-subtle)] rounded-lg focus:outline-none focus:ring-2 focus:ring-[var(--focus-border)] text-[var(--text-primary)]"
                        />
                      </div>
                    </div>
                    <button
                      onClick={handleAddEnvVar}
                      disabled={addingEnvVar || !newEnvKey.trim()}
                      className="px-4 py-2 bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-[var(--app-bg)] font-medium rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      {addingEnvVar ? 'Adding...' : 'Add Variable'}
                    </button>
                  </div>
                )}

                {loadingEnvVars ? (
                  <div className="text-center py-8 text-[var(--text-muted)]">
                    <div className="inline-block animate-spin rounded-full h-6 w-6 border-b-2 border-[var(--primary)] mb-2"></div>
                    <p>Loading environment variables...</p>
                  </div>
                ) : envVars.length === 0 ? (
                  <div className="text-center py-8 text-[var(--text-muted)]">
                    <p>No environment variables configured</p>
                  </div>
                ) : (
                  <div className="space-y-2">
                    {envVars.map((envVar) => (
                      <div
                        key={envVar.id}
                        className="flex items-center justify-between bg-[var(--elevated)] rounded-lg p-4 border border-[var(--border-subtle)]"
                      >
                        <div className="flex-1">
                          <div className="flex items-center space-x-4">
                            <span className="font-mono font-semibold text-[var(--text-primary)]">{envVar.key}</span>
                            <span className="text-[var(--text-muted)]">=</span>
                            <span className="font-mono text-[var(--text-secondary)]">{envVar.value}</span>
                          </div>
                        </div>
                        <button
                          onClick={() => handleDeleteEnvVar(envVar.key)}
                          className="ml-4 px-3 py-1 text-sm bg-[var(--error)] hover:bg-[var(--error)]/80 text-white rounded transition-colors"
                        >
                          Delete
                        </button>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          )}
        </div>

        {/* Helpful Hints */}
        {(app.status === 'failed' || app.status === 'error') && (
          <div className="mt-6 bg-[var(--warning)]/10 rounded-lg border border-[var(--warning)] p-4">
            <div className="flex items-start gap-3">
              <svg className="w-5 h-5 text-[var(--warning)] mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
              </svg>
              <div>
                <div className="font-medium text-[var(--text-primary)] mb-1">App deployment failed</div>
                <div className="text-sm text-[var(--text-secondary)]">
                  Check the Logs tab for error details. Common issues include missing Dockerfile, build errors, or configuration problems. 
                  After fixing the issue, use the Redeploy button to try again.
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
