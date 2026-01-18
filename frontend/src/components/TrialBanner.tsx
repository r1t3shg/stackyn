import { useMemo, useState } from 'react';
import type { UserProfile } from '@/lib/types';
import UpgradePlanModal from './UpgradePlanModal';

interface TrialBannerProps {
  userProfile: UserProfile | null;
}

export default function TrialBanner({ userProfile }: TrialBannerProps) {
  const [showUpgradeModal, setShowUpgradeModal] = useState(false);
  
  const trialInfo = useMemo(() => {
    // Debug: Log subscription data to help diagnose
    if (process.env.NODE_ENV === 'development') {
      console.log('[TrialBanner] userProfile:', userProfile);
      console.log('[TrialBanner] subscription:', userProfile?.subscription);
    }
    
    if (!userProfile?.subscription) {
      if (process.env.NODE_ENV === 'development') {
        console.log('[TrialBanner] No subscription found in userProfile');
      }
      return null;
    }

    const subscription = userProfile.subscription;
    
    if (subscription.status !== 'trial') {
      if (process.env.NODE_ENV === 'development') {
        console.log('[TrialBanner] Subscription status is not "trial":', subscription.status);
      }
      return null;
    }

    if (!subscription.trial_started_at || !subscription.trial_ends_at) {
      if (process.env.NODE_ENV === 'development') {
        console.log('[TrialBanner] Missing trial dates:', {
          trial_started_at: subscription.trial_started_at,
          trial_ends_at: subscription.trial_ends_at,
        });
      }
      return null;
    }

    const now = new Date();
    const startedAt = new Date(subscription.trial_started_at);
    const endsAt = new Date(subscription.trial_ends_at);

    // Calculate days elapsed and days remaining
    const daysElapsed = Math.floor((now.getTime() - startedAt.getTime()) / (1000 * 60 * 60 * 24));
    const daysRemaining = Math.ceil((endsAt.getTime() - now.getTime()) / (1000 * 60 * 60 * 24));

    // If trial has expired, don't show banner
    if (daysRemaining < 0) {
      return null;
    }

    return {
      day: daysElapsed + 1, // Current day (1-indexed)
      daysRemaining: Math.max(0, daysRemaining),
      trialEndsAt: endsAt,
    };
  }, [userProfile]);

  // Always render something in development to help debug
  if (!trialInfo) {
    if (process.env.NODE_ENV === 'development' && userProfile) {
      // Show debug info in development
      return (
        <div className="bg-yellow-50 border-b border-yellow-200 text-xs p-2">
          <strong>Debug:</strong> Trial banner not showing. 
          Subscription: {userProfile.subscription ? JSON.stringify(userProfile.subscription, null, 2) : 'null'}
        </div>
      );
    }
    return null;
  }

  return (
    <div className="bg-gradient-to-r from-blue-50 to-indigo-50 border-b border-blue-200">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-3">
            <div className="flex-shrink-0">
              <svg
                className="h-5 w-5 text-blue-600"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                />
              </svg>
            </div>
            <div className="flex-1">
              <p className="text-sm font-medium text-blue-900">
                <span className="font-semibold">Free Trial Active</span>
                {' - '}
                Day {trialInfo.day} of 7
                {trialInfo.daysRemaining > 0 && (
                  <>
                    {' - '}
                    {trialInfo.daysRemaining === 1
                      ? '1 day remaining'
                      : `${trialInfo.daysRemaining} days remaining`}
                  </>
                )}
                {trialInfo.daysRemaining === 0 && ' - Last day!'}
              </p>
            </div>
          </div>
          <div className="flex-shrink-0">
            <button
              onClick={() => setShowUpgradeModal(true)}
              className="text-sm font-medium text-blue-600 hover:text-blue-800 underline"
            >
              Upgrade now →
            </button>
          </div>
        </div>
      </div>
      
      {/* Upgrade Plan Modal */}
      <UpgradePlanModal
        isOpen={showUpgradeModal}
        onClose={() => setShowUpgradeModal(false)}
      />
    </div>
  );
}

