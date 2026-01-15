# Troubleshooting SSL Certificate Issues

## Error: ERR_CERT_AUTHORITY_INVALID

This means Let's Encrypt hasn't generated a certificate yet. Here's how to fix it:

## Step 1: Check Traefik Logs

```bash
docker-compose logs traefik | grep -i cert
docker-compose logs traefik | grep -i acme
docker-compose logs traefik | grep -i error
```

## Step 2: Verify DNS Configuration

Make sure your domain points to your VPS IP:

```bash
# Check DNS
dig staging.stackyn.com
dig api.staging.stackyn.com

# Should return your VPS IP address
```

If DNS isn't configured, Let's Encrypt can't verify domain ownership.

## Step 3: Verify Port 80 is Accessible

Let's Encrypt needs port 80 to verify domain ownership:

```bash
# Check if port 80 is open
sudo ufw status
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# Test from outside (use another machine or online tool)
curl -I http://staging.stackyn.com
```

## Step 4: Check Traefik Certificate Storage

```bash
# Check if acme.json exists and has correct permissions
docker-compose exec traefik ls -la /letsencrypt/
docker-compose exec traefik cat /letsencrypt/acme.json
```

## Step 5: Restart Traefik to Retry Certificate Generation

```bash
docker-compose restart traefik
docker-compose logs -f traefik
```

## Step 6: Verify Environment Variables

Make sure `.env` file has:

```env
FRONTEND_DOMAIN=staging.stackyn.com
API_DOMAIN=api.staging.stackyn.com
ACME_EMAIL=your-email@example.com
```

## Common Issues:

1. **DNS not configured**: Domain must point to VPS IP before SSL works
2. **Port 80 blocked**: Firewall must allow port 80 for Let's Encrypt verification
3. **Certificate generation in progress**: Wait 1-2 minutes after first access
4. **Rate limiting**: Let's Encrypt has rate limits (5 certs per week per domain)

## Quick Fix: Use HTTP Temporarily

If you need immediate access, you can temporarily access via HTTP:

- http://staging.stackyn.com (HTTP, no SSL)
- Wait for Let's Encrypt to generate certificate (check logs)

## Force Certificate Generation

```bash
# Restart Traefik to trigger certificate generation
docker-compose restart traefik

# Watch logs for certificate generation
docker-compose logs -f traefik
```

Look for messages like:
- "Certificate obtained"
- "Retrieving ACME certificate"
- "Unable to obtain ACME certificate" (error)

