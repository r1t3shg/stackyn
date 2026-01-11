import { useState, useEffect } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { useAuth } from '@/contexts/AuthContext';
import { authApi } from '@/lib/api';
import Logo from '@/components/Logo';

type LoginMethod = 'password' | 'otp';

export default function Login() {
  const [loginMethod, setLoginMethod] = useState<LoginMethod>('password');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [otp, setOtp] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [otpSent, setOtpSent] = useState(false);
  const [resendCooldown, setResendCooldown] = useState(0);
  const { user, login, loginWithToken, isLoading } = useAuth();
  const navigate = useNavigate();

  // Redirect if already logged in
  useEffect(() => {
    if (!isLoading && user) {
      navigate('/apps', { replace: true });
    }
  }, [user, isLoading, navigate]);

  // Resend cooldown timer
  useEffect(() => {
    if (resendCooldown > 0) {
      const timer = setTimeout(() => setResendCooldown(resendCooldown - 1), 1000);
      return () => clearTimeout(timer);
    }
  }, [resendCooldown]);

  const handlePasswordLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setLoading(true);

    try {
      await login(email, password);
      navigate('/apps');
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'An error occurred during login';
      
      // If password not set, switch to OTP login mode
      if (errorMessage.includes('Password not set')) {
        setError('Password not set for this account. Please use OTP login instead.');
        setLoginMethod('otp');
      } else {
        setError(errorMessage);
      }
      console.error('Login error:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleSendOTP = async () => {
    if (!email || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
      setError('Please enter a valid email address');
      return;
    }

    setError(null);
    setLoading(true);
    try {
      await authApi.sendOTP(email);
      setOtpSent(true);
      setResendCooldown(60); // 60 second cooldown
      setError(null);
    } catch (err: any) {
      setError(err.message || 'Failed to send OTP');
    } finally {
      setLoading(false);
    }
  };

  const handleOTPLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    if (!otp || otp.length !== 6) {
      setError('Please enter a valid 6-digit OTP');
      return;
    }

    setLoading(true);
    try {
      const response = await authApi.verifyOTP(email, otp);
      // Store token and update auth context
      loginWithToken(response.token, response.user);
      navigate('/apps');
    } catch (err: any) {
      const errorMessage = err.message || 'Failed to verify OTP';
      if (errorMessage.includes('expired')) {
        setError('OTP has expired. Please request a new one.');
      } else if (errorMessage.includes('Invalid')) {
        setError('Invalid OTP. Please check and try again.');
      } else {
        setError(errorMessage);
      }
    } finally {
      setLoading(false);
    }
  };

  const handleResendOTP = async () => {
    if (resendCooldown > 0) return;
    await handleSendOTP();
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
            Sign in to Stackyn
          </h1>
          <p className="text-lg text-[var(--text-secondary)]">
            Welcome back. Sign in to manage your applications.
          </p>
        </div>

        {/* Login Method Toggle */}
        <div className="flex items-center justify-center gap-4 mb-6">
          <button
            type="button"
            onClick={() => {
              setLoginMethod('password');
              setOtpSent(false);
              setOtp('');
              setError(null);
            }}
            className={`px-4 py-2 rounded-lg font-medium transition-colors ${
              loginMethod === 'password'
                ? 'bg-[var(--primary)] text-[var(--app-bg)]'
                : 'bg-[var(--surface)] text-[var(--text-secondary)] hover:bg-[var(--elevated)]'
            }`}
          >
            Password
          </button>
          <button
            type="button"
            onClick={() => {
              setLoginMethod('otp');
              setPassword('');
              setError(null);
            }}
            className={`px-4 py-2 rounded-lg font-medium transition-colors ${
              loginMethod === 'otp'
                ? 'bg-[var(--primary)] text-[var(--app-bg)]'
                : 'bg-[var(--surface)] text-[var(--text-secondary)] hover:bg-[var(--elevated)]'
            }`}
          >
            OTP
          </button>
        </div>

        {/* Login Form */}
        <div className="bg-[var(--surface)] rounded-lg border border-[var(--border-subtle)] p-8">
          {error && (
            <div className="bg-[var(--error)]/10 border border-[var(--error)] rounded-lg p-4 mb-6">
              <p className="text-[var(--error)] text-sm">{error}</p>
            </div>
          )}

          {loginMethod === 'password' ? (
            <form className="space-y-6" onSubmit={handlePasswordLogin}>
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
                  className="w-full px-4 py-3 bg-[var(--elevated)] border border-[var(--border-subtle)] rounded-lg focus:outline-none focus:border-[var(--focus-border)] text-[var(--text-primary)] placeholder-[var(--text-muted)] transition-colors"
                  placeholder="you@example.com"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                />
              </div>

              <div>
                <div className="flex items-center justify-between mb-2">
                  <label htmlFor="password" className="block text-sm font-medium text-[var(--text-secondary)]">
                    Password
                  </label>
                  <Link
                    to="/forgot-password"
                    className="text-sm text-[var(--primary)] hover:text-[var(--primary-hover)] transition-colors"
                  >
                    Forgot password?
                  </Link>
                </div>
                <div className="relative">
                  <input
                    id="password"
                    name="password"
                    type={showPassword ? 'text' : 'password'}
                    autoComplete="current-password"
                    required
                    className="w-full px-4 py-3 pr-12 bg-[var(--elevated)] border border-[var(--border-subtle)] rounded-lg focus:outline-none focus:border-[var(--focus-border)] text-[var(--text-primary)] placeholder-[var(--text-muted)] transition-colors"
                    placeholder="Enter your password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                  />
                  <button
                    type="button"
                    onClick={() => setShowPassword(!showPassword)}
                    className="absolute right-3 top-1/2 -translate-y-1/2 text-[var(--text-muted)] hover:text-[var(--text-primary)] transition-colors focus:outline-none"
                    aria-label={showPassword ? 'Hide password' : 'Show password'}
                  >
                    {showPassword ? (
                      <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
                      </svg>
                    ) : (
                      <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                      </svg>
                    )}
                  </button>
                </div>
              </div>

              <button
                type="submit"
                disabled={loading}
                className="w-full flex justify-center py-3 px-4 border border-transparent text-sm font-medium rounded-lg text-[var(--app-bg)] bg-[var(--primary)] hover:bg-[var(--primary-hover)] focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-[var(--primary)] disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                {loading ? 'Signing in...' : 'Sign in'}
              </button>
            </form>
          ) : (
            <div className="space-y-6">
              {!otpSent ? (
                <form className="space-y-6" onSubmit={(e) => { e.preventDefault(); handleSendOTP(); }}>
                  <div>
                    <label htmlFor="email-otp" className="block text-sm font-medium text-[var(--text-secondary)] mb-2">
                      Email address
                    </label>
                    <input
                      id="email-otp"
                      name="email"
                      type="email"
                      autoComplete="email"
                      required
                      className="w-full px-4 py-3 bg-[var(--elevated)] border border-[var(--border-subtle)] rounded-lg focus:outline-none focus:border-[var(--focus-border)] text-[var(--text-primary)] placeholder-[var(--text-muted)] transition-colors"
                      placeholder="you@example.com"
                      value={email}
                      onChange={(e) => setEmail(e.target.value)}
                    />
                  </div>

                  <button
                    type="submit"
                    disabled={loading || !email}
                    className="w-full flex justify-center py-3 px-4 border border-transparent text-sm font-medium rounded-lg text-[var(--app-bg)] bg-[var(--primary)] hover:bg-[var(--primary-hover)] focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-[var(--primary)] disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                  >
                    {loading ? 'Sending code...' : 'Send verification code'}
                  </button>
                </form>
              ) : (
                <form className="space-y-6" onSubmit={handleOTPLogin}>
                  <div>
                    <label htmlFor="otp" className="block text-sm font-medium text-[var(--text-secondary)] mb-2">
                      Verification code
                    </label>
                    <input
                      id="otp"
                      name="otp"
                      type="text"
                      inputMode="numeric"
                      pattern="[0-9]*"
                      maxLength={6}
                      required
                      value={otp}
                      onChange={(e) => {
                        const value = e.target.value.replace(/\D/g, '').slice(0, 6);
                        setOtp(value);
                      }}
                      className="w-full px-4 py-3 bg-[var(--elevated)] border border-[var(--border-subtle)] rounded-lg focus:outline-none focus:border-[var(--focus-border)] text-[var(--text-primary)] placeholder-[var(--text-muted)] transition-colors text-center text-2xl tracking-widest font-mono"
                      placeholder="000000"
                      autoFocus
                    />
                    <p className="mt-2 text-sm text-[var(--text-muted)] text-center">
                      Enter the 6-digit code sent to {email}
                    </p>
                  </div>

                  <button
                    type="submit"
                    disabled={loading || otp.length !== 6}
                    className="w-full flex justify-center py-3 px-4 border border-transparent text-sm font-medium rounded-lg text-[var(--app-bg)] bg-[var(--primary)] hover:bg-[var(--primary-hover)] focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-[var(--primary)] disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                  >
                    {loading ? 'Verifying...' : 'Verify and sign in'}
                  </button>

                  <div className="text-center">
                    <button
                      type="button"
                      onClick={handleResendOTP}
                      disabled={resendCooldown > 0 || loading}
                      className="text-sm text-[var(--primary)] hover:text-[var(--primary-hover)] disabled:text-[var(--text-muted)] disabled:cursor-not-allowed transition-colors"
                    >
                      {resendCooldown > 0
                        ? `Resend code in ${resendCooldown}s`
                        : "Didn't receive code? Resend"}
                    </button>
                  </div>

                  <button
                    type="button"
                    onClick={() => {
                      setOtpSent(false);
                      setOtp('');
                      setError(null);
                    }}
                    className="w-full py-3 px-4 border border-[var(--border-subtle)] text-sm font-medium rounded-lg text-[var(--text-primary)] bg-[var(--surface)] hover:bg-[var(--elevated)] focus:outline-none transition-colors"
                  >
                    Change email
                  </button>
                </form>
              )}
            </div>
          )}
        </div>

        {/* Sign Up Link */}
        <div className="mt-6 text-center">
          <p className="text-sm text-[var(--text-secondary)]">
            Don't have an account?{' '}
            <Link to="/signup" className="font-medium text-[var(--primary)] hover:text-[var(--primary-hover)] transition-colors">
              Sign up
            </Link>
          </p>
        </div>
      </div>
    </div>
  );
}
