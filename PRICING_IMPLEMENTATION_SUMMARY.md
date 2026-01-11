# Pricing Plans & Trial Implementation Summary

## Overview
This document summarizes the MVP pricing plans, 7-day free trial, resource limits, subscription enforcement, and onboarding emails implementation for Stackyn.

## Database Changes

### Migrations Created

1. **000007_add_trial_fields_to_subscriptions.up.sql**
   - Adds `trial_started_at`, `trial_ends_at` (TIMESTAMP, nullable)
   - Adds `ram_limit_mb`, `disk_limit_gb` (INTEGER, NOT NULL, defaults: 512 MB, 5 GB)
   - Renames `subscription_id` to `lemon_subscription_id` (nullable)
   - Creates unique index for one active/trial subscription per user
   - Creates index for trial lifecycle management (cron job queries)

2. **000008_add_resource_fields_to_apps.up.sql**
   - Adds `ram_mb` (INTEGER, NOT NULL, default: 256 MB)
   - Adds `disk_gb` (INTEGER, NOT NULL, default: 1 GB)
   - Creates index for resource usage queries

### Subscription Table Schema

```sql
subscriptions
- id (UUID, PRIMARY KEY)
- user_id (UUID, FOREIGN KEY)
- lemon_subscription_id (VARCHAR, nullable)
- plan (VARCHAR) -- 'starter' | 'pro'
- status (VARCHAR) -- 'trial' | 'active' | 'expired' | 'cancelled'
- trial_started_at (TIMESTAMP, nullable)
- trial_ends_at (TIMESTAMP, nullable)
- ram_limit_mb (INTEGER, NOT NULL)
- disk_limit_gb (INTEGER, NOT NULL)
- created_at (TIMESTAMP)
- updated_at (TIMESTAMP)
```

### Apps Table Schema (Updated)

```sql
apps
- id (UUID, PRIMARY KEY)
- user_id (UUID, FOREIGN KEY)
- name (VARCHAR)
- slug (VARCHAR)
- ram_mb (INTEGER, NOT NULL, default: 256)
- disk_gb (INTEGER, NOT NULL, default: 1)
- ... (existing fields)
```

## Go Structs

### Subscription Struct (`server/internal/api/repositories.go`)
```go
type Subscription struct {
    ID                 string
    UserID             string
    LemonSubscriptionID *string    // nullable
    Plan               string      // starter | pro
    Status             string      // trial | active | expired | cancelled
    TrialStartedAt     *time.Time  // nullable
    TrialEndsAt        *time.Time  // nullable
    RAMLimitMB         int
    DiskLimitGB        int
    CreatedAt          time.Time
    UpdatedAt          time.Time
}
```

## Trial Logic

### On User Signup (`server/internal/api/auth_handlers.go`)
1. User is created via `VerifyOTP` handler
2. Trial subscription is automatically created:
   - Status: `"trial"`
   - Plan: `"pro"` (trial gets Pro features)
   - `trial_started_at`: now
   - `trial_ends_at`: now + 7 days
   - `ram_limit_mb`: 2048 (2 GB)
   - `disk_limit_gb`: 20
