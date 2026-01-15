import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { appsApi } from '@/lib/api';

interface NewAppModalProps {
  isOpen: boolean;
  onClose: () => void;
  onAppCreated?: (appId: string) => void; // Optional callback for when an app is created
}

export default function NewAppModal({ isOpen, onClose, onAppCreated }: NewAppModalProps) {
  const navigate = useNavigate();
  const [formData, setFormData] = useState({
    name: '',
    repo_url: '',
    branch: '',
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleClose = () => {
    if (!loading) {
      setFormData({ name: '', repo_url: '', branch: '' });
      setError(null);
      onClose();
    }
  };

  // Handle Escape key to close modal
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isOpen && !loading) {
        handleClose();
      }
    };

    if (isOpen) {
      document.addEventListener('keydown', handleEscape);
      // Prevent body scroll when modal is open
      document.body.style.overflow = 'hidden';
    }

    return () => {
      document.removeEventListener('keydown', handleEscape);
      document.body.style.overflow = 'unset';
    };
  }, [isOpen, loading, onClose]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);

    try {
      const response = await appsApi.create(formData);
      if (response.error) {
        setError(response.error);
      } else {
        // Reset form and close modal
        setFormData({ name: '', repo_url: '', branch: '' });
        setError(null);
        onClose();
        // Call optional callback or navigate to the new app details page
        if (onAppCreated) {
          onAppCreated(response.app.id);
        } else {
          navigate(`/apps/${response.app.id}`);
        }
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create app');
      console.error('Error creating app:', err);
    } finally {
      setLoading(false);
    }
  };

  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-4"
      onClick={handleClose}
    >
      {/* Backdrop */}
      <div className="fixed inset-0 bg-black/60 backdrop-blur-sm" />

      {/* Modal */}
      <div
        className="relative w-full max-w-2xl bg-[var(--elevated)] rounded-lg border border-[var(--border-subtle)] shadow-xl max-h-[90vh] overflow-y-auto"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-[var(--border-subtle)]">
          <h2 className="text-2xl font-bold text-[var(--text-primary)]">Create New Application</h2>
          <button
            onClick={handleClose}
            disabled={loading}
            className="text-[var(--text-muted)] hover:text-[var(--text-primary)] transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            aria-label="Close modal"
          >
            <svg
              className="w-6 h-6"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M6 18L18 6M6 6l12 12"
              />
            </svg>
          </button>
        </div>

        {/* Content */}
        <div className="p-6">
          {error && (
            <div className="bg-[var(--error)]/10 border border-[var(--error)] rounded-lg p-4 mb-6">
              <p className="text-[var(--error)]">{error}</p>
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-6">
            <div>
              <label htmlFor="modal-name" className="block text-sm font-medium text-[var(--text-secondary)] mb-2">
                App Name
              </label>
              <input
                type="text"
                id="modal-name"
                required
                value={formData.name}
                onChange={(e) => {
                  // Replace spaces with hyphens and convert to lowercase
                  const transformedValue = e.target.value
                    .replace(/\s+/g, '-')  // Replace spaces with hyphens
                    .toLowerCase();         // Convert to lowercase
                  setFormData({ ...formData, name: transformedValue });
                }}
                className="w-full px-4 py-2 bg-[var(--surface)] border border-[var(--border-subtle)] rounded-lg focus:border-[var(--focus-border)] text-[var(--text-primary)]"
                placeholder="my-awesome-app"
                disabled={loading}
              />
            </div>

            <div>
              <label htmlFor="modal-repo_url" className="block text-sm font-medium text-[var(--text-secondary)] mb-2">
                Repository URL
              </label>
              <input
                type="url"
                id="modal-repo_url"
                required
                value={formData.repo_url}
                onChange={(e) => setFormData({ ...formData, repo_url: e.target.value })}
                className="w-full px-4 py-2 bg-[var(--surface)] border border-[var(--border-subtle)] rounded-lg focus:border-[var(--focus-border)] text-[var(--text-primary)]"
                placeholder="https://github.com/username/repo.git"
                disabled={loading}
              />
              <p className="mt-1 text-sm text-[var(--text-muted)]">
                Make sure your repository contains a Dockerfile in the root directory
              </p>
            </div>

            <div>
              <label htmlFor="modal-branch" className="block text-sm font-medium text-[var(--text-secondary)] mb-2">
                Branch
              </label>
              <input
                type="text"
                id="modal-branch"
                required
                value={formData.branch}
                onChange={(e) => setFormData({ ...formData, branch: e.target.value })}
                className="w-full px-4 py-2 bg-[var(--surface)] border border-[var(--border-subtle)] rounded-lg focus:border-[var(--focus-border)] text-[var(--text-primary)]"
                placeholder="main"
                disabled={loading}
              />
            </div>

            <div className="flex items-center justify-end space-x-4 pt-4 border-t border-[var(--border-subtle)]">
              <button
                type="button"
                onClick={handleClose}
                disabled={loading}
                className="px-4 py-2 border border-[var(--border-subtle)] rounded-lg text-[var(--text-primary)] hover:bg-[var(--surface)] transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Cancel
              </button>
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

