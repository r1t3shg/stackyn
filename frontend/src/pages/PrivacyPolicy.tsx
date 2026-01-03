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
        window.location.href = 'https://console.staging.stackyn.com/';
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
                    href={window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1' ? '/' : 'https://console.staging.stackyn.com/'}
                    onClick={(e) => {
                      e.preventDefault();
                      if (window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1') {
                        navigate('/');
                      } else {
                        window.location.href = 'https://console.staging.stackyn.com/';
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
          <p className="text-[var(--text-secondary)] mb-6">Last updated: {new Date().toLocaleDateString()}</p>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">1. Introduction</h2>
            <p className="text-[var(--text-primary)] mb-4">
              Stackyn ("we", "our", or "us") is committed to protecting your privacy. This Privacy Policy
              explains how we collect, use, disclose, and safeguard your information when you use our
              Platform-as-a-Service.
            </p>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">2. Information We Collect</h2>
            <p className="text-[var(--text-primary)] mb-4">We collect information that you provide directly to us:</p>
            <ul className="list-disc list-inside text-[var(--text-primary)] mb-4 space-y-2">
              <li><strong>Account Information:</strong> Email address, password, and other registration details</li>
              <li><strong>Application Data:</strong> Information about your applications, deployments, and configurations</li>
              <li><strong>Usage Data:</strong> Information about how you use the Service, including logs and metrics</li>
              <li><strong>Payment Information:</strong> Billing details processed through secure payment processors</li>
            </ul>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">3. How We Use Your Information</h2>
            <p className="text-[var(--text-primary)] mb-4">We use the information we collect to:</p>
            <ul className="list-disc list-inside text-[var(--text-primary)] mb-4 space-y-2">
              <li>Provide, maintain, and improve our Service</li>
              <li>Process transactions and send related information</li>
              <li>Send technical notices, updates, and support messages</li>
              <li>Respond to your comments, questions, and requests</li>
              <li>Monitor and analyze trends, usage, and activities</li>
              <li>Detect, prevent, and address technical issues</li>
            </ul>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">4. Information Sharing and Disclosure</h2>
            <p className="text-[var(--text-primary)] mb-4">
              We do not sell, trade, or rent your personal information to third parties. We may share your
              information only in the following circumstances:
            </p>
            <ul className="list-disc list-inside text-[var(--text-primary)] mb-4 space-y-2">
              <li><strong>Service Providers:</strong> With third-party vendors who perform services on our behalf</li>
              <li><strong>Legal Requirements:</strong> When required by law or to protect our rights</li>
              <li><strong>Business Transfers:</strong> In connection with any merger, sale, or acquisition</li>
              <li><strong>With Your Consent:</strong> When you explicitly authorize us to share information</li>
            </ul>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">5. Data Security</h2>
            <p className="text-[var(--text-primary)] mb-4">
              We implement appropriate technical and organizational security measures to protect your personal
              information. However, no method of transmission over the Internet or electronic storage is
              100% secure, and we cannot guarantee absolute security.
            </p>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">6. Data Retention</h2>
            <p className="text-[var(--text-primary)] mb-4">
              We retain your personal information for as long as necessary to provide the Service and
              fulfill the purposes outlined in this Privacy Policy, unless a longer retention period is
              required or permitted by law.
            </p>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">7. Your Rights</h2>
            <p className="text-[var(--text-primary)] mb-4">You have the right to:</p>
            <ul className="list-disc list-inside text-[var(--text-primary)] mb-4 space-y-2">
              <li>Access and receive a copy of your personal information</li>
              <li>Rectify inaccurate or incomplete information</li>
              <li>Request deletion of your personal information</li>
              <li>Object to processing of your personal information</li>
              <li>Request restriction of processing your personal information</li>
              <li>Data portability - receive your data in a structured format</li>
            </ul>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">8. Cookies and Tracking Technologies</h2>
            <p className="text-[var(--text-primary)] mb-4">
              We use cookies and similar tracking technologies to track activity on our Service and hold
              certain information. You can instruct your browser to refuse all cookies or to indicate when
              a cookie is being sent.
            </p>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">9. Third-Party Links</h2>
            <p className="text-[var(--text-primary)] mb-4">
              Our Service may contain links to third-party websites or services. We are not responsible
              for the privacy practices of these third parties. We encourage you to read their privacy
              policies.
            </p>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">10. Children's Privacy</h2>
            <p className="text-[var(--text-primary)] mb-4">
              Our Service is not intended for children under the age of 13. We do not knowingly collect
              personal information from children under 13. If you are a parent or guardian and believe
              your child has provided us with personal information, please contact us.
            </p>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">11. Changes to This Privacy Policy</h2>
            <p className="text-[var(--text-primary)] mb-4">
              We may update our Privacy Policy from time to time. We will notify you of any changes by
              posting the new Privacy Policy on this page and updating the "Last updated" date.
            </p>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">12. Contact Us</h2>
            <p className="text-[var(--text-primary)] mb-4">
              If you have any questions about this Privacy Policy, please contact us through our support
              channels.
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

