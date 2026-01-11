import { useState, useEffect } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { authApi } from '@/lib/api';
import { API_BASE_URL } from '@/lib/config';
import Logo from '@/components/Logo';

type SignupStep = 'email' | 'otp' | 'password' | 'details';

export default function SignUp() {
  const [step, setStep] = useState<SignupStep>('email');
  const [email, setEmail] = useState('');
  const [otp, setOtp] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [fullName, setFullName] = useState('');
  const [companyName, setCompanyName] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [emailError, setEmailError] = useState<string | null>(null);
  const [passwordError, setPasswordError] = useState<string | null>(null);
  const [resendCooldown, setResendCooldown] = useState(0);
  const navigate = useNavigate();

  // Redirect if already logged in
  useEffect(() => {
    const token = localStorage.getItem('auth_token');
    if (token) {
      navigate('/apps', { replace: true });
    }
  }, [navigate]);

  // Resend cooldown timer
  useEffect(() => {
    if (resendCooldown > 0) {
      const timer = setTimeout(() => setResendCooldown(resendCooldown - 1), 1000);
      return () => clearTimeout(timer);
    }
  }, [resendCooldown]);

  // Real-time email validation
  useEffect(() => {
    if (email && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
      setEmailError('Please enter a valid email address');
    } else {
      setEmailError(null);
    }
  }, [email]);

  // Step 1: Send OTP to email
  const handleEmailSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    if (!email || emailError) {
      setError('Please enter a valid email address');
      return;
    }

    setLoading(true);
    try {
      await authApi.sendOTP(email);
      setStep('otp');
      setResendCooldown(60); // 60 second cooldown
      setError(null);
    } catch (err: any) {
      const errorMessage = err.message || 'Failed to send OTP';
      if (errorMessage.includes('already registered') || errorMessage.includes('exists')) {
        setError('This email is already registered. Please sign in instead.');
      } else {
        setError(errorMessage);
      }
    } finally {
      setLoading(false);
    }
  };

  // Step 2: Verify OTP (password not included here)
  const handleOTPSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    if (!otp || otp.length !== 6) {
      setError('Please enter a valid 6-digit OTP');
      return;
    }

    setLoading(true);
    try {
      const response = await authApi.verifyOTP(email, otp);
      // Store token and user
      localStorage.setItem('auth_token', response.token);
      localStorage.setItem('auth_user', JSON.stringify(response.user));
      
      // Move to password step (optional)
      setStep('password');
      setError(null);
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

  // Handle resend OTP
  const handleResendOTP = async () => {
    if (resendCooldown > 0) return;

    setError(null);
    setLoading(true);
    try {
      await authApi.sendOTP(email);
      setResendCooldown(60); // 60 second cooldown
      setError(null);
      // Show success message
      alert('OTP sent! Please check your inbox.');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to resend OTP');
    } finally {
      setLoading(false);
    }
  };

  // Step 3: Set password (optional - can skip)
  const handlePasswordSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    // Password is optional - if provided, validate it
    if (password) {
      if (password.length < 8) {
        setError('Password must be at least 8 characters');
        return;
      }
      if (password !== confirmPassword) {
        setError('Passwords do not match');
        return;
      }

      // Update password via API
      setLoading(true);
      try {
        const token = localStorage.getItem('auth_token');
        
        if (!token) {
          throw new Error('Session expired. Please start over.');
        }

        const response = await fetch(`${API_BASE_URL}/api/auth/update-profile`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`,
          },
          body: JSON.stringify({
            password: password,
          }),
        });

        if (!response.ok) {
          const error = await response.json().catch(() => ({ error: 'Failed to set password' }));
          throw new Error(error.error || 'Failed to set password');
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to set password');
        setLoading(false);
        return;
      } finally {
        setLoading(false);
      }
    }

    // Move to details step
    setStep('details');
  };

  // Step 4: Complete signup with user details
  const handleDetailsSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    if (!fullName) {
      setError('Please enter your full name');
      return;
    }

    setLoading(true);
    try {
      const token = localStorage.getItem('auth_token');
      
      if (!token) {
        throw new Error('Session expired. Please start over.');
      }

      // Update user profile via API
      const response = await fetch(`${API_BASE_URL}/api/auth/update-profile`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({
          full_name: fullName,
          company_name: companyName,
        }),
      });

      if (!response.ok) {
        const error = await response.json().catch(() => ({ error: 'Failed to update profile' }));
        throw new Error(error.error || 'Failed to update profile');
      }

      const updatedUser = await response.json();
      localStorage.setItem('auth_user', JSON.stringify(updatedUser));

      // Redirect to apps page
      navigate('/apps');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to complete signup');
    } finally {
      setLoading(false);
    }
  };

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
            {step === 'email' && 'Create account'}
            {step === 'otp' && 'Verify your email'}
            {step === 'password' && 'Set password (optional)'}
            {step === 'details' && 'Complete your account'}
          </h1>
          <p className="text-lg text-[var(--text-secondary)]">
            {step === 'email' && 'Go from code to production without managing servers. Get started in seconds.'}
            {step === 'otp' && `We sent a verification code to ${email}. Please check your inbox and enter the code below.`}
            {step === 'password' && 'Set a password to login without OTP in the future (optional)'}
            {step === 'details' && 'Just a few more details to get you started'}
          </p>
        </div>

        {/* Progress Indicator */}
        <div className="mb-8">
          <div className="flex items-center justify-center gap-2">
            <div className={`flex-1 h-1 rounded-full ${step !== 'email' ? 'bg-[var(--primary)]' : 'bg-[var(--primary)]'}`}></div>
            <div className={`flex-1 h-1 rounded-full ${step === 'otp' || step === 'password' || step === 'details' ? 'bg-[var(--primary)]' : 'bg-[var(--border-subtle)]'}`}></div>
            <div className={`flex-1 h-1 rounded-full ${step === 'password' || step === 'details' ? 'bg-[var(--primary)]' : 'bg-[var(--border-subtle)]'}`}></div>
            <div className={`flex-1 h-1 rounded-full ${step === 'details' ? 'bg-[var(--primary)]' : 'bg-[var(--border-subtle)]'}`}></div>
          </div>
          <div className="flex justify-between mt-2 text-xs text-[var(--text-muted)]">
            <span>Email</span>
            <span>Verify</span>
            <span>Password</span>
            <span>Details</span>
          </div>
        </div>

        {/* Sign-up Form */}
        <div className="bg-[var(--surface)] rounded-lg border border-[var(--border-subtle)] p-8">
          {error && (
            <div className="bg-[var(--error)]/10 border border-[var(--error)] rounded-lg p-4 mb-6">
              <p className="text-[var(--error)] text-sm">{error}</p>
            </div>
          )}

          {/* Step 1: Email */}
          {step === 'email' && (
            <form className="space-y-6" onSubmit={handleEmailSubmit}>
              <div>
                <label htmlFor="email" className="block text-sm font-medium text-[var(--text-secondary)] mb-2">
                  Work email address
                </label>
                <input
                  id="email"
                  name="email"
                  type="email"
                  autoComplete="email"
                  required
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  className={`w-full px-4 py-3 bg-[var(--elevated)] border rounded-lg focus:outline-none focus:border-[var(--focus-border)] text-[var(--text-primary)] placeholder-[var(--text-muted)] transition-colors ${
                    emailError ? 'border-[var(--error)]' : 'border-[var(--border-subtle)]'
                  }`}
                  placeholder="you@company.com"
                />
                {emailError && (
                  <p className="mt-1 text-sm text-[var(--error)]">{emailError}</p>
                )}
              </div>

              <button
                type="submit"
                disabled={loading || !!emailError || !email}
                className="w-full flex justify-center py-3 px-4 border border-transparent text-sm font-medium rounded-lg text-[var(--app-bg)] bg-[var(--primary)] hover:bg-[var(--primary-hover)] focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-[var(--primary)] disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                {loading ? 'Sending code...' : 'Continue'}
              </button>
            </form>
          )}

          {/* Step 2: OTP Verification (password fields removed) */}
          {step === 'otp' && (
            <div className="space-y-6">
              <form className="space-y-6" onSubmit={handleOTPSubmit}>
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
                  {loading ? 'Verifying...' : 'Verify code'}
                </button>
              </form>

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
                onClick={() => {
                  setStep('email');
                  setOtp('');
                  setError(null);
                }}
                className="w-full py-3 px-4 border border-[var(--border-subtle)] text-sm font-medium rounded-lg text-[var(--text-primary)] bg-[var(--surface)] hover:bg-[var(--elevated)] focus:outline-none transition-colors"
              >
                Back
              </button>
            </div>
          )}

          {/* Step 3: Set Password (optional) */}
          {step === 'password' && (
            <form className="space-y-6" onSubmit={handlePasswordSubmit}>
              <div>
                <label htmlFor="password" className="block text-sm font-medium text-[var(--text-secondary)] mb-2">
                  Password <span className="text-[var(--text-muted)]">(optional)</span>
                </label>
                <input
                  id="password"
                  name="password"
                  type="password"
                  autoComplete="new-password"
                  value={password}
                  onChange={(e) => {
                    setPassword(e.target.value);
                    if (e.target.value && e.target.value.length < 8) {
                      setPasswordError('Password must be at least 8 characters');
                    } else if (e.target.value && confirmPassword && e.target.value !== confirmPassword) {
                      setPasswordError('Passwords do not match');
                    } else {
                      setPasswordError(null);
                    }
                  }}
                  className="w-full px-4 py-3 bg-[var(--elevated)] border border-[var(--border-subtle)] rounded-lg focus:outline-none focus:border-[var(--focus-border)] text-[var(--text-primary)] placeholder-[var(--text-muted)] transition-colors"
                  placeholder="Enter a password (optional)"
                />
                {passwordError && (
                  <p className="mt-1 text-sm text-[var(--error)]">{passwordError}</p>
                )}
                <p className="mt-1 text-sm text-[var(--text-muted)]">
                  Set a password to login without OTP in the future. You can skip this step.
                </p>
              </div>

              {password && (
                <div>
                  <label htmlFor="confirmPassword" className="block text-sm font-medium text-[var(--text-secondary)] mb-2">
                    Confirm Password
                  </label>
                  <input
                    id="confirmPassword"
                    name="confirmPassword"
                    type="password"
                    autoComplete="new-password"
                    value={confirmPassword}
                    onChange={(e) => {
                      setConfirmPassword(e.target.value);
                      if (password && e.target.value !== password) {
                        setPasswordError('Passwords do not match');
                      } else {
                        setPasswordError(null);
                      }
                    }}
                    className="w-full px-4 py-3 bg-[var(--elevated)] border border-[var(--border-subtle)] rounded-lg focus:outline-none focus:border-[var(--focus-border)] text-[var(--text-primary)] placeholder-[var(--text-muted)] transition-colors"
                    placeholder="Confirm your password"
                  />
                </div>
              )}

              <button
                type="submit"
                disabled={loading || !!passwordError || (password ? password.length < 8 : false) || (password ? password !== confirmPassword : false)}
                className="w-full flex justify-center py-3 px-4 border border-transparent text-sm font-medium rounded-lg text-[var(--app-bg)] bg-[var(--primary)] hover:bg-[var(--primary-hover)] focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-[var(--primary)] disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                {loading ? 'Setting password...' : password ? 'Continue with password' : 'Skip password'}
              </button>
            </form>
          )}

          {/* Step 4: Account Details */}
          {step === 'details' && (
            <form className="space-y-6" onSubmit={handleDetailsSubmit}>
              <div>
                <label htmlFor="fullName" className="block text-sm font-medium text-[var(--text-secondary)] mb-2">
                  Full name <span className="text-[var(--error)]">*</span>
                </label>
                <input
                  id="fullName"
                  name="fullName"
                  type="text"
                  autoComplete="name"
                  required
                  value={fullName}
                  onChange={(e) => setFullName(e.target.value)}
                  className="w-full px-4 py-3 bg-[var(--elevated)] border border-[var(--border-subtle)] rounded-lg focus:outline-none focus:border-[var(--focus-border)] text-[var(--text-primary)] placeholder-[var(--text-muted)] transition-colors"
                  placeholder="John Doe"
                />
              </div>

              <div>
                <label htmlFor="companyName" className="block text-sm font-medium text-[var(--text-secondary)] mb-2">
                  Company / Project name
                </label>
                <input
                  id="companyName"
                  name="companyName"
                  type="text"
                  autoComplete="organization"
                  value={companyName}
                  onChange={(e) => setCompanyName(e.target.value)}
                  className="w-full px-4 py-3 bg-[var(--elevated)] border border-[var(--border-subtle)] rounded-lg focus:outline-none focus:border-[var(--focus-border)] text-[var(--text-primary)] placeholder-[var(--text-muted)] transition-colors"
                  placeholder="Acme Inc."
                />
              </div>

              <button
                type="submit"
                disabled={loading || !fullName}
                className="w-full flex justify-center py-3 px-4 border border-transparent text-sm font-medium rounded-lg text-[var(--app-bg)] bg-[var(--primary)] hover:bg-[var(--primary-hover)] focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-[var(--primary)] disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                {loading ? 'Completing signup...' : 'Go to console'}
              </button>
            </form>
          )}
        </div>

        {/* Login Link */}
        <div className="mt-6 text-center">
          <p className="text-sm text-[var(--text-secondary)]">
            Already have an account?{' '}
            <Link to="/login" className="font-medium text-[var(--primary)] hover:text-[var(--primary-hover)] transition-colors">
              Sign in
            </Link>
          </p>
        </div>

        {/* Terms and Privacy */}
        <div className="mt-4 text-center text-xs text-[var(--text-muted)]">
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
  );
}
