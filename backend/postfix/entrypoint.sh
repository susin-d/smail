#!/bin/bash
set -e

DOMAIN="${PRIMARY_DOMAIN:-example.com}"
FQDN="${HOSTNAME:-mail.$DOMAIN}"

echo "Starting Postfix for domain: $DOMAIN (hostname: $FQDN)"

# Set hostname
postconf -e "myhostname=$FQDN"
postconf -e "mydomain=$DOMAIN"
postconf -e "myorigin=\$mydomain"

# Set message size limit
postconf -e "message_size_limit=${MAX_MESSAGE_SIZE:-10485760}"

# Update TLS paths and generate fallback certs if needed
CERT_DIR="/etc/letsencrypt/live/$FQDN"
CERT_FILE="$CERT_DIR/fullchain.pem"
KEY_FILE="$CERT_DIR/privkey.pem"
mkdir -p "$CERT_DIR"

if [ ! -f "$CERT_FILE" ] || [ ! -f "$KEY_FILE" ]; then
    echo "WARNING: No TLS certificates found at $CERT_DIR. Generating temporary self-signed certificate."
    openssl req -x509 -nodes -newkey rsa:2048 -days 30 \
      -subj "/CN=$FQDN" \
      -keyout "$KEY_FILE" \
      -out "$CERT_FILE" >/dev/null 2>&1
fi

postconf -e "smtpd_tls_cert_file=$CERT_FILE"
postconf -e "smtpd_tls_key_file=$KEY_FILE"
postconf -e "smtpd_tls_security_level=may"
echo "TLS certificates configured at $CERT_DIR"

# Create virtual domain/mailbox files if they don't exist
touch /etc/postfix/virtual_domains
touch /etc/postfix/virtual_mailbox
touch /etc/postfix/virtual_alias

# Add primary domain
echo "$DOMAIN OK" > /etc/postfix/virtual_domains
postmap /etc/postfix/virtual_domains 2>/dev/null || true

# Ensure hash maps
postmap /etc/postfix/virtual_mailbox 2>/dev/null || true
postmap /etc/postfix/virtual_alias 2>/dev/null || true

# Create mail directories
mkdir -p /maildata
chown -R postfix:postfix /maildata 2>/dev/null || true

# Generate aliases
newaliases 2>/dev/null || true

echo "Postfix configuration complete. Starting..."

# Start postfix in foreground
exec postfix start-fg
