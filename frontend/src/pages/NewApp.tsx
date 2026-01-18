import { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { appsApi } from '@/lib/api';

interface EnvVar {
  key: string;
  value: string;
}

export default function NewAppPage() {
  const navigate = useNavigate();
  const [formData, setFormData] = useState({
    name: '',
    slug: '',
    repo_url: '',
    branch: '',
  });
  const [envVars, setEnvVars] = useState<EnvVar[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);

    try {
      // Prepare environment variables for the request
      const validEnvVars = envVars
        .filter(env => env.key.trim() !== '')
        .map(env => ({ key: env.key.trim(), value: env.value }));
      
      // Include environment variables in the create request
      const createData = {
        ...formData,
        env_vars: validEnvVars.length > 0 ? validEnvVars : undefined,
      };
      
      const response = await appsApi.create(createData);
      if (response.error) {
        setError(response.error);
      } else {
        // Navigate to app details page
        navigate(`/apps/${response.app.id}`);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create app');
      console.error('Error creating app:', err);
    } finally {
      setLoading(false);
    }
  };

  const addEnvVar = () => {
    setEnvVars([...envVars, { key: '', value: '' }]);
  };

  const removeEnvVar = (index: number) => {
    setEnvVars(envVars.filter((_, i) => i !== index));
  };

  const updateEnvVar = (index: number, field: 'key' | 'value', value: string) => {
    const updated = [...envVars];
    updated[index] = { ...updated[index], [field]: value };
    setEnvVars(updated);
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
                onChange={(e) => {
                  const nameValue = e.target.value;
                  // Auto-generate slug from name if slug is empty or matches the auto-generated pattern
                  const autoSlug = nameValue
                    .toLowerCase()
                    .replace(/[^a-z0-9]+/g, '-')
                    .replace(/^-+|-+$/g, '');
                  
                  setFormData((prev) => ({
                    ...prev,
                    name: nameValue,
                    slug: prev.slug === '' || prev.slug === formData.name.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '') 
                      ? autoSlug 
                      : prev.slug,
                  }));
                }}
                className="w-full px-4 py-2 bg-[var(--elevated)] border border-[var(--border-subtle)] rounded-lg focus:border-[var(--focus-border)] text-[var(--text-primary)]"
                placeholder="My Awesome App"
              />
            </div>

            <div>
              <label htmlFor="slug" className="block text-sm font-medium text-[var(--text-secondary)] mb-2">
                Slug (Subdomain) <span className="text-[var(--text-muted)] text-xs">(optional)</span>
              </label>
              <input
                type="text"
                id="slug"
                value={formData.slug}
                onChange={(e) => {
                  // Validate slug format: only lowercase letters, numbers, and hyphens
                  const slugValue = e.target.value
                    .toLowerCase()
                    .replace(/[^a-z0-9-]/g, '')
                    .replace(/-+/g, '-')
                    .replace(/^-+|-+$/g, '');
                  setFormData({ ...formData, slug: slugValue });
                }}
                className="w-full px-4 py-2 bg-[var(--elevated)] border border-[var(--border-subtle)] rounded-lg focus:border-[var(--focus-border)] text-[var(--text-primary)]"
                placeholder="my-awesome-app"
              />
              <p className="mt-1 text-sm text-[var(--text-muted)]">
                Your app will be available at <span className="font-mono text-[var(--primary)]">https://{formData.slug || formData.name.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '') || 'your-slug'}.stackyn.com</span>
              </p>
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
                className="w-full px-4 py-2 bg-[var(--elevated)] border border-[var(--border-subtle)] rounded-lg focus:border-[var(--focus-border)] text-[var(--text-primary)]"
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
                className="w-full px-4 py-2 bg-[var(--elevated)] border border-[var(--border-subtle)] rounded-lg focus:border-[var(--focus-border)] text-[var(--text-primary)]"
                placeholder="main"
              />
            </div>

            {/* Environment Variables Section */}
            <div>
              <div className="flex items-center justify-between mb-2">
                <label className="block text-sm font-medium text-[var(--text-secondary)]">
                  Environment Variables
                </label>
                <button
                  type="button"
                  onClick={addEnvVar}
                  className="text-sm px-3 py-1 bg-[var(--surface)] hover:bg-[var(--elevated)] text-[var(--primary)] border border-[var(--border-subtle)] rounded-lg transition-colors"
                >
                  + Add Variable
                </button>
              </div>
              <p className="text-sm text-[var(--text-muted)] mb-3">
                Add environment variables that will be injected into your app container
              </p>
              
              {envVars.length === 0 ? (
                <div className="text-center py-4 text-[var(--text-muted)] bg-[var(--elevated)] border border-[var(--border-subtle)] rounded-lg">
                  <p className="text-sm">No environment variables added</p>
                </div>
              ) : (
                <div className="space-y-2">
                  {envVars.map((envVar, index) => (
                    <div key={index} className="flex gap-2 items-start">
                      <div className="flex-1">
                        <input
                          type="text"
                          value={envVar.key}
                          onChange={(e) => updateEnvVar(index, 'key', e.target.value)}
                          placeholder="KEY"
                          className="w-full px-3 py-2 bg-[var(--elevated)] border border-[var(--border-subtle)] rounded-lg focus:border-[var(--focus-border)] text-[var(--text-primary)] font-mono text-sm"
                        />
                      </div>
                      <div className="flex-1">
                        <input
                          type="text"
                          value={envVar.value}
                          onChange={(e) => updateEnvVar(index, 'value', e.target.value)}
                          placeholder="value"
                          className="w-full px-3 py-2 bg-[var(--elevated)] border border-[var(--border-subtle)] rounded-lg focus:border-[var(--focus-border)] text-[var(--text-primary)] font-mono text-sm"
                        />
                      </div>
                      <button
                        type="button"
                        onClick={() => removeEnvVar(index)}
                        className="px-3 py-2 bg-[var(--error)]/10 hover:bg-[var(--error)]/20 text-[var(--error)] rounded-lg transition-colors"
                        title="Remove variable"
                      >
                        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                        </svg>
                      </button>
                    </div>
                  ))}
                </div>
              )}
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


