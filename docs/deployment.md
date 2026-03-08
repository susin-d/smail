# Deployment Guide

## Prerequisites

- A VPS with at least 1 GB RAM (Ubuntu 22.04 or Debian 12 recommended)
- A domain with DNS access
- A Vercel account (for frontend)

## Step 1: VPS Initial Setup

```bash
# Update system
apt update && apt upgrade -y

# Install Docker
curl -fsSL https://get.docker.com | sh

# Install Docker Compose
apt install -y docker-compose-plugin

# Verify
docker --version
docker compose version
```

## Step 2: Open Required Ports

```bash
# Firewall rules (ufw)
ufw allow 22/tcp    # SSH
ufw allow 25/tcp    # SMTP
ufw allow 80/tcp    # HTTP (redirect)
ufw allow 443/tcp   # HTTPS
ufw allow 587/tcp   # SMTP Submission
ufw allow 993/tcp   # IMAPS
ufw enable
```

## Step 3: Clone and Configure

```bash
# Clone the project
cd /opt
git clone YOUR_REPO_URL smail
cd smail/backend

# Create environment file
cp .env.example .env

# Edit with your values
nano .env
```

**Critical `.env` settings:**

```env
MYSQL_ROOT_PASSWORD=your_secure_root_password
MYSQL_DATABASE=smail
MYSQL_USER=smail_user
MYSQL_PASSWORD=your_secure_db_password
JWT_SECRET=your_32_char_jwt_secret
SMAIL_DEV=0
PRIMARY_DOMAIN=yourdomain.com
HOSTNAME=mail.yourdomain.com
CORS_ORIGINS=https://your-app.vercel.app
```

Also ensure domain and cert paths are consistent in these files:

- `backend/nginx/nginx.conf`
- `backend/postfix/main.cf`
- `backend/dovecot/dovecot.conf`

## Step 4: TLS Certificates

```bash
# Install Certbot
apt install -y certbot

# Get certificates
certbot certonly --standalone -d mail.yourdomain.com

# Certificates are stored at:
# /etc/letsencrypt/live/mail.yourdomain.com/fullchain.pem
# /etc/letsencrypt/live/mail.yourdomain.com/privkey.pem
```

Mount the certs into Docker by updating docker-compose.yml to point the `certs` volume:

```yaml
volumes:
  certs:
    driver: local
    driver_opts:
      type: none
      o: bind
      device: /etc/letsencrypt
```

## Step 5: Build and Start

```bash
cd /opt/smail/backend

# Build all containers
docker compose build

# Start all services
docker compose up -d

# Check all containers are running
docker compose ps

# Show unhealthy/exited containers too
docker compose ps -a

# View logs
docker compose logs -f

# API logs
docker compose logs -f api-go
```

## Step 6: Verify Services

```bash
# Check memory usage
docker stats --no-stream

# Test API
curl http://localhost:8000/health
curl http://localhost:8000/

# Check Postfix
docker exec smail-postfix postconf myhostname

# Check Dovecot
docker exec smail-dovecot dovecot --version

# Get DKIM key
docker exec smail-opendkim cat /etc/opendkim/keys/yourdomain.com/mail.txt
```

## Step 7: Create First Admin

```bash
# Add your domain once (idempotent)
docker compose exec mariadb mariadb -u"$MYSQL_USER" -p"$MYSQL_PASSWORD" "$MYSQL_DATABASE" \
  -e "INSERT IGNORE INTO domains (domain, is_verified) VALUES ('yourdomain.com', 1);"

# Register the first account through API
curl -X POST http://localhost:8000/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@yourdomain.com","password":"yourpassword","display_name":"Admin"}'

# Promote it to admin
docker compose exec mariadb mariadb -u"$MYSQL_USER" -p"$MYSQL_PASSWORD" "$MYSQL_DATABASE" \
  -e "UPDATE users SET is_admin=1 WHERE email='admin@yourdomain.com';"
```

## Step 8: Configure DNS

Follow the [DNS Setup Guide](dns-setup.md) to configure all required DNS records.

## Step 9: Deploy Frontend

```bash
# From your local machine
cd frontend

# Install dependencies
npm install

# Set environment variable
# In Vercel dashboard or .env.local:
NEXT_PUBLIC_API_URL=https://mail.yourdomain.com/api

# Deploy to Vercel
npx vercel --prod
```

## Step 10: Set Up Auto-Renewal

```bash
# TLS certificate auto-renewal
crontab -e
# Add:
0 3 * * * certbot renew --quiet && docker compose -f /opt/smail/backend/docker-compose.yml restart nginx postfix dovecot
```

## Post-Deployment Checklist

- [ ] All containers running (`docker compose ps`)
- [ ] Total RAM < 1 GB (`docker stats --no-stream`)
- [ ] API responds (`curl https://mail.yourdomain.com/api/health`)
- [ ] DNS records configured (A, MX, SPF, DKIM, DMARC)
- [ ] DNS propagated (`dig MX yourdomain.com`)
- [ ] TLS working (`curl -I https://mail.yourdomain.com`)
- [ ] Can send test email
- [ ] Can receive test email
- [ ] Frontend deployed and accessible
- [ ] Reverse DNS (PTR) configured via VPS provider
