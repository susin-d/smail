#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
ENV_FILE="$BACKEND_DIR/.env"

DOMAIN=""
MAIL_HOST=""
VPS_IP=""
SELECTOR=""
POSTMASTER_EMAIL=""
TTL="3600"
FETCH_DKIM="1"

usage() {
  cat <<'EOF'
Generate DNS records for smail domains.

Usage:
  ./scripts/dns-setup.sh [options]

Options:
  --domain <domain>          Mail domain (example.com)
  --mail-host <host>         SMTP host (mail.example.com)
  --ip <ipv4>                Public VPS IPv4 for A record
  --selector <selector>      DKIM selector (default: mail)
  --postmaster <email>       DMARC aggregate report mailbox
  --ttl <seconds>            DNS TTL (default: 3600)
  --no-dkim-fetch            Do not try to read DKIM key from container
  -h, --help                 Show this help

Notes:
  - If options are omitted, values are read from backend/.env where possible.
  - DKIM TXT is fetched from container 'smail-opendkim' when available.
EOF
}

load_env_defaults() {
  if [[ -f "$ENV_FILE" ]]; then
    set -a
    # shellcheck disable=SC1090
    source "$ENV_FILE"
    set +a
  fi

  if [[ -z "$DOMAIN" && -n "${PRIMARY_DOMAIN:-}" ]]; then
    DOMAIN="$PRIMARY_DOMAIN"
  fi

  if [[ -z "$MAIL_HOST" ]]; then
    if [[ -n "${HOSTNAME:-}" ]]; then
      MAIL_HOST="$HOSTNAME"
    elif [[ -n "$DOMAIN" ]]; then
      MAIL_HOST="mail.$DOMAIN"
    fi
  fi

  if [[ -z "$SELECTOR" ]]; then
    SELECTOR="${DKIM_SELECTOR:-mail}"
  fi

  if [[ -z "$POSTMASTER_EMAIL" ]]; then
    if [[ -n "${POSTMASTER_EMAIL:-}" ]]; then
      POSTMASTER_EMAIL="$POSTMASTER_EMAIL"
    elif [[ -n "$DOMAIN" ]]; then
      POSTMASTER_EMAIL="postmaster@$DOMAIN"
    fi
  fi

  if [[ -z "$VPS_IP" && -n "${VPS_PUBLIC_IP:-}" ]]; then
    VPS_IP="$VPS_PUBLIC_IP"
  fi
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --domain)
        DOMAIN="${2:-}"
        shift 2
        ;;
      --mail-host)
        MAIL_HOST="${2:-}"
        shift 2
        ;;
      --ip)
        VPS_IP="${2:-}"
        shift 2
        ;;
      --selector)
        SELECTOR="${2:-}"
        shift 2
        ;;
      --postmaster)
        POSTMASTER_EMAIL="${2:-}"
        shift 2
        ;;
      --ttl)
        TTL="${2:-}"
        shift 2
        ;;
      --no-dkim-fetch)
        FETCH_DKIM="0"
        shift
        ;;
      -h|--help)
        usage
        exit 0
        ;;
      *)
        echo "Unknown option: $1" >&2
        usage
        exit 1
        ;;
    esac
  done
}

is_ipv4() {
  local ip="$1"
  [[ "$ip" =~ ^([0-9]{1,3}\.){3}[0-9]{1,3}$ ]] || return 1
  IFS='.' read -r a b c d <<< "$ip"
  for octet in "$a" "$b" "$c" "$d"; do
    (( octet >= 0 && octet <= 255 )) || return 1
  done
}

extract_dkim_value() {
  local raw="$1"
  # Common opendkim format: mail._domainkey IN TXT ( "v=DKIM1; ..." "..." )
  printf '%s' "$raw" | tr -d '\n' | sed -E 's/.*TXT[[:space:]]*\((.*)\).*/\1/' | tr -d '"' | tr -d '[:space:]'
}

get_dkim_txt() {
  local txt_path="/etc/opendkim/keys/$DOMAIN/$SELECTOR.txt"
  local raw=""

  if ! command -v docker >/dev/null 2>&1; then
    return 0
  fi

  if docker ps --format '{{.Names}}' | grep -qx 'smail-opendkim'; then
    if raw="$(docker exec smail-opendkim sh -lc "cat '$txt_path'" 2>/dev/null)"; then
      extract_dkim_value "$raw"
    fi
  fi
}

print_records() {
  local dkim_value="$1"

  cat <<EOF

smail DNS records for: $DOMAIN

A record:
  type: A
  name: $MAIL_HOST
  value: ${VPS_IP:-YOUR_VPS_IP}
  ttl: $TTL

MX record:
  type: MX
  name: $DOMAIN
  value: $MAIL_HOST
  priority: 10
  ttl: $TTL

SPF record:
  type: TXT
  name: $DOMAIN
  value: v=spf1 mx a ~all
  ttl: $TTL

DKIM record:
  type: TXT
  name: $SELECTOR._domainkey.$DOMAIN
  value: ${dkim_value:-v=DKIM1; k=rsa; p=YOUR_DKIM_PUBLIC_KEY}
  ttl: $TTL

DMARC record:
  type: TXT
  name: _dmarc.$DOMAIN
  value: v=DMARC1; p=quarantine; rua=mailto:$POSTMASTER_EMAIL; fo=1
  ttl: $TTL

PTR (set at VPS provider):
  ${VPS_IP:-YOUR_VPS_IP} -> $MAIL_HOST

Verification commands:
  dig A $MAIL_HOST
  dig MX $DOMAIN
  dig TXT $DOMAIN
  dig TXT $SELECTOR._domainkey.$DOMAIN
  dig TXT _dmarc.$DOMAIN
  dig -x ${VPS_IP:-YOUR_VPS_IP}
EOF
}

main() {
  parse_args "$@"
  load_env_defaults

  if [[ -z "$DOMAIN" ]]; then
    echo "Error: domain is required. Set PRIMARY_DOMAIN in backend/.env or pass --domain." >&2
    exit 1
  fi

  if [[ -z "$MAIL_HOST" ]]; then
    MAIL_HOST="mail.$DOMAIN"
  fi

  if [[ -z "$SELECTOR" ]]; then
    SELECTOR="mail"
  fi

  if [[ -z "$POSTMASTER_EMAIL" ]]; then
    POSTMASTER_EMAIL="postmaster@$DOMAIN"
  fi

  if [[ -n "$VPS_IP" ]] && ! is_ipv4 "$VPS_IP"; then
    echo "Error: invalid IPv4 address: $VPS_IP" >&2
    exit 1
  fi

  local dkim_value=""
  if [[ "$FETCH_DKIM" == "1" ]]; then
    dkim_value="$(get_dkim_txt || true)"
  fi

  print_records "$dkim_value"
}

main "$@"
