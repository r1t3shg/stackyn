# VPS .env File Template

Copy this to your VPS at `/opt/stackyn/.env` and replace placeholder values with actual generated secrets.

## Quick Setup Commands

On your VPS, run:

```bash
cd /opt/stackyn

# Generate passwords and secrets
POSTGRES_PASS=$(openssl rand -base64 32)
REDIS_PASS=$(openssl rand -base64 32)
JWT_SECRET=$(openssl rand -base64 32)

# Create .env file
cat > .env << EOF
# Database
POSTGRES_PASSWORD=$POSTGRES_PASS

# Redis
REDIS_PASSWORD=$REDIS_PASS

# JWT Secret
JWT_SECRET=$JWT_SECRET

# Frontend Configuration
FRONTEND_API_URL=https://api.staging.stackyn.com
FRONTEND_DOMAIN=staging.stackyn.com
CONSOLE_DOMAIN=console.staging.stackyn.com

# API Domain
API_DOMAIN=api.staging.stackyn.com

# Base Domain for User Apps (REQUIRED)
APP_BASE_DOMAIN=staging.stackyn.com

# Let's Encrypt Email
ACME_EMAIL=your-email@example.com

# Email Configuration
RESEND_API_KEY=re_6iU1KmCf_3p6MzQRbsDyerP736x1WWExj
EMAIL_FROM_EMAIL=noreply@stackyn.com
EOF

# Secure the file
chmod 600 .env

echo "✅ .env file created with generated secrets"
echo "⚠️  Don't forget to set ACME_EMAIL to your actual email!"
```

## Manual Setup

If you prefer to edit manually:

1. Copy `env.example` to `.env`:
   ```bash
   cp env.example .env
   nano .env
   ```

2. Generate and set these values:
   - `POSTGRES_PASSWORD`: `openssl rand -base64 32`
   - `REDIS_PASSWORD`: `openssl rand -base64 32`
   - `JWT_SECRET`: `openssl rand -base64 32`

3. Update `ACME_EMAIL` to your actual email address

4. Verify `APP_BASE_DOMAIN` is set to `staging.stackyn.com`

## Required Variables Checklist

- [ ] `POSTGRES_PASSWORD` - Generated strong password
- [ ] `REDIS_PASSWORD` - Generated strong password  
- [ ] `JWT_SECRET` - Generated strong secret
- [ ] `API_DOMAIN` - Set to `api.staging.stackyn.com`
- [ ] `APP_BASE_DOMAIN` - Set to `staging.stackyn.com` (REQUIRED)
- [ ] `ACME_EMAIL` - Your email for SSL certificates
- [ ] `FRONTEND_API_URL` - Set to `https://api.staging.stackyn.com`
- [ ] `FRONTEND_DOMAIN` - Set to `staging.stackyn.com`

## Security Notes

- 🔒 Never commit `.env` file to git
- 🔒 Use `chmod 600 .env` to restrict file permissions
- 🔒 Generate unique passwords/secrets for production
- 🔒 Keep backups of your `.env` file in a secure location

