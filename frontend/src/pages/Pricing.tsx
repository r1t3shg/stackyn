import { Link, useNavigate } from 'react-router-dom';
import { useAuth } from '@/contexts/AuthContext';
import Logo from '@/components/Logo';

export default function Pricing() {
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
            <Logo height={40} />
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
                className="text-blue-600 font-medium"
              >
                Pricing
              </Link>
            </nav>
            <a
              href={user ? "https://console.staging.stackyn.com/" : "/login"}
              onClick={handleSignInClick}
              className="bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-[var(--app-bg)] font-medium py-2 px-6 rounded-lg transition-colors"
            >
              Sign in
            </a>
          </div>
        </div>
      </header>

      {/* Content */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-16">
        <div className="text-center mb-16">
          <h1 className="text-4xl font-bold text-[var(--text-primary)] mb-4">Pricing</h1>
          <p className="text-xl text-[var(--text-secondary)] max-w-2xl mx-auto">
            Choose the plan that's right for you. All plans include our core features.
          </p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-8 mb-16">
          {/* Starter Plan */}
          <div className="border border-[var(--border-subtle)] rounded-xl p-8 hover:shadow-lg transition-shadow">
            <div className="mb-6">
              <h3 className="text-2xl font-bold text-[var(--text-primary)] mb-2">Starter</h3>
              <div className="flex items-baseline">
                <span className="text-4xl font-bold text-[var(--text-primary)]">$0</span>
                <span className="text-[var(--text-secondary)] ml-2">/month</span>
              </div>
              <p className="text-[var(--text-secondary)] mt-2">Perfect for trying out Stackyn</p>
            </div>
            <ul className="space-y-3 mb-8">
              <li className="flex items-start">
                <svg className="w-5 h-5 text-[var(--success)] mr-2 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                <span className="text-[var(--text-primary)]">Up to 2 applications</span>
              </li>
              <li className="flex items-start">
                <svg className="w-5 h-5 text-[var(--success)] mr-2 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                <span className="text-[var(--text-primary)]">512 MB RAM per app</span>
              </li>
              <li className="flex items-start">
                <svg className="w-5 h-5 text-[var(--success)] mr-2 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                <span className="text-[var(--text-primary)]">Community support</span>
              </li>
              <li className="flex items-start">
                <svg className="w-5 h-5 text-[var(--success)] mr-2 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                <span className="text-[var(--text-primary)]">Basic monitoring</span>
              </li>
            </ul>
            <a
              href={user ? "https://console.staging.stackyn.com/" : "/login"}
              onClick={handleSignInClick}
              className="block w-full text-center bg-[var(--surface)] hover:bg-[var(--elevated)] text-[var(--text-primary)] font-medium py-3 px-6 rounded-lg transition-colors"
            >
              Sign in
            </a>
          </div>

          {/* Pro Plan */}
          <div className="border-2 border-[var(--primary)] rounded-xl p-8 hover:shadow-lg transition-shadow relative">
            <div className="absolute top-0 right-0 bg-[var(--primary)] text-[var(--app-bg)] px-4 py-1 rounded-bl-lg rounded-tr-xl text-sm font-medium">
              Popular
            </div>
            <div className="mb-6">
              <h3 className="text-2xl font-bold text-[var(--text-primary)] mb-2">Pro</h3>
              <div className="flex items-baseline">
                <span className="text-4xl font-bold text-[var(--text-primary)]">$29</span>
                <span className="text-[var(--text-secondary)] ml-2">/month</span>
              </div>
              <p className="text-[var(--text-secondary)] mt-2">For growing teams and projects</p>
            </div>
            <ul className="space-y-3 mb-8">
              <li className="flex items-start">
                <svg className="w-5 h-5 text-[var(--success)] mr-2 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                <span className="text-[var(--text-primary)]">Up to 10 applications</span>
              </li>
              <li className="flex items-start">
                <svg className="w-5 h-5 text-[var(--success)] mr-2 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                <span className="text-[var(--text-primary)]">2 GB RAM per app</span>
              </li>
              <li className="flex items-start">
                <svg className="w-5 h-5 text-[var(--success)] mr-2 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                <span className="text-[var(--text-primary)]">Priority support</span>
              </li>
              <li className="flex items-start">
                <svg className="w-5 h-5 text-[var(--success)] mr-2 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                <span className="text-[var(--text-primary)]">Advanced monitoring</span>
              </li>
              <li className="flex items-start">
                <svg className="w-5 h-5 text-[var(--success)] mr-2 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                <span className="text-[var(--text-primary)]">Custom domains</span>
              </li>
              <li className="flex items-start">
                <svg className="w-5 h-5 text-[var(--success)] mr-2 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                <span className="text-[var(--text-primary)]">Environment variables</span>
              </li>
            </ul>
            <a
              href={user ? "https://console.staging.stackyn.com/" : "/login"}
              onClick={handleSignInClick}
              className="block w-full text-center bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-[var(--app-bg)] font-medium py-3 px-6 rounded-lg transition-colors"
            >
              Sign in
            </a>
          </div>

          {/* Enterprise Plan */}
          <div className="border border-[var(--border-subtle)] rounded-xl p-8 hover:shadow-lg transition-shadow">
            <div className="mb-6">
              <h3 className="text-2xl font-bold text-[var(--text-primary)] mb-2">Enterprise</h3>
              <div className="flex items-baseline">
                <span className="text-4xl font-bold text-[var(--text-primary)]">Custom</span>
              </div>
              <p className="text-[var(--text-secondary)] mt-2">For large-scale deployments</p>
            </div>
            <ul className="space-y-3 mb-8">
              <li className="flex items-start">
                <svg className="w-5 h-5 text-[var(--success)] mr-2 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                <span className="text-[var(--text-primary)]">Unlimited applications</span>
              </li>
              <li className="flex items-start">
                <svg className="w-5 h-5 text-[var(--success)] mr-2 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                <span className="text-[var(--text-primary)]">Custom resource limits</span>
              </li>
              <li className="flex items-start">
                <svg className="w-5 h-5 text-[var(--success)] mr-2 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                <span className="text-[var(--text-primary)]">24/7 dedicated support</span>
              </li>
              <li className="flex items-start">
                <svg className="w-5 h-5 text-[var(--success)] mr-2 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                <span className="text-[var(--text-primary)]">SLA guarantee</span>
              </li>
              <li className="flex items-start">
                <svg className="w-5 h-5 text-[var(--success)] mr-2 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                <span className="text-[var(--text-primary)]">Advanced security features</span>
              </li>
              <li className="flex items-start">
                <svg className="w-5 h-5 text-[var(--success)] mr-2 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                <span className="text-[var(--text-primary)]">Custom integrations</span>
              </li>
            </ul>
            <a
              href={user ? "https://console.staging.stackyn.com/" : "/login"}
              onClick={handleSignInClick}
              className="block w-full text-center bg-[var(--surface)] hover:bg-[var(--elevated)] text-[var(--text-primary)] font-medium py-3 px-6 rounded-lg transition-colors"
            >
              Sign in
            </a>
          </div>
        </div>

        {/* FAQ Section */}
        <div className="mt-16">
          <h2 className="text-3xl font-bold text-[var(--text-primary)] text-center mb-12">Frequently Asked Questions</h2>
          <div className="max-w-3xl mx-auto space-y-6">
            <div className="border-b border-[var(--border-subtle)] pb-6">
              <h3 className="text-xl font-semibold text-[var(--text-primary)] mb-2">Can I change plans later?</h3>
              <p className="text-[var(--text-secondary)]">
                Yes, you can upgrade or downgrade your plan at any time. Changes will be reflected in your
                next billing cycle.
              </p>
            </div>
            <div className="border-b border-[var(--border-subtle)] pb-6">
              <h3 className="text-xl font-semibold text-[var(--text-primary)] mb-2">What payment methods do you accept?</h3>
              <p className="text-[var(--text-secondary)]">
                We accept all major credit cards and support annual billing for Enterprise plans.
              </p>
            </div>
            <div className="border-b border-[var(--border-subtle)] pb-6">
              <h3 className="text-xl font-semibold text-[var(--text-primary)] mb-2">Is there a free trial?</h3>
              <p className="text-[var(--text-secondary)]">
                Yes, our Starter plan is free forever. You can also try Pro features with a 14-day free trial.
              </p>
            </div>
            <div className="pb-6">
              <h3 className="text-xl font-semibold text-[var(--text-primary)] mb-2">What happens if I exceed my plan limits?</h3>
              <p className="text-[var(--text-secondary)]">
                We'll notify you before you reach your limits. You can upgrade your plan or we can work
                with you to find a solution that fits your needs.
              </p>
            </div>
          </div>
        </div>
      </div>

      {/* Footer */}
      <footer className="bg-[var(--sidebar)] text-[var(--text-muted)] py-12 mt-16">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center text-sm">
            <p>&copy; {new Date().getFullYear()} Stackyn. All rights reserved.</p>
          </div>
        </div>
      </footer>
    </div>
  );
}

