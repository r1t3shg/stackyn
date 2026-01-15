# Billing & Trial Implementation Summary

## Overview
Implemented 7-day free trial, subscription enforcement, app stopping, and email notifications for Stackyn.

## Database Changes

### Migration 000009: Add Billing Fields to Users Table
- Added `billing_status` (trial | active | expired)
- Added `plan` (free_trial | starter | pro)
- Added `trial_started_at` TIMESTAMP
- Added `trial_ends_at` TIMESTAMP
- Added `subscription_id` VARCHAR(255)
- Created indexes for fast billing checks

### Migration 000010: Add Disabled Status to Apps
- Documented `disabled` status for apps (billing expired)

## Core Implementation

### 1. User Signup → Trial Start ✅
- **Location**: `server/internal/api/auth_handlers.go`
- When user signs up via `VerifyOTP`:
  - Creates 7-day free trial subscription
  - Sets `billing_status = 'trial'`, `plan = 'free_trial'`
  - Sets `trial_started_at` and `trial_ends_at` (7 days from now)
  - Sends trial started email (non-blocking)
  - Syncs billing fields to users table

### 2. Trial Enforcement ✅
- **Location**: `server/internal/workers/billing_worker.go`
- Background worker runs every 30 minutes
- Queries users table: `WHERE billing_status = 'trial' AND trial_ends_at < NOW()`
- For each expired user:
  - Stops all Docker containers
  - Marks apps as `disabled` in database
  - Sets `billing_status = 'expired'`
  - Sends trial ended email

### 3. Subscription Activation ✅
- **Location**: `server/internal/api/webhook_handlers.go`
- Handles Lemon Squeezy webhooks:
  - `subscription_created`
  - `subscription_updated`
  - `invoice_paid`
- When subscription activated:
  - Sets `plan` (starter or pro)
  - Sets `billing_status = 'active'`
  - Re-enables dashboard + deployments
  - Sends subscription activated email

### 4. Subscription Expiry / Failure ✅
- **Location**: `server/internal/api/webhook_handlers.go`
- Handles webhooks:
  - `invoice_failed` → Calls `ExpireSubscription`
  - `subscription_cancelled` → Calls `CancelSubscription`
- When expired:
  - Marks `billing_status = 'expired'`
  - Stops all apps
  - Disables dashboard
  - Sends payment failed / subscription expired email

### 5. API Enforcement ✅
- **Location**: `server/internal/api/middleware.go`
- `RequireActiveBilling(user)` function:
  - Returns nil if `billing_status == 'active'`
  - Returns nil if `billing_status == 'trial'` AND `trial_ends_at > NOW()`
  - Returns error otherwise
- `BillingMiddleware` applies guard to protected endpoints
- Applied to:
  - `POST /api/v1/apps` (Create app)
  - `POST /api/v1/apps/{id}/redeploy` (Redeploy)
  - `GET /api/v1/apps/{id}/logs/*` (View logs)
  - All app modification endpoints

### 6. Background Worker ✅
- **Location**: `server/internal/workers/billing_worker.go`
- Runs every 30 minutes
- Queries users table directly (not subscriptions table)
- Processes expired trials
- Idempotent (safe to run multiple times)

### 7. App Stop Logic ✅
- **Location**: `server/internal/api/app_stopper.go`
- `StopAllAppsForUser(userID)`:
  - Gets all apps for user
  - Marks each app as `disabled` in database
  - Stops Docker containers via deployment service
  - Idempotent (safe to run multiple times)

### 8. Email Sending ✅
- **Location**: `server/internal/services/email.go`
- Uses Resend API for transactional emails
- Templates implemented:
  - ✅ Trial started
  - ✅ Trial ended
  - ✅ Subscription activated
  - ✅ Payment failed
  - ✅ Subscription expired
- All emails sent asynchronously (non-blocking)

### 9. Lemon Squeezy Webhook Handler ✅
- **Location**: `server/internal/api/webhook_handlers.go`
- Handles events:
  - `subscription_created` → Activate subscription
  - `subscription_updated` → Activate subscription
  - `invoice_paid` → Activate subscription
  - `invoice_failed` → Expire subscription
  - `subscription_cancelled` → Cancel subscription
- Verifies webhook signature (HMAC-SHA256)
- Maps webhook → user via customer_id (email for MVP)

### 10. Dashboard Behavior ✅
- Billing middleware blocks API requests when billing inactive
- Returns HTTP 402 (Payment Required) with error message
- Frontend should show paywall when receiving 402 response

## Files Modified/Created

### New Files
- `server/internal/db/migrations/000009_add_billing_fields_to_users.up.sql`
- `server/internal/db/migrations/000009_add_billing_fields_to_users.down.sql`
- `server/internal/db/migrations/000010_add_disabled_status_to_apps.up.sql`
- `server/internal/db/migrations/000010_add_disabled_status_to_apps.down.sql`
- `server/internal/workers/billing_worker.go`
- `BILLING_IMPLEMENTATION.md`

### Modified Files
- `server/internal/api/auth_handlers.go` - Added trial creation on signup
- `server/internal/api/repositories.go` - Added billing fields to User struct and queries
- `server/internal/api/middleware.go` - Added RequireActiveBilling and BillingMiddleware
- `server/internal/api/router.go` - Applied billing middleware to app endpoints, started billing worker
- `server/internal/api/webhook_handlers.go` - Added invoice_paid and invoice_failed handling
- `server/internal/api/app_stopper.go` - Added app disabling logic
- `server/internal/api/subscription_repo_adapter.go` - Added UpdateUserBilling implementation
- `server/internal/services/subscription.go` - Added ExpireSubscription, billing field syncing
- `server/internal/services/email.go` - Added payment failed and subscription expired emails

## Testing Checklist

- [ ] User signup creates trial automatically
- [ ] Trial started email is sent
- [ ] Apps can be deployed during trial
- [ ] Trial expires after 7 days
- [ ] Apps are stopped when trial expires
- [ ] Apps are marked as disabled
- [ ] Trial expired email is sent
- [ ] Dashboard is blocked after trial expires
- [ ] Subscription activation via webhook works
- [ ] Subscription activated email is sent
- [ ] Payment failed stops apps
- [ ] Payment failed email is sent
- [ ] Billing middleware blocks deployments when inactive
- [ ] Billing worker runs every 30 minutes

## Environment Variables Required

- `RESEND_API_KEY` - Resend API key for emails
- `EMAIL_FROM_EMAIL` - From email address
- `LEMON_SQUEEZY_WEBHOOK_SECRET` - Webhook signing secret (optional for dev)

## Notes

- Email failures do NOT block signup or subscription operations
- App stopping failures do NOT block trial expiration
- Billing fields in users table are synced from subscriptions table (subscriptions is source of truth)
- All operations are idempotent (safe to retry)
- No polling of Lemon Squeezy - relies fully on webhooks

