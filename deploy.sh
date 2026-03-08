#!/usr/bin/env bash
# MaaS Production Deployment Script
# Optimized for a 1GB RAM Debian/Ubuntu VPS

set -e

echo "======================================================"
echo " Starting MaaS Production Deployment"
echo "======================================================"

# 1. System Requirements & Docker
if ! command -v docker &> /dev/null; then
    echo "[*] Installing Docker..."
    curl -fsSL https://get.docker.com -o get-docker.sh
    sudo sh get-docker.sh
    rm get-docker.sh
fi

if ! command -v sysctl &> /dev/null || ! sysctl vm.overcommit_memory | grep -q "1"; then
    echo "[*] Tuning kernel specifically for 1GB RAM limits (allowing slight overcommit for Redis/MySQL bursts)"
    sudo sysctl vm.overcommit_memory=1
    echo "vm.overcommit_memory=1" | sudo tee -a /etc/sysctl.conf
fi

cd backend

# 2. Environment Variables
if [ ! -f .env ]; then
    echo "[*] Creating production .env file from template..."
    cp .env.example .env
    
    # Generate secure passwords and secrets
    sed -i "s/changeme_root_password/$(openssl rand -hex 16)/" .env
    sed -i "s/changeme_db_password/$(openssl rand -hex 16)/" .env
    sed -i "s/changeme_jwt_secret_key_at_least_32_chars/$(openssl rand -hex 32)/" .env
    
    echo "⚠️  IMPORTANT: Please update PRIMARY_DOMAIN in backend/.env before accessing the site!"
fi

# 3. SSL/TLS Certificates (Let's Encrypt / Certbot)
echo "[*] Checking TLS Certificates..."
source .env
if [ ! -d "/etc/letsencrypt/live/$PRIMARY_DOMAIN" ]; then
    echo "[!] No TLS certificates found for $PRIMARY_DOMAIN."
    echo "    To automatically provision them, ensure your DNS A record points to this VPS."
    echo "    Run: sudo apt install certbot -y && sudo certbot certonly --standalone -d $PRIMARY_DOMAIN"
    echo "    Or you can place your own certs at /etc/letsencrypt/live/$PRIMARY_DOMAIN/"
    echo "    Waiting 5 seconds before continuing..."
    sleep 5
fi

# 4. Storage Directories Permissions
echo "[*] Ensuring storage directories exist and have correct permissions..."
mkdir -p maildata opendkim_keys
chmod 777 maildata opendkim_keys

# 5. Build and Run
echo "[*] Building multi-stage Docker images (This will compile the Next.js frontend to static HTML)"
echo "    This step may take a few minutes on a 1GB VPS."
docker compose build --parallel

echo "[*] Starting MaaS Platform..."
docker compose up -d

echo "======================================================"
echo " MaaS Deployment Complete! 🚀"
echo "======================================================"
echo ""
echo "System Status:"
docker compose ps
echo ""
echo "Monitor logs with: docker compose logs -f"
echo ""
echo "To create the first admin user:"
echo "1) Register a user via API (/auth/register), then promote it in DB:"
echo 'docker compose exec mariadb mariadb -u"$MYSQL_USER" -p"$MYSQL_PASSWORD" "$MYSQL_DATABASE" -e "UPDATE users SET is_admin=1 WHERE email=\"admin@yourdomain.com\";"'
