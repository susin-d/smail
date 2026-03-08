#!/bin/bash
set -e

MYSQL_PASSWORD="${MYSQL_PASSWORD:-changeme}"

echo "Configuring Dovecot..."

# Replace password placeholder in SQL config
sed -i "s/MYSQL_PASSWORD_PLACEHOLDER/$MYSQL_PASSWORD/g" /etc/dovecot/dovecot-sql.conf

# Update TLS paths if certs exist
CERT_DIR="/etc/letsencrypt/live/${HOSTNAME:-mail.example.com}"
if [ -f "$CERT_DIR/fullchain.pem" ]; then
    sed -i "s|ssl_cert = .*|ssl_cert = <$CERT_DIR/fullchain.pem|" /etc/dovecot/dovecot.conf
    sed -i "s|ssl_key = .*|ssl_key = <$CERT_DIR/privkey.pem|" /etc/dovecot/dovecot.conf
    echo "TLS certificates configured."
else
    echo "WARNING: No TLS certs found. Disabling SSL requirement."
    sed -i "s|ssl = required|ssl = no|" /etc/dovecot/dovecot.conf
fi

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