3. Trial Started email is sent (non-blocking - failures don't block signup)

### Trial Service (`server/internal/services/subscription.go`)
- `CreateTrial()`: Creates 7-day trial with Pro limits
- `ProcessTrialLifecycle()`: Processes trial lifecycle (called by cron)
- `ExpireTrial()`: Expires trial after 7 days
- `ActivateSubscription()`: Activates subscription from trial/payment
- `CheckResourceLimits()`: Validates resource usage against limits

## Email Service (`server/internal/services/email.go`)

### Email Templates Implemented

1. **Trial Started**
   - Subject: "Your Stackyn 7-day trial has started"
   - Sent: On user signup (after trial creation)
   - Content: Welcome, trial end date, resource limits (2GB RAM / 20GB Disk), CTA

2. **Trial Ending**
   - Subject: "Your Stackyn trial ends tomorrow"
   - Sent: At day 6 (24 hours before expiration)
   - Content: Reminder, pricing options, resource comparison, upgrade link

3. **Trial Expired**
   - Subject: "Your Stackyn trial has ended"
   - Sent: When trial expires (via cron job)
   - Content: Trial ended, new deploys blocked, existing apps keep running, upgrade CTA

4. **Subscription Activated**
   - Subject: "Welcome to Stackyn 🎉"
   - Sent: When subscription is activated (via Lemon Squeezy webhook)
   - Content: Plan name, RAM & Disk limits, support contact

### Email Service Features
- Uses Resend API
- Email failures do NOT block signup or deploy (async, non-blocking)
- All emails logged
- HTML templates with proper styling

## Cron Job / Scheduler (`server/internal/tasks/trial_lifecycle.go`)

### Daily Processing
Runs daily to process trial subscriptions:

1. **Trial Expired** (now >= trial_ends_at):
   - Set status to `"expired"`
   - Send Trial Expired email (once)

2. **Trial Ending Soon** (now >= trial_ends_at - 24h):
   - Send Trial Ending email (once, at day 6)

### Implementation
- `TrialLifecycleTask`: Task for processing trial lifecycle
- `Run()`: Processes all trial subscriptions
- `StartPeriodicTask()`: Starts periodic task (runs daily at specified hour)

**Note**: For MVP, cron job can be run manually or via a scheduled task. In production, consider running as a separate worker service.

## Resource Limit Enforcement

### Subscription Status Check
Before deploy or app creation:
- Verify subscription status is `"trial"` or `"active"`
- Reject if status is `"expired"` or `"cancelled"`

### Resource Limit Checks (`server/internal/api/handlers.go`)

#### App Creation (`CreateApp`)
1. Calculate current resource usage (sum of all user's apps)
2. Check if adding new app exceeds limits:
   - Total RAM <= `ram_limit_mb`
   - Total Disk <= `disk_limit_gb`
3. Reject with clear error: "Plan limit exceeded. Upgrade to continue."

#### Default App Resources
- RAM: 256 MB per app
- Disk: 1 GB per app

**Note**: For MVP, resource allocation is fixed per app. In future, this could be configurable during app creation.

## Docker Runtime Configuration

### Memory Limits (`server/internal/services/deployment.go`)
When starting containers, memory limit is set via Docker API:
```go
Resources: container.Resources{
    Memory: opts.Limits.MemoryMB * 1024 * 1024, // Convert MB to bytes
    ...
}
```

**Example**:
- Starter plan: 512 MB → `docker run --memory=512m`
- Pro plan: 2 GB → `docker run --memory=2048m`

### Disk Limits
Disk limits are enforced at volume creation time. For MVP, disk usage is tracked in the database. Future implementations should enforce disk limits at the volume/filesystem level.

## Lemon Squeezy Webhook Handler (`server/internal/api/webhook_handlers.go`)

### Webhook Endpoint
- Route: `POST /api/webhooks/lemon-squeezy`
- Signature Verification: HMAC-SHA256 (using webhook secret)

### Events Handled

1. **subscription_created / subscription_updated**
   - Extract plan name, status, customer ID
   - Activate subscription:
     - Update status to `"active"`
     - Set plan limits:
       - Starter: 512 MB RAM, 5 GB Disk
       - Pro: 2048 MB RAM, 20 GB Disk
     - Save `lemon_subscription_id`
     - Send Subscription Activated email

2. **subscription_cancelled / subscription_expired**
   - Set status to `"cancelled"`

### Webhook Payload Structure
```go
type LemonSqueezyWebhookPayload struct {
    Meta struct {
        EventName string `json:"event_name"`
    }
    Data struct {
        Type       string
        Attributes struct {
            SubscriptionID string
            CustomerID     string
            PlanName       string
            Status         string
        }
    }
}
```

**Note**: Adjust payload structure based on actual Lemon Squeezy webhook format.

## Pricing Plans

### Starter - $19/month
- 1 app
- 1 VPS
- Max RAM: 512 MB
- Max Disk: 5 GB

### Pro - $49/month
- Up to 3 apps
- 1 VPS
- Max RAM: 2 GB (shared across apps)
- Max Disk: 20 GB

### Trial (7 days)
- Defaults to Pro limits
- 2 GB RAM, 20 GB Disk
- Up to 3 apps
- No credit card required
- Auto-expires after 7 days

## Authorization & Limit Enforcement Flow

### App Creation Flow
1. User authenticates (JWT token)
2. Check subscription status (`trial` or `active` only)
3. Check app count limit (Starter: 1, Pro/Trial: 3)
4. Check resource limits:
   - Calculate total RAM usage (current + new app)
   - Calculate total disk usage (current + new app)
   - Compare against subscription limits
5. If all checks pass, create app
6. If any check fails, return clear error message

### Deployment Flow
1. Before deployment, verify subscription status
2. Set Docker memory limit based on app's RAM allocation
3. Mount disk volume (enforce size if possible)
4. Start container with limits

## Implementation Notes

### Email Idempotency
For MVP, emails are sent on schedule without strict idempotency checks. Future improvements:
- Add `trial_ending_email_sent` flag to subscriptions table
- Add `trial_expired_email_sent` flag
- Only send emails once per event

### Resource Tracking
- Current implementation: Sum of default values per app (256 MB RAM, 1 GB Disk)
- Future: Store actual usage in apps table, update on container metrics

### Error Messages
All limit errors return clear messages:
- "Plan limit exceeded. Total RAM usage (X MB) exceeds limit (Y MB). Upgrade to continue."
- "Plan limit exceeded. Total disk usage (X GB) exceeds limit (Y GB). Upgrade to continue."
- "Subscription is not active (status: expired). Upgrade to continue."

## Files Created/Modified

### New Files
- `server/internal/db/migrations/000007_add_trial_fields_to_subscriptions.up.sql`
- `server/internal/db/migrations/000007_add_trial_fields_to_subscriptions.down.sql`
- `server/internal/db/migrations/000008_add_resource_fields_to_apps.up.sql`
- `server/internal/db/migrations/000008_add_resource_fields_to_apps.down.sql`
- `server/internal/services/subscription.go`
- `server/internal/tasks/trial_lifecycle.go`
- `server/internal/api/webhook_handlers.go`

### Modified Files
- `server/internal/api/repositories.go` (updated Subscription struct, repository methods)
- `server/internal/services/email.go` (added trial email methods)
- `server/internal/api/auth_handlers.go` (added trial creation on signup)
- `server/internal/api/handlers.go` (added resource limit enforcement)
- `server/internal/api/router.go` (added subscription service, webhook route)

## Testing Checklist

- [ ] Run migrations on test database
- [ ] Test user signup → trial creation
- [ ] Test trial email sending
- [ ] Test resource limit enforcement (create app beyond limits)
- [ ] Test subscription status check (expired trial blocks deploy)
- [ ] Test cron job (trial expiration, reminder emails)
- [ ] Test Lemon Squeezy webhook (subscription activation)
- [ ] Test Docker memory limits (verify containers have correct limits)
- [ ] Test error messages (clear and user-friendly)

## Environment Variables

Required environment variables:
- `RESEND_API_KEY`: Resend API key for emails
- `EMAIL_FROM_EMAIL`: From email address (e.g., noreply@stackyn.com)
- `LEMON_SQUEEZY_WEBHOOK_SECRET`: Webhook signing secret (optional for development)

## Next Steps (Future Enhancements)

1. **Resource Usage Tracking**: Track actual RAM/disk usage from container metrics
2. **Email Idempotency**: Add flags to prevent duplicate emails
3. **Plan Upgrades/Downgrades**: Handle plan changes mid-cycle
4. **Usage Alerts**: Warn users when approaching limits
5. **Disk Enforcement**: Enforce disk limits at volume/filesystem level
6. **Separate Cron Worker**: Run trial lifecycle task as separate service
7. **Customer ID Mapping**: Store Lemon Squeezy customer ID in users table

## Notes

- All email failures are logged but do NOT block signup or deploy
- Trial creation happens automatically on signup (no credit card required)
- Existing apps continue running after trial expires (only new deploys blocked)
- Resource limits are enforced at app creation time (preventive) and deployment time (runtime)
- Docker memory limits are enforced at container startup
- Disk limits are tracked in database (filesystem enforcement can be added later)

