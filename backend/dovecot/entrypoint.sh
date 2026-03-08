#!/bin/bash
set -e

MYSQL_PASSWORD="${MYSQL_PASSWORD:-changeme}"
FQDN="${HOSTNAME:-mail.example.com}"

echo "Configuring Dovecot..."

# Replace password placeholder in SQL config
sed -i "s/MYSQL_PASSWORD_PLACEHOLDER/$MYSQL_PASSWORD/g" /etc/dovecot/dovecot-sql.conf

# Update TLS paths and generate fallback certs if needed
CERT_DIR="/etc/letsencrypt/live/${FQDN}"
CERT_FILE="$CERT_DIR/fullchain.pem"
KEY_FILE="$CERT_DIR/privkey.pem"
mkdir -p "$CERT_DIR"

if [ ! -f "$CERT_FILE" ] || [ ! -f "$KEY_FILE" ]; then
    echo "WARNING: No TLS certs found for ${FQDN}. Generating temporary self-signed certificate."
    openssl req -x509 -nodes -newkey rsa:2048 -days 30 \
      -subj "/CN=${FQDN}" \
      -keyout "$KEY_FILE" \
      -out "$CERT_FILE" >/dev/null 2>&1
fi

sed -i "s|ssl_cert = .*|ssl_cert = <$CERT_FILE|" /etc/dovecot/dovecot.conf
sed -i "s|ssl_key = .*|ssl_key = <$KEY_FILE|" /etc/dovecot/dovecot.conf
sed -i "s|ssl = no|ssl = required|" /etc/dovecot/dovecot.conf
echo "TLS certificates configured at $CERT_DIR."

# Create vmail user/group
addgroup -g 5000 vmail 2>/dev/null || true
adduser -D -u 5000 -G vmail -s /sbin/nologin -h /maildata vmail 2>/dev/null || true

# Ensure required folders
mkdir -p /maildata /etc/dovecot/sieve/global
chown -R dovecot:vmail /maildata /etc/dovecot/sieve
chmod -R 770 /maildata

# Compile Sieve scripts if present
if [ -f /etc/dovecot/sieve/global/spam.sieve ]; then
    sievec /etc/dovecot/sieve/global/spam.sieve
fi

# Fix permissions on SQL config (contains password)
chmod 600 /etc/dovecot/dovecot-sql.conf

echo "Starting Dovecot..."
exec dovecot -F
