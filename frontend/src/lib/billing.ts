import type { UserProfile } from './types';

/**
 * Check if user has active billing (trial or paid subscription)
 */
export function isBillingActive(userProfile: UserProfile | null): boolean {
  if (!userProfile) {
    return false;
  }

  const subscription = userProfile.subscription;
  const billingStatus = userProfile.billing_status || subscription?.status;

  // Active paid subscription
  if (billingStatus === 'active' || subscription?.status === 'active') {
    return true;
  }

  // Check if trial is still active
  if (billingStatus === 'trial' || subscription?.status === 'trial') {
    if (subscription?.trial_ends_at) {
      const trialEndsAt = new Date(subscription.trial_ends_at);
      return trialEndsAt >= new Date(); // Trial is active if it hasn't ended yet
    }
    // If no trial_ends_at, assume trial is active (shouldn't happen, but be safe)
    return true;
  }

  // Expired or cancelled
  return false;
}

/**
 * Check if user should see the paywall
 */
export function shouldShowPaywall(userProfile: UserProfile | null): boolean {
  return !isBillingActive(userProfile);
}

