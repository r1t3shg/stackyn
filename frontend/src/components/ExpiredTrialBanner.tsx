import { useState } from 'react';
import type { UserProfile } from '@/lib/types';
import UpgradePlanModal from './UpgradePlanModal';

interface ExpiredTrialBannerProps {
  userProfile: UserProfile | null;
}

export default function ExpiredTrialBanner({ userProfile }: ExpiredTrialBannerProps) {
  const [showUpgradeModal, setShowUpgradeModal] = useState(false);

  if (!userProfile?.subscription) {
    return null;
  }

  const subscription = userProfile.subscription;
  
  // Only show for expired status or trial that has passed end date
  const isExpired = subscription.status === 'expired' || 
    (subscription.status === 'trial' && subscription.trial_ends_at && new Date(subscription.trial_ends_at) < new Date());

  if (!isExpired) {
    return null;
  }

  return (
    <div className="bg-gradient-to-r from-orange-50 to-red-50 border-b border-orange-200">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-3">
            <div className="flex-shrink-0">
              <svg
                className="h-5 w-5 text-orange-600"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
                />
              </svg>
            </div>
            <div className="flex-1">
              <p className="text-sm font-medium text-orange-900">
                <span className="font-semibold">Your free trial has ended.</span>
                {' '}Upgrade to continue deploying apps and access all features.
              </p>
            </div>
          </div>
          <div className="flex-shrink-0">
            <button
              onClick={() => setShowUpgradeModal(true)}
              className="text-sm font-medium text-white bg-orange-600 hover:bg-orange-700 px-4 py-2 rounded-lg transition-colors"
            >
              Upgrade Plan →
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

