import { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { appsApi } from '@/lib/api';

export default function NewAppPage() {
  const navigate = useNavigate();
  const [formData, setFormData] = useState({
    name: '',
    repo_url: '',
    branch: '',
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);

    try {
      const response = await appsApi.create(formData);
      if (response.error) {
        setError(response.error);
      } else {
        navigate(`/apps/${response.app.id}`);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create app');
      console.error('Error creating app:', err);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-[var(--app-bg)]">
      <div className="max-w-2xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <Link
          to="/"
          className="text-[var(--info)] hover:text-[var(--primary)] mb-6 inline-block transition-colors"
        >
          ← Back to Apps
        </Link>

        <div className="bg-[var(--surface)] rounded-lg border border-[var(--border-subtle)] p-8">
          <h1 className="text-3xl font-bold text-[var(--text-primary)] mb-6">Create New Application</h1>

          {error && (
            <div className="bg-[var(--error)]/10 border border-[var(--error)] rounded-lg p-4 mb-6">
              <p className="text-[var(--error)]">{error}</p>
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-6">
            <div>
              <label htmlFor="name" className="block text-sm font-medium text-[var(--text-secondary)] mb-2">
                App Name
              </label>
              <input
                type="text"
                id="name"
                required
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                className="w-full px-4 py-2 bg-[var(--elevated)] border border-[var(--border-subtle)] rounded-lg focus:ring-2 focus:ring-[var(--focus-border)] focus:border-[var(--focus-border)] text-[var(--text-primary)]"
                placeholder="my-awesome-app"
              />
            </div>

            <div>
              <label htmlFor="repo_url" className="block text-sm font-medium text-[var(--text-secondary)] mb-2">
                Repository URL
              </label>
              <input
                type="url"
                id="repo_url"
                required
                value={formData.repo_url}
                onChange={(e) => setFormData({ ...formData, repo_url: e.target.value })}
                className="w-full px-4 py-2 bg-[var(--elevated)] border border-[var(--border-subtle)] rounded-lg focus:ring-2 focus:ring-[var(--focus-border)] focus:border-[var(--focus-border)] text-[var(--text-primary)]"
                placeholder="https://github.com/username/repo.git"
              />
              <p className="mt-1 text-sm text-[var(--text-muted)]">
                Make sure your repository contains a Dockerfile in the root directory
              </p>
            </div>

            <div>
              <label htmlFor="branch" className="block text-sm font-medium text-[var(--text-secondary)] mb-2">
                Branch
              </label>
              <input
                type="text"
                id="branch"
                required
                value={formData.branch}
                onChange={(e) => setFormData({ ...formData, branch: e.target.value })}
                className="w-full px-4 py-2 bg-[var(--elevated)] border border-[var(--border-subtle)] rounded-lg focus:ring-2 focus:ring-[var(--focus-border)] focus:border-[var(--focus-border)] text-[var(--text-primary)]"
                placeholder="main"
              />
            </div>

            <div className="flex items-center justify-end space-x-4">
              <Link
                to="/"
                className="px-4 py-2 border border-[var(--border-subtle)] rounded-lg text-[var(--text-primary)] hover:bg-[var(--elevated)] transition-colors"
              >
                Cancel
              </Link>
              <button
                type="submit"
                disabled={loading}
                className="px-6 py-2 bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-[var(--app-bg)] font-medium rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {loading ? 'Creating...' : 'Create App'}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
}


