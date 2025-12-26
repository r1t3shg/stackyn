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
            {/* Logo */}
            <div className="flex-shrink-0">
              <Logo height={40} />
            </div>

            {/* Desktop Navigation Links */}
            <nav className="hidden md:flex items-center space-x-8">
              <Link
                to="/terms"
                className="text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors font-medium"
              >
                Terms of Service
              </Link>
              <Link
                to="/privacy"
                className="text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors font-medium"
              >
                Privacy Policy
              </Link>
              <Link
                to="/pricing"
                className="text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors font-medium"
              >
                Pricing
              </Link>
            </nav>

            {/* Desktop Sign In CTA */}
            <div className="hidden md:flex items-center">
              <a
                href={user ? "https://console.staging.stackyn.com/" : "/login"}
                onClick={handleSignInClick}
                className="bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-[var(--app-bg)] font-medium py-2 px-6 rounded-lg transition-colors"
              >
                Sign in
              </a>
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
                <Link
                  to="/terms"
                  className="text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors font-medium"
                  onClick={() => setMobileMenuOpen(false)}
                >
                  Terms of Service
                </Link>
                <Link
                  to="/privacy"
                  className="text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors font-medium"
                  onClick={() => setMobileMenuOpen(false)}
                >
                  Privacy Policy
                </Link>
                <Link
                  to="/pricing"
                  className="text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors font-medium"
                  onClick={() => setMobileMenuOpen(false)}
                >
                  Pricing
                </Link>
                <a
                  href={user ? "https://console.staging.stackyn.com/" : "/login"}
                  onClick={handleSignInClick}
                  className="bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-[var(--app-bg)] font-medium py-2 px-6 rounded-lg transition-colors text-center"
                >
                  Sign in
                </a>
              </nav>
            </div>
          )}
        </div>
      </header>

      {/* Hero Section */}
      <section className="relative overflow-hidden bg-[var(--surface)]">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-24 sm:py-32">
          <div className="text-center">
            <h1 className="text-5xl sm:text-6xl font-bold text-[var(--text-primary)] mb-6">
              Deploy Your Apps with Ease
            </h1>
            <p className="text-xl text-[var(--text-secondary)] max-w-3xl mx-auto mb-8">
              Stackyn is a modern Platform-as-a-Service that makes deploying applications
              from Git repositories simple and fast. Focus on building, we handle the infrastructure.
            </p>
            <div className="flex justify-center gap-4">
              <a
                href={user ? "https://console.staging.stackyn.com/" : "/login"}
                onClick={handleSignInClick}
                className="bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-[var(--app-bg)] font-semibold py-3 px-8 rounded-lg transition-colors text-lg"
              >
                Sign in
              </a>
              <Link
                to="/pricing"
                className="bg-[var(--app-bg)] hover:bg-[var(--elevated)] text-[var(--text-primary)] font-semibold py-3 px-8 rounded-lg transition-colors text-lg border border-gray-300"
              >
                View Pricing
              </Link>
            </div>
          </div>
        </div>
      </section>

      {/* Features Section */}
      <section className="py-24 bg-[var(--app-bg)]">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center mb-16">
            <h2 className="text-3xl sm:text-4xl font-bold text-[var(--text-primary)] mb-4">
              Why Choose Stackyn?
            </h2>
            <p className="text-lg text-[var(--text-secondary)] max-w-2xl mx-auto">
              Everything you need to deploy and manage your applications in one platform
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
            <div className="p-6 rounded-xl border border-[var(--border-subtle)] hover:shadow-lg transition-shadow">
              <div className="w-12 h-12 bg-[var(--primary-muted)] rounded-lg flex items-center justify-center mb-4">
                <svg className="w-6 h-6 text-[var(--primary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
                </svg>
              </div>
              <h3 className="text-xl font-semibold text-[var(--text-primary)] mb-2">Git-Based Deployments</h3>
              <p className="text-[var(--text-secondary)]">
                Connect your Git repository and deploy with a single click. We automatically
                build and deploy your application using Docker containers.
              </p>
            </div>

            <div className="p-6 rounded-xl border border-[var(--border-subtle)] hover:shadow-lg transition-shadow">
              <div className="w-12 h-12 bg-[var(--primary-muted)] rounded-lg flex items-center justify-center mb-4">
                <svg className="w-6 h-6 text-[var(--success)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                </svg>
              </div>
              <h3 className="text-xl font-semibold text-[var(--text-primary)] mb-2">Fast & Scalable</h3>
              <p className="text-[var(--text-secondary)]">
                Built on Docker and modern infrastructure, Stackyn ensures your applications
                are fast, reliable, and can scale with your needs.
              </p>
            </div>

            <div className="p-6 rounded-xl border border-[var(--border-subtle)] hover:shadow-lg transition-shadow">
              <div className="w-12 h-12 bg-[var(--elevated)] rounded-lg flex items-center justify-center mb-4">
                <svg className="w-6 h-6 text-[var(--info)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
                </svg>
              </div>
              <h3 className="text-xl font-semibold text-[var(--text-primary)] mb-2">Secure by Default</h3>
              <p className="text-[var(--text-secondary)]">
                Your applications are deployed with automatic SSL/TLS certificates and
                follow security best practices out of the box.
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* How It Works Section */}
      <section className="py-24 bg-[var(--elevated)]">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center mb-16">
            <h2 className="text-3xl sm:text-4xl font-bold text-[var(--text-primary)] mb-4">
              How It Works
            </h2>
            <p className="text-lg text-[var(--text-secondary)] max-w-2xl mx-auto">
              Deploy your application in three simple steps
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
            <div className="text-center">
              <div className="w-16 h-16 bg-[var(--primary)] text-[var(--app-bg)] rounded-full flex items-center justify-center text-2xl font-bold mx-auto mb-4">
                1
              </div>
              <h3 className="text-xl font-semibold text-[var(--text-primary)] mb-2">Connect Your Repository</h3>
              <p className="text-[var(--text-secondary)]">
                Link your Git repository to Stackyn. We support any repository with a Dockerfile.
              </p>
            </div>

            <div className="text-center">
              <div className="w-16 h-16 bg-[var(--primary)] text-[var(--app-bg)] rounded-full flex items-center justify-center text-2xl font-bold mx-auto mb-4">
                2
              </div>
              <h3 className="text-xl font-semibold text-[var(--text-primary)] mb-2">Configure Your App</h3>
              <p className="text-[var(--text-secondary)]">
                Set your app name, environment variables, and deployment settings in our console.
              </p>
            </div>

            <div className="text-center">
              <div className="w-16 h-16 bg-[var(--primary)] text-[var(--app-bg)] rounded-full flex items-center justify-center text-2xl font-bold mx-auto mb-4">
                3
              </div>
              <h3 className="text-xl font-semibold text-[var(--text-primary)] mb-2">Deploy & Monitor</h3>
              <p className="text-[var(--text-secondary)]">
                Deploy with one click and monitor your application's logs and status in real-time.
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="py-24 bg-[var(--primary)]">
        <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 text-center">
          <h2 className="text-3xl sm:text-4xl font-bold text-[var(--app-bg)] mb-4">
            Ready to Deploy Your First App?
          </h2>
          <p className="text-xl text-[var(--text-secondary)] mb-8">
            Join developers who are already using Stackyn to simplify their deployment process.
          </p>
          <a
            href={user ? "https://console.staging.stackyn.com/" : "/login"}
            onClick={handleSignInClick}
            className="inline-block bg-[var(--app-bg)] hover:bg-gray-100 text-[var(--primary)] font-semibold py-3 px-8 rounded-lg transition-colors text-lg"
          >
            Sign in
          </a>
        </div>
      </section>

      {/* Footer */}
      <footer className="bg-[var(--sidebar)] text-[var(--text-muted)] py-12">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="grid grid-cols-1 md:grid-cols-4 gap-8">
            <div>
              <div className="mb-4">
                <Logo height={32} />
              </div>
              <p className="text-sm">
                Modern Platform-as-a-Service for deploying applications from Git repositories.
              </p>
            </div>
            <div>
              <h4 className="text-[var(--app-bg)] font-semibold mb-4">Product</h4>
              <ul className="space-y-2 text-sm">
                <li>
                  <Link to="/pricing" className="hover:text-[var(--app-bg)] transition-colors">
                    Pricing
                  </Link>
                </li>
                <li>
                  <a 
                    href={user ? "https://console.staging.stackyn.com/" : "/login"} 
                    onClick={handleSignInClick}
                    className="hover:text-[var(--app-bg)] transition-colors"
                  >
                    Sign in
                  </a>
                </li>
              </ul>
            </div>
            <div>
              <h4 className="text-[var(--app-bg)] font-semibold mb-4">Legal</h4>
              <ul className="space-y-2 text-sm">
                <li>
                  <Link to="/terms" className="hover:text-[var(--app-bg)] transition-colors">
                    Terms of Service
                  </Link>
                </li>
                <li>
                  <Link to="/privacy" className="hover:text-[var(--app-bg)] transition-colors">
                    Privacy Policy
                  </Link>
                </li>
              </ul>
            </div>
            <div>
              <h4 className="text-[var(--app-bg)] font-semibold mb-4">Get Started</h4>
              <a
                href={user ? "https://console.staging.stackyn.com/" : "/login"}
                onClick={handleSignInClick}
                className="inline-block bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-[var(--app-bg)] font-medium py-2 px-6 rounded-lg transition-colors text-sm"
              >
                Sign in
              </a>
            </div>
          </div>
          <div className="mt-8 pt-8 border-t border-[var(--border-subtle)] text-center text-sm">
            <p>&copy; {new Date().getFullYear()} Stackyn. All rights reserved.</p>
          </div>
        </div>
      </footer>
    </div>
  );
}

