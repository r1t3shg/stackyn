import { Link, useNavigate } from 'react-router-dom';
import { useAuth } from '@/contexts/AuthContext';
import Logo from '@/components/Logo';

export default function RefundPolicy() {
    const { user } = useAuth();
    const navigate = useNavigate();

    const handleSignInClick = (e: React.MouseEvent<HTMLAnchorElement>) => {
        e.preventDefault();
        if (user) {
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

                            {/* Desktop Navigation Links */}
                            <nav className="hidden md:flex items-center space-x-6">
                                <a href="/#why-stackyn" className="text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors font-medium">Why Stackyn?</a>
                                <a href="/#features" className="text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors font-medium">Features</a>
                                <a href="/#pricing" className="text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors font-medium">Pricing</a>
                            </nav>
                        </div>

                        {/* Desktop CTA */}
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
                            <button className="text-[var(--text-secondary)] hover:text-[var(--text-primary)] p-2">
                                <svg className="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M4 6h16M4 12h16M4 18h16" />
                                </svg>
                            </button>
                        </div>
                    </div>
                </div>
            </header>

            {/* Content */}
            <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-16">
                <h1 className="text-4xl font-bold text-[var(--text-primary)] mb-8">Refund Policy</h1>
                <div className="prose prose-lg max-w-none">
                    <p className="text-[var(--text-secondary)] mb-6">Effective Date: January 2026</p>

                    <p className="text-[var(--text-primary)] mb-4">
                        Thank you for using Stackyn ("Service"). This Refund Policy explains how refunds are handled for subscriptions and payments made through our platform.
                    </p>

                    <section className="mb-8">
                        <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">1. Subscription Plans</h2>
                        <p className="text-[var(--text-primary)] mb-4">
                            Stackyn operates on a subscription-based model. By purchasing a subscription, you agree to the pricing, billing cycle, and this Refund Policy.
                        </p>
                    </section>

                    <section className="mb-8">
                        <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">2. Free Trial</h2>
                        <ul className="list-disc list-inside text-[var(--text-primary)] mb-4 space-y-2">
                            <li>Stackyn may offer a free trial on selected plans.</li>
                            <li>No payment is required during the trial period.</li>
                            <li>Once the trial ends, your subscription will begin automatically unless canceled.</li>
                        </ul>
                    </section>

                    <section className="mb-8">
                        <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">3. Refund Eligibility</h2>
                        <p className="text-[var(--text-primary)] mb-4">
                            Refunds are not guaranteed and are handled on a case-by-case basis.
                            You may be eligible for a refund if:
                        </p>
                        <ul className="list-disc list-inside text-[var(--text-primary)] mb-4 space-y-2">
                            <li>You were charged incorrectly due to a billing error</li>
                            <li>The Service was unavailable for an extended period due to a verified platform issue</li>
                        </ul>
                    </section>

                    <section className="mb-8">
                        <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">4. Non-Refundable Cases</h2>
                        <p className="text-[var(--text-primary)] mb-4">
                            We do not provide refunds for:
                        </p>
                        <ul className="list-disc list-inside text-[var(--text-primary)] mb-4 space-y-2">
                            <li>Partial usage of a billing period</li>
                            <li>Unused time after cancellation</li>
                            <li>Failure to cancel before the renewal date</li>
                            <li>Change of mind after payment</li>
                            <li>Violation of our Terms of Service</li>
                        </ul>
                    </section>

                    <section className="mb-8">
                        <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">5. Cancellation</h2>
                        <ul className="list-disc list-inside text-[var(--text-primary)] mb-4 space-y-2">
                            <li>You may cancel your subscription at any time from your account dashboard.</li>
                            <li>Cancellation stops future billing but does not trigger an automatic refund.</li>
                            <li>Your subscription remains active until the end of the billing period.</li>
                        </ul>
                    </section>

                    <section className="mb-8">
                        <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">6. Payment Processor</h2>
                        <ul className="list-disc list-inside text-[var(--text-primary)] mb-4 space-y-2">
                            <li>All payments and refunds are processed by Lemon Squeezy.</li>
                            <li>Stackyn does not store payment information.</li>
                        </ul>
                    </section>

                    <section className="mb-8">
                        <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">7. Requesting a Refund</h2>
                        <p className="text-[var(--text-primary)] mb-4">
                            To request a refund, contact us within 7 days of payment:
                        </p>
                        <p className="text-[var(--text-primary)] mb-4">
                            <strong><a href="mailto:support@stackyn.com" className="text-[var(--primary)] hover:underline">support@stackyn.com</a></strong>
                        </p>
                        <p className="text-[var(--text-primary)] mb-4">
                            Please include:
                        </p>
                        <ul className="list-disc list-inside text-[var(--text-primary)] mb-4 space-y-2">
                            <li>Account email</li>
                            <li>Invoice or order ID</li>
                            <li>Reason for the refund request</li>
                        </ul>
                    </section>

                    <section className="mb-8">
                        <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">8. Policy Updates</h2>
                        <ul className="list-disc list-inside text-[var(--text-primary)] mb-4 space-y-2">
                            <li>We reserve the right to update this Refund Policy at any time.</li>
                            <li>Changes will be effective immediately upon posting.</li>
                        </ul>
                    </section>

                    <section className="mb-8">
                        <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">9. Contact</h2>
                        <p className="text-[var(--text-primary)] mb-4">
                            For refund-related questions:
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
                            <Link to="/about" className="hover:text-[var(--text-primary)] transition-colors">About Us</Link>
                            <Link to="/terms" className="hover:text-[var(--text-primary)] transition-colors">Terms of Service</Link>
                            <Link to="/privacy" className="hover:text-[var(--text-primary)] transition-colors">Privacy Policy</Link>
                            <Link to="/refund" className="hover:text-[var(--text-primary)] transition-colors">Refund Policy</Link>
                        </div>
                    </div>
                </div>
            </footer>
        </div>
    );
}
