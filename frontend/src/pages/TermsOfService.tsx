import { Link } from 'react-router-dom';

export default function TermsOfService() {
  return (
    <div className="min-h-screen bg-white">
      {/* Header */}
      <header className="border-b border-gray-200 bg-white sticky top-0 z-50">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <Link to="/" className="text-2xl font-bold text-gray-900">
              Stackyn
            </Link>
            <nav className="hidden md:flex items-center space-x-8">
              <Link
                to="/terms"
                className="text-blue-600 font-medium"
              >
                Terms of Service
              </Link>
              <Link
                to="/privacy"
                className="text-gray-600 hover:text-gray-900 transition-colors font-medium"
              >
                Privacy Policy
              </Link>
              <Link
                to="/pricing"
                className="text-gray-600 hover:text-gray-900 transition-colors font-medium"
              >
                Pricing
              </Link>
            </nav>
            <a
              href="https://console.staging.stackyn.com/"
              className="bg-blue-600 hover:bg-blue-700 text-white font-medium py-2 px-6 rounded-lg transition-colors"
            >
              Console
            </a>
          </div>
        </div>
      </header>

      {/* Content */}
      <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-16">
        <h1 className="text-4xl font-bold text-gray-900 mb-8">Terms of Service</h1>
        <div className="prose prose-lg max-w-none">
          <p className="text-gray-600 mb-6">Last updated: {new Date().toLocaleDateString()}</p>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-gray-900 mb-4">1. Acceptance of Terms</h2>
            <p className="text-gray-700 mb-4">
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
            <ul className="list-disc list-inside text-gray-700 mb-4 space-y-2">
              <li>Modify or copy the materials</li>
              <li>Use the materials for any commercial purpose or for any public display</li>
              <li>Attempt to reverse engineer any software contained in Stackyn</li>
              <li>Remove any copyright or other proprietary notations from the materials</li>
            </ul>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-gray-900 mb-4">3. Service Availability</h2>
            <p className="text-gray-700 mb-4">
              Stackyn strives to provide reliable service, but we do not guarantee uninterrupted or
              error-free service. We reserve the right to modify, suspend, or discontinue any part of the
              Service at any time with or without notice.
            </p>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-gray-900 mb-4">4. User Responsibilities</h2>
            <p className="text-gray-700 mb-4">
              You are responsible for:
            </p>
            <ul className="list-disc list-inside text-gray-700 mb-4 space-y-2">
              <li>Maintaining the confidentiality of your account credentials</li>
              <li>All activities that occur under your account</li>
              <li>Ensuring your applications comply with applicable laws and regulations</li>
              <li>Not using the Service for any illegal or unauthorized purpose</li>
            </ul>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-gray-900 mb-4">5. Prohibited Uses</h2>
            <p className="text-gray-700 mb-4">
              You may not use Stackyn:
            </p>
            <ul className="list-disc list-inside text-gray-700 mb-4 space-y-2">
              <li>In any way that violates any applicable law or regulation</li>
              <li>To transmit any malicious code, viruses, or harmful data</li>
              <li>To interfere with or disrupt the Service or servers</li>
              <li>To attempt to gain unauthorized access to any part of the Service</li>
            </ul>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-gray-900 mb-4">6. Limitation of Liability</h2>
            <p className="text-gray-700 mb-4">
              In no event shall Stackyn or its suppliers be liable for any damages (including, without
              limitation, damages for loss of data or profit, or due to business interruption) arising out
              of the use or inability to use the Service, even if Stackyn or a Stackyn authorized
              representative has been notified orally or in writing of the possibility of such damage.
            </p>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-gray-900 mb-4">7. Termination</h2>
            <p className="text-gray-700 mb-4">
              We may terminate or suspend your account and access to the Service immediately, without prior
              notice or liability, for any reason whatsoever, including without limitation if you breach
              the Terms. Upon termination, your right to use the Service will immediately cease.
            </p>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-gray-900 mb-4">8. Changes to Terms</h2>
            <p className="text-gray-700 mb-4">
              Stackyn reserves the right to revise these terms of service at any time without notice. By
              using this Service you are agreeing to be bound by the then current version of these terms
              of service.
            </p>
          </section>

          <section className="mb-8">
            <h2 className="text-2xl font-semibold text-gray-900 mb-4">9. Contact Information</h2>
            <p className="text-gray-700 mb-4">
              If you have any questions about these Terms of Service, please contact us through our
              support channels.
            </p>
          </section>
        </div>
      </div>

      {/* Footer */}
      <footer className="bg-gray-900 text-gray-400 py-12 mt-16">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center text-sm">
            <p>&copy; {new Date().getFullYear()} Stackyn. All rights reserved.</p>
          </div>
        </div>
      </footer>
    </div>
  );
}

