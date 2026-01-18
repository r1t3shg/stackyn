import { useEffect, useState, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { appsApi, userApi } from '@/lib/api';
import type { App, UserProfile } from '@/lib/types';
import StatusBadge from '@/components/StatusBadge';
import TrialBanner from '@/components/TrialBanner';
import Paywall from '@/components/Paywall';
import BillingTestPanel from '@/components/BillingTestPanel';
import ConfirmModal from '@/components/ConfirmModal';
import { shouldShowPaywall } from '@/lib/billing';
import { useAuth } from '@/contexts/AuthContext';

type SortField = 'name' | 'status' | 'last_deployed' | 'created_at';
type SortDirection = 'asc' | 'desc';
type StatusFilter = 'all' | 'running' | 'deploying' | 'stopped' | 'error';

export default function Home() {
  const [apps, setApps] = useState<App[]>([]);
  const [userProfile, setUserProfile] = useState<UserProfile | null>(null);
  const [loading, setLoading] = useState(true);
  const [profileLoading, setProfileLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');
  const [sortField, setSortField] = useState<SortField>('created_at');
  const [sortDirection, setSortDirection] = useState<SortDirection>('desc');
  const [showLogoutConfirm, setShowLogoutConfirm] = useState(false);
  const { user, logout } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    loadApps();
    loadUserProfile();
  }, []);

  const loadApps = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await appsApi.list();
      setApps(Array.isArray(data) ? data : []);
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to load apps';
      
      // If it's a 401 error, redirect to login using React Router (no page refresh)
      // Check for various 401 error message formats
      if (errorMessage.includes('401') || 
          errorMessage.includes('Unauthorized') || 
          errorMessage.toLowerCase().includes('unauthorized')) {
        console.log('Unauthorized access - redirecting to login');
        // Clear any stale auth data
        localStorage.removeItem('auth_token');
        localStorage.removeItem('auth_user');
        // Use React Router navigation (no page refresh)
        navigate('/login', { replace: true });
        return;
      }
      
      setError(errorMessage);
      console.error('Error loading apps:', err);
      setApps([]);
    } finally {
      setLoading(false);
    }
  };

  const loadUserProfile = async () => {
    try {
      setProfileLoading(true);
      const profile = await userApi.getProfile();
      console.log('Loaded user profile:', profile);
      console.log('Subscription data:', profile.subscription);
      setUserProfile(profile);
    } catch (err) {
      console.error('Error loading user profile:', err);
      // Don't show error for profile - it's not critical
    } finally {
      setProfileLoading(false);
    }
  };


  // Calculate actual RAM and disk usage from all apps
  // Falls back to resource_limits if usage_stats is not available
  const actualUsage = useMemo(() => {
    let totalRamMb = 0;
    let totalDiskMb = 0;

    apps.forEach((app) => {
      if (app.deployment?.usage_stats) {
        // Only sum actual usage stats, not allocated resources
        const ramUsage = app.deployment.usage_stats.memory_usage_mb || 0;
        totalRamMb += ramUsage;
        // Sum disk usage (convert GB to MB)
        const diskUsageGb = app.deployment.usage_stats.disk_usage_gb || 0;
        totalDiskMb += diskUsageGb * 1024;
      }
    });

    return {
      ramMb: Math.round(totalRamMb),
      diskMb: Math.round(totalDiskMb),
    };
  }, [apps]);

  // Filter and sort apps
  const filteredAndSortedApps = useMemo(() => {
    let filtered = apps;

    // Apply search filter
    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      filtered = filtered.filter(
        (app) =>
          app.name.toLowerCase().includes(query) ||
          app.repo_url.toLowerCase().includes(query) ||
          app.url?.toLowerCase().includes(query)
      );
    }

    // Apply status filter
    if (statusFilter !== 'all') {
      filtered = filtered.filter((app) => {
        const status = app.status?.toLowerCase() || '';
        if (statusFilter === 'running') return status === 'running' || status === 'healthy';
        if (statusFilter === 'deploying') return status === 'pending' || status === 'building' || status === 'deploying';
        if (statusFilter === 'stopped') return status === 'stopped';
        if (statusFilter === 'error') return status === 'failed' || status === 'error';
        return true;
      });
    }

    // Apply sorting
    filtered = [...filtered].sort((a, b) => {
      let aValue: any;
      let bValue: any;

      if (sortField === 'name') {
        aValue = a.name.toLowerCase();
        bValue = b.name.toLowerCase();
      } else if (sortField === 'status') {
        aValue = a.status?.toLowerCase() || '';
        bValue = b.status?.toLowerCase() || '';
      } else if (sortField === 'last_deployed') {
        aValue = a.deployment?.last_deployed_at ? new Date(a.deployment.last_deployed_at).getTime() : 0;
        bValue = b.deployment?.last_deployed_at ? new Date(b.deployment.last_deployed_at).getTime() : 0;
      } else {
        // created_at
        aValue = new Date(a.created_at).getTime();
        bValue = new Date(b.created_at).getTime();
      }

      if (aValue < bValue) return sortDirection === 'asc' ? -1 : 1;
      if (aValue > bValue) return sortDirection === 'asc' ? 1 : -1;
      return 0;
    });

    return filtered;
  }, [apps, searchQuery, statusFilter, sortField, sortDirection]);

  const formatDate = (dateString: string | null | undefined) => {
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


  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc');
    } else {
      setSortField(field);
      setSortDirection('desc');
    }
  };

  const getPlanDisplayName = (plan: string) => {
    const planNames: Record<string, string> = {
      free: 'Free',
      starter: 'Starter',
      builder: 'Builder',
      pro: 'Pro',
    };
    return planNames[plan.toLowerCase()] || plan;
  };

  return (
    <div className="min-h-screen bg-[var(--app-bg)]">
      {/* Trial Banner */}
      <TrialBanner userProfile={userProfile} />
      
      {/* Billing Test Panel (dev only) */}
      <BillingTestPanel onUpdate={loadUserProfile} />
      
      {/* Show paywall if billing is inactive - only show after profile is loaded */}
      {!profileLoading && shouldShowPaywall(userProfile) && (
        <Paywall userProfile={userProfile} />
      )}
      
      {/* Hide main content if paywall is shown - wait for profile to load */}
      {!profileLoading && !shouldShowPaywall(userProfile) && (
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* User Profile CTA Section */}
        {userProfile && !profileLoading && (
          <div className="mb-8 bg-gradient-to-r from-[var(--primary)]/10 to-[var(--primary)]/5 border border-[var(--primary)]/20 rounded-xl p-6">
            <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
              <div className="flex-1">
                <div className="flex items-center gap-3 mb-3">
                  <div className="w-12 h-12 rounded-full bg-[var(--primary)]/20 flex items-center justify-center">
                    <svg className="w-6 h-6 text-[var(--primary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
                    </svg>
                  </div>
                  <div>
                    <h2 className="text-xl font-bold text-[var(--text-primary)]">
                      {userProfile.full_name || userProfile.email}
                    </h2>
                    {userProfile.company_name && (
                      <p className="text-sm text-[var(--text-secondary)]">{userProfile.company_name}</p>
                    )}
                  </div>
                </div>
                <div className="flex flex-wrap gap-4 text-sm">
                  <div className="flex items-center gap-2">
                    <span className="text-[var(--text-secondary)]">Plan:</span>
                    <span className="font-semibold text-[var(--primary)]">
                      {getPlanDisplayName(userProfile.plan)}
                    </span>
                  </div>
                  {userProfile.quota && (
                    <>
                      <div className="flex items-center gap-2">
                        <span className="text-[var(--text-secondary)]">Apps:</span>
                        <span className="font-semibold text-[var(--text-primary)]">
                          {userProfile.quota.app_count} / {userProfile.quota.plan.max_apps}
                        </span>
                      </div>
                      <div className="flex items-center gap-2">
                        <span className="text-[var(--text-secondary)]">RAM:</span>
                        <span className="font-semibold text-[var(--text-primary)]">
                          {actualUsage.ramMb} MB / {userProfile.quota.plan.max_ram_mb} MB
                        </span>
                      </div>
                      <div className="flex items-center gap-2">
                        <span className="text-[var(--text-secondary)]">Disk:</span>
                        <span className="font-semibold text-[var(--text-primary)]">
                          {actualUsage.diskMb} MB / {userProfile.quota.plan.max_disk_mb} MB
                        </span>
                      </div>
                    </>
                  )}
                </div>
              </div>
              <div className="flex gap-3">
                {userProfile.subscription?.status === 'expired' && (
                  <button
                    onClick={() => window.location.href = '/pricing'}
                    className="bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-[var(--app-bg)] font-medium py-2 px-6 rounded-lg transition-colors flex items-center gap-2"
                  >
                    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                    </svg>
                    Upgrade Plan
                  </button>
                )}
              </div>
            </div>
          </div>
        )}

        {/* Header */}
        <div className="flex items-center justify-between mb-8">
          <div>
            <h1 className="text-3xl font-bold text-[var(--text-primary)]">My Applications</h1>
            {user && (
              <p className="text-sm text-[var(--text-secondary)] mt-1">Signed in as {user.email}</p>
            )}
          </div>
          <div className="flex gap-3">
            <button
              onClick={() => navigate('/apps/new')}
              className="bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-[var(--app-bg)] font-medium py-2 px-6 rounded-lg transition-colors flex items-center gap-2"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
              </svg>
              New App
            </button>
            <button
              onClick={() => setShowLogoutConfirm(true)}
              className="bg-[var(--surface)] hover:bg-[var(--elevated)] text-[var(--text-primary)] border border-[var(--border-subtle)] font-medium py-2 px-4 rounded-lg transition-colors"
            >
              Logout
            </button>
          </div>
        </div>

        {/* Logout Confirmation Modal */}
        <ConfirmModal
          isOpen={showLogoutConfirm}
          title="Confirm Logout"
          message="Are you sure you want to log out? You'll need to sign in again to access your applications."
          confirmText="Logout"
          cancelText="Cancel"
          onConfirm={async () => {
            setShowLogoutConfirm(false);
            await logout();
          }}
          onCancel={() => setShowLogoutConfirm(false)}
          variant="default"
        />

        {/* Search, Filter, and Sort Controls */}
        <div className="mb-6 flex flex-col sm:flex-row gap-4">
          <div className="flex-1">
            <div className="relative">
              <svg
                className="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-[var(--text-muted)]"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
              </svg>
              <input
                type="text"
                placeholder="Search apps by name, repository, or URL..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="w-full pl-10 pr-4 py-2 bg-[var(--surface)] border border-[var(--border-subtle)] rounded-lg focus:border-[var(--focus-border)] text-[var(--text-primary)]"
              />
            </div>
          </div>
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value as StatusFilter)}
            className="px-4 py-2 bg-[var(--surface)] border border-[var(--border-subtle)] rounded-lg focus:border-[var(--focus-border)] text-[var(--text-primary)]"
          >
            <option value="all">All Status</option>
            <option value="running">Running</option>
            <option value="deploying">Deploying</option>
            <option value="stopped">Stopped</option>
            <option value="error">Error</option>
          </select>
          <select
            value={`${sortField}_${sortDirection}`}
            onChange={(e) => {
              const [field, direction] = e.target.value.split('_');
              setSortField(field as SortField);
              setSortDirection(direction as SortDirection);
            }}
            className="px-4 py-2 bg-[var(--surface)] border border-[var(--border-subtle)] rounded-lg focus:border-[var(--focus-border)] text-[var(--text-primary)]"
          >
            <option value="created_at_desc">Newest First</option>
            <option value="created_at_asc">Oldest First</option>
            <option value="name_asc">Name (A-Z)</option>
            <option value="name_desc">Name (Z-A)</option>
            <option value="status_asc">Status</option>
            <option value="last_deployed_desc">Last Deployed</option>
          </select>
        </div>

        {/* Loading State */}
        {loading && (
          <div className="text-center py-12">
            <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-[var(--primary)]"></div>
            <p className="mt-4 text-[var(--text-secondary)]">Loading apps...</p>
          </div>
        )}

        {/* Error State */}
        {error && (
          <div className="bg-[var(--error)]/10 border border-[var(--error)] rounded-lg p-6 mb-6">
            <p className="text-[var(--error)] font-medium mb-2">Error: {error}</p>
            <button
              onClick={loadApps}
              className="bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-[var(--app-bg)] font-medium py-2 px-4 rounded-lg transition-colors"
            >
              Try again
            </button>
          </div>
        )}

        {/* Empty State */}
        {!loading && !error && filteredAndSortedApps.length === 0 && (
          <div className="text-center py-16 bg-[var(--surface)] rounded-lg border border-[var(--border-subtle)]">
            <svg
              className="mx-auto h-16 w-16 text-[var(--text-muted)] mb-4"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
            </svg>
            <h3 className="text-xl font-semibold text-[var(--text-primary)] mb-2">
              {searchQuery || statusFilter !== 'all' ? 'No apps match your filters' : 'No applications yet'}
            </h3>
            <p className="text-[var(--text-secondary)] mb-6 max-w-md mx-auto">
              {searchQuery || statusFilter !== 'all'
                ? 'Try adjusting your search or filter criteria.'
                : 'Deploy your first app in seconds by connecting a Git repository. Stackyn will build, run, and expose your app automatically.'}
            </p>
            {(!searchQuery && statusFilter === 'all') && (
              <button
                onClick={() => navigate('/apps/new')}
                className="inline-flex items-center gap-2 bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-[var(--app-bg)] font-medium py-3 px-6 rounded-lg transition-colors"
              >
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                </svg>
                Deploy Your First App
              </button>
            )}
          </div>
        )}

        {/* Apps Table */}
        {!loading && !error && filteredAndSortedApps.length > 0 && (
          <div className="bg-[var(--surface)] rounded-lg border border-[var(--border-subtle)] overflow-hidden">
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead className="bg-[var(--elevated)] border-b border-[var(--border-subtle)]">
                  <tr>
                    <th
                      className="px-6 py-3 text-left text-xs font-medium text-[var(--text-secondary)] uppercase tracking-wider cursor-pointer hover:bg-[var(--surface)] transition-colors"
                      onClick={() => handleSort('name')}
                    >
                      <div className="flex items-center gap-2">
                        Name
                        {sortField === 'name' && (
                          <svg
                            className={`w-4 h-4 ${sortDirection === 'asc' ? '' : 'transform rotate-180'}`}
                            fill="none"
                            stroke="currentColor"
                            viewBox="0 0 24 24"
                          >
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 15l7-7 7 7" />
                          </svg>
                        )}
                      </div>
                    </th>
                    <th
                      className="px-6 py-3 text-left text-xs font-medium text-[var(--text-secondary)] uppercase tracking-wider cursor-pointer hover:bg-[var(--surface)] transition-colors"
                      onClick={() => handleSort('status')}
                    >
                      <div className="flex items-center gap-2">
                        Status
                        {sortField === 'status' && (
                          <svg
                            className={`w-4 h-4 ${sortDirection === 'asc' ? '' : 'transform rotate-180'}`}
                            fill="none"
                            stroke="currentColor"
                            viewBox="0 0 24 24"
                          >
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 15l7-7 7 7" />
                          </svg>
                        )}
                      </div>
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-[var(--text-secondary)] uppercase tracking-wider">
                      Live URL
                    </th>
                    <th
                      className="px-6 py-3 text-left text-xs font-medium text-[var(--text-secondary)] uppercase tracking-wider cursor-pointer hover:bg-[var(--surface)] transition-colors"
                      onClick={() => handleSort('last_deployed')}
                    >
                      <div className="flex items-center gap-2">
                        Last Deployed
                        {sortField === 'last_deployed' && (
                          <svg
                            className={`w-4 h-4 ${sortDirection === 'asc' ? '' : 'transform rotate-180'}`}
                            fill="none"
                            stroke="currentColor"
                            viewBox="0 0 24 24"
                          >
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 15l7-7 7 7" />
                          </svg>
                        )}
                      </div>
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-[var(--border-subtle)]">
                  {filteredAndSortedApps.map((app) => (
                    <tr
                      key={app.id}
                      className="hover:bg-[var(--elevated)] transition-colors cursor-pointer"
                      onClick={() => navigate(`/apps/${app.id}`)}
                    >
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div className="font-medium text-[var(--text-primary)]">{app.name}</div>
                        <div className="text-sm text-[var(--text-muted)] mt-1">{app.repo_url}</div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <StatusBadge status={app.status || 'unknown'} />
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        {app.url ? (
                          <a
                            href={app.url}
                            target="_blank"
                            rel="noopener noreferrer"
                            onClick={(e) => e.stopPropagation()}
                            className="text-[var(--info)] hover:text-[var(--primary)] text-sm transition-colors flex items-center gap-1"
                          >
                            {app.url}
                            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                            </svg>
                          </a>
                        ) : (
                          <span className="text-[var(--text-muted)] text-sm">—</span>
                        )}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-[var(--text-secondary)]">
                        {formatDate(app.deployment?.last_deployed_at || app.updated_at)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {/* Results Count */}
        {!loading && !error && filteredAndSortedApps.length > 0 && (
          <div className="mt-4 text-sm text-[var(--text-muted)]">
            Showing {filteredAndSortedApps.length} of {apps.length} app{apps.length !== 1 ? 's' : ''}
            {userProfile?.quota && (
              <span className="ml-4">
                ({userProfile.quota.app_count} published)
              </span>
            )}
          </div>
        )}
      </div>
      )}
    </div>
  );
}
