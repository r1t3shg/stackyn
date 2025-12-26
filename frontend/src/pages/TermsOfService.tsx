import { Link, useNavigate } from 'react-router-dom';
import { useAuth } from '@/contexts/AuthContext';
import Logo from '@/components/Logo';

export default function TermsOfService() {
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
                  href="/#why-stackyn"
                  className="text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors font-medium"
                >
                  Why Stackyn?
                </a>
                <a
                  href="/#features"
                  className="text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors font-medium"
                >
                  Features
                </a>
                <a
                  href="/#pricing"
                  className="text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors font-medium"
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
                  Sign in
                </a>
              )}
            </div>

            {/* Mobile menu button */}
            <div className="md:hidden flex items-center">
              <button
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
                  <path d="M4 6h16M4 12h16M4 18h16" />
                </svg>
              </button>
            </div>
          </div>
        </div>
      </header>

      {/* Content */}
      <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-16">
        <h1 className="text-4xl font-bold text-[var(--text-primary)] mb-8">Terms of Service</h1>
        <div className="prose prose-lg max-w-none">
          <p className="text-[var(--text-secondary)] mb-6">Last updated: {new Date().toLocaleDateString()}</p>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">1. Acceptance of Terms</h2>
            <p className="text-[var(--text-primary)] mb-4">
              By accessing and using Stackyn ("the Service"), you accept and agree to be bound by the terms
              and provision of this agreement. If you do not agree to abide by the above, please do not use
              this service.
            </p>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-gray-900 mb-4">2. Use License</h2>
            <p className="text-gray-700 mb-4">
              Permission is granted to temporarily use Stackyn for personal and commercial purposes. This is
              the grant of a license, not a transfer of title, and under this license you may not:
            </p>
            <ul className="list-disc list-inside text-[var(--text-primary)] mb-4 space-y-2">
              <li>Modify or copy the materials</li>
              <li>Use the materials for any commercial purpose or for any public display</li>
              <li>Attempt to reverse engineer any software contained in Stackyn</li>
              <li>Remove any copyright or other proprietary notations from the materials</li>
            </ul>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">3. Service Availability</h2>
            <p className="text-[var(--text-primary)] mb-4">
              Stackyn strives to provide reliable service, but we do not guarantee uninterrupted or
              error-free service. We reserve the right to modify, suspend, or discontinue any part of the
              Service at any time with or without notice.
            </p>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">4. User Responsibilities</h2>
            <p className="text-[var(--text-primary)] mb-4">
              You are responsible for:
            </p>
            <ul className="list-disc list-inside text-[var(--text-primary)] mb-4 space-y-2">
              <li>Maintaining the confidentiality of your account credentials</li>
              <li>All activities that occur under your account</li>
              <li>Ensuring your applications comply with applicable laws and regulations</li>
              <li>Not using the Service for any illegal or unauthorized purpose</li>
            </ul>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">5. Prohibited Uses</h2>
            <p className="text-[var(--text-primary)] mb-4">
              You may not use Stackyn:
            </p>
            <ul className="list-disc list-inside text-[var(--text-primary)] mb-4 space-y-2">
              <li>In any way that violates any applicable law or regulation</li>
              <li>To transmit any malicious code, viruses, or harmful data</li>
              <li>To interfere with or disrupt the Service or servers</li>
              <li>To attempt to gain unauthorized access to any part of the Service</li>
            </ul>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">6. Limitation of Liability</h2>
            <p className="text-[var(--text-primary)] mb-4">
              In no event shall Stackyn or its suppliers be liable for any damages (including, without
              limitation, damages for loss of data or profit, or due to business interruption) arising out
              of the use or inability to use the Service, even if Stackyn or a Stackyn authorized
              representative has been notified orally or in writing of the possibility of such damage.
            </p>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">7. Termination</h2>
            <p className="text-[var(--text-primary)] mb-4">
              We may terminate or suspend your account and access to the Service immediately, without prior
              notice or liability, for any reason whatsoever, including without limitation if you breach
              the Terms. Upon termination, your right to use the Service will immediately cease.
            </p>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">8. Changes to Terms</h2>
            <p className="text-[var(--text-primary)] mb-4">
              Stackyn reserves the right to revise these terms of service at any time without notice. By
              using this Service you are agreeing to be bound by the then current version of these terms
              of service.
            </p>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">9. Contact Information</h2>
            <p className="text-[var(--text-primary)] mb-4">
              If you have any questions about these Terms of Service, please contact us through our
              support channels.
            </p>
          </section>
        </div>
      </div>

      {/* Footer */}
      <footer className="bg-[var(--sidebar)] text-[var(--text-muted)] py-12 mt-16 border-t border-[var(--border-subtle)]">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex flex-col md:flex-row justify-between items-center gap-4">
            <div className="text-sm">
              <p>&copy; {new Date().getFullYear()} Stackyn. All rights reserved.</p>
            </div>
            <div className="flex items-center gap-6 text-sm">
              <Link to="/terms" className="hover:text-[var(--text-primary)] transition-colors">
                Terms of Service
              </Link>
              <Link to="/privacy" className="hover:text-[var(--text-primary)] transition-colors">
                Privacy Policy
              </Link>
            </div>
          </div>
        </div>
      </footer>
    </div>
  );
}

