import { Link } from 'react-router-dom';
import Logo from '@/components/Logo';

export default function Contact() {
  return (
    <div className="min-h-screen bg-[var(--app-bg)]">
      {/* Header */}
      <header className="border-b border-[var(--border-subtle)] bg-[var(--app-bg)] sticky top-0 z-50">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <Link to="/" className="flex items-center">
              <Logo height={32} showText={true} />
            </Link>
            <nav className="hidden md:flex items-center space-x-6">
              <Link to="/#pricing" className="text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors">
                Pricing
              </Link>
              <Link to="/about" className="text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors">
                About
              </Link>
              <Link to="/contact" className="text-[var(--text-primary)] font-medium">
                Contact
              </Link>
            </nav>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-16">
        <div className="text-center mb-12">
          <h1 className="text-4xl sm:text-5xl font-bold text-[var(--text-primary)] mb-4">
            Get in Touch
          </h1>
          <p className="text-lg text-[var(--text-secondary)]">
            Have a question or need support? We're here to help.
          </p>
        </div>

        <div className="bg-[var(--surface)] rounded-lg border border-[var(--border-subtle)] p-8 mb-8">
          <div className="flex items-start gap-4 mb-6">
            <div className="flex-shrink-0 w-12 h-12 bg-[var(--primary-muted)] rounded-lg flex items-center justify-center">
              <svg className="w-6 h-6 text-[var(--primary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
              </svg>
            </div>
            <div>
              <h2 className="text-xl font-semibold text-[var(--text-primary)] mb-2">Email Support</h2>
              <p className="text-[var(--text-secondary)] mb-3">
                Send us an email and we'll get back to you as soon as possible.
              </p>
              <a
                href="mailto:support@stackyn.com"
                className="text-[var(--primary)] hover:text-[var(--primary-hover)] font-medium text-lg inline-flex items-center gap-2"
              >
                support@stackyn.com
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                </svg>
              </a>
            </div>
          </div>
        </div>

        {/* Social Media Links */}
        <div className="bg-[var(--surface)] rounded-lg border border-[var(--border-subtle)] p-8">
          <h2 className="text-2xl font-semibold text-[var(--text-primary)] mb-6">Connect With Us</h2>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-6">
            {/* Discord */}
            <a
              href="https://discord.gg/stackyn"
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-4 p-4 bg-[var(--elevated)] rounded-lg border border-[var(--border-subtle)] hover:border-[var(--primary)] transition-colors group"
            >
              <div className="flex-shrink-0 w-12 h-12 bg-[#5865F2]/20 rounded-lg flex items-center justify-center group-hover:bg-[#5865F2]/30 transition-colors">
                <svg className="w-6 h-6 text-[#5865F2]" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M20.317 4.37a19.791 19.791 0 00-4.885-1.515.074.074 0 00-.079.037c-.21.375-.444.864-.608 1.25a18.27 18.27 0 00-5.487 0 12.64 12.64 0 00-.617-1.25.077.077 0 00-.079-.037A19.736 19.736 0 003.677 4.37a.07.07 0 00-.032.027C.533 9.046-.32 13.58.099 18.057a.082.082 0 00.031.057 19.9 19.9 0 005.993 3.03.078.078 0 00.084-.028c.462-.63.874-1.295 1.226-1.994a.076.076 0 00-.041-.106 13.107 13.107 0 01-1.872-.892.077.077 0 01-.008-.128 10.2 10.2 0 00.372-.292.074.074 0 01.077-.01c3.928 1.793 8.18 1.793 12.062 0a.074.074 0 01.078.01c.12.098.246.198.373.292a.077.077 0 01-.006.127 12.299 12.299 0 01-1.873.892.077.077 0 00-.041.107c.36.698.772 1.362 1.225 1.993a.076.076 0 00.084.028 19.839 19.839 0 006.002-3.03.077.077 0 00.032-.054c.5-5.177-.838-9.674-3.549-13.66a.061.061 0 00-.031-.03zM8.02 15.33c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.956-2.419 2.157-2.419 1.21 0 2.176 1.096 2.157 2.42 0 1.333-.956 2.418-2.157 2.418zm7.975 0c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.955-2.419 2.157-2.419 1.21 0 2.176 1.096 2.157 2.42 0 1.333-.946 2.418-2.157 2.418z"/>
                </svg>
              </div>
              <div>
                <h3 className="font-semibold text-[var(--text-primary)] group-hover:text-[var(--primary)] transition-colors">Discord</h3>
                <p className="text-sm text-[var(--text-muted)]">Join our community</p>
              </div>
            </a>

            {/* Twitter */}
            <a
              href="https://twitter.com/stackyn"
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-4 p-4 bg-[var(--elevated)] rounded-lg border border-[var(--border-subtle)] hover:border-[var(--primary)] transition-colors group"
            >
              <div className="flex-shrink-0 w-12 h-12 bg-[#1DA1F2]/20 rounded-lg flex items-center justify-center group-hover:bg-[#1DA1F2]/30 transition-colors">
                <svg className="w-6 h-6 text-[#1DA1F2]" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M8.29 20.251c7.547 0 11.675-6.253 11.675-11.675 0-.178 0-.355-.012-.53A8.348 8.348 0 0022 5.92a8.19 8.19 0 01-2.357.646 4.118 4.118 0 001.804-2.27 8.224 8.224 0 01-2.605.996 4.107 4.107 0 00-6.993 3.743 11.65 11.65 0 01-8.457-4.287 4.106 4.106 0 001.27 5.477A4.072 4.072 0 012.8 9.713v.052a4.105 4.105 0 003.292 4.022 4.095 4.095 0 01-1.853.07 4.108 4.108 0 003.834 2.85A8.233 8.233 0 012 18.407a11.616 11.616 0 006.29 1.84" />
                </svg>
              </div>
              <div>
                <h3 className="font-semibold text-[var(--text-primary)] group-hover:text-[var(--primary)] transition-colors">Twitter</h3>
                <p className="text-sm text-[var(--text-muted)]">Follow us for updates</p>
              </div>
            </a>

            {/* LinkedIn */}
            <a
              href="https://linkedin.com/company/stackyn"
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-4 p-4 bg-[var(--elevated)] rounded-lg border border-[var(--border-subtle)] hover:border-[var(--primary)] transition-colors group"
            >
              <div className="flex-shrink-0 w-12 h-12 bg-[#0077B5]/20 rounded-lg flex items-center justify-center group-hover:bg-[#0077B5]/30 transition-colors">
                <svg className="w-6 h-6 text-[#0077B5]" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M19 0h-14c-2.761 0-5 2.239-5 5v14c0 2.761 2.239 5 5 5h14c2.762 0 5-2.239 5-5v-14c0-2.761-2.238-5-5-5zm-11 19h-3v-11h3v11zm-1.5-12.268c-.966 0-1.75-.79-1.75-1.764s.784-1.764 1.75-1.764 1.75.79 1.75 1.764-.783 1.764-1.75 1.764zm13.5 12.268h-3v-5.604c0-3.368-4-3.113-4 0v5.604h-3v-11h3v1.765c1.396-2.586 7-2.777 7 2.476v6.759z" />
                </svg>
              </div>
              <div>
                <h3 className="font-semibold text-[var(--text-primary)] group-hover:text-[var(--primary)] transition-colors">LinkedIn</h3>
                <p className="text-sm text-[var(--text-muted)]">Connect with us</p>
              </div>
            </a>

            {/* Instagram */}
            <a
              href="https://instagram.com/stackyn"
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-4 p-4 bg-[var(--elevated)] rounded-lg border border-[var(--border-subtle)] hover:border-[var(--primary)] transition-colors group"
            >
              <div className="flex-shrink-0 w-12 h-12 bg-gradient-to-br from-[#833AB4]/20 via-[#FD1D1D]/20 to-[#FCB045]/20 rounded-lg flex items-center justify-center group-hover:from-[#833AB4]/30 group-hover:via-[#FD1D1D]/30 group-hover:to-[#FCB045]/30 transition-colors">
                <svg className="w-6 h-6 text-[#E4405F]" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M12 2.163c3.204 0 3.584.012 4.85.07 3.252.148 4.771 1.691 4.919 4.919.058 1.265.069 1.645.069 4.849 0 3.205-.012 3.584-.069 4.849-.149 3.225-1.664 4.771-4.919 4.919-1.266.058-1.644.07-4.85.07-3.204 0-3.584-.012-4.849-.07-3.26-.149-4.771-1.699-4.919-4.92-.058-1.265-.07-1.644-.07-4.849 0-3.204.013-3.583.07-4.849.149-3.227 1.664-4.771 4.919-4.919 1.266-.057 1.645-.069 4.849-.069zm0-2.163c-3.259 0-3.667.014-4.947.072-4.358.2-6.78 2.618-6.98 6.98-.059 1.281-.073 1.689-.073 4.948 0 3.259.014 3.668.072 4.948.2 4.358 2.618 6.78 6.98 6.98 1.281.058 1.689.072 4.948.072 3.259 0 3.668-.014 4.948-.072 4.354-.2 6.782-2.618 6.979-6.98.059-1.28.073-1.689.073-4.948 0-3.259-.014-3.667-.072-4.947-.196-4.354-2.617-6.78-6.979-6.98-1.281-.059-1.69-.073-4.949-.073zm0 5.838c-3.403 0-6.162 2.759-6.162 6.162s2.759 6.163 6.162 6.163 6.162-2.759 6.162-6.163c0-3.403-2.759-6.162-6.162-6.162zm0 10.162c-2.209 0-4-1.79-4-4 0-2.209 1.791-4 4-4s4 1.791 4 4c0 2.21-1.791 4-4 4zm6.406-11.845c-.796 0-1.441.645-1.441 1.44s.645 1.44 1.441 1.44c.795 0 1.439-.645 1.439-1.44s-.644-1.44-1.439-1.44z" />
                </svg>
              </div>
              <div>
                <h3 className="font-semibold text-[var(--text-primary)] group-hover:text-[var(--primary)] transition-colors">Instagram</h3>
                <p className="text-sm text-[var(--text-muted)]">Follow our journey</p>
              </div>
            </a>
          </div>
        </div>

        <div className="mt-8 text-center">
          <Link
            to="/"
            className="inline-flex items-center gap-2 text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 19l-7-7m0 0l7-7m-7 7h18" />
            </svg>
            Back to Home
          </Link>
        </div>
      </main>
    </div>
  );
}

