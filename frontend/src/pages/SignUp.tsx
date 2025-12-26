import { useState, useEffect } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { useAuth } from '@/contexts/AuthContext';
import Logo from '@/components/Logo';

export default function SignUp() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [emailError, setEmailError] = useState<string | null>(null);
  const [passwordError, setPasswordError] = useState<string | null>(null);
  const [confirmPasswordError, setConfirmPasswordError] = useState<string | null>(null);
  const { user, signup, isLoading } = useAuth();
  const navigate = useNavigate();

  // Redirect if already logged in
  useEffect(() => {
    if (!isLoading && user) {
      navigate('/apps', { replace: true });
    }
  }, [user, isLoading, navigate]);

  // Real-time email validation
  useEffect(() => {
    if (email && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
      setEmailError('Please enter a valid email address');
    } else {
      setEmailError(null);
    }
  }, [email]);

  // Real-time password validation
  useEffect(() => {
    if (password) {
      if (password.length < 8) {
        setPasswordError('Password must be at least 8 characters');
      } else {
        setPasswordError(null);
      }
    } else {
      setPasswordError(null);
    }
  }, [password]);

  // Real-time confirm password validation
  useEffect(() => {
    if (confirmPassword && password && confirmPassword !== password) {
      setConfirmPasswordError('Passwords do not match');
    } else {
      setConfirmPasswordError(null);
    }
  }, [confirmPassword, password]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    // Validate all fields
    if (!email || !password || !confirmPassword) {
      setError('Please fill in all fields');
      return;
    }

    if (emailError || passwordError || confirmPasswordError) {
      setError('Please fix the validation errors');
      return;
    }

    if (password !== confirmPassword) {
      setConfirmPasswordError('Passwords do not match');
      return;
    }

    setLoading(true);
    try {
      await signup(email, password);
      navigate('/apps');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'An error occurred during signup');
    } finally {
      setLoading(false);
    }
  };

  const handleGitHubSignUp = () => {
    // Placeholder for GitHub OAuth
    alert('GitHub sign-up coming soon! For now, please use email and password.');
  };

  const handleGitLabSignUp = () => {
    // Placeholder for GitLab OAuth
    alert('GitLab sign-up coming soon! For now, please use email and password.');
  };

  // Show loading state while checking auth
  if (isLoading) {
    return (
      <div className="min-h-screen bg-[var(--app-bg)] flex items-center justify-center">
        <div className="text-center">
          <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-[var(--primary)]"></div>
          <p className="mt-4 text-[var(--text-secondary)]">Loading...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-[var(--app-bg)] flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
      <div className="max-w-md w-full">
        {/* Logo */}
        <div className="flex justify-center mb-8">
          <Logo />
        </div>

        {/* Header */}
        <div className="text-center mb-8">
          <h1 className="text-3xl sm:text-4xl font-bold text-[var(--text-primary)] mb-3">
            Deploy your app in one click
          </h1>
          <p className="text-lg text-[var(--text-secondary)]">
            Go from code to production without managing servers. Get started in seconds.
          </p>
        </div>

        {/* Sign-up Form */}
        <div className="bg-[var(--surface)] rounded-lg border border-[var(--border-subtle)] p-8">
          <form className="space-y-6" onSubmit={handleSubmit}>
            {error && (
              <div className="bg-[var(--error)]/10 border border-[var(--error)] rounded-lg p-4">
                <p className="text-[var(--error)] text-sm">{error}</p>
              </div>
            )}

            {/* Email Field */}
            <div>
              <label htmlFor="email" className="block text-sm font-medium text-[var(--text-secondary)] mb-2">
                Email address
              </label>
              <input
                id="email"
                name="email"
                type="email"
                autoComplete="email"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className={`w-full px-4 py-3 bg-[var(--elevated)] border rounded-lg focus:outline-none focus:ring-2 focus:ring-[var(--focus-border)] focus:border-[var(--focus-border)] text-[var(--text-primary)] placeholder-[var(--text-muted)] transition-colors ${
                  emailError ? 'border-[var(--error)]' : 'border-[var(--border-subtle)]'
                }`}
                placeholder="you@example.com"
              />
              {emailError && (
                <p className="mt-1 text-sm text-[var(--error)]">{emailError}</p>
              )}
            </div>

            {/* Password Field */}
            <div>
              <label htmlFor="password" className="block text-sm font-medium text-[var(--text-secondary)] mb-2">
                Password
              </label>
              <input
                id="password"
                name="password"
                type="password"
                autoComplete="new-password"
                required
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className={`w-full px-4 py-3 bg-[var(--elevated)] border rounded-lg focus:outline-none focus:ring-2 focus:ring-[var(--focus-border)] focus:border-[var(--focus-border)] text-[var(--text-primary)] placeholder-[var(--text-muted)] transition-colors ${
                  passwordError ? 'border-[var(--error)]' : 'border-[var(--border-subtle)]'
                }`}
                placeholder="At least 8 characters"
              />
              {passwordError && (
                <p className="mt-1 text-sm text-[var(--error)]">{passwordError}</p>
              )}
            </div>

            {/* Confirm Password Field */}
            <div>
              <label htmlFor="confirmPassword" className="block text-sm font-medium text-[var(--text-secondary)] mb-2">
                Confirm Password
              </label>
              <input
                id="confirmPassword"
                name="confirmPassword"
                type="password"
                autoComplete="new-password"
                required
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                className={`w-full px-4 py-3 bg-[var(--elevated)] border rounded-lg focus:outline-none focus:ring-2 focus:ring-[var(--focus-border)] focus:border-[var(--focus-border)] text-[var(--text-primary)] placeholder-[var(--text-muted)] transition-colors ${
                  confirmPasswordError ? 'border-[var(--error)]' : 'border-[var(--border-subtle)]'
                }`}
                placeholder="Confirm your password"
              />
              {confirmPasswordError && (
                <p className="mt-1 text-sm text-[var(--error)]">{confirmPasswordError}</p>
              )}
            </div>

            {/* Submit Button */}
            <button
              type="submit"
              disabled={loading || !!emailError || !!passwordError || !!confirmPasswordError}
              className="w-full flex justify-center py-3 px-4 border border-transparent text-sm font-medium rounded-lg text-[var(--app-bg)] bg-[var(--primary)] hover:bg-[var(--primary-hover)] focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-[var(--primary)] disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              {loading ? 'Creating account...' : 'Create account'}
            </button>

            {/* Divider */}
            <div className="relative">
              <div className="absolute inset-0 flex items-center">
                <div className="w-full border-t border-[var(--border-subtle)]"></div>
              </div>
              <div className="relative flex justify-center text-sm">
                <span className="px-2 bg-[var(--surface)] text-[var(--text-muted)]">Or continue with</span>
              </div>
            </div>

            {/* OAuth Buttons */}
            <div className="grid grid-cols-2 gap-3">
              <button
                type="button"
                onClick={handleGitHubSignUp}
                className="flex items-center justify-center gap-2 px-4 py-3 border border-[var(--border-subtle)] rounded-lg bg-[var(--elevated)] hover:bg-[var(--surface)] text-[var(--text-primary)] transition-colors"
              >
                <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
                  <path fillRule="evenodd" d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z" clipRule="evenodd" />
                </svg>
                GitHub
              </button>
              <button
                type="button"
                onClick={handleGitLabSignUp}
                className="flex items-center justify-center gap-2 px-4 py-3 border border-[var(--border-subtle)] rounded-lg bg-[var(--elevated)] hover:bg-[var(--surface)] text-[var(--text-primary)] transition-colors"
              >
                <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M12 0L8.5 9.5H0L6 15.5L2.5 24L12 18L21.5 24L18 15.5L24 9.5H15.5L12 0Z"/>
                </svg>
                GitLab
              </button>
            </div>
          </form>
        </div>

        {/* Trust Indicators */}
        <div className="mt-6 space-y-4">
          <div className="text-center space-y-2">
            <div className="flex items-center justify-center gap-6 text-xs text-[var(--text-muted)]">
              <div className="flex items-center gap-1">
                <svg className="w-4 h-4 text-[var(--success)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                </svg>
                Secure authentication
              </div>
              <div className="flex items-center gap-1">
                <svg className="w-4 h-4 text-[var(--success)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                </svg>
                Privacy protected
              </div>
              <div className="flex items-center gap-1">
                <svg className="w-4 h-4 text-[var(--success)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z" />
                </svg>
                No credit card required
              </div>
            </div>
          </div>

          {/* What Happens Next */}
          <div className="bg-[var(--primary-muted)]/20 rounded-lg border border-[var(--primary-muted)] p-4">
            <div className="flex items-start gap-3">
              <svg className="w-5 h-5 text-[var(--primary)] mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <div>
                <div className="font-medium text-[var(--text-primary)] mb-1">What happens next?</div>
                <div className="text-sm text-[var(--text-secondary)]">
                  After signing up, you'll connect a Git repository and deploy your first app. Stackyn will automatically build, run, and expose your application.
                </div>
              </div>
            </div>
          </div>

          {/* Login Link */}
          <div className="text-center">
            <p className="text-sm text-[var(--text-secondary)]">
              Already have an account?{' '}
              <Link to="/login" className="font-medium text-[var(--primary)] hover:text-[var(--primary-hover)] transition-colors">
                Sign in
              </Link>
            </p>
          </div>

          {/* Terms and Privacy */}
          <div className="text-center text-xs text-[var(--text-muted)]">
            By creating an account, you agree to our{' '}
            <Link to="/terms" className="text-[var(--primary)] hover:text-[var(--primary-hover)] transition-colors">
              Terms of Service
            </Link>
            {' '}and{' '}
            <Link to="/privacy" className="text-[var(--primary)] hover:text-[var(--primary-hover)] transition-colors">
              Privacy Policy
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
}

