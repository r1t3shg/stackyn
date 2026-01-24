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
 * Note: Expired trials should NOT show paywall - users can still access console
 */
export function shouldShowPaywall(userProfile: UserProfile | null): boolean {
  if (!userProfile) {
    return false; // Don't show paywall if no profile loaded yet
  }

  const subscription = userProfile.subscription;
  const billingStatus = userProfile.billing_status || subscription?.status;

  // Active paid subscription - no paywall
  if (billingStatus === 'active' || subscription?.status === 'active') {
    return false;
  }

  // Active trial - no paywall
  if (billingStatus === 'trial' || subscription?.status === 'trial') {
    if (subscription?.trial_ends_at) {
      const trialEndsAt = new Date(subscription.trial_ends_at);
      if (trialEndsAt >= new Date()) {
        return false; // Trial is still active
      }
    } else {
      return false; // No trial end date, assume active
    }
  }

  // Expired trial - allow console access (no paywall)
  if (billingStatus === 'expired' || subscription?.status === 'expired') {
    return false;
  }

  // No subscription at all - show paywall
  return true;
}

/**
 * Check if user's trial has expired
 */
export function isTrialExpired(userProfile: UserProfile | null): boolean {
  if (!userProfile) {
    return false; // Can't determine if expired without profile
  }

  const subscription = userProfile.subscription;
  const billingStatus = userProfile.billing_status || subscription?.status;

  // Check if status is explicitly expired
  if (billingStatus === 'expired' || subscription?.status === 'expired') {
    return true;
  }

  // Check if trial has passed its end date
  if (billingStatus === 'trial' || subscription?.status === 'trial') {
    if (subscription?.trial_ends_at) {
      const trialEndsAt = new Date(subscription.trial_ends_at);
      return trialEndsAt < new Date(); // Trial expired if end date is in the past
    }
  }

  return false;
}

