import { useEffect, useState, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { appsApi } from '@/lib/api';
import type { App } from '@/lib/types';
import NewAppModal from '@/components/NewAppModal';
import StatusBadge from '@/components/StatusBadge';
import { useAuth } from '@/contexts/AuthContext';

type SortField = 'name' | 'status' | 'last_deployed' | 'created_at';
type SortDirection = 'asc' | 'desc';
type StatusFilter = 'all' | 'running' | 'deploying' | 'stopped' | 'error';

export default function Home() {
  const [apps, setApps] = useState<App[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isNewAppModalOpen, setIsNewAppModalOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');
  const [sortField, setSortField] = useState<SortField>('created_at');
  const [sortDirection, setSortDirection] = useState<SortDirection>('desc');
  const { user, logout } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    loadApps();
  }, []);

  const loadApps = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await appsApi.list();
      setApps(Array.isArray(data) ? data : []);
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to load apps';
      setError(errorMessage);
      console.error('Error loading apps:', err);
      setApps([]);
    } finally {
      setLoading(false);
    }
  };


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

  return (
    <div className="min-h-screen bg-[var(--app-bg)]">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
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
              onClick={() => setIsNewAppModalOpen(true)}
              className="bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-[var(--app-bg)] font-medium py-2 px-6 rounded-lg transition-colors flex items-center gap-2"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
              </svg>
              New App
            </button>
            <button
              onClick={logout}
              className="bg-[var(--surface)] hover:bg-[var(--elevated)] text-[var(--text-primary)] border border-[var(--border-subtle)] font-medium py-2 px-4 rounded-lg transition-colors"
            >
              Logout
            </button>
          </div>
        </div>

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
                className="w-full pl-10 pr-4 py-2 bg-[var(--surface)] border border-[var(--border-subtle)] rounded-lg focus:ring-1 focus:ring-[var(--focus-border)] focus:border-[var(--focus-border)] text-[var(--text-primary)]"
              />
            </div>
          </div>
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value as StatusFilter)}
            className="px-4 py-2 bg-[var(--surface)] border border-[var(--border-subtle)] rounded-lg focus:ring-1 focus:ring-[var(--focus-border)] focus:border-[var(--focus-border)] text-[var(--text-primary)]"
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
            className="px-4 py-2 bg-[var(--surface)] border border-[var(--border-subtle)] rounded-lg focus:ring-1 focus:ring-[var(--focus-border)] focus:border-[var(--focus-border)] text-[var(--text-primary)]"
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
                onClick={() => setIsNewAppModalOpen(true)}
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
          </div>
        )}
      </div>

      {/* New App Modal */}
      <NewAppModal
        isOpen={isNewAppModalOpen}
        onClose={() => setIsNewAppModalOpen(false)}
        onAppCreated={async () => {
          setIsNewAppModalOpen(false);
          await loadApps();
        }}
      />
    </div>
  );
}
