# VPS Hosting Guide (smail)

This guide shows how to host the full smail stack on a Linux VPS (Ubuntu 22.04/24.04 or Debian 12).

## 1. Requirements

- VPS: minimum 2 vCPU, 2 GB RAM, 30+ GB disk
- OS: Ubuntu 22.04/24.04 or Debian 12
- Domain:
  - root domain: `susindran.in`
  - mail host: `mail.susindran.in`
- Open inbound ports from internet: `25`, `80`, `443`, `587`, `993`
- Repository access (GitHub)

## 2. Prepare VPS

```bash
# Login as root (or sudo user)
apt update && apt upgrade -y

# Base packages
apt install -y ca-certificates curl git ufw

# Docker
curl -fsSL https://get.docker.com | sh
apt install -y docker-compose-plugin

# Verify
docker --version
docker compose version
```

## 3. Firewall

```bash
ufw allow 22/tcp
ufw allow 25/tcp
ufw allow 80/tcp
ufw allow 443/tcp
ufw allow 587/tcp
ufw allow 993/tcp
ufw --force enable
ufw status
```

## 4. Clone Project

```bash
cd /opt
sudo git clone https://github.com/susin-d/smail.git
cd smail/backend
```

## 5. Configure Environment

```bash
cp .env.example .env
nano .env
```

Set at least these values in `backend/.env`:

```env
MYSQL_ROOT_PASSWORD=<strong-root-password>
MYSQL_DATABASE=smail
MYSQL_USER=smail_user
MYSQL_PASSWORD=<strong-db-password>

JWT_SECRET=<long-random-secret>
JWT_ALGORITHM=HS256
JWT_EXPIRE_MINUTES=1440
SMAIL_DEV=0

PRIMARY_DOMAIN=susindran.in
HOSTNAME=mail.susindran.in
CORS_ORIGINS=https://mail.susindran.in
```

## 6. Build and Start Stack

```bash
cd /opt/smail/backend
docker compose up -d --build
```

Check services:

```bash
docker compose ps
docker compose ps -a
docker compose logs --tail=80
```

## 7. First Health Checks

```bash
curl http://localhost:8000/
curl http://localhost:8000/health
curl -k -I https://localhost
```

Expected:

- API returns JSON on `/` and `/health`
- HTTPS returns `200 OK`

## 8. DNS Setup

Use `docs/dns-setup.md` for complete records. Minimum required:

- `A` record: `mail.susindran.in -> <VPS_IP>`
- `MX` record: `susindran.in -> mail.susindran.in` priority `10`
- SPF TXT: `v=spf1 mx a ~all`
- DKIM TXT: `mail._domainkey.susindran.in`
- DMARC TXT: `_dmarc.susindran.in`
- PTR/reverse DNS on VPS provider: `<VPS_IP> -> mail.susindran.in`

Quick generator:

```bash
cd /opt/smail/backend
chmod +x scripts/dns-setup.sh
./scripts/dns-setup.sh --domain susindran.in --ip <VPS_IP>
```

## 9. Bootstrap Domain and Admin

Registering users requires domain row in database first.

```bash
cd /opt/smail/backend

# Insert domain once
docker compose exec mariadb mariadb -u"$MYSQL_USER" -p"$MYSQL_PASSWORD" "$MYSQL_DATABASE" \
  -e "INSERT IGNORE INTO domains (domain, is_verified) VALUES ('susindran.in', 1);"
```

Create first user:

```bash
curl -X POST http://localhost:8000/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@susindran.in","password":"StrongPass123!","display_name":"Admin"}'
```

Promote to admin:

```bash
docker compose exec mariadb mariadb -u"$MYSQL_USER" -p"$MYSQL_PASSWORD" "$MYSQL_DATABASE" \
  -e "UPDATE users SET is_admin=1 WHERE email='admin@susindran.in';"
```

## 10. TLS for Public Production

Current stack can auto-generate temporary self-signed certs if real certs are missing. This keeps services up but is not trusted by browsers/mail clients.

For real production, use valid certificates for `mail.susindran.in`.

### Option A: Use host Certbot and mount `/etc/letsencrypt`

```bash
apt install -y certbot

# Stop web stack briefly if needed for standalone challenge
cd /opt/smail/backend
docker compose stop nginx

certbot certonly --standalone -d mail.susindran.in

# Start nginx again
docker compose start nginx
```

Then restart TLS consumers:

```bash
docker compose restart nginx postfix dovecot
```

### Option B: Keep fallback only (not recommended for internet-facing production)

No action needed, but clients will show certificate warnings.

## 11. Verify Mail and API Externally

From your local machine:

```bash
curl -I https://mail.susindran.in
curl https://mail.susindran.in/api/health
```

Check SMTP/IMAP reachability:

```bash
nc -vz mail.susindran.in 25
nc -vz mail.susindran.in 587
nc -vz mail.susindran.in 993
```

## 12. Frontend Deployment

If you host frontend behind same nginx in this repo, it is already served from `https://mail.susindran.in`.

If you use separate frontend hosting (for example Vercel), set:

- `NEXT_PUBLIC_API_URL=https://mail.susindran.in/api`
- `CORS_ORIGINS` in backend `.env` to your frontend domain

## 13. Daily Operations

Useful commands:

```bash
cd /opt/smail/backend

# Logs
docker compose logs -f
docker compose logs -f api-go

# Restart key services
docker compose restart nginx postfix dovecot api-go

# Update deployment
git pull
docker compose up -d --build

# Resource usage
docker stats --no-stream
```

## 14. Troubleshooting

### `nginx` or `dovecot` keeps restarting

```bash
docker compose logs --tail=120 nginx dovecot
```

- If certificate file errors appear, issue real certs with Certbot or rely on fallback self-signed cert (temporary).

### `/auth/register` says domain not registered

Insert domain row (Step 9) and retry.

### Public domain not reachable

- Check DNS A record points to VPS IP
- Check cloud firewall/security group allows 80/443/25/587/993
- Check `ufw status`

## 15. Go-Live Checklist

- [ ] All containers `Up` in `docker compose ps`
- [ ] `http://localhost:8000/health` is healthy
- [ ] `https://mail.susindran.in` returns 200
- [ ] Valid public TLS cert installed (not self-signed fallback)
- [ ] DNS records (A/MX/SPF/DKIM/DMARC/PTR) verified
- [ ] Admin user can login and send test mail
- [ ] Test receive flow works on external mailbox
