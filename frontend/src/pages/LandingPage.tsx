import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useAuth } from '@/contexts/AuthContext';
import Logo from '@/components/Logo';

export default function LandingPage() {
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const { user } = useAuth();
  const navigate = useNavigate();

  const handleSignInClick = (e: React.MouseEvent<HTMLAnchorElement>) => {
    e.preventDefault();
    if (user) {
      window.location.href = 'https://console.staging.stackyn.com/';
    } else {
      navigate('/login');
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
            <div className="hidden md:flex items-center gap-3">
              {user ? (
                <>
                  <a
                    href="https://console.staging.stackyn.com/"
                    className="bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-[var(--app-bg)] font-medium py-2 px-6 rounded-lg transition-colors"
                  >
                    Go to Console
                  </a>
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
                Login
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
                    <a
                      href="https://console.staging.stackyn.com/"
                      className="bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-[var(--app-bg)] font-medium py-2 px-6 rounded-lg transition-colors text-center"
                    >
                      Go to Console
                    </a>
                  </>
                ) : (
                  <a
                    href="/login"
                  onClick={handleSignInClick}
                    className="bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-[var(--app-bg)] font-medium py-2 px-6 rounded-lg transition-colors text-center"
                >
                  Login
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
              <a
                href={user ? "https://console.staging.stackyn.com/" : "/signup"}
                onClick={(e) => {
                  if (!user) {
                    e.preventDefault();
                    navigate('/signup');
                  } else {
                    handleSignInClick(e);
                  }
                }}
                className="bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-[var(--app-bg)] font-semibold py-4 px-8 rounded-lg transition-colors text-lg"
              >
                Get Started Free
              </a>
              <button
                className="bg-[var(--elevated)] hover:bg-[var(--surface)] text-[var(--text-primary)] font-semibold py-4 px-8 rounded-lg transition-colors text-lg border border-[var(--border-subtle)]"
              >
                Watch Demo
              </button>
            </div>
            {/* Hero Visual Placeholder - Dashboard mockup */}
            <div className="mt-16 rounded-lg border border-[var(--border-subtle)] bg-[var(--elevated)] p-8 overflow-hidden">
              <div className="bg-[var(--terminal-bg)] rounded-lg p-6 font-mono text-sm text-[var(--text-primary)]">
                <div className="flex items-center gap-2 mb-4">
                  <div className="w-3 h-3 rounded-full bg-[var(--success)]"></div>
                  <span className="text-[var(--text-muted)]">app.example.com</span>
                  <span className="text-[var(--text-muted)]">•</span>
                  <span className="text-[var(--success)]">Healthy</span>
                </div>
                <div className="space-y-2">
                  <div className="text-[var(--text-secondary)]">$ git push origin main</div>
                  <div className="text-[var(--success)]">✓ Building...</div>
                  <div className="text-[var(--success)]">✓ Deploying...</div>
                  <div className="text-[var(--success)]">✓ Live at https://app.example.com</div>
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
                Push your code from GitHub, GitLab, or Bitbucket.
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
                Real-time logs, health status, and error tracking.
              </p>
            </div>

            <div className="p-6 rounded-lg border border-[var(--border-subtle)] bg-[var(--elevated)] hover:border-[var(--border-strong)] transition-colors">
              <div className="w-10 h-10 bg-[var(--primary-muted)] rounded-lg flex items-center justify-center mb-4">
                <svg className="w-6 h-6 text-[var(--primary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
                </svg>
              </div>
              <h3 className="text-xl font-semibold text-[var(--text-primary)] mb-2">Git-First Workflow</h3>
              <p className="text-[var(--text-secondary)]">
                Automatic builds on push to main/develop.
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
            <h1 className="text-4xl sm:text-5xl font-bold text-[var(--text-primary)] mb-4">Pricing</h1>
            <h2 className="text-2xl sm:text-3xl font-semibold text-[var(--text-primary)] mb-4">
              Simple pricing. No surprises.
            </h2>
          </div>

          {/* Plans Grid */}
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-20">
            {/* Free Plan */}
            <div className="border border-[var(--border-subtle)] rounded-lg p-6 bg-[var(--elevated)]">
              <h3 className="text-2xl font-bold text-[var(--text-primary)] mb-2">Free</h3>
              <ul className="space-y-3 mb-8">
                <li className="flex items-start text-sm">
                  <svg className="w-5 h-5 text-[var(--success)] mr-2 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                  </svg>
                  <span className="text-[var(--text-secondary)]">128 MB RAM</span>
                </li>
                <li className="flex items-start text-sm">
                  <svg className="w-5 h-5 text-[var(--success)] mr-2 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                  </svg>
                  <span className="text-[var(--text-secondary)]">1 app</span>
                </li>
                <li className="flex items-start text-sm">
                  <svg className="w-5 h-5 text-[var(--text-muted)] mr-2 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                  <span className="text-[var(--text-secondary)]">Sleeps</span>
                </li>
                <li className="flex items-start text-sm">
                  <svg className="w-5 h-5 text-[var(--text-muted)] mr-2 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                  <span className="text-[var(--text-secondary)]">No workers</span>
                </li>
                <li className="flex items-start text-sm">
                  <svg className="w-5 h-5 text-[var(--text-muted)] mr-2 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                  <span className="text-[var(--text-secondary)]">No domains</span>
                </li>
              </ul>
              <div className="mb-6">
                <div className="text-3xl font-bold text-[var(--text-primary)] mb-2">$0</div>
              </div>
              <a
                href={user ? "https://console.staging.stackyn.com/" : "/signup?plan=free"}
                onClick={(e) => {
                  if (!user) {
                    e.preventDefault();
                    navigate('/signup?plan=free');
                  } else {
                    handleSignInClick(e);
                  }
                }}
                className="block w-full text-center bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-[var(--app-bg)] font-medium py-3 px-6 rounded-lg transition-colors"
              >
                Get Started
              </a>
            </div>

            {/* Starter Plan */}
            <div className="border border-[var(--border-subtle)] rounded-lg p-6 bg-[var(--elevated)]">
              <h3 className="text-2xl font-bold text-[var(--text-primary)] mb-2">Starter</h3>
              <ul className="space-y-3 mb-8">
                <li className="flex items-start text-sm">
                  <svg className="w-5 h-5 text-[var(--success)] mr-2 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                  </svg>
                  <span className="text-[var(--text-secondary)]">256 MB RAM</span>
                </li>
                <li className="flex items-start text-sm">
                  <svg className="w-5 h-5 text-[var(--success)] mr-2 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                  </svg>
                  <span className="text-[var(--text-secondary)]">Always-on</span>
                </li>
                <li className="flex items-start text-sm">
                  <svg className="w-5 h-5 text-[var(--success)] mr-2 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                  </svg>
                  <span className="text-[var(--text-secondary)]">1 app</span>
                </li>
                <li className="flex items-start text-sm">
                  <svg className="w-5 h-5 text-[var(--success)] mr-2 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                  </svg>
                  <span className="text-[var(--text-secondary)]">Custom domain</span>
                </li>
                <li className="flex items-start text-sm">
                  <svg className="w-5 h-5 text-[var(--success)] mr-2 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                  </svg>
                  <span className="text-[var(--text-secondary)]">Manual deploy</span>
                </li>
              </ul>
              <div className="mb-6">
                <div className="text-3xl font-bold text-[var(--text-primary)] mb-2">$5 <span className="text-lg font-normal text-[var(--text-muted)]">/ month</span></div>
              </div>
              <a
                href={user ? "https://console.staging.stackyn.com/" : "/signup?plan=starter"}
                onClick={(e) => {
                  if (!user) {
                    e.preventDefault();
                    navigate('/signup?plan=starter');
                  } else {
                    handleSignInClick(e);
                  }
                }}
                className="block w-full text-center bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-[var(--app-bg)] font-medium py-3 px-6 rounded-lg transition-colors"
              >
                Start Starter
              </a>
            </div>

            {/* Builder Plan */}
            <div className="border-2 border-[var(--primary)] rounded-lg p-6 bg-[var(--elevated)] relative">
              <div className="absolute top-0 right-0 bg-[var(--primary)] text-[var(--app-bg)] px-3 py-1 rounded-bl-lg rounded-tr-lg text-xs font-medium">
                Most Popular
              </div>
              <h3 className="text-2xl font-bold text-[var(--text-primary)] mb-2">Builder</h3>
              <ul className="space-y-3 mb-8">
                <li className="flex items-start text-sm">
                  <svg className="w-5 h-5 text-[var(--success)] mr-2 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                  </svg>
                  <span className="text-[var(--text-secondary)]">512 MB RAM</span>
                </li>
                <li className="flex items-start text-sm">
                  <svg className="w-5 h-5 text-[var(--success)] mr-2 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                  </svg>
                  <span className="text-[var(--text-secondary)]">3 apps</span>
                </li>
                <li className="flex items-start text-sm">
                  <svg className="w-5 h-5 text-[var(--success)] mr-2 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                  </svg>
                  <span className="text-[var(--text-secondary)]">Auto deploy</span>
                </li>
                <li className="flex items-start text-sm">
                  <svg className="w-5 h-5 text-[var(--success)] mr-2 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                  </svg>
                  <span className="text-[var(--text-secondary)]">Health checks</span>
                </li>
                <li className="flex items-start text-sm">
                  <svg className="w-5 h-5 text-[var(--success)] mr-2 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                  </svg>
                  <span className="text-[var(--text-secondary)]">Logs</span>
                </li>
              </ul>
              <div className="mb-6">
                <div className="text-3xl font-bold text-[var(--text-primary)] mb-2">$15 <span className="text-lg font-normal text-[var(--text-muted)]">/ month</span></div>
              </div>
              <a
                href={user ? "https://console.staging.stackyn.com/" : "/signup?plan=builder"}
                onClick={(e) => {
                  if (!user) {
                    e.preventDefault();
                    navigate('/signup?plan=builder');
                  } else {
                    handleSignInClick(e);
                  }
                }}
                className="block w-full text-center bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-[var(--app-bg)] font-medium py-3 px-6 rounded-lg transition-colors"
              >
                Start Builder
              </a>
            </div>

            {/* Pro Plan */}
            <div className="border border-[var(--border-subtle)] rounded-lg p-6 bg-[var(--elevated)]">
              <h3 className="text-2xl font-bold text-[var(--text-primary)] mb-2">Pro</h3>
              <ul className="space-y-3 mb-8">
                <li className="flex items-start text-sm">
                  <svg className="w-5 h-5 text-[var(--success)] mr-2 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                  </svg>
                  <span className="text-[var(--text-secondary)]">1 GB RAM</span>
                </li>
                <li className="flex items-start text-sm">
                  <svg className="w-5 h-5 text-[var(--success)] mr-2 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                  </svg>
                  <span className="text-[var(--text-secondary)]">5 apps</span>
                </li>
                <li className="flex items-start text-sm">
                  <svg className="w-5 h-5 text-[var(--success)] mr-2 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                  </svg>
                  <span className="text-[var(--text-secondary)]">Zero-downtime</span>
                </li>
                <li className="flex items-start text-sm">
                  <svg className="w-5 h-5 text-[var(--success)] mr-2 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                  </svg>
                  <span className="text-[var(--text-secondary)]">Workers</span>
                </li>
                <li className="flex items-start text-sm">
                  <svg className="w-5 h-5 text-[var(--success)] mr-2 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                  </svg>
                  <span className="text-[var(--text-secondary)]">Priority builds</span>
                </li>
              </ul>
              <div className="mb-6">
                <div className="text-3xl font-bold text-[var(--text-primary)] mb-2">$29 <span className="text-lg font-normal text-[var(--text-muted)]">/ month</span></div>
              </div>
              <a
                href={user ? "https://console.staging.stackyn.com/" : "/signup?plan=pro"}
                onClick={(e) => {
                  if (!user) {
                    e.preventDefault();
                    navigate('/signup?plan=pro');
                  } else {
                    handleSignInClick(e);
                  }
                }}
                className="block w-full text-center bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-[var(--app-bg)] font-medium py-3 px-6 rounded-lg transition-colors"
              >
                Start Pro
              </a>
            </div>
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
            <a
              href={user ? "https://console.staging.stackyn.com/" : "/signup"}
              onClick={(e) => {
                if (!user) {
                  e.preventDefault();
                  navigate('/signup');
                } else {
                  handleSignInClick(e);
                }
              }}
              className="inline-block bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-[var(--app-bg)] font-semibold py-4 px-8 rounded-lg transition-colors text-lg mb-4"
            >
              Get Started Free
            </a>
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

      {/* Testimonials Section */}
      <section className="py-24 bg-[var(--app-bg)]">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center mb-16">
            <h2 className="text-3xl sm:text-4xl lg:text-5xl font-bold text-[var(--text-primary)] mb-4">
              Loved by Developers Around the World
            </h2>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-8 max-w-4xl mx-auto">
            <div className="p-8 rounded-lg border border-[var(--border-subtle)] bg-[var(--surface)]">
              <div className="flex items-center mb-4">
                <div className="w-12 h-12 rounded-full bg-[var(--primary-muted)] flex items-center justify-center mr-4">
                  <span className="text-[var(--primary)] font-semibold">IH</span>
                </div>
                <div>
                  <div className="font-semibold text-[var(--text-primary)]">Indie Hacker</div>
                  <div className="text-sm text-[var(--text-muted)]">Developer</div>
                </div>
              </div>
              <p className="text-[var(--text-secondary)] italic">
                "Stackyn made deploying my API effortless. I saved hours of setup."
              </p>
            </div>

            <div className="p-8 rounded-lg border border-[var(--border-subtle)] bg-[var(--surface)]">
              <div className="flex items-center mb-4">
                <div className="w-12 h-12 rounded-full bg-[var(--primary-muted)] flex items-center justify-center mr-4">
                  <span className="text-[var(--primary)] font-semibold">SF</span>
                </div>
                <div>
                  <div className="font-semibold text-[var(--text-primary)]">SaaS Founder</div>
                  <div className="text-sm text-[var(--text-muted)]">Founder</div>
                </div>
              </div>
              <p className="text-[var(--text-secondary)] italic">
                "Finally, a PaaS that feels designed for backend developers."
              </p>
            </div>
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
            <a
            href={user ? "https://console.staging.stackyn.com/" : "/signup"}
            onClick={(e) => {
              if (!user) {
                e.preventDefault();
                navigate('/signup');
              } else {
                handleSignInClick(e);
              }
            }}
              className="bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-[var(--app-bg)] font-semibold py-4 px-8 rounded-lg transition-colors text-lg"
            >
              Get Started Free
            </a>
            <a
              href="#pricing"
              className="bg-[var(--elevated)] hover:bg-[var(--surface)] text-[var(--text-primary)] font-semibold py-4 px-8 rounded-lg transition-colors text-lg border border-[var(--border-subtle)]"
              onClick={(e) => {
                e.preventDefault();
                document.getElementById('pricing')?.scrollIntoView({ behavior: 'smooth' });
              }}
            >
              See How It Works
            </a>
          </div>
          <p className="text-sm text-[var(--text-muted)]">
            No credit card required • Works with GitHub, GitLab, Bitbucket
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
                  <a href="#" className="hover:text-[var(--text-primary)] transition-colors">
                    Contact
                  </a>
                </li>
              </ul>
            </div>
            <div>
              <h4 className="text-[var(--text-primary)] font-semibold mb-4">Connect</h4>
              <div className="flex space-x-4">
                <a href="#" className="hover:text-[var(--text-primary)] transition-colors" aria-label="GitHub">
                  <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
                    <path fillRule="evenodd" d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z" clipRule="evenodd" />
                  </svg>
                </a>
                <a href="#" className="hover:text-[var(--text-primary)] transition-colors" aria-label="Twitter">
                  <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
                    <path d="M8.29 20.251c7.547 0 11.675-6.253 11.675-11.675 0-.178 0-.355-.012-.53A8.348 8.348 0 0022 5.92a8.19 8.19 0 01-2.357.646 4.118 4.118 0 001.804-2.27 8.224 8.224 0 01-2.605.996 4.107 4.107 0 00-6.993 3.743 11.65 11.65 0 01-8.457-4.287 4.106 4.106 0 001.27 5.477A4.072 4.072 0 012.8 9.713v.052a4.105 4.105 0 003.292 4.022 4.095 4.095 0 01-1.853.07 4.108 4.108 0 003.834 2.85A8.233 8.233 0 012 18.407a11.616 11.616 0 006.29 1.84" />
                  </svg>
                </a>
                <a href="#" className="hover:text-[var(--text-primary)] transition-colors" aria-label="LinkedIn">
                  <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
                    <path d="M19 0h-14c-2.761 0-5 2.239-5 5v14c0 2.761 2.239 5 5 5h14c2.762 0 5-2.239 5-5v-14c0-2.761-2.238-5-5-5zm-11 19h-3v-11h3v11zm-1.5-12.268c-.966 0-1.75-.79-1.75-1.764s.784-1.764 1.75-1.764 1.75.79 1.75 1.764-.783 1.764-1.75 1.764zm13.5 12.268h-3v-5.604c0-3.368-4-3.113-4 0v5.604h-3v-11h3v1.765c1.396-2.586 7-2.777 7 2.476v6.759z" />
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
    </div>
  );
}
