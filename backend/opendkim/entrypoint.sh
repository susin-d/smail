#!/bin/bash
set -e

DOMAIN="${PRIMARY_DOMAIN:-example.com}"
SELECTOR="${DKIM_SELECTOR:-mail}"
KEY_DIR="/etc/opendkim/keys/$DOMAIN"

echo "Configuring OpenDKIM for domain: $DOMAIN (selector: $SELECTOR)"

# Generate DKIM key if not exists
if [ ! -f "$KEY_DIR/$SELECTOR.private" ]; then
    echo "Generating DKIM key pair..."
    mkdir -p "$KEY_DIR"
    opendkim-genkey -b 2048 -d "$DOMAIN" -D "$KEY_DIR" -s "$SELECTOR" -v
    chown -R opendkim:opendkim "$KEY_DIR"
    echo ""
    echo "═══════════════════════════════════════════════════"
    echo "  DKIM DNS Record for $DOMAIN"
    echo "═══════════════════════════════════════════════════"
    echo ""
    echo "Add the following TXT record to your DNS:"
    echo "Name: $SELECTOR._domainkey.$DOMAIN"
    echo ""
    cat "$KEY_DIR/$SELECTOR.txt"
    echo ""
    echo "═══════════════════════════════════════════════════"
else
    echo "DKIM key already exists for $DOMAIN"
fi

# Update domain in config
sed -i "s/^Domain.*/Domain          $DOMAIN/" /etc/opendkim/opendkim.conf
sed -i "s/^Selector.*/Selector        $SELECTOR/" /etc/opendkim/opendkim.conf

# Create key table
echo "$SELECTOR._domainkey.$DOMAIN $DOMAIN:$SELECTOR:$KEY_DIR/$SELECTOR.private" > /etc/opendkim/key.table

# Create signing table
echo "*@$DOMAIN $SELECTOR._domainkey.$DOMAIN" > /etc/opendkim/signing.table

# Create trusted hosts
cat > /etc/opendkim/trusted.hosts <<EOF
127.0.0.1
localhost
172.16.0.0/12
$DOMAIN
EOF

# Fix permissions
chown -R opendkim:opendkim /etc/opendkim
chmod 600 "$KEY_DIR/$SELECTOR.private" 2>/dev/null || true

echo "Starting OpenDKIM..."
exec opendkim -f -x /etc/opendkim/opendkim.conf
