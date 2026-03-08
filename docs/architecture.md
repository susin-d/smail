# smail architecture

## System Overview

smail (Mail as a Service) is a lightweight, multi-user, multi-domain email platform designed to run within a 1 GB RAM budget on a single VPS.

## Architecture Diagram

```
┌──────────────────────────────────────────────────────────────────────┐
│                        CLIENT LAYER                                  │
│                                                                      │
│  ┌──────────────────────────────────────┐                           │
│  │         Next.js Frontend              │  ← Vercel (CDN)          │
│  │  Login │ Inbox │ Compose │ Settings   │                           │
│  └─────────────────┬────────────────────┘                           │
│                    │ HTTPS (REST API)                                │
└────────────────────┼─────────────────────────────────────────────────┘
                     │
┌────────────────────┼─────────────────────────────────────────────────┐
│                    ▼         VPS (1 GB RAM)                          │
│  ┌─────────────────────────┐                                        │
│  │   Nginx (30 MB)         │ ← TLS termination, rate limiting       │
│  │   :443 / :80            │                                        │
│  └────────┬────────────────┘                                        │
│           │                                                          │
│  ┌────────▼────────────────┐      ┌──────────────────┐              │
│  │  Go API Backend (150MB) │─────►│  Redis (50 MB)   │              │
│  │  :8000                  │      │  Job Queue       │              │
│  │  Auth │ Mail │ Domains  │      └──────────────────┘              │
│  └───┬────────┬────────────┘                                        │
│      │        │                                                      │
│      │   ┌────▼────────────────┐                                    │
│      │   │  MariaDB (80 MB)    │                                    │
│      │   │  Users, Domains     │                                    │
│      │   │  Mail Metadata      │                                    │
│      │   └─────────────────────┘                                    │
│      │                                                              │
│  ┌───▼──────────────────────────────────────────────┐               │
│  │              MAIL PROCESSING LAYER                │               │
│  │                                                   │               │
│  │  ┌──────────────┐  ┌─────────────┐  ┌─────────┐ │               │
│  │  │ Postfix      │  │ Dovecot     │  │OpenDKIM │ │               │
│  │  │ (100 MB)     │  │ (150 MB)    │  │ (50 MB) │ │               │
│  │  │ SMTP :25/587 │  │ IMAP :993   │  │ :8891   │ │               │
│  │  │              │──│ LMTP :24    │  │ (milter)│ │               │
│  │  │  Sends mail  │  │ Stores mail │  │  Signs  │ │               │
│  │  └──────┬───────┘  └──────┬──────┘  └─────────┘ │               │
│  │         │                 │                       │               │
│  │         └────────┬────────┘                       │               │
│  │                  ▼                                │               │
│  │    ┌──────────────────────┐                       │               │
│  │    │  /maildata (Volume)  │                       │               │
│  │    │  Maildir format      │                       │               │
│  │    │  /domain/user/       │                       │               │
│  │    └──────────────────────┘                       │               │
│  └───────────────────────────────────────────────────┘               │
│                                                                      │
│  Total Memory Budget: ~610 MB / 1024 MB                              │
└──────────────────────────────────────────────────────────────────────┘
```

## Data Flow

### Sending Email
```
User → Frontend → Go API → Redis Queue → Worker → Postfix (SMTP) → OpenDKIM (sign) → Internet
                                                         ↓
                                                   MariaDB (save metadata)
```

### Receiving Email
```
Internet → Postfix (:25) → OpenDKIM (verify) → Dovecot (LMTP :24) → Maildir storage
                                                         ↓
User → Frontend → Go API → Dovecot (IMAP :993) → Read email
                     ↓
              MariaDB (index metadata)
```

## Container Specifications

| Service    | Base Image        | RAM Limit | Ports      | Purpose                  |
|------------|-------------------|-----------|------------|--------------------------|
| nginx      | nginx:1.25-alpine | 30 MB     | 80, 443    | Reverse proxy, TLS       |
| api-go     | golang:1.24-alpine| 150 MB    | 8000       | REST API                 |
| mariadb    | mariadb:10.11     | 80 MB     | 3306       | User/domain/mail metadata|
| redis      | redis:7-alpine    | 50 MB     | 6379       | Job queue, caching       |
| postfix    | alpine:3.19       | 100 MB    | 25, 587    | SMTP (send/receive)      |
| dovecot    | alpine:3.19       | 150 MB    | 993        | IMAP, LMTP, auth         |
| opendkim   | alpine:3.19       | 50 MB     | 8891       | DKIM signing             |
| **Total**  |                   | **610 MB**|            |                          |

## Security Layers

1. **TLS** — All external connections encrypted (Let's Encrypt)
2. **JWT** — Stateless authentication with configurable expiry
3. **bcrypt** — Password hashing (12 rounds)
4. **Rate Limiting** — Ngin middleware + Nginx (edge + API-level)
5. **DKIM/SPF/DMARC** — Email authentication
6. **SASL** — SMTP authentication via Dovecot
7. **Security Headers** — X-Frame-Options, HSTS, XSS protection
