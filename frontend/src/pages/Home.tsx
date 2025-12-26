import { useEffect, useState } from 'react';
import { appsApi, healthCheck } from '@/lib/api';
import type { App } from '@/lib/types';
import AppCard from '@/components/AppCard';
import NewAppModal from '@/components/NewAppModal';
import { API_BASE_URL } from '@/lib/config';
import { useAuth } from '@/contexts/AuthContext';

export default function Home() {
  const [apps, setApps] = useState<App[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [healthStatus, setHealthStatus] = useState<string | null>(null);
  const [isNewAppModalOpen, setIsNewAppModalOpen] = useState(false);
  const { user, logout } = useAuth();

  useEffect(() => {
    // Test backend connection first
    testBackendConnection();
    loadApps();
  }, []);

  const testBackendConnection = async () => {
    try {
      const health = await healthCheck();
      setHealthStatus(`Backend is healthy: ${health.status}`);
      console.log('Backend health check passed:', health);
    } catch (err) {
      setHealthStatus('Backend health check failed');
      console.error('Backend health check error:', err);
    }
  };

  const loadApps = async () => {
    try {
      setLoading(true);
      setError(null);
      console.log('Loading apps from:', `${API_BASE_URL}/api/apps`);
      const data = await appsApi.list();
      console.log('Apps loaded successfully:', data);
      // Ensure data is always an array, never null or undefined
      setApps(Array.isArray(data) ? data : []);
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to load apps';
      setError(errorMessage);
      console.error('Error loading apps:', err);
      // Set empty array on error to prevent null reference
      setApps([]);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-[var(--app-bg)]">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
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
              className="bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-[var(--app-bg)] font-medium py-2 px-4 rounded-lg transition-colors"
            >
              + Create New App
            </button>
            <button
              onClick={logout}
              className="bg-[var(--surface)] hover:bg-[var(--elevated)] text-[var(--text-primary)] border border-[var(--border-subtle)] font-medium py-2 px-4 rounded-lg transition-colors"
            >
              Logout
            </button>
          </div>
        </div>

        {loading && (
          <div className="text-center py-12">
            <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-[var(--primary)]"></div>
            <p className="mt-4 text-[var(--text-secondary)]">Loading apps...</p>
          </div>
        )}

        {healthStatus && (
          <div className={`border rounded-lg p-3 mb-4 text-sm ${
            healthStatus.includes('healthy') 
              ? 'bg-[var(--primary-muted)] border-[var(--success)] text-[var(--success)]' 
              : 'bg-[var(--warning)]/10 border-[var(--warning)] text-[var(--warning)]'
          }`}>
            {healthStatus}
          </div>
        )}

        {error && (
          <div className="bg-[var(--error)]/10 border border-[var(--error)] rounded-lg p-6 mb-6">
            <p className="text-[var(--error)] font-medium mb-2">Error: {error}</p>
            <p className="text-[var(--error)]/90 text-sm mb-4">
              Make sure the backend server is running on port 8080. Check:
            </p>
            <ul className="text-[var(--error)]/90 text-sm list-disc list-inside mb-4 space-y-1">
              <li>Backend API is running: <code className="bg-[var(--surface)] px-1 rounded">go run cmd/api/main.go</code></li>
              <li>Backend is accessible at: <code className="bg-[var(--surface)] px-1 rounded">http://localhost:8080</code></li>
              <li>API endpoint: <code className="bg-[var(--surface)] px-1 rounded">{API_BASE_URL}/api/apps</code></li>
              <li>Check browser console (F12) for more details</li>
              <li>Verify CORS is enabled in backend (should be after recent update)</li>
            </ul>
            <div className="flex gap-3">
              <button
                onClick={testBackendConnection}
                className="bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-[var(--app-bg)] font-medium py-2 px-4 rounded-lg transition-colors"
              >
                Test Connection
              </button>
              <button
                onClick={loadApps}
                className="bg-[var(--error)] hover:bg-[var(--error)]/80 text-white font-medium py-2 px-4 rounded-lg transition-colors"
              >
                Try again
              </button>
            </div>
          </div>
        )}

        {!loading && !error && apps && apps.length === 0 && (
          <div className="text-center py-12 bg-[var(--surface)] rounded-lg border border-[var(--border-subtle)]">
            <p className="text-[var(--text-secondary)] mb-4">No applications found</p>
            <button
              onClick={() => setIsNewAppModalOpen(true)}
              className="inline-block bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-[var(--app-bg)] font-medium py-2 px-4 rounded-lg transition-colors"
            >
              Create your first app
            </button>
          </div>
        )}

        {!loading && !error && apps && apps.length > 0 && (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {apps.map((app) => (
              <AppCard key={app.id} app={app} />
            ))}
          </div>
        )}
      </div>

      {/* New App Modal */}
      <NewAppModal
        isOpen={isNewAppModalOpen}
        onClose={() => setIsNewAppModalOpen(false)}
      />
    </div>
  );
}


