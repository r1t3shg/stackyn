import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useAuth } from '@/contexts/AuthContext';
import Logo from '@/components/Logo';

export default function LandingPage() {
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const [videoModalOpen, setVideoModalOpen] = useState(false);
  const { user } = useAuth();
  const navigate = useNavigate();

  const handleSignInClick = (e: React.MouseEvent<HTMLElement>) => {
    e.preventDefault();
    e.stopPropagation();
    if (user) {
      // In local development, navigate to apps page. In production, redirect to console subdomain
      const hostname = window.location.hostname;
      const isLocal = hostname === 'localhost' || hostname === '127.0.0.1' || hostname === '' || hostname.startsWith('192.168.') || hostname.startsWith('10.') || hostname.startsWith('172.');
      if (isLocal || import.meta.env.DEV) {
        navigate('/apps');
      } else {
        // Pass auth token when redirecting to console subdomain
        const token = localStorage.getItem('auth_token');
        const consoleUrl = token 
          ? `https://console.stackyn.com/?token=${encodeURIComponent(token)}`
          : 'https://console.stackyn.com/';
        window.location.href = consoleUrl;
      }
    } else {
      navigate('/login');
    }
  };

  const handleConsoleRedirect = () => {
    const hostname = window.location.hostname;
    const isLocal = hostname === 'localhost' || hostname === '127.0.0.1' || hostname === '' || hostname.startsWith('192.168.') || hostname.startsWith('10.') || hostname.startsWith('172.');
    if (isLocal || import.meta.env.DEV) {
      navigate('/apps');
    } else {
      // Pass auth token when redirecting to console subdomain
      const token = localStorage.getItem('auth_token');
      const consoleUrl = token 
        ? `https://console.stackyn.com/?token=${encodeURIComponent(token)}`
        : 'https://console.stackyn.com/';
      window.location.href = consoleUrl;
    }
  };

  return (
    <div className="min-h-screen bg-[var(--app-bg)]">
      {/* Header */}
      <header className="border-b border-[var(--border-subtle)] bg-[var(--app-bg)] sticky top-0 z-50">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center gap-8">
              {/* Logo */}
              <div className="flex-shrink-0">
                <Logo />
              </div>

              {/* Desktop Navigation Links - Left Aligned */}
              <nav className="hidden md:flex items-center space-x-6">
                <a
                  href="#why-stackyn"
                  className="text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors font-medium"
                  onClick={(e) => {
                    e.preventDefault();
                    document.getElementById('why-stackyn')?.scrollIntoView({ behavior: 'smooth' });
                  }}
                >
                  Why Stackyn?
                </a>
                <a
                  href="#features"
                  className="text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors font-medium"
                  onClick={(e) => {
                    e.preventDefault();
                    document.getElementById('features')?.scrollIntoView({ behavior: 'smooth' });
                  }}
                >
                  Features
                </a>
                <a
                  href="#pricing"
                  className="text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors font-medium"
                  onClick={(e) => {
                    e.preventDefault();
                    document.getElementById('pricing')?.scrollIntoView({ behavior: 'smooth' });
                  }}
                >
                  Pricing
                </a>
              </nav>
            </div>

            {/* Desktop CTA - Right Side */}
            <div className="hidden md:flex items-center gap-3 relative z-10">
              {user ? (
                <>
                  <button
                    type="button"
                    onClick={(e) => {
                      e.preventDefault();
                      e.stopPropagation();
                      const hostname = window.location.hostname;
                      const isLocal = hostname === 'localhost' || hostname === '127.0.0.1' || hostname === '' || hostname.startsWith('192.168.') || hostname.startsWith('10.') || hostname.startsWith('172.');
                      if (isLocal || import.meta.env.DEV) {
                        navigate('/apps');
                      } else {
                        // Pass auth token when redirecting to console subdomain
                        const token = localStorage.getItem('auth_token');
                        const consoleUrl = token 
                          ? `https://console.stackyn.com/?token=${encodeURIComponent(token)}`
                          : 'https://console.stackyn.com/';
                        window.location.href = consoleUrl;
                      }
                    }}
                    style={{ cursor: 'pointer', pointerEvents: 'auto', position: 'relative', zIndex: 100 }}
                    className="bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-[var(--app-bg)] font-medium py-2 px-6 rounded-lg transition-colors"
                  >
                    Go to Console
                  </button>
                  <button
                    className="w-10 h-10 rounded-full bg-[var(--primary-muted)] flex items-center justify-center text-[var(--primary)] font-semibold hover:bg-[var(--elevated)] transition-colors"
                    aria-label="User menu"
                  >
                    {user.email?.charAt(0).toUpperCase() || 'U'}
                  </button>
                </>
              ) : (
                <a
                  href="/login"
                  onClick={handleSignInClick}
                  className="bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-[var(--app-bg)] font-medium py-2 px-6 rounded-lg transition-colors"
                >
                  Sign In
                </a>
              )}
            </div>

            {/* Mobile menu button */}
            <div className="md:hidden flex items-center">
              <button
                onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
                className="text-[var(--text-secondary)] hover:text-[var(--text-primary)] p-2"
                aria-label="Toggle menu"
              >
                <svg
                  className="h-6 w-6"
                  fill="none"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth="2"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  {mobileMenuOpen ? (
                    <path d="M6 18L18 6M6 6l12 12" />
                  ) : (
                    <path d="M4 6h16M4 12h16M4 18h16" />
                  )}
                </svg>
              </button>
            </div>
          </div>

          {/* Mobile menu */}
          {mobileMenuOpen && (
            <div className="md:hidden py-4 border-t border-[var(--border-subtle)]">
              <nav className="flex flex-col space-y-4">
                <a
                  href="#why-stackyn"
                  className="text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors font-medium"
                  onClick={(e) => {
                    e.preventDefault();
                    setMobileMenuOpen(false);
                    document.getElementById('why-stackyn')?.scrollIntoView({ behavior: 'smooth' });
                  }}
                >
                  Why Stackyn?
                </a>
                <a
                  href="#features"
                  className="text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors font-medium"
                  onClick={(e) => {
                    e.preventDefault();
                    setMobileMenuOpen(false);
                    document.getElementById('features')?.scrollIntoView({ behavior: 'smooth' });
                  }}
                >
                  Features
                </a>
                <a
                  href="#pricing"
                  className="text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors font-medium"
                  onClick={(e) => {
                    e.preventDefault();
                    setMobileMenuOpen(false);
                    document.getElementById('pricing')?.scrollIntoView({ behavior: 'smooth' });
                  }}
                >
                  Pricing
                </a>
                {user ? (
                  <>
                    <button
                      type="button"
                      onClick={(e) => {
                        e.preventDefault();
                        e.stopPropagation();
                        handleConsoleRedirect();
                      }}
                      style={{ cursor: 'pointer', pointerEvents: 'auto', position: 'relative', zIndex: 100 }}
                      className="bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-[var(--app-bg)] font-medium py-2 px-6 rounded-lg transition-colors text-center w-full"
                    >
                      Go to Console
                    </button>
                  </>
                ) : (
                  <a
                    href="/login"
                    onClick={handleSignInClick}
                    className="bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-[var(--app-bg)] font-medium py-2 px-6 rounded-lg transition-colors text-center"
                  >
                    Sign In
                  </a>
                )}
              </nav>
            </div>
          )}
        </div>
      </header>

      {/* Hero Section */}
      <section className="relative overflow-hidden bg-[var(--surface)]">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-24 sm:py-32">
          <div className="text-center">
            <h1 className="text-5xl sm:text-6xl lg:text-7xl font-bold text-[var(--text-primary)] mb-10 leading-tight">
              Deploy your backend apps in <span className="text-[var(--primary)]">one click</span> — no DevOps, no servers, no hassle.
            </h1>
            <div className="flex flex-col sm:flex-row justify-center gap-4">
              {user ? (
                <button
                  type="button"
                  onClick={(e) => {
                    e.preventDefault();
                    handleConsoleRedirect();
                  }}
                  className="bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-[var(--app-bg)] font-semibold py-4 px-8 rounded-lg transition-colors text-lg"
                >
                  Go to Console
                </button>
              ) : (
                <a
                  href="/signup"
                  onClick={(e) => {
                    e.preventDefault();
                    navigate('/signup');
                  }}
                  className="bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-[var(--app-bg)] font-semibold py-4 px-8 rounded-lg transition-colors text-lg"
                >
                  Get Started Free
                </a>
              )}
              <button
                onClick={() => setVideoModalOpen(true)}
                className="bg-[var(--elevated)] hover:bg-[var(--surface)] text-[var(--text-primary)] font-semibold py-4 px-8 rounded-lg transition-colors text-lg border border-[var(--border-subtle)]"
              >
                See How It Works
              </button>
            </div>
            {/* Hero Visual Placeholder - Dashboard mockup */}
            <div className="mt-16 rounded-lg border border-[var(--border-subtle)] bg-[var(--elevated)] p-8 overflow-hidden">
              <div className="bg-[var(--terminal-bg)] rounded-lg p-6 font-mono text-sm text-[var(--text-primary)]">
                <div className="flex items-center gap-2 mb-4">
                  <div className="w-3 h-3 rounded-full bg-[var(--success)]"></div>
                  <span className="text-[var(--text-muted)]">app.stackyn.com</span>
                  <span className="text-[var(--text-muted)]">•</span>
                  <span className="text-[var(--success)]">Healthy</span>
                </div>
                <div className="space-y-2">
                  <div className="text-[var(--text-secondary)]">$ git push origin main</div>
                  <div className="text-[var(--success)]">✓ Building...</div>
                  <div className="text-[var(--success)]">✓ Deploying...</div>
                  <div className="text-[var(--success)]">✓ Live at https://app.stackyn.com</div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* How It Works / 3-Step Section */}
      <section className="py-24 bg-[var(--app-bg)]">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center mb-16">
            <h2 className="text-3xl sm:text-4xl lg:text-5xl font-bold text-[var(--text-primary)] mb-4">
              From Code to Live in Minutes
            </h2>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-12">
            <div className="text-center">
              <div className="w-20 h-20 bg-[var(--primary-muted)] rounded-full flex items-center justify-center mx-auto mb-6">
                <svg className="w-10 h-10 text-[var(--primary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
                </svg>
              </div>
              <h3 className="text-2xl font-semibold text-[var(--text-primary)] mb-3">1. Connect Your Repo</h3>
              <p className="text-lg text-[var(--text-secondary)]">
                Push your code from GitHub.
              </p>
            </div>

            <div className="text-center">
              <div className="w-20 h-20 bg-[var(--primary-muted)] rounded-full flex items-center justify-center mx-auto mb-6">
                <svg className="w-10 h-10 text-[var(--primary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                </svg>
              </div>
              <h3 className="text-2xl font-semibold text-[var(--text-primary)] mb-3">2. One-Click Deploy</h3>
              <p className="text-lg text-[var(--text-secondary)]">
                Stackyn builds, runs, and exposes your app instantly.
              </p>
            </div>

            <div className="text-center">
              <div className="w-20 h-20 bg-[var(--primary-muted)] rounded-full flex items-center justify-center mx-auto mb-6">
                <svg className="w-10 h-10 text-[var(--primary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
                </svg>
              </div>
              <h3 className="text-2xl font-semibold text-[var(--text-primary)] mb-3">3. Monitor & Manage</h3>
              <p className="text-lg text-[var(--text-secondary)]">
                Logs, status, and redeploys — all in one dashboard.
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* Core Features Section */}
      <section id="features" className="py-24 bg-[var(--surface)]">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center mb-16">
            <h2 className="text-3xl sm:text-4xl lg:text-5xl font-bold text-[var(--text-primary)] mb-4">
              Everything You Need to Ship Your Backend
            </h2>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            <div className="p-6 rounded-lg border border-[var(--border-subtle)] bg-[var(--elevated)] hover:border-[var(--border-strong)] transition-colors">
              <div className="w-10 h-10 bg-[var(--primary-muted)] rounded-lg flex items-center justify-center mb-4">
                <svg className="w-6 h-6 text-[var(--primary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                </svg>
              </div>
              <h3 className="text-xl font-semibold text-[var(--text-primary)] mb-2">One-Click Deployment</h3>
              <p className="text-[var(--text-secondary)]">
                Deploy APIs, websites, and microservices instantly.
              </p>
            </div>

            <div className="p-6 rounded-lg border border-[var(--border-subtle)] bg-[var(--elevated)] hover:border-[var(--border-strong)] transition-colors">
              <div className="w-10 h-10 bg-[var(--primary-muted)] rounded-lg flex items-center justify-center mb-4">
                <svg className="w-6 h-6 text-[var(--primary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                </svg>
              </div>
              <h3 className="text-xl font-semibold text-[var(--text-primary)] mb-2">HTTPS & Domains</h3>
              <p className="text-[var(--text-secondary)]">
                Auto SSL, subdomain routing, and secure URLs.
              </p>
            </div>

            <div className="p-6 rounded-lg border border-[var(--border-subtle)] bg-[var(--elevated)] hover:border-[var(--border-strong)] transition-colors">
              <div className="w-10 h-10 bg-[var(--primary-muted)] rounded-lg flex items-center justify-center mb-4">
                <svg className="w-6 h-6 text-[var(--primary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
                </svg>
              </div>
              <h3 className="text-xl font-semibold text-[var(--text-primary)] mb-2">Logs & Monitoring</h3>
              <p className="text-[var(--text-secondary)]">
                Real-time logs and health status.
              </p>
            </div>

            <div className="p-6 rounded-lg border border-[var(--border-subtle)] bg-[var(--elevated)] hover:border-[var(--border-strong)] transition-colors">
              <div className="w-10 h-10 bg-[var(--primary-muted)] rounded-lg flex items-center justify-center mb-4">
                <svg className="w-6 h-6 text-[var(--primary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
                </svg>
              </div>
              <h3 className="text-xl font-semibold text-[var(--text-primary)] mb-2">Manual Deployment Control</h3>
              <p className="text-[var(--text-secondary)]">
                Deploy on your schedule with one-click redeploy.
              </p>
            </div>

            <div className="p-6 rounded-lg border border-[var(--border-subtle)] bg-[var(--elevated)] hover:border-[var(--border-strong)] transition-colors">
              <div className="w-10 h-10 bg-[var(--primary-muted)] rounded-lg flex items-center justify-center mb-4">
                <svg className="w-6 h-6 text-[var(--primary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
                </svg>
              </div>
              <h3 className="text-xl font-semibold text-[var(--text-primary)] mb-2">Resource Limits</h3>
              <p className="text-[var(--text-secondary)]">
                Containers are isolated with safe memory caps.
              </p>
            </div>

            <div className="p-6 rounded-lg border border-[var(--border-subtle)] bg-[var(--elevated)] hover:border-[var(--border-strong)] transition-colors">
              <div className="w-10 h-10 bg-[var(--primary-muted)] rounded-lg flex items-center justify-center mb-4">
                <svg className="w-6 h-6 text-[var(--primary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4" />
                </svg>
              </div>
              <h3 className="text-xl font-semibold text-[var(--text-primary)] mb-2">Multi-Language Support</h3>
              <p className="text-[var(--text-secondary)]">
                Go, Node.js, Python, and more.
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* Why Stackyn Section */}
      <section id="why-stackyn" className="py-24 bg-[var(--app-bg)]">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center mb-16">
            <h2 className="text-3xl sm:text-4xl lg:text-5xl font-bold text-[var(--text-primary)] mb-6">
              Focus on Features, Not Infrastructure
            </h2>
            <p className="text-xl text-[var(--text-secondary)] max-w-3xl mx-auto mb-8">
              Traditional cloud providers and PaaS platforms can be complex, expensive, or restrictive. Stackyn removes the friction:
            </p>
            <div className="flex flex-wrap justify-center gap-6 text-lg text-[var(--text-primary)]">
              <div className="flex items-center gap-2">
                <svg className="w-5 h-5 text-[var(--success)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                <span>No Docker configs</span>
              </div>
              <div className="flex items-center gap-2">
                <svg className="w-5 h-5 text-[var(--success)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                <span>No YAML files</span>
              </div>
              <div className="flex items-center gap-2">
                <svg className="w-5 h-5 text-[var(--success)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                <span>No DevOps expertise required</span>
              </div>
            </div>
          </div>

          {/* Comparison Table */}
          <div className="overflow-x-auto">
            <table className="w-full border-collapse">
              <thead>
                <tr className="border-b border-[var(--border-subtle)]">
                  <th className="text-left py-4 px-6 text-[var(--text-primary)] font-semibold">Feature</th>
                  <th className="text-center py-4 px-6 text-[var(--text-primary)] font-semibold">Stackyn</th>
                  <th className="text-center py-4 px-6 text-[var(--text-muted)] font-semibold">AWS</th>
                  <th className="text-center py-4 px-6 text-[var(--text-muted)] font-semibold">Heroku</th>
                </tr>
              </thead>
              <tbody>
                <tr className="border-b border-[var(--border-subtle)]">
                  <td className="py-4 px-6 text-[var(--text-secondary)]">Setup Time</td>
                  <td className="py-4 px-6 text-center text-[var(--success)]">Minutes</td>
                  <td className="py-4 px-6 text-center text-[var(--text-muted)]">Hours</td>
                  <td className="py-4 px-6 text-center text-[var(--text-muted)]">Hours</td>
                </tr>
                <tr className="border-b border-[var(--border-subtle)]">
                  <td className="py-4 px-6 text-[var(--text-secondary)]">Configuration</td>
                  <td className="py-4 px-6 text-center text-[var(--success)]">None</td>
                  <td className="py-4 px-6 text-center text-[var(--text-muted)]">Complex</td>
                  <td className="py-4 px-6 text-center text-[var(--text-muted)]">Moderate</td>
                </tr>
                <tr className="border-b border-[var(--border-subtle)]">
                  <td className="py-4 px-6 text-[var(--text-secondary)]">Pricing</td>
                  <td className="py-4 px-6 text-center text-[var(--success)]">Transparent</td>
                  <td className="py-4 px-6 text-center text-[var(--text-muted)]">Variable</td>
                  <td className="py-4 px-6 text-center text-[var(--text-muted)]">High</td>
                </tr>
                <tr>
                  <td className="py-4 px-6 text-[var(--text-secondary)]">Git Integration</td>
                  <td className="py-4 px-6 text-center text-[var(--success)]">✓ Native</td>
                  <td className="py-4 px-6 text-center text-[var(--text-muted)]">Manual</td>
                  <td className="py-4 px-6 text-center text-[var(--text-muted)]">✓ Native</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </section>

      {/* Pricing Section */}
      <section id="pricing" className="py-24 bg-[var(--surface)] scroll-mt-16">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          {/* Header Section */}
          <div className="text-center mb-16">
            <h1 className="text-4xl sm:text-5xl font-bold text-[var(--text-primary)] mb-4">
              Deploy backend apps without DevOps headaches
            </h1>
            <p className="text-xl text-[var(--text-secondary)] max-w-3xl mx-auto">
              Build, deploy, scale — Stackyn handles servers, SSL, and restarts.
            </p>
          </div>

          {/* Plans Grid */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-8 max-w-5xl mx-auto mb-16">
            {/* Starter Plan */}
            <div className="border border-[var(--border-subtle)] rounded-lg p-8 bg-[var(--elevated)]">
              <div className="mb-6">
                <div className="flex items-center gap-2 mb-2">
                  <h3 className="text-2xl font-bold text-[var(--text-primary)]">Starter</h3>
                </div>
                <div className="text-4xl font-bold text-[var(--text-primary)] mb-1">$19 <span className="text-xl font-normal text-[var(--text-muted)]">/ month</span></div>
                <p className="text-sm text-[var(--text-muted)]">For solo developers & side projects</p>
              </div>

              <div className="mb-6">
                <h4 className="text-sm font-semibold text-[var(--text-primary)] mb-3 uppercase tracking-wide">What you get</h4>
                <ul className="space-y-3">
                  <li className="flex items-start">
                    <svg className="w-5 h-5 text-[var(--success)] mr-3 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                    </svg>
                    <span className="text-[var(--text-secondary)] text-sm">Deploy 1 backend app</span>
                  </li>
                  <li className="flex items-start">
                    <svg className="w-5 h-5 text-[var(--success)] mr-3 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                    </svg>
                    <span className="text-[var(--text-secondary)] text-sm">512 MB RAM</span>
                  </li>
                  <li className="flex items-start">
                    <svg className="w-5 h-5 text-[var(--success)] mr-3 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                    </svg>
                    <span className="text-[var(--text-secondary)] text-sm">5 GB persistent disk</span>
                  </li>
                  <li className="flex items-start">
                    <svg className="w-5 h-5 text-[var(--success)] mr-3 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                    </svg>
                    <span className="text-[var(--text-secondary)] text-sm">One-click deploy to VPS</span>
                  </li>
                  <li className="flex items-start">
                    <svg className="w-5 h-5 text-[var(--success)] mr-3 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                    </svg>
                    <span className="text-[var(--text-secondary)] text-sm">Free SSL (auto-managed)</span>
                  </li>
                  <li className="flex items-start">
                    <svg className="w-5 h-5 text-[var(--success)] mr-3 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                    </svg>
                    <span className="text-[var(--text-secondary)] text-sm">Environment variables</span>
                  </li>
                  <li className="flex items-start">
                    <svg className="w-5 h-5 text-[var(--success)] mr-3 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                    </svg>
                    <span className="text-[var(--text-secondary)] text-sm">App logs (basic)</span>
                  </li>
                  <li className="flex items-start">
                    <svg className="w-5 h-5 text-[var(--success)] mr-3 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                    </svg>
                    <span className="text-[var(--text-secondary)] text-sm">Restart & redeploy anytime</span>
                  </li>
                  <li className="flex items-start">
                    <svg className="w-5 h-5 text-[var(--success)] mr-3 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                    </svg>
                    <span className="text-[var(--text-secondary)] text-sm">7-day free trial</span>
                  </li>
                </ul>
              </div>

              <div className="mb-8 pt-6 border-t border-[var(--border-subtle)]">
                <p className="text-sm font-medium text-[var(--text-primary)] mb-1">Perfect for</p>
                <p className="text-sm text-[var(--text-secondary)]">MVPs, personal projects, early testing</p>
              </div>

              <button
                disabled
                className="block w-full text-center bg-[var(--primary)] text-[var(--app-bg)] font-semibold py-3 px-6 rounded-lg opacity-50 cursor-not-allowed"
              >
                Billing Launching Soon
              </button>
            </div>

            {/* Pro Plan */}
            <div className="border-2 border-[var(--primary)] rounded-lg p-8 bg-[var(--elevated)] relative">
              <div className="absolute top-0 right-0 bg-[var(--primary)] text-[var(--app-bg)] px-4 py-1 rounded-bl-lg rounded-tr-lg text-xs font-semibold">
                Most Popular
              </div>
              <div className="mb-6">
                <div className="flex items-center gap-2 mb-2">
                  <h3 className="text-2xl font-bold text-[var(--text-primary)]">Pro</h3>
                </div>
                <div className="text-4xl font-bold text-[var(--text-primary)] mb-1">$49 <span className="text-xl font-normal text-[var(--text-muted)]">/ month</span></div>
                <p className="text-sm text-[var(--text-muted)]">For serious builders & small teams</p>
              </div>

              <div className="mb-6">
                <h4 className="text-sm font-semibold text-[var(--text-primary)] mb-3 uppercase tracking-wide">Everything in Starter, plus</h4>
                <ul className="space-y-3">
                  <li className="flex items-start">
                    <svg className="w-5 h-5 text-[var(--success)] mr-3 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                    </svg>
                    <span className="text-[var(--text-secondary)] text-sm">Deploy up to 3 apps</span>
                  </li>
                  <li className="flex items-start">
                    <svg className="w-5 h-5 text-[var(--success)] mr-3 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                    </svg>
                    <span className="text-[var(--text-secondary)] text-sm">2 GB RAM</span>
                  </li>
                  <li className="flex items-start">
                    <svg className="w-5 h-5 text-[var(--success)] mr-3 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                    </svg>
                    <span className="text-[var(--text-secondary)] text-sm">20 GB persistent disk</span>
                  </li>
                  <li className="flex items-start">
                    <svg className="w-5 h-5 text-[var(--success)] mr-3 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                    </svg>
                    <span className="text-[var(--text-secondary)] text-sm">One-click deploy to VPS</span>
                  </li>
                  <li className="flex items-start">
                    <svg className="w-5 h-5 text-[var(--success)] mr-3 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                    </svg>
                    <span className="text-[var(--text-secondary)] text-sm">Free SSL (auto-managed)</span>
                  </li>
                  <li className="flex items-start">
                    <svg className="w-5 h-5 text-[var(--success)] mr-3 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                    </svg>
                    <span className="text-[var(--text-secondary)] text-sm">Environment variables</span>
                  </li>
                  <li className="flex items-start">
                    <svg className="w-5 h-5 text-[var(--success)] mr-3 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                    </svg>
                    <span className="text-[var(--text-secondary)] text-sm">App logs</span>
                  </li>
                  <li className="flex items-start">
                    <svg className="w-5 h-5 text-[var(--success)] mr-3 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                    </svg>
                    <span className="text-[var(--text-secondary)] text-sm">Restart & redeploy anytime</span>
                  </li>
                  <li className="flex items-start">
                    <svg className="w-5 h-5 text-[var(--success)] mr-3 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                    </svg>
                    <span className="text-[var(--text-secondary)] text-sm">7-day free trial</span>
                  </li>
                </ul>
              </div>

              <div className="mb-8 pt-6 border-t border-[var(--border-subtle)]">
                <p className="text-sm font-medium text-[var(--text-primary)] mb-1">Perfect for</p>
                <p className="text-sm text-[var(--text-secondary)]">Startups, paid products, production workloads</p>
              </div>

              <button
                disabled
                className="block w-full text-center bg-[var(--primary)] text-[var(--app-bg)] font-semibold py-3 px-6 rounded-lg opacity-50 cursor-not-allowed"
              >
                Billing Launching Soon
              </button>
            </div>
          </div>

          {/* Trial Notice */}
          <div className="text-center mb-12 p-6 rounded-lg border border-[var(--primary)]/20 bg-[var(--primary-muted)]/10">
            <p className="text-lg text-[var(--text-primary)]">
              <strong>7-day free trial</strong> — No credit card required. Try Pro features for free!
            </p>
          </div>

          {/* FAQ Section */}
          <div className="mb-20">
            <h2 className="text-2xl sm:text-3xl font-bold text-[var(--text-primary)] text-center mb-12">
              Frequently Asked Questions
            </h2>
            <div className="max-w-3xl mx-auto space-y-8">
              <div>
                <h3 className="text-xl font-semibold text-[var(--text-primary)] mb-3">What counts as an app?</h3>
                <p className="text-[var(--text-secondary)]">
                  One Git repository deployed as a single running service.
                </p>
              </div>
              <div>
                <h3 className="text-xl font-semibold text-[var(--text-primary)] mb-3">Can I upgrade or downgrade anytime?</h3>
                <p className="text-[var(--text-secondary)]">
                  Yes. Plans are flexible and can be changed at any time.
                </p>
              </div>
              <div>
                <h3 className="text-xl font-semibold text-[var(--text-primary)] mb-3">What happens if I exceed my limits?</h3>
                <p className="text-[var(--text-secondary)]">
                  We'll notify you and help you upgrade — your app won't be shut down suddenly.
                </p>
              </div>
              <div>
                <h3 className="text-xl font-semibold text-[var(--text-primary)] mb-3">Is this suitable for production?</h3>
                <p className="text-[var(--text-secondary)]">
                  Yes. Stackyn is built for real-world backend apps with reliability and isolation.
                </p>
              </div>
            </div>
          </div>

          {/* Final CTA Section */}
          <div className="text-center mb-20 p-12 rounded-lg border border-[var(--border-subtle)] bg-[var(--elevated)]">
            <h2 className="text-3xl sm:text-4xl font-bold text-[var(--text-primary)] mb-6">
              Start shipping, not configuring servers.
            </h2>
            {user ? (
              <button
                type="button"
                onClick={(e) => {
                  e.preventDefault();
                  const hostname = window.location.hostname;
                  const isLocal = hostname === 'localhost' || hostname === '127.0.0.1' || hostname === '' || hostname.startsWith('192.168.') || hostname.startsWith('10.') || hostname.startsWith('172.');
                  if (isLocal || import.meta.env.DEV) {
                    navigate('/apps');
                  } else {
                    window.location.href = 'https://console.stackyn.com/';
                  }
                }}
                className="inline-block bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-[var(--app-bg)] font-semibold py-4 px-8 rounded-lg transition-colors text-lg mb-4"
              >
                Go to Console
              </button>
            ) : (
              <a
                href="/signup"
                onClick={(e) => {
                  e.preventDefault();
                  navigate('/signup');
                }}
                className="inline-block bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-[var(--app-bg)] font-semibold py-4 px-8 rounded-lg transition-colors text-lg mb-4"
              >
                Get Started Free
              </a>
            )}
            <p className="text-sm text-[var(--text-muted)]">No credit card required</p>
          </div>

          {/* Founder Note */}
          <div className="max-w-3xl mx-auto p-8 rounded-lg border-l-4 border-[var(--primary)] bg-[var(--primary-muted)]/20">
            <p className="text-[var(--text-primary)] italic text-lg">
              "Stackyn is built for developers who want clarity, speed, and control — without becoming DevOps engineers."
            </p>
          </div>
        </div>
      </section>

      {/* CTA / Closing Section */}
      <section className="py-24 bg-[var(--surface)]">
        <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 text-center">
          <h2 className="text-3xl sm:text-4xl lg:text-5xl font-bold text-[var(--text-primary)] mb-6">
            Launch your app in minutes. Stop managing infrastructure.
          </h2>
          <div className="flex flex-col sm:flex-row justify-center gap-4 mb-6">
            {user ? (
              <button
                type="button"
                onClick={(e) => {
                  e.preventDefault();
                  const hostname = window.location.hostname;
                  const isLocal = hostname === 'localhost' || hostname === '127.0.0.1' || hostname === '' || hostname.startsWith('192.168.') || hostname.startsWith('10.') || hostname.startsWith('172.');
                  if (isLocal || import.meta.env.DEV) {
                    navigate('/apps');
                  } else {
                    window.location.href = 'https://console.stackyn.com/';
                  }
                }}
                className="bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-[var(--app-bg)] font-semibold py-4 px-8 rounded-lg transition-colors text-lg"
              >
                Go to Console
              </button>
            ) : (
              <a
                href="/signup"
                onClick={(e) => {
                  e.preventDefault();
                  navigate('/signup');
                }}
                className="bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-[var(--app-bg)] font-semibold py-4 px-8 rounded-lg transition-colors text-lg"
              >
                Get Started Free
              </a>
            )}
            <button
              onClick={() => setVideoModalOpen(true)}
              className="bg-[var(--elevated)] hover:bg-[var(--surface)] text-[var(--text-primary)] font-semibold py-4 px-8 rounded-lg transition-colors text-lg border border-[var(--border-subtle)]"
            >
              See How It Works
            </button>
          </div>
          <p className="text-sm text-[var(--text-muted)]">
            No credit card required • Works with GitHub
          </p>
        </div>
      </section>

      {/* Footer */}
      <footer className="bg-[var(--sidebar)] text-[var(--text-muted)] py-12 border-t border-[var(--border-subtle)]">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-8 mb-8">
            <div>
              <div className="mb-4">
                <Logo height={20} showText={true} />
              </div>
              <p className="text-sm text-[var(--text-secondary)]">
                Deploy your backend apps in one click — no DevOps, no servers, no hassle.
              </p>
            </div>
            <div>
              <h4 className="text-[var(--text-primary)] font-semibold mb-4">Links</h4>
              <ul className="space-y-2 text-sm">
                <li>
                  <a
                    href="#pricing"
                    className="hover:text-[var(--text-primary)] transition-colors"
                    onClick={(e) => {
                      e.preventDefault();
                      document.getElementById('pricing')?.scrollIntoView({ behavior: 'smooth' });
                    }}
                  >
                    Pricing
                  </a>
                </li>
                <li>
                  <Link to="/about" className="hover:text-[var(--text-primary)] transition-colors">
                    About Us
                  </Link>
                </li>
                <li>
                  <Link to="/terms" className="hover:text-[var(--text-primary)] transition-colors">
                    Terms of Service
                  </Link>
                </li>
                <li>
                  <Link to="/privacy" className="hover:text-[var(--text-primary)] transition-colors">
                    Privacy Policy
                  </Link>
                </li>
                <li>
                  <Link to="/refund" className="hover:text-[var(--text-primary)] transition-colors">
                    Refund Policy
                  </Link>
                </li>
                <li>
                  <Link to="/contact" className="hover:text-[var(--text-primary)] transition-colors">
                    Contact
                  </Link>
                </li>
              </ul>
            </div>
            <div>
              <h4 className="text-[var(--text-primary)] font-semibold mb-4">Connect</h4>
              <div className="space-y-3 mb-4">
                <a
                  href="mailto:support@stackyn.com"
                  className="flex items-center gap-2 text-[var(--text-secondary)] hover:text-[var(--primary)] transition-colors"
                >
                  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
                  </svg>
                  support@stackyn.com
                </a>
              </div>
              <div className="flex space-x-4">
                <a href="https://discord.gg/zmeKCbXKn3" target="_blank" rel="noopener noreferrer" className="hover:text-[var(--text-primary)] transition-colors" aria-label="Discord">
                  <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
                    <path d="M20.317 4.37a19.791 19.791 0 00-4.885-1.515.074.074 0 00-.079.037c-.21.375-.444.864-.608 1.25a18.27 18.27 0 00-5.487 0 12.64 12.64 0 00-.617-1.25.077.077 0 00-.079-.037A19.736 19.736 0 003.677 4.37a.07.07 0 00-.032.027C.533 9.046-.32 13.58.099 18.057a.082.082 0 00.031.057 19.9 19.9 0 005.993 3.03.078.078 0 00.084-.028c.462-.63.874-1.295 1.226-1.994a.076.076 0 00-.041-.106 13.107 13.107 0 01-1.872-.892.077.077 0 01-.008-.128 10.2 10.2 0 00.372-.292.074.074 0 01.077-.01c3.928 1.793 8.18 1.793 12.062 0a.074.074 0 01.078.01c.12.098.246.198.373.292a.077.077 0 01-.006.127 12.299 12.299 0 01-1.873.892.077.077 0 00-.041.107c.36.698.772 1.362 1.225 1.993a.076.076 0 00.084.028 19.839 19.839 0 006.002-3.03.077.077 0 00.032-.054c.5-5.177-.838-9.674-3.549-13.66a.061.061 0 00-.031-.03zM8.02 15.33c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.956-2.419 2.157-2.419 1.21 0 2.176 1.096 2.157 2.42 0 1.333-.956 2.418-2.157 2.418zm7.975 0c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.955-2.419 2.157-2.419 1.21 0 2.176 1.096 2.157 2.42 0 1.333-.946 2.418-2.157 2.418z"/>
                  </svg>
                </a>
                <a href="https://x.com/stackynio" target="_blank" rel="noopener noreferrer" className="hover:text-[var(--text-primary)] transition-colors" aria-label="X (Twitter)">
                  <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
                    <path d="M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-5.214-6.817L4.99 21.75H1.68l7.73-8.835L1.254 2.25H8.08l4.713 6.231zm-1.161 17.52h1.833L7.084 4.126H5.117z"/>
                  </svg>
                </a>
                <a href="https://www.linkedin.com/company/stackyn/" target="_blank" rel="noopener noreferrer" className="hover:text-[var(--text-primary)] transition-colors" aria-label="LinkedIn">
                  <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
                    <path d="M19 0h-14c-2.761 0-5 2.239-5 5v14c0 2.761 2.239 5 5 5h14c2.762 0 5-2.239 5-5v-14c0-2.761-2.238-5-5-5zm-11 19h-3v-11h3v11zm-1.5-12.268c-.966 0-1.75-.79-1.75-1.764s.784-1.764 1.75-1.764 1.75.79 1.75 1.764-.783 1.764-1.75 1.764zm13.5 12.268h-3v-5.604c0-3.368-4-3.113-4 0v5.604h-3v-11h3v1.765c1.396-2.586 7-2.777 7 2.476v6.759z" />
                  </svg>
                </a>
                <a href="https://www.instagram.com/stackyn.io/" target="_blank" rel="noopener noreferrer" className="hover:text-[var(--text-primary)] transition-colors" aria-label="Instagram">
                  <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
                    <path d="M12 2.163c3.204 0 3.584.012 4.85.07 3.252.148 4.771 1.691 4.919 4.919.058 1.265.069 1.645.069 4.849 0 3.205-.012 3.584-.069 4.849-.149 3.225-1.664 4.771-4.919 4.919-1.266.058-1.644.07-4.85.07-3.204 0-3.584-.012-4.849-.07-3.26-.149-4.771-1.699-4.919-4.92-.058-1.265-.07-1.644-.07-4.849 0-3.204.013-3.583.07-4.849.149-3.227 1.664-4.771 4.919-4.919 1.266-.057 1.645-.069 4.849-.069zm0-2.163c-3.259 0-3.667.014-4.947.072-4.358.2-6.78 2.618-6.98 6.98-.059 1.281-.073 1.689-.073 4.948 0 3.259.014 3.668.072 4.948.2 4.358 2.618 6.78 6.98 6.98 1.281.058 1.689.072 4.948.072 3.259 0 3.668-.014 4.948-.072 4.354-.2 6.782-2.618 6.979-6.98.059-1.28.073-1.689.073-4.948 0-3.259-.014-3.667-.072-4.947-.196-4.354-2.617-6.78-6.979-6.98-1.281-.059-1.69-.073-4.949-.073zm0 5.838c-3.403 0-6.162 2.759-6.162 6.162s2.759 6.163 6.162 6.163 6.162-2.759 6.162-6.163c0-3.403-2.759-6.162-6.162-6.162zm0 10.162c-2.209 0-4-1.79-4-4 0-2.209 1.791-4 4-4s4 1.791 4 4c0 2.21-1.791 4-4 4zm6.406-11.845c-.796 0-1.441.645-1.441 1.44s.645 1.44 1.441 1.44c.795 0 1.439-.645 1.439-1.44s-.644-1.44-1.439-1.44z" />
                  </svg>
                </a>
              </div>
            </div>
          </div>
          <div className="pt-8 border-t border-[var(--border-subtle)] text-center text-sm">
            <p>&copy; {new Date().getFullYear()} Stackyn. All rights reserved.</p>
          </div>
        </div>
      </footer>

      {/* Video Modal */}
      {videoModalOpen && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-sm"
          onClick={() => setVideoModalOpen(false)}
        >
          <div
            className="relative w-full max-w-5xl mx-4 bg-[var(--app-bg)] rounded-lg overflow-hidden shadow-2xl"
            onClick={(e) => e.stopPropagation()}
          >
            <button
              onClick={() => setVideoModalOpen(false)}
              className="absolute top-4 right-4 z-10 text-[var(--text-primary)] hover:text-[var(--text-secondary)] transition-colors bg-[var(--surface)] rounded-full p-2 hover:bg-[var(--elevated)]"
              aria-label="Close video"
            >
              <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
            <div className="relative w-full" style={{ paddingTop: '56.25%' }}>
              <video
                className="absolute top-0 left-0 w-full h-full"
                controls
                autoPlay
                src="/Stackyn-intro.mp4"
              >
                Your browser does not support the video tag.
              </video>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
