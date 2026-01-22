# Billing Setup Guide

## Overview

The billing system uses Lemon Squeezy for payment processing. To enable checkout functionality, you need to configure Lemon Squeezy API credentials.

## Required Environment Variables

Add these to your `.env` file:

```bash
# Lemon Squeezy API Key
LEMON_API_KEY=your_lemon_squeezy_api_key_here

# Lemon Squeezy Store ID
LEMON_STORE_ID=your_lemon_squeezy_store_id_here

# Frontend Base URL (for checkout redirects)
FRONTEND_BASE_URL=https://console.staging.stackyn.com
```

## Optional Environment Variables

```bash
# Enable test mode (uses test variant IDs)
LEMON_TEST_MODE=false

# Test variant IDs (comma-separated, format: starter:variant_id,pro:variant_id)
# Example: starter:12345,pro:67890
LEMON_TEST_VARIANT_IDS=

# Live variant IDs (comma-separated, format: starter:variant_id,pro:variant_id)
# Example: starter:12345,pro:67890
LEMON_LIVE_VARIANT_IDS=

# Webhook secret for verifying webhook signatures
LEMON_WEBHOOK_SECRET=
```

## How to Get Lemon Squeezy Credentials

### 1. Create a Lemon Squeezy Account

1. Go to https://lemonsqueezy.com
2. Sign up for an account
3. Complete the onboarding process

### 2. Get Your API Key

1. Go to https://app.lemonsqueezy.com/settings/api
2. Click "Create API Key"
3. Give it a name (e.g., "Stackyn Production")
4. Copy the API key (you'll only see it once!)
5. Add it to your `.env` file as `LEMON_API_KEY`

### 3. Get Your Store ID

1. Go to https://app.lemonsqueezy.com/stores
2. Click on your store
3. The Store ID is in the URL: `https://app.lemonsqueezy.com/stores/{STORE_ID}`
4. Or check the store settings page
5. Add it to your `.env` file as `LEMON_STORE_ID`

### 4. Create Products and Get Variant IDs

1. Go to https://app.lemonsqueezy.com/products
2. Create a product for "Starter" plan ($19/month)
3. Create a product for "Pro" plan ($49/month)
4. For each product, create a variant (monthly subscription)
5. Copy the Variant ID from the product page
6. Add to `.env`:
   - `LEMON_LIVE_VARIANT_IDS=starter:12345,pro:67890` (replace with actual IDs)

### 5. Set Up Webhooks (Optional but Recommended)

1. Go to https://app.lemonsqueezy.com/settings/webhooks
2. Click "Create Webhook"
3. Set the URL to: `https://api.staging.stackyn.com/api/billing/webhook`
4. Select events:
   - `subscription_created`
   - `subscription_updated`
   - `subscription_cancelled`
   - `invoice_paid`
   - `invoice_payment_failed`
5. Copy the webhook secret
6. Add to `.env` as `LEMON_WEBHOOK_SECRET`

## Testing

### Test Mode

1. Enable test mode in Lemon Squeezy dashboard (toggle in bottom left)
2. Set `LEMON_TEST_MODE=true` in `.env`
3. Create test products and get test variant IDs
4. Set `LEMON_TEST_VARIANT_IDS=starter:test_id,pro:test_id`

### Test Cards

Use these test card numbers in test mode:
- **Visa**: `4242 4242 4242 4242`
- Any future expiration date (e.g., 12/25)
- Any 3-digit CVC (e.g., 123)

## Verification

After setting up, restart your API container:

```bash
docker-compose restart api
```

Then test the checkout endpoint:

```bash
curl -X POST https://api.staging.stackyn.com/api/billing/checkout \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{"plan": "starter"}'
```

You should get a `checkout_url` response instead of "Billing service not configured" error.

## Troubleshooting

### Error: "Billing service not configured"

- Check that `LEMON_API_KEY` is set in `.env`
- Check that `LEMON_STORE_ID` is set in `.env`
- Check that `FRONTEND_BASE_URL` is set in `.env`
- Restart the API container after adding variables

### Error: "Invalid variant ID"

- Check that `LEMON_LIVE_VARIANT_IDS` or `LEMON_TEST_VARIANT_IDS` are set correctly
- Format should be: `starter:variant_id,pro:variant_id`
- Make sure variant IDs match your Lemon Squeezy products

### Webhooks not working

- Check that `LEMON_WEBHOOK_SECRET` is set
- Verify webhook URL is correct: `https://api.staging.stackyn.com/api/billing/webhook`
- Check API logs: `docker-compose logs api | grep webhook`

