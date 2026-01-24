import { Link } from 'react-router-dom';
import { useState } from 'react';
import type { UserProfile } from '@/lib/types';
import { billingApi } from '@/lib/api';

interface PaywallProps {
  userProfile: UserProfile | null;
  message?: string;
}

export default function Paywall({ userProfile, message }: PaywallProps) {
  const [loadingPlan, setLoadingPlan] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Handle plan selection and start checkout
  const handlePlanSelect = async (plan: 'starter' | 'pro') => {
    try {
      setLoadingPlan(plan);
      setError(null);
      const response = await billingApi.createCheckout(plan);
      // Redirect to Lemon Squeezy checkout
      window.location.href = response.checkout_url;
    } catch (err) {
      console.error('Error creating checkout session:', err);
      setError(err instanceof Error ? err.message : 'Failed to start checkout. Please try again.');
      setLoadingPlan(null);
    }
  };
  // Determine the paywall message based on billing status
  const getPaywallMessage = () => {
    if (message) return message;

    if (!userProfile) {
      return "Unable to verify billing status. Please contact support.";
    }

    const subscription = userProfile.subscription;
    const billingStatus = userProfile.billing_status || subscription?.status;

    if (billingStatus === 'expired' || subscription?.status === 'expired') {
      return "Your free trial has ended. Upgrade to continue deploying apps.";
    }

    if (billingStatus === 'cancelled' || subscription?.status === 'cancelled') {
      return "Your subscription has been cancelled. Resubscribe to continue.";
    }

    // Check if trial has expired
    if (subscription?.status === 'trial' && subscription.trial_ends_at) {
      const trialEndsAt = new Date(subscription.trial_ends_at);
      if (trialEndsAt < new Date()) {
        return "Your free trial has ended. Upgrade to continue deploying apps.";
      }
    }

    return "Billing is inactive. Upgrade to continue.";
  };

  return (
    <div className="fixed inset-0 bg-[var(--app-bg)] z-50 flex items-center justify-center p-4">
      <div className="max-w-2xl w-full bg-[var(--elevated)] border border-[var(--border-subtle)] rounded-lg p-8 shadow-xl">
        <div className="text-center mb-8">
          <div className="w-16 h-16 bg-[var(--primary-muted)] rounded-full flex items-center justify-center mx-auto mb-4">
            <svg className="w-8 h-8 text-[var(--primary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
            </svg>
          </div>
          <h2 className="text-3xl font-bold text-[var(--text-primary)] mb-4">
            Upgrade Required
          </h2>
          <p className="text-lg text-[var(--text-secondary)] mb-6">
            {getPaywallMessage()}
          </p>
        </div>

        <div className="bg-[var(--surface)] rounded-lg p-6 mb-6">
          <h3 className="text-lg font-semibold text-[var(--text-primary)] mb-4">
            Choose Your Plan
          </h3>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {/* Starter Plan */}
            <div className="border border-[var(--border-subtle)] rounded-lg p-4 hover:border-[var(--primary)] transition-colors">
              <div className="flex items-center gap-2 mb-2">
                <span className="text-xl">🟢</span>
                <h4 className="font-semibold text-[var(--text-primary)]">Starter</h4>
              </div>
              <div className="text-2xl font-bold text-[var(--text-primary)] mb-1">
                $19 <span className="text-sm font-normal text-[var(--text-muted)]">/ month</span>
              </div>
              <ul className="text-sm text-[var(--text-secondary)] space-y-1 mb-4">
                <li>• 1 app</li>
                <li>• 512 MB RAM</li>
                <li>• 5 GB Disk</li>
              </ul>
              <button
                onClick={() => handlePlanSelect('starter')}
                disabled={loadingPlan !== null}
                className="block w-full text-center bg-[var(--primary)] hover:bg-[var(--primary-hover)] disabled:opacity-50 disabled:cursor-not-allowed text-[var(--app-bg)] font-semibold py-2 px-4 rounded-lg transition-colors text-sm"
              >
                {loadingPlan === 'starter' ? 'Loading...' : 'Upgrade to Starter'}
              </button>
            </div>

            {/* Pro Plan */}
            <div className="border-2 border-[var(--primary)] rounded-lg p-4 relative">
              <div className="absolute top-0 right-0 bg-[var(--primary)] text-[var(--app-bg)] px-2 py-1 rounded-bl-lg rounded-tr-lg text-xs font-semibold">
                Popular
              </div>
              <div className="flex items-center gap-2 mb-2">
                <span className="text-xl">🔵</span>
                <h4 className="font-semibold text-[var(--text-primary)]">Pro</h4>
              </div>
              <div className="text-2xl font-bold text-[var(--text-primary)] mb-1">
                $49 <span className="text-sm font-normal text-[var(--text-muted)]">/ month</span>
              </div>
              <ul className="text-sm text-[var(--text-secondary)] space-y-1 mb-4">
                <li>• 3 apps</li>
                <li>• 2 GB RAM</li>
                <li>• 20 GB Disk</li>
              </ul>
              <button
                onClick={() => handlePlanSelect('pro')}
                disabled={loadingPlan !== null}
                className="block w-full text-center bg-[var(--primary)] hover:bg-[var(--primary-hover)] disabled:opacity-50 disabled:cursor-not-allowed text-[var(--app-bg)] font-semibold py-2 px-4 rounded-lg transition-colors text-sm"
              >
                {loadingPlan === 'pro' ? 'Loading...' : 'Upgrade to Pro'}
              </button>
            </div>
          </div>
        </div>

        {error && (
          <div className="mb-4 p-3 bg-red-100 border border-red-400 text-red-700 rounded-lg text-sm">
            {error}
          </div>
        )}
        <div className="text-center">
          <p className="text-sm text-[var(--text-muted)] mb-4">
            💡 <strong>Testing?</strong> Use Lemon Squeezy test mode with card <code className="bg-[var(--surface)] px-2 py-1 rounded">4242 4242 4242 4242</code>
          </p>
          <Link
            to="/pricing"
            className="text-[var(--primary)] hover:text-[var(--primary-hover)] font-medium"
          >
            View full pricing details →
          </Link>
        </div>
      </div>
    </div>
  );
}

