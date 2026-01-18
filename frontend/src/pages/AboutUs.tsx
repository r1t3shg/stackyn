import { Link, useNavigate } from 'react-router-dom';
import { useAuth } from '@/contexts/AuthContext';
import Logo from '@/components/Logo';

export default function AboutUs() {
    const { user } = useAuth();
    const navigate = useNavigate();

    const handleSignInClick = (e: React.MouseEvent<HTMLAnchorElement>) => {
        e.preventDefault();
        if (user) {
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
                <h1 className="text-4xl font-bold text-[var(--text-primary)] mb-8">About Stackyn</h1>
                <div className="prose prose-lg max-w-none">
                    <p className="text-[var(--text-primary)] mb-6">
                        Stackyn is a developer-focused cloud platform that helps individuals, startups, and small teams deploy and manage applications with speed and simplicity.
                    </p>
                    <p className="text-[var(--text-primary)] mb-8">
                        Our mission is to remove infrastructure complexity so developers can focus on building products instead of managing servers.
                    </p>

                    <section className="mb-8">
                        <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">What We Do</h2>
                        <p className="text-[var(--text-primary)] mb-4">
                            Stackyn provides an easy-to-use platform for deploying backend and full-stack applications without dealing with manual server setup, complex DevOps workflows, or infrastructure headaches.
                        </p>
                        <p className="text-[var(--text-primary)] mb-4">
                            With Stackyn, users can:
                        </p>
                        <ul className="list-disc list-inside text-[var(--text-primary)] mb-4 space-y-2">
                            <li>Deploy applications in minutes</li>
                            <li>Manage deployments from a single dashboard</li>
                            <li>Scale resources based on their plan</li>
                            <li>Focus on development instead of infrastructure</li>
                        </ul>
                    </section>

                    <section className="mb-8">
                        <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">Who We Serve</h2>
                        <p className="text-[var(--text-primary)] mb-4">
                            Stackyn is built for:
                        </p>
                        <ul className="list-disc list-inside text-[var(--text-primary)] mb-4 space-y-2">
                            <li>Indie developers</li>
                            <li>Startup founders</li>
                            <li>Freelancers</li>
                            <li>Small engineering teams</li>
                        </ul>
                        <p className="text-[var(--text-primary)] mb-4">
                            We aim to provide a reliable and affordable platform tailored for early-stage products and growing businesses.
                        </p>
                    </section>

                    <section className="mb-8">
                        <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">Our Vision</h2>
                        <p className="text-[var(--text-primary)] mb-4">
                            We believe cloud infrastructure should be:
                        </p>
                        <ul className="list-disc list-inside text-[var(--text-primary)] mb-4 space-y-2">
                            <li>Simple</li>
                            <li>Transparent</li>
                            <li>Affordable</li>
                            <li>Developer-first</li>
                        </ul>
                        <p className="text-[var(--text-primary)] mb-4">
                            Stackyn is designed to grow alongside your product, from early MVPs to production workloads.
                        </p>
                    </section>

                    <section className="mb-8">
                        <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">Business Information</h2>
                        <ul className="list-disc list-inside text-[var(--text-primary)] mb-4 space-y-2">
                            <li><strong>Business Name:</strong> Stackyn</li>
                            <li><strong>Website:</strong> <a href="https://stackyn.com" className="text-[var(--primary)] hover:underline">https://stackyn.com</a></li>
                            <li><strong>Business Type:</strong> Software as a Service (SaaS)</li>
                            <li><strong>Industry:</strong> Cloud Hosting / Application Deployment</li>
                            <li><strong>Founded:</strong> 2026</li>
                            <li><strong>Founder:</strong> Ritesh Gupta</li>
                            <li><strong>Operating Model:</strong> Subscription-based</li>
                        </ul>
                    </section>

                    <section className="mb-8">
                        <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-4">Support & Contact</h2>
                        <p className="text-[var(--text-primary)] mb-4">
                            For questions, support, or business inquiries:
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
