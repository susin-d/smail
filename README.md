# Smail (A small mail server)

MaaS is a lightweight, self-hosted multi-domain mail platform designed to run on a single VPS. It includes SMTP/IMAP infrastructure, anti-spam and DKIM services, a Go API, and a Next.js frontend.

## Repository Structure

- `backend/`: Dockerized mail stack and API
- `frontend/`: Next.js web client
- `docs/`: architecture, deployment, DNS, API testing, and optimization guides
- `deploy.sh`: helper script for production-style deployment

## Stack

- API: Go + Gin (`backend/api-go`)
- Frontend: Next.js 14 (`frontend`)
- Mail services: Postfix, Dovecot, OpenDKIM, Rspamd
- Data services: MariaDB, Redis
- Reverse proxy: Nginx
- Orchestration: Docker Compose

## Quick Start (Docker)

1. Create environment file:

```bash
cd backend
cp .env.example .env
```

2. Edit `backend/.env` and set at minimum:

- `MYSQL_ROOT_PASSWORD`
- `MYSQL_PASSWORD`
- `JWT_SECRET`
- `PRIMARY_DOMAIN`
- `HOSTNAME`
- `CORS_ORIGINS`

3. Build and run:

```bash
docker compose build
docker compose up -d
```

4. Verify services:

```bash
docker compose ps
curl http://localhost:8000/
curl http://localhost:8000/health
```

## Local Frontend Development

```bash
cd frontend
npm install
npm run dev
```

Default local API URL is `http://localhost:8000` via `NEXT_PUBLIC_API_URL` in `frontend/next.config.js`.

## API Endpoints

Base URL (direct): `http://localhost:8000`

- `GET /` service info
- `GET /health` health status
- `POST /auth/register`
- `POST /auth/login`
- `GET/POST/DELETE /domains`
- `GET/POST/DELETE /users`
- `GET/POST /mail/...`

For full request/response examples, see `docs/api-testing.md`.

## Ports

- `80`, `443`: Nginx
- `25`, `587`: SMTP (Postfix)
- `993`: IMAPS (Dovecot)
- `8000`: API

## Production Notes

- Obtain TLS certificates and make them available to the `certs` volume used by Docker Compose.
- Configure required DNS records (A, MX, SPF, DKIM, DMARC, PTR).
- Keep memory usage under budget with container limits in `backend/docker-compose.yml`.

Detailed instructions:

- `docs/deployment.md`
- `docs/dns-setup.md`
- `docs/architecture.md`
- `docs/memory-optimization.md`

## Useful Commands

From `backend/`:

```bash
docker compose logs -f
docker compose logs -f api-go
docker compose restart nginx postfix dovecot
docker compose down
```

## Security Checklist

- Use strong secrets in `.env`
- Restrict `CORS_ORIGINS` in production
- Keep TLS certificates renewed
- Set correct reverse DNS (PTR)
- Ensure DKIM keys are published and valid

## License

Add your license information here.
