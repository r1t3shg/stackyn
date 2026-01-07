import { useEffect, useState, useCallback } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { appsApi, deploymentsApi } from '@/lib/api';
import type { App, Deployment, EnvVar, DeploymentLogs } from '@/lib/types';
import StatusBadge from '@/components/StatusBadge';
import LogsViewer from '@/components/LogsViewer';
import { extractString } from '@/lib/types';

type Tab = 'overview' | 'deployments' | 'metrics' | 'settings';

export default function AppDetailsPage() {
  const params = useParams<{ id: string }>();
  const navigate = useNavigate();
  const appId = params.id!;

  const [app, setApp] = useState<App | null>(null);
  const [deployments, setDeployments] = useState<Deployment[]>([]);
  const [envVars, setEnvVars] = useState<EnvVar[]>([]);
  const [logs, setLogs] = useState<DeploymentLogs | null>(null);
  const [buildLogs, setBuildLogs] = useState<DeploymentLogs | null>(null);
  const [activeTab, setActiveTab] = useState<Tab>('overview');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [newEnvVars, setNewEnvVars] = useState<Array<{ key: string; value: string }>>([]);
  const [addingEnvVar, setAddingEnvVar] = useState(false);
  const [envVarsError, setEnvVarsError] = useState<string | null>(null);
  const [loadingEnvVars, setLoadingEnvVars] = useState(false);
  // Poll for app/deployment status updates (DB is single source of truth)
  useEffect(() => {
    // Only poll if app is in building/deploying state
    const hasActiveBuild = deployments.some(d => 
      d.status === 'building' || d.status === 'pending'
    );
    
    if (!hasActiveBuild) {
      return;
    }

    // Poll every 2 seconds while building
    const interval = setInterval(() => {
      loadApp();
      loadDeployments();
    }, 2000);

    return () => clearInterval(interval);
  }, [appId, deployments]);

  const loadBuildLogs = useCallback(async (deploymentId: number) => {
    try {
      const data = await deploymentsApi.getLogs(deploymentId);
      setBuildLogs(data);
    } catch (err) {
      console.error('Error loading build logs:', err);
    }
  }, []);

  // Poll for build logs during deployment
  useEffect(() => {
    // Find the active deployment that's building or pending
    const activeBuildingDeployment = deployments.find(d => 
      (d.status === 'building' || d.status === 'pending') &&
      d.id.toString() === app?.deployment?.active_deployment_id?.replace('dep_', '')
    );

    if (!activeBuildingDeployment) {
      // Clear build logs if no active building deployment
      setBuildLogs(null);
      return;
    }

    // Load build logs immediately
    loadBuildLogs(activeBuildingDeployment.id);

    // Poll for build logs every 2 seconds while building
    const interval = setInterval(() => {
      loadBuildLogs(activeBuildingDeployment.id);
    }, 2000);

    return () => clearInterval(interval);
  }, [appId, deployments, app?.deployment?.active_deployment_id, loadBuildLogs]);

  useEffect(() => {
    if (appId) {
      loadApp();
      loadDeployments();
      loadEnvVars();
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

  const loadLogs = useCallback(async () => {
    if (!app?.deployment?.active_deployment_id) return;
    try {
      const deploymentId = app.deployment.active_deployment_id.replace('dep_', '');
      const data = await deploymentsApi.getLogs(deploymentId);
      setLogs(data);
    } catch (err) {
      console.error('Error loading logs:', err);
      // Ensure UI doesn't get stuck on "Loading logs..." if request fails (e.g., 401)
      setLogs({
        deployment_id: 0,
        status: 'unknown',
        build_log: null,
        runtime_log: null,
        error_message: null,
      });
    }
  }, [app?.deployment?.active_deployment_id]);

  // Load runtime logs automatically when app is loaded
  useEffect(() => {
    if (app?.deployment?.active_deployment_id) {
      loadLogs();
      // Auto-refresh logs every 5 seconds
      const interval = setInterval(() => {
        loadLogs();
      }, 5000);
      return () => clearInterval(interval);
    } else {
      // Clear logs if no active deployment
      setLogs(null);
    }
  }, [app?.deployment?.active_deployment_id, loadLogs]);

  // Refresh app data periodically when viewing metrics tab to get fresh usage stats
  useEffect(() => {
    if (activeTab === 'metrics') {
      // Load fresh app data immediately when switching to metrics tab
      loadApp();
      // Auto-refresh metrics every 10 seconds
      const interval = setInterval(() => {
        loadApp();
        loadDeployments();
      }, 10000);
      return () => clearInterval(interval);
    }
  }, [activeTab, appId]);

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

  const addNewEnvVarField = () => {
    setNewEnvVars([...newEnvVars, { key: '', value: '' }]);
  };

  const removeNewEnvVarField = (index: number) => {
    setNewEnvVars(newEnvVars.filter((_, i) => i !== index));
  };

  const updateNewEnvVar = (index: number, field: 'key' | 'value', value: string) => {
    const updated = [...newEnvVars];
    updated[index] = { ...updated[index], [field]: value };
    setNewEnvVars(updated);
  };

  const handleAddEnvVars = async () => {
    const validEnvVars = newEnvVars.filter(env => env.key.trim() !== '');
    if (validEnvVars.length === 0) {
      alert('Please add at least one environment variable with a key');
      return;
    }
    
    setAddingEnvVar(true);
    setEnvVarsError(null);
    try {
      const promises = validEnvVars.map(env => 
        appsApi.createEnvVar(appId, { key: env.key.trim(), value: env.value })
      );
      await Promise.all(promises);
      setNewEnvVars([]);
      await loadEnvVars();
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to add environment variables';
      setEnvVarsError(errorMessage);
      console.error('Error adding environment variables:', err);
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
                {/* Show deployment status if building/pending */}
                {deployments.length > 0 && (deployments[0].status === 'building' || deployments[0].status === 'pending') && (
                  <div className="flex items-center gap-2 text-sm text-[var(--text-secondary)]">
                    <div className="inline-block animate-spin rounded-full h-4 w-4 border-b-2 border-[var(--primary)]"></div>
                    <span>Building...</span>
                  </div>
                )}
                {/* Show error message if app failed or in error state */}
                {(app.status === 'failed' || app.status === 'error') && (
                  <div className="flex items-center gap-2 text-sm text-[var(--error)]">
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                    <span>{app.status === 'error' ? 'Application not accessible' : 'Application failed to start'}</span>
                  </div>
                )}
                {/* Show error message if deployment failed */}
                {deployments.length > 0 && deployments[0].status === 'failed' && deployments[0].error_message && (
                  <div className="flex items-center gap-2 text-sm text-[var(--error)]">
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                    <span>{extractString(deployments[0].error_message)}</span>
                  </div>
                )}
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
              <div className="text-xs text-[var(--text-muted)] mb-1">RAM Used</div>
              <div className="text-lg font-semibold text-[var(--text-primary)]">
                {app.deployment?.usage_stats?.memory_usage_mb !== undefined
                  ? `${app.deployment.usage_stats.memory_usage_mb} MB`
                  : 'N/A'}
              </div>
            </div>
            <div>
              <div className="text-xs text-[var(--text-muted)] mb-1">Disk Used</div>
              <div className="text-lg font-semibold text-[var(--text-primary)]">
                {app.deployment?.usage_stats?.disk_usage_gb !== undefined
                  ? `${app.deployment.usage_stats.disk_usage_gb.toFixed(2)} GB`
                  : 'N/A'}
              </div>
            </div>
            <div>
              <div className="text-xs text-[var(--text-muted)] mb-1">Container Status</div>
              <div className="text-lg font-semibold text-[var(--text-primary)]">
                {app.status || 'Unknown'}
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

        {/* Runtime Logs Section */}
        <div className="bg-[var(--surface)] rounded-lg border border-[var(--border-subtle)] p-6 mb-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-xl font-semibold text-[var(--text-primary)]">Runtime Logs</h2>
            <button
              onClick={loadLogs}
              className="px-3 py-1 text-sm bg-[var(--surface)] hover:bg-[var(--elevated)] text-[var(--text-primary)] border border-[var(--border-subtle)] rounded transition-colors flex items-center gap-2"
              title="Refresh logs"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
              Refresh
            </button>
          </div>
          {logs ? (
            <div>
              {extractString(logs.runtime_log) ? (
                <LogsViewer logs={extractString(logs.runtime_log)} title="" />
              ) : (
                <div className="text-center py-8 text-[var(--text-muted)] bg-[var(--elevated)] rounded-lg border border-[var(--border-subtle)]">
                  <p>No runtime logs available yet. Logs will appear here once the application starts.</p>
                </div>
              )}
            </div>
          ) : (
            <div className="text-center py-8 text-[var(--text-muted)] bg-[var(--elevated)] rounded-lg border border-[var(--border-subtle)]">
              <div className="flex items-center justify-center gap-2">
                <div className="inline-block animate-spin rounded-full h-4 w-4 border-b-2 border-[var(--primary)]"></div>
                <p>Loading logs...</p>
              </div>
            </div>
          )}
        </div>

        {/* Build Logs Section - Show during deployment */}
        {(() => {
          const activeBuildingDeployment = deployments.find(d => 
            (d.status === 'building' || d.status === 'pending') &&
            d.id.toString() === app?.deployment?.active_deployment_id?.replace('dep_', '')
          );

          if (!activeBuildingDeployment) {
            return null;
          }

          return (
            <div className="bg-[var(--surface)] rounded-lg border border-[var(--border-subtle)] p-6 mb-6">
              <div className="flex items-center justify-between mb-4">
                <div className="flex items-center gap-3">
                  <div className="inline-block animate-spin rounded-full h-5 w-5 border-b-2 border-[var(--primary)]"></div>
                  <h2 className="text-xl font-semibold text-[var(--text-primary)]">
                    Build Logs - Deployment #{activeBuildingDeployment.id}
                  </h2>
                  <StatusBadge status={activeBuildingDeployment.status} />
                </div>
              </div>
              
              {buildLogs && extractString(buildLogs.build_log) ? (
                <LogsViewer logs={extractString(buildLogs.build_log)} title="Build Logs" />
              ) : buildLogs && buildLogs.build_log === null ? (
                <div className="bg-[var(--elevated)] rounded-lg p-4 border border-[var(--border-subtle)]">
                  <p className="text-sm text-[var(--text-muted)]">Build logs will appear here as the build progresses...</p>
                </div>
              ) : (
                <div className="bg-[var(--elevated)] rounded-lg p-4 border border-[var(--border-subtle)]">
                  <div className="flex items-center gap-2">
                    <div className="inline-block animate-spin rounded-full h-4 w-4 border-b-2 border-[var(--primary)]"></div>
                    <p className="text-sm text-[var(--text-muted)]">Loading build logs...</p>
                  </div>
                </div>
              )}

              {buildLogs?.error_message && extractString(buildLogs.error_message) && (
                <div className="mt-4 p-4 bg-[var(--error)]/10 border border-[var(--error)] rounded-lg">
                  <h3 className="text-sm font-medium text-[var(--error)] mb-2">Build Error</h3>
                  <p className="text-sm text-[var(--error)]">{extractString(buildLogs.error_message)}</p>
                </div>
              )}
            </div>
          );
        })()}

        {/* Tabs */}
        <div className="mb-6">
          <div className="border-b border-[var(--border-subtle)]">
            <nav className="flex space-x-8 overflow-x-auto">
              {(['overview', 'deployments', 'metrics', 'settings'] as Tab[]).map((tab) => (
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

          {/* Metrics Tab */}
          {activeTab === 'metrics' && (
            <div className="space-y-6">
              <div className="bg-[var(--surface)] rounded-lg border border-[var(--border-subtle)] p-6">
                <div className="flex items-center justify-between mb-4">
                  <h2 className="text-xl font-semibold text-[var(--text-primary)]">Resource Usage</h2>
                  <button
                    onClick={() => {
                      loadApp();
                      loadDeployments();
                    }}
                    className="px-3 py-1 text-sm bg-[var(--surface)] hover:bg-[var(--elevated)] text-[var(--text-primary)] border border-[var(--border-subtle)] rounded transition-colors flex items-center gap-2"
                    title="Refresh metrics"
                  >
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                    </svg>
                    Refresh
                  </button>
                </div>
              {loading ? (
                <div className="text-center py-8">
                  <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-[var(--primary)] mb-4"></div>
                  <p className="text-[var(--text-muted)]">Loading metrics...</p>
                </div>
              ) : app.deployment?.usage_stats ? (
                <>
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
                          {app.deployment.usage_stats.memory_usage_mb} MB {app.deployment.resource_limits?.memory_mb ? `/ ${app.deployment.resource_limits.memory_mb} MB` : ''}
                        </div>
                      </div>

                      <div className="bg-[var(--elevated)] rounded-lg p-4 border border-[var(--border-subtle)]">
                        <div className="text-sm text-[var(--text-muted)] mb-1">CPU Allocation</div>
                        <div className="text-2xl font-bold text-[var(--text-primary)]">
                          {app.deployment.resource_limits?.cpu ? `${app.deployment.resource_limits.cpu} vCPU` : 'N/A'}
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

                  <div className="bg-[var(--elevated)] rounded-lg p-4 border border-[var(--border-subtle)] mt-4">
                    <h3 className="text-lg font-semibold text-[var(--text-primary)] mb-4">Disk Usage</h3>
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
                      {app.deployment.usage_stats.disk_usage_gb.toFixed(2)} GB {app.deployment.resource_limits?.disk_gb ? `/ ${app.deployment.resource_limits.disk_gb} GB` : ''}
                    </div>
                  </div>
                </>
              ) : (
                <div className="text-center py-8">
                  <p className="text-[var(--text-muted)]">No metrics available yet. Metrics will appear after the app is deployed.</p>
                </div>
              )}
              </div>
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
                </div>

                {envVarsError && (
                  <div className="bg-[var(--error)]/10 border border-[var(--error)] rounded-lg p-4 mb-4">
                    <p className="text-[var(--error)] text-sm">{envVarsError}</p>
                  </div>
                )}

                {/* Add New Environment Variables */}
                <div className="mb-6">
                  <div className="flex items-center justify-between mb-3">
                    <label className="block text-sm font-medium text-[var(--text-secondary)]">
                      Add New Variables
                    </label>
                    <button
                      type="button"
                      onClick={addNewEnvVarField}
                      className="text-sm px-3 py-1 bg-[var(--surface)] hover:bg-[var(--elevated)] text-[var(--primary)] border border-[var(--border-subtle)] rounded-lg transition-colors"
                    >
                      + Add Variable
                    </button>
                  </div>
                  
                  {newEnvVars.length === 0 ? (
                    <div className="text-center py-4 text-[var(--text-muted)] bg-[var(--elevated)] border border-[var(--border-subtle)] rounded-lg">
                      <p className="text-sm">Click "+ Add Variable" to add environment variables</p>
                    </div>
                  ) : (
                    <div className="bg-[var(--elevated)] rounded-lg p-4 border border-[var(--border-subtle)] space-y-3">
                      {newEnvVars.map((envVar, index) => (
                        <div key={index} className="flex gap-2 items-start">
                          <div className="flex-1">
                            <input
                              type="text"
                              value={envVar.key}
                              onChange={(e) => updateNewEnvVar(index, 'key', e.target.value)}
                              placeholder="KEY"
                              className="w-full px-3 py-2 bg-[var(--surface)] border border-[var(--border-subtle)] rounded-lg focus:outline-none focus:border-[var(--focus-border)] text-[var(--text-primary)] font-mono text-sm"
                            />
                          </div>
                          <div className="flex-1">
                            <input
                              type="text"
                              value={envVar.value}
                              onChange={(e) => updateNewEnvVar(index, 'value', e.target.value)}
                              placeholder="value"
                              className="w-full px-3 py-2 bg-[var(--surface)] border border-[var(--border-subtle)] rounded-lg focus:outline-none focus:border-[var(--focus-border)] text-[var(--text-primary)] font-mono text-sm"
                            />
                          </div>
                          <button
                            type="button"
                            onClick={() => removeNewEnvVarField(index)}
                            className="px-3 py-2 bg-[var(--error)]/10 hover:bg-[var(--error)]/20 text-[var(--error)] rounded-lg transition-colors"
                            title="Remove variable"
                          >
                            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                            </svg>
                          </button>
                        </div>
                      ))}
                      <button
                        type="button"
                        onClick={handleAddEnvVars}
                        disabled={addingEnvVar || newEnvVars.filter(env => env.key.trim() !== '').length === 0}
                        className="w-full px-4 py-2 bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-[var(--app-bg)] font-medium rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                      >
                        {addingEnvVar ? 'Adding Variables...' : `Add ${newEnvVars.filter(env => env.key.trim() !== '').length} Variable(s)`}
                      </button>
                    </div>
                  )}
                </div>

                {/* Existing Environment Variables */}
                <div className="border-t border-[var(--border-subtle)] pt-4">
                  <h3 className="text-sm font-medium text-[var(--text-secondary)] mb-3">Existing Variables</h3>

                  {loadingEnvVars ? (
                    <div className="text-center py-8 text-[var(--text-muted)]">
                      <div className="inline-block animate-spin rounded-full h-6 w-6 border-b-2 border-[var(--primary)] mb-2"></div>
                      <p>Loading environment variables...</p>
                    </div>
                  ) : envVars.length === 0 ? (
                    <div className="text-center py-4 text-[var(--text-muted)] bg-[var(--elevated)] border border-[var(--border-subtle)] rounded-lg">
                      <p className="text-sm">No environment variables configured</p>
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
            </div>
          )}
        </div>

        {/* Error Message Section */}
        {(app.status === 'failed' || app.status === 'error') && (
          <div className="mt-6 bg-[var(--error)]/10 rounded-lg border border-[var(--error)] p-4">
            <div className="flex items-start gap-3">
              <svg className="w-5 h-5 text-[var(--error)] mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <div className="flex-1">
                <div className="font-medium text-[var(--error)] mb-2">
                  {app.status === 'error' ? 'Application Not Accessible' : 'Application Failed to Start'}
                </div>
                {deployments.length > 0 && deployments[0].error_message && extractString(deployments[0].error_message) && (
                  <div className="bg-[var(--surface)] rounded border border-[var(--border-subtle)] p-3 mb-2 font-mono text-sm text-[var(--text-primary)] whitespace-pre-wrap break-words">
                    {extractString(deployments[0].error_message)}
                  </div>
                )}
                <div className="text-sm text-[var(--text-secondary)] mb-3">
                  {app.status === 'error' ? (
                    <>
                      The application container is running but cannot be accessed through its URL. This could be due to SSL certificate issues, routing problems, or the application not responding correctly.
                    </>
                  ) : (
                    <>
                      The application container crashed during startup. Check the build logs and runtime logs tabs for detailed error information.
                    </>
                  )}
                </div>
                <div className="text-sm text-[var(--text-secondary)] mb-3">
                  <strong>Common causes:</strong> {
                    app.status === 'error' 
                      ? 'SSL certificate not issued, Traefik routing misconfigured, application not listening on expected port, DNS issues, or container health check failures.'
                      : 'Missing configuration files, environment variables not set, incorrect port binding, application startup errors, or missing dependencies.'
                  }
                </div>
                <button
                  onClick={handleRedeploy}
                  disabled={actionLoading !== null}
                  className="px-4 py-2 bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-[var(--app-bg)] font-medium rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {actionLoading === 'redeploy' ? 'Redeploying...' : 'Redeploy App'}
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
