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

# Update TLS paths if certs exist
if [ -f "/etc/letsencrypt/live/$FQDN/fullchain.pem" ]; then
    postconf -e "smtpd_tls_cert_file=/etc/letsencrypt/live/$FQDN/fullchain.pem"
    postconf -e "smtpd_tls_key_file=/etc/letsencrypt/live/$FQDN/privkey.pem"
    echo "TLS certificates found and configured."
else
    echo "WARNING: No TLS certificates found at /etc/letsencrypt/live/$FQDN/"
    postconf -e "smtpd_tls_security_level=none"
fi

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
