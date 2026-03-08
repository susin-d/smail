#!/bin/sh
set -eu

HOST_FQDN="${HOSTNAME:-mail.susindran.in}"
CERT_DIR="/etc/letsencrypt/live/${HOST_FQDN}"
CERT_FILE="${CERT_DIR}/fullchain.pem"
KEY_FILE="${CERT_DIR}/privkey.pem"

mkdir -p "${CERT_DIR}"

if [ ! -f "${CERT_FILE}" ] || [ ! -f "${KEY_FILE}" ]; then
  echo "[nginx] TLS certs not found for ${HOST_FQDN}. Generating temporary self-signed certificate."
  openssl req -x509 -nodes -newkey rsa:2048 -days 30 \
    -subj "/CN=${HOST_FQDN}" \
    -keyout "${KEY_FILE}" \
    -out "${CERT_FILE}" >/dev/null 2>&1
fi

exec nginx -g "daemon off;"
