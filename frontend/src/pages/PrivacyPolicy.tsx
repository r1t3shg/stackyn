import { Link, useNavigate } from 'react-router-dom';
import { useAuth } from '@/contexts/AuthContext';
import Logo from '@/components/Logo';

export default function PrivacyPolicy() {
  const { user } = useAuth();
  const navigate = useNavigate();

  const handleSignInClick = (e: React.MouseEvent<HTMLAnchorElement>) => {
    e.preventDefault();
    if (user) {
      // In local development, navigate to home. In production, redirect to console subdomain
      if (window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1') {
        navigate('/');
      } else {
        window.location.href = 'https://console.stackyn.com/';
      }
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
                    href={window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1' ? '/' : 'https://console.stackyn.com/'}
                    onClick={(e) => {
                      e.preventDefault();
                      if (window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1') {
                        navigate('/');
                      } else {
                        window.location.href = 'https://console.stackyn.com/';
                      }
                    }}
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
        <h1 className="text-4xl font-bold text-[var(--text-primary)] mb-8">Privacy Policy</h1>
        <div className="prose prose-lg max-w-none">
          <p className="text-[var(--text-secondary)] mb-6">Effective Date: January 2026</p>

          <p className="text-[var(--text-primary)] mb-4">
            Stackyn ("we", "our", or "us") respects your privacy. This Privacy Policy explains how we collect, use, and protect your information when you use the Stackyn platform ("Service").
          </p>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">1. Information We Collect</h2>

            <h3 className="text-xl font-medium text-[var(--text-primary)] mb-2">a. Information You Provide</h3>
            <ul className="list-disc list-inside text-[var(--text-primary)] mb-4 space-y-2">
              <li>Email address</li>
              <li>Account details</li>
              <li>Application configuration data</li>
              <li>Support communications</li>
            </ul>

            <h3 className="text-xl font-medium text-[var(--text-primary)] mb-2">b. Automatically Collected Information</h3>
            <ul className="list-disc list-inside text-[var(--text-primary)] mb-4 space-y-2">
              <li>IP address</li>
              <li>Browser type and device information</li>
              <li>Usage data and logs related to deployments</li>
            </ul>

            <h3 className="text-xl font-medium text-[var(--text-primary)] mb-2">c. Payment Information</h3>
            <ul className="list-disc list-inside text-[var(--text-primary)] mb-4 space-y-2">
              <li>Payments are processed by Lemon Squeezy.</li>
              <li>We do not store or process credit card details.</li>
            </ul>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">2. How We Use Your Information</h2>
            <p className="text-[var(--text-primary)] mb-4">
              We use your information to:
            </p>
            <ul className="list-disc list-inside text-[var(--text-primary)] mb-4 space-y-2">
              <li>Create and manage your account</li>
              <li>Provide and operate the Service</li>
              <li>Manage subscriptions and billing status</li>
              <li>Send transactional emails (trial start, trial end, billing)</li>
              <li>Improve platform performance and reliability</li>
            </ul>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">3. Email Communications</h2>
            <p className="text-[var(--text-primary)] mb-4">
              We send service-related emails using Resend, including:
            </p>
            <ul className="list-disc list-inside text-[var(--text-primary)] mb-4 space-y-2">
              <li>Account verification</li>
              <li>Trial expiration notices</li>
              <li>Subscription and billing updates</li>
              <li>Critical service notifications</li>
            </ul>
            <p className="text-[var(--text-primary)] mb-4">
              You may not opt out of essential service emails.
            </p>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">4. Data Sharing</h2>
            <p className="text-[var(--text-primary)] mb-4">
              We only share data with trusted third parties necessary to operate Stackyn, including:
            </p>
            <ul className="list-disc list-inside text-[var(--text-primary)] mb-4 space-y-2">
              <li>Lemon Squeezy (payments & subscriptions)</li>
              <li>Infrastructure providers (hosting & networking)</li>
              <li>Email service providers (transactional emails)</li>
            </ul>
            <p className="text-[var(--text-primary)] mb-4">
              We do not sell or rent your personal data.
            </p>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">5. Cookies & Tracking</h2>
            <p className="text-[var(--text-primary)] mb-4">
              We use cookies for:
            </p>
            <ul className="list-disc list-inside text-[var(--text-primary)] mb-4 space-y-2">
              <li>Authentication</li>
              <li>Session management</li>
              <li>Basic analytics</li>
            </ul>
            <p className="text-[var(--text-primary)] mb-4">
              You may disable cookies, but some features may not function properly.
            </p>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">6. Data Security</h2>
            <p className="text-[var(--text-primary)] mb-4">
              We take reasonable steps to protect your data using industry-standard security measures.
              However, no method of transmission or storage is 100% secure.
            </p>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">7. Data Retention</h2>
            <ul className="list-disc list-inside text-[var(--text-primary)] mb-4 space-y-2">
              <li>We retain data while your account is active</li>
              <li>Trial accounts may be deleted after extended inactivity</li>
              <li>You may request account deletion by contacting support</li>
            </ul>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">8. Your Rights</h2>
            <p className="text-[var(--text-primary)] mb-4">
              Depending on your location, you may have the right to:
            </p>
            <ul className="list-disc list-inside text-[var(--text-primary)] mb-4 space-y-2">
              <li>Access your personal data</li>
              <li>Request correction or deletion</li>
              <li>Withdraw consent where applicable</li>
            </ul>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">9. Children's Privacy</h2>
            <p className="text-[var(--text-primary)] mb-4">
              Stackyn is not intended for individuals under the age of 18.
              We do not knowingly collect data from minors.
            </p>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">10. Changes to This Policy</h2>
            <p className="text-[var(--text-primary)] mb-4">
              We may update this Privacy Policy from time to time.
              Continued use of the Service constitutes acceptance of any changes.
            </p>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">11. Contact Us</h2>
            <p className="text-[var(--text-primary)] mb-4">
              If you have questions about this Privacy Policy:
            </p>
            <p className="text-[var(--text-primary)] mb-4">
              <strong><a href="mailto:support@stackyn.com" className="text-[var(--primary)] hover:underline">support@stackyn.com</a></strong>
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
              <Link to="/about" className="hover:text-[var(--text-primary)] transition-colors">
                About Us
              </Link>
              <Link to="/terms" className="hover:text-[var(--text-primary)] transition-colors">
                Terms of Service
              </Link>
              <Link to="/privacy" className="hover:text-[var(--text-primary)] transition-colors">
                Privacy Policy
              </Link>
              <Link to="/refund" className="hover:text-[var(--text-primary)] transition-colors">
                Refund Policy
              </Link>
            </div>
          </div>
        </div>
      </footer>
    </div>
  );
}
