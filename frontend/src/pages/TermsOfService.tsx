import { Link, useNavigate } from 'react-router-dom';
import { useAuth } from '@/contexts/AuthContext';
import Logo from '@/components/Logo';

export default function TermsOfService() {
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
        <h1 className="text-4xl font-bold text-[var(--text-primary)] mb-8">Terms of Service</h1>
        <div className="prose prose-lg max-w-none">
          <p className="text-[var(--text-secondary)] mb-6">Effective Date: January 2026</p>

          <p className="text-[var(--text-primary)] mb-4">
            Welcome to <strong>Stackyn</strong>. These Terms of Service ("Terms") govern your access to and use of the Stackyn platform and services ("Service"). By using Stackyn, you agree to these Terms.
          </p>

          <p className="text-[var(--text-primary)] mb-8">
            If you do not agree, do not use the Service.
          </p>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">1. About Stackyn</h2>
            <p className="text-[var(--text-primary)] mb-4">
              Stackyn is a developer-focused Platform-as-a-Service (PaaS) that enables users to deploy, manage, and run backend applications on virtual private servers using container-based infrastructure.
            </p>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">2. Eligibility</h2>
            <p className="text-[var(--text-primary)] mb-4">
              You must:
            </p>
            <ul className="list-disc list-inside text-[var(--text-primary)] mb-4 space-y-2">
              <li>Be at least 18 years old</li>
              <li>Provide accurate account information</li>
              <li>Be legally permitted to use this Service in your jurisdiction</li>
            </ul>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">3. Account Responsibility</h2>
            <ul className="list-disc list-inside text-[var(--text-primary)] mb-4 space-y-2">
              <li>You are responsible for maintaining account security</li>
              <li>All activity under your account is your responsibility</li>
              <li>You must notify us immediately of unauthorized use</li>
            </ul>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">4. Free Trial</h2>
            <ul className="list-disc list-inside text-[var(--text-primary)] mb-4 space-y-2">
              <li>Stackyn provides a <strong>7-day free trial</strong></li>
              <li>No payment method is required to start the trial</li>
              <li>After the trial ends:
                <ul className="list-disc list-inside kv-pl-5 mt-2 space-y-1">
                  <li>Deployed applications will be <strong>paused</strong></li>
                  <li>Dashboard access may be restricted</li>
                </ul>
              </li>
              <li>Data may be deleted after prolonged inactivity</li>
            </ul>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">5. Paid Subscriptions</h2>
            <ul className="list-disc list-inside text-[var(--text-primary)] mb-4 space-y-2">
              <li>Paid plans are billed monthly via <strong>Lemon Squeezy</strong></li>
              <li>Subscription fees are charged in advance</li>
              <li>Plan limits (RAM, disk, apps) are enforced automatically</li>
              <li>Failure to pay may result in service suspension</li>
            </ul>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">6. Acceptable Use Policy</h2>
            <p className="text-[var(--text-primary)] mb-4">
              You agree <strong>NOT</strong> to use Stackyn for:
            </p>
            <ul className="list-disc list-inside text-[var(--text-primary)] mb-4 space-y-2">
              <li>Illegal activities</li>
              <li>Malware, phishing, or crypto mining</li>
              <li>Abuse of system resources</li>
              <li>Hosting copyrighted or prohibited content</li>
            </ul>
            <p className="text-[var(--text-primary)] mb-4">
              We reserve the right to suspend accounts violating these rules.
            </p>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">7. Service Availability</h2>
            <p className="text-[var(--text-primary)] mb-4">
              Stackyn is provided on an <strong>"as-is"</strong> and <strong>"as-available"</strong> basis.
            </p>
            <p className="text-[var(--text-primary)] mb-4">
              We do not guarantee:
            </p>
            <ul className="list-disc list-inside text-[var(--text-primary)] mb-4 space-y-2">
              <li>100% uptime</li>
              <li>Error-free operation</li>
              <li>Continuous availability</li>
            </ul>
            <p className="text-[var(--text-primary)] mb-4">
              Maintenance or outages may occur without prior notice.
            </p>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">8. Data & Backups</h2>
            <ul className="list-disc list-inside text-[var(--text-primary)] mb-4 space-y-2">
              <li>You are responsible for backing up your data</li>
              <li>Stackyn is not liable for data loss or corruption</li>
              <li>We do not guarantee data retention after account termination</li>
            </ul>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">9. Termination</h2>
            <p className="text-[var(--text-primary)] mb-4">
              We may suspend or terminate your account if:
            </p>
            <ul className="list-disc list-inside text-[var(--text-primary)] mb-4 space-y-2">
              <li>These Terms are violated</li>
              <li>Abuse or illegal activity is detected</li>
              <li>Required payments are not made</li>
            </ul>
            <p className="text-[var(--text-primary)] mb-4">
              You may cancel your subscription at any time.
            </p>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">10. Refund Policy</h2>
            <ul className="list-disc list-inside text-[var(--text-primary)] mb-4 space-y-2">
              <li>Free trial users are not charged</li>
              <li>Paid subscriptions are <strong>non-refundable</strong></li>
              <li>You may cancel to prevent future charges</li>
            </ul>
            <p className="text-[var(--text-primary)] mb-4">
              Refunds may be issued at our discretion for technical failures.
            </p>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">11. Limitation of Liability</h2>
            <p className="text-[var(--text-primary)] mb-4">
              To the maximum extent permitted by law:
            </p>
            <ul className="list-disc list-inside text-[var(--text-primary)] mb-4 space-y-2">
              <li>Stackyn is not liable for indirect or consequential damages</li>
              <li>Total liability will not exceed the amount paid in the last billing cycle</li>
            </ul>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">12. Changes to Terms</h2>
            <p className="text-[var(--text-primary)] mb-4">
              We may update these Terms from time to time. Continued use of the Service means acceptance of updated Terms.
            </p>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">13. Contact Information</h2>
            <p className="text-[var(--text-primary)] mb-4">
              For questions or support:
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

