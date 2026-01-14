# Billing Test Guide

This guide explains how to test the billing system and paywall functionality.

## Overview

The billing system includes:
- **7-day free trial** that starts automatically on signup
- **Paywall** that blocks dashboard actions when billing is inactive
- **Test endpoint** to simulate different billing states
- **Lemon Squeezy test mode** for testing checkout flows

## Lemon Squeezy Test Mode

Lemon Squeezy provides a test mode that allows you to test the entire checkout flow without processing real payments.

### Enabling Test Mode

1. Go to your Lemon Squeezy dashboard
2. Toggle the "Test Mode" switch in the bottom left corner
3. Create test products (they won't automatically transfer to live mode)

### Test Card Numbers

Use these test card numbers in test mode:
- **Visa**: `4242 4242 4242 4242`
- Use any future expiration date (e.g., 12/25)
- Use any 3-digit CVC (e.g., 123)

### Test Webhooks

Webhooks are sent in test mode just like in live mode, allowing you to test:
- Subscription creation
- Payment success/failure
- Subscription cancellation

## Testing Billing States

### Using the Test Endpoint

A test endpoint is available at `POST /api/v1/test/billing` to simulate different billing states.

**⚠️ Warning**: This endpoint is disabled in production (`ENV=production`). It updates the actual database.

#### Example: Set billing to expired

```bash
curl -X POST http://localhost:8080/api/v1/test/billing \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "billing_status": "expired",
    "plan": "starter"
  }'
```

#### Example: Set billing to active

```bash
curl -X POST http://localhost:8080/api/v1/test/billing \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "billing_status": "active",
    "plan": "pro"
  }'
```

#### Example: Set trial with custom end date

```bash
curl -X POST http://localhost:8080/api/v1/test/billing \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "billing_status": "trial",
    "plan": "free_trial",
    "trial_ends_at": "2024-01-01T00:00:00Z"
  }'
```

### Using the Billing Test Panel (Frontend)

A floating test panel is available in development mode on the Home page:

1. Log in to the dashboard
2. Look for the yellow "🧪 Billing Test Panel" in the bottom-right corner
3. Select a billing status (trial, active, expired)
4. Select a plan (free_trial, starter, pro)
5. Click "Update Billing State"
6. The paywall will appear/disappear based on the billing state

**Note**: The test panel only appears when `NODE_ENV !== 'production'`.

## Testing the Paywall

### When the Paywall Appears

The paywall is shown when:
- `billing_status === 'expired'` OR
- `billing_status === 'trial'` AND `trial_ends_at < now()`

### Testing Scenarios

1. **Expired Trial**
   - Set `billing_status: "expired"`
   - Paywall should appear
   - All app actions should be blocked (API returns 402)

2. **Active Trial**
   - Set `billing_status: "trial"` with `trial_ends_at` in the future
   - Paywall should NOT appear
   - App actions should work normally

3. **Active Subscription**
   - Set `billing_status: "active"`
   - Paywall should NOT appear
   - App actions should work normally

## API Enforcement

The billing middleware (`BillingMiddleware`) enforces active billing on these endpoints:
- `POST /api/v1/apps` - Create app
- `POST /api/v1/apps/{id}/redeploy` - Redeploy app
- `GET /api/v1/apps/{id}/logs/*` - View logs
- `GET /api/v1/apps/{id}` - Get app details
- All other app management endpoints

When billing is inactive, these endpoints return:
```json
{
  "error": "Billing is inactive. Please upgrade your plan to continue.",
  "status": 402
}
```

## Background Worker

The billing worker runs every 30 minutes to:
1. Find users with `billing_status = 'trial'` AND `trial_ends_at < NOW()`
2. Stop all their apps
3. Mark apps as `disabled`
4. Set `billing_status = 'expired'`
5. Send trial ended email

To test this manually, you can:
1. Set a user's `trial_ends_at` to a past date
2. Set `billing_status = 'trial'`
3. Wait for the worker to run (or trigger it manually)

## Email Testing

Email templates are sent for:
- Trial started (on signup)
- Trial ended (when trial expires)
- Subscription activated (on webhook)
- Payment failed (on webhook)
- Subscription expired (on webhook)

Emails are sent via Resend API. Check your Resend dashboard for sent emails.

## Checklist for Testing

- [ ] Sign up new user → Trial starts automatically
- [ ] Trial started email is sent
- [ ] Set billing to expired → Paywall appears
- [ ] Try to create app with expired billing → API returns 402
- [ ] Set billing to active → Paywall disappears
- [ ] Create app with active billing → Works normally
- [ ] Test Lemon Squeezy checkout in test mode
- [ ] Verify webhook updates billing status
- [ ] Test background worker expiration logic

## Production Notes

- Test endpoint is automatically disabled when `ENV=production`
- Billing test panel is hidden in production builds
- Always test in a staging environment before production

