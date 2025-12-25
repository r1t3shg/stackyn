import { Link } from 'react-router-dom';

export default function Pricing() {
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
                className="text-gray-600 hover:text-gray-900 transition-colors font-medium"
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
                className="text-blue-600 font-medium"
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
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-16">
        <div className="text-center mb-16">
          <h1 className="text-4xl font-bold text-gray-900 mb-4">Pricing</h1>
          <p className="text-xl text-gray-600 max-w-2xl mx-auto">
            Choose the plan that's right for you. All plans include our core features.
          </p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-8 mb-16">
          {/* Starter Plan */}
          <div className="border border-gray-200 rounded-xl p-8 hover:shadow-lg transition-shadow">
            <div className="mb-6">
              <h3 className="text-2xl font-bold text-gray-900 mb-2">Starter</h3>
              <div className="flex items-baseline">
                <span className="text-4xl font-bold text-gray-900">$0</span>
                <span className="text-gray-600 ml-2">/month</span>
              </div>
              <p className="text-gray-600 mt-2">Perfect for trying out Stackyn</p>
            </div>
            <ul className="space-y-3 mb-8">
              <li className="flex items-start">
                <svg className="w-5 h-5 text-green-500 mr-2 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                <span className="text-gray-700">Up to 2 applications</span>
              </li>
              <li className="flex items-start">
                <svg className="w-5 h-5 text-green-500 mr-2 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                <span className="text-gray-700">512 MB RAM per app</span>
              </li>
              <li className="flex items-start">
                <svg className="w-5 h-5 text-green-500 mr-2 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                <span className="text-gray-700">Community support</span>
              </li>
              <li className="flex items-start">
                <svg className="w-5 h-5 text-green-500 mr-2 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                <span className="text-gray-700">Basic monitoring</span>
              </li>
            </ul>
            <a
              href="https://console.staging.stackyn.com/"
              className="block w-full text-center bg-gray-100 hover:bg-gray-200 text-gray-900 font-medium py-3 px-6 rounded-lg transition-colors"
            >
              Get Started
            </a>
          </div>

          {/* Pro Plan */}
          <div className="border-2 border-blue-600 rounded-xl p-8 hover:shadow-lg transition-shadow relative">
            <div className="absolute top-0 right-0 bg-blue-600 text-white px-4 py-1 rounded-bl-lg rounded-tr-xl text-sm font-medium">
              Popular
            </div>
            <div className="mb-6">
              <h3 className="text-2xl font-bold text-gray-900 mb-2">Pro</h3>
              <div className="flex items-baseline">
                <span className="text-4xl font-bold text-gray-900">$29</span>
                <span className="text-gray-600 ml-2">/month</span>
              </div>
              <p className="text-gray-600 mt-2">For growing teams and projects</p>
            </div>
            <ul className="space-y-3 mb-8">
              <li className="flex items-start">
                <svg className="w-5 h-5 text-green-500 mr-2 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                <span className="text-gray-700">Up to 10 applications</span>
              </li>
              <li className="flex items-start">
                <svg className="w-5 h-5 text-green-500 mr-2 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                <span className="text-gray-700">2 GB RAM per app</span>
              </li>
              <li className="flex items-start">
                <svg className="w-5 h-5 text-green-500 mr-2 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                <span className="text-gray-700">Priority support</span>
              </li>
              <li className="flex items-start">
                <svg className="w-5 h-5 text-green-500 mr-2 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                <span className="text-gray-700">Advanced monitoring</span>
              </li>
              <li className="flex items-start">
                <svg className="w-5 h-5 text-green-500 mr-2 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                <span className="text-gray-700">Custom domains</span>
              </li>
              <li className="flex items-start">
                <svg className="w-5 h-5 text-green-500 mr-2 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                <span className="text-gray-700">Environment variables</span>
              </li>
            </ul>
            <a
              href="https://console.staging.stackyn.com/"
              className="block w-full text-center bg-blue-600 hover:bg-blue-700 text-white font-medium py-3 px-6 rounded-lg transition-colors"
            >
              Get Started
            </a>
          </div>

          {/* Enterprise Plan */}
          <div className="border border-gray-200 rounded-xl p-8 hover:shadow-lg transition-shadow">
            <div className="mb-6">
              <h3 className="text-2xl font-bold text-gray-900 mb-2">Enterprise</h3>
              <div className="flex items-baseline">
                <span className="text-4xl font-bold text-gray-900">Custom</span>
              </div>
              <p className="text-gray-600 mt-2">For large-scale deployments</p>
            </div>
            <ul className="space-y-3 mb-8">
              <li className="flex items-start">
                <svg className="w-5 h-5 text-green-500 mr-2 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                <span className="text-gray-700">Unlimited applications</span>
              </li>
              <li className="flex items-start">
                <svg className="w-5 h-5 text-green-500 mr-2 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                <span className="text-gray-700">Custom resource limits</span>
              </li>
              <li className="flex items-start">
                <svg className="w-5 h-5 text-green-500 mr-2 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                <span className="text-gray-700">24/7 dedicated support</span>
              </li>
              <li className="flex items-start">
                <svg className="w-5 h-5 text-green-500 mr-2 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                <span className="text-gray-700">SLA guarantee</span>
              </li>
              <li className="flex items-start">
                <svg className="w-5 h-5 text-green-500 mr-2 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                <span className="text-gray-700">Advanced security features</span>
              </li>
              <li className="flex items-start">
                <svg className="w-5 h-5 text-green-500 mr-2 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                <span className="text-gray-700">Custom integrations</span>
              </li>
            </ul>
            <a
              href="https://console.staging.stackyn.com/"
              className="block w-full text-center bg-gray-100 hover:bg-gray-200 text-gray-900 font-medium py-3 px-6 rounded-lg transition-colors"
            >
              Contact Sales
            </a>
          </div>
        </div>

        {/* FAQ Section */}
        <div className="mt-16">
          <h2 className="text-3xl font-bold text-gray-900 text-center mb-12">Frequently Asked Questions</h2>
          <div className="max-w-3xl mx-auto space-y-6">
            <div className="border-b border-gray-200 pb-6">
              <h3 className="text-xl font-semibold text-gray-900 mb-2">Can I change plans later?</h3>
              <p className="text-gray-600">
                Yes, you can upgrade or downgrade your plan at any time. Changes will be reflected in your
                next billing cycle.
              </p>
            </div>
            <div className="border-b border-gray-200 pb-6">
              <h3 className="text-xl font-semibold text-gray-900 mb-2">What payment methods do you accept?</h3>
              <p className="text-gray-600">
                We accept all major credit cards and support annual billing for Enterprise plans.
              </p>
            </div>
            <div className="border-b border-gray-200 pb-6">
              <h3 className="text-xl font-semibold text-gray-900 mb-2">Is there a free trial?</h3>
              <p className="text-gray-600">
                Yes, our Starter plan is free forever. You can also try Pro features with a 14-day free trial.
              </p>
            </div>
            <div className="pb-6">
              <h3 className="text-xl font-semibold text-gray-900 mb-2">What happens if I exceed my plan limits?</h3>
              <p className="text-gray-600">
                We'll notify you before you reach your limits. You can upgrade your plan or we can work
                with you to find a solution that fits your needs.
              </p>
            </div>
          </div>
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

