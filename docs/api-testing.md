# MaaS API — Complete Testing Guide

> **Base URL:** `http://localhost:8000` (direct) or `https://mail.yourdomain.com/api` (via Nginx)
>
> **Swagger UI:** `http://localhost:8000/docs`

---

## 0. Health Check

```bash
# Health check
curl http://localhost:8000/

# Expected: {"service":"MaaS API","version":"1.0.0","status":"operational"}
```

```bash
# Detailed health
curl http://localhost:8000/health

# Expected: {"status":"healthy","services":{"api":"up","database":"configured",...}}
```

---

## 1. Authentication

### POST /auth/register — Create Account

```bash
curl -X POST http://localhost:8000/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "securepassword123",
    "display_name": "Admin User"
  }'
```

**Expected 201:**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "token_type": "bearer",
  "expires_in": 86400,
  "user": {
    "id": 1,
    "email": "admin@example.com",
    "display_name": "Admin User",
    "domain_id": 1,
    "domain_name": "example.com",
    "is_admin": false,
    "is_active": true,
    "storage_quota_mb": 100,
    "storage_used_mb": 0,
    "created_at": "2026-03-07T08:00:00"
  }
}
```

**Error cases:**
```bash
# Duplicate email → 409 Conflict
# Missing domain → 400 "Domain 'unknown.com' is not registered on this platform"
# Short password → 422 Validation Error
```

---

### POST /auth/login — Sign In

```bash
curl -X POST http://localhost:8000/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "securepassword123"
  }'
```

**Expected 200:** Same structure as register response.

**Error cases:**
```bash
# Wrong password → 401 "Invalid email or password"
# Disabled account → 403 "Account is disabled"
```

> **Save the token** for all subsequent requests:
> ```bash
> TOKEN="eyJhbGciOiJIUzI1NiIs..."
> ```

---

## 2. Domains (Admin Only for create/delete)

### GET /domains — List All Domains

```bash
curl http://localhost:8000/domains \
  -H "Authorization: Bearer $TOKEN"
```

**Expected 200:**
```json
[
  {
    "id": 1,
    "domain": "example.com",
    "is_verified": false,
    "created_at": "2026-03-07T08:00:00"
  }
]
```

---

### POST /domains — Add Domain (Admin)

```bash
curl -X POST http://localhost:8000/domains \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"domain": "mydomain.com"}'
```

**Expected 201:**
```json
{
  "domain": "mydomain.com",
  "records": [
    {"record_type": "MX",  "name": "mydomain.com",                   "value": "mail.mydomain.com", "priority": 10},
    {"record_type": "A",   "name": "mail.mydomain.com",              "value": "YOUR_VPS_IP",       "priority": null},
    {"record_type": "TXT", "name": "mydomain.com",                   "value": "v=spf1 mx a ~all",  "priority": null},
    {"record_type": "TXT", "name": "mail._domainkey.mydomain.com",   "value": "v=DKIM1; k=rsa; p=YOUR_DKIM_PUBLIC_KEY", "priority": null},
    {"record_type": "TXT", "name": "_dmarc.mydomain.com",            "value": "v=DMARC1; p=quarantine; ...", "priority": null}
  ]
}
```

**Error cases:**
```bash
# Duplicate domain → 409 "Domain already registered"
# Non-admin user → 403 "Admin access required"
```

---

### GET /domains/{domain_id}/dns — Get DNS Records

```bash
curl http://localhost:8000/domains/1/dns \
  -H "Authorization: Bearer $TOKEN"
```

**Expected 200:** Same `DomainDnsResponse` structure as create.

---

### DELETE /domains/{domain_id} — Delete Domain (Admin)

```bash
curl -X DELETE http://localhost:8000/domains/1 \
  -H "Authorization: Bearer $TOKEN"
```

**Expected 200:**
```json
{"message": "Domain 'mydomain.com' deleted", "success": true}
```

---

## 3. Users

### GET /users — List All Users (Admin)

```bash
curl http://localhost:8000/users \
  -H "Authorization: Bearer $TOKEN"
```

**Expected 200:**
```json
[
  {
    "id": 1,
    "email": "admin@example.com",
    "display_name": "Admin User",
    "domain_id": 1,
    "domain_name": "example.com",
    "is_admin": true,
    "is_active": true,
    "storage_quota_mb": 100,
    "storage_used_mb": 0,
    "created_at": "2026-03-07T08:00:00"
  }
]
```

---

### POST /users — Create User (Admin)

```bash
curl -X POST http://localhost:8000/users \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "userpass1234",
    "display_name": "Regular User",
    "domain_id": 1,
    "is_admin": false
  }'
```

**Expected 201:** Returns `UserResponse`.

**Error cases:**
```bash
# Email domain mismatch → 400 "Email domain must match 'example.com'"
# Domain not found → 404 "Domain not found"
# Duplicate email → 409 "Email already registered"
```

---

### GET /users/me — Get Current Profile

```bash
curl http://localhost:8000/users/me \
  -H "Authorization: Bearer $TOKEN"
```

**Expected 200:** Returns `UserResponse` for the authenticated user.

---

### PATCH /users/me — Update Profile

```bash
# Update display name
curl -X PATCH http://localhost:8000/users/me \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"display_name": "New Name"}'

# Change password
curl -X PATCH http://localhost:8000/users/me \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"password": "newpassword123"}'
```

**Expected 200:** Updated `UserResponse`.

---

### DELETE /users/{user_id} — Delete User (Admin)

```bash
curl -X DELETE http://localhost:8000/users/2 \
  -H "Authorization: Bearer $TOKEN"
```

**Expected 200:**
```json
{"message": "User 'user@example.com' deleted", "success": true}
```

---

## 4. Mail

### GET /mail/folders — List Folders

```bash
curl http://localhost:8000/mail/folders \
  -H "Authorization: Bearer $TOKEN"
```

**Expected 200:**
```json
[
  {"name": "INBOX",  "count": 12, "unread": 3},
  {"name": "Sent",   "count": 5,  "unread": 0},
  {"name": "Drafts", "count": 0,  "unread": 0},
  {"name": "Trash",  "count": 1,  "unread": 0},
  {"name": "Spam",   "count": 2,  "unread": 2}
]
```

---

### GET /mail/inbox — Get Inbox (Paginated)

```bash
# Default inbox
curl "http://localhost:8000/mail/inbox" \
  -H "Authorization: Bearer $TOKEN"

# Specific folder + pagination
curl "http://localhost:8000/mail/inbox?folder=Sent&page=1&per_page=20" \
  -H "Authorization: Bearer $TOKEN"
```

**Query params:**

| Param    | Type | Default | Description |
|----------|------|---------|-------------|
| folder   | str  | INBOX   | Folder name |
| page     | int  | 1       | Page (≥ 1) |
| per_page | int  | 50      | Items (1–100) |

**Expected 200:**
```json
{
  "total": 12,
  "page": 1,
  "per_page": 50,
  "mails": [
    {
      "id": 1,
      "sender": "someone@gmail.com",
      "recipient": "admin@example.com",
      "subject": "Hello",
      "folder": "INBOX",
      "is_read": false,
      "is_starred": false,
      "has_attachments": false,
      "size": 1024,
      "timestamp": "2026-03-07T10:30:00"
    }
  ]
}
```

---

### GET /mail/{mail_id} — Read Email

```bash
curl http://localhost:8000/mail/1 \
  -H "Authorization: Bearer $TOKEN"
```

**Expected 200:**
```json
{
  "id": 1,
  "sender": "someone@gmail.com",
  "recipient": "admin@example.com",
  "subject": "Hello",
  "folder": "INBOX",
  "is_read": true,
  "is_starred": false,
  "has_attachments": false,
  "body": "Email body text here...",
  "html_body": null,
  "timestamp": "2026-03-07T10:30:00",
  "attachments": []
}
```

> **Note:** Reading an email automatically marks it as read.

**Error cases:**
```bash
# Not found / not owned → 404 "Email not found"
```

---

### POST /mail/send — Send Email

```bash
curl -X POST http://localhost:8000/mail/send \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "to": "recipient@gmail.com",
    "subject": "Hello from MaaS",
    "body": "This is a test email sent from MaaS.",
    "html_body": "<h1>Hello!</h1><p>This is a test.</p>"
  }'
```

**Expected 202 (Accepted):**
```json
{
  "message": "Email queued for delivery (job: mail:1709812345.123)",
  "success": true
}
```

> The email is queued in Redis and processed asynchronously by the worker.

**Fields:**

| Field     | Type   | Required | Description |
|-----------|--------|----------|-------------|
| to        | email  | ✅       | Recipient email |
| subject   | string | ✅       | Max 500 chars |
| body      | string | ✅       | Plain text body |
| html_body | string | ❌       | Optional HTML version |

---

### POST /mail/{mail_id}/action — Perform Action

```bash
# Mark as read
curl -X POST http://localhost:8000/mail/1/action \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"action": "read"}'

# Mark as unread
curl -X POST http://localhost:8000/mail/1/action \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"action": "unread"}'

# Star
curl -X POST http://localhost:8000/mail/1/action \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"action": "star"}'

# Unstar
curl -X POST http://localhost:8000/mail/1/action \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"action": "unstar"}'

# Delete (moves to Trash)
curl -X POST http://localhost:8000/mail/1/action \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"action": "delete"}'

# Move to folder
curl -X POST http://localhost:8000/mail/1/action \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"action": "move", "folder": "Archive"}'
```

**Available actions:** `read`, `unread`, `star`, `unstar`, `delete`, `move`

**Expected 200:**
```json
{"message": "Action 'star' applied to email 1", "success": true}
```

---

## 5. Error Responses

All errors follow this structure:

```json
{"detail": "Error message here"}
```

| Status | Meaning |
|--------|---------|
| 400 | Bad request / validation failure |
| 401 | Missing or invalid JWT token |
| 403 | Insufficient permissions (not admin) |
| 404 | Resource not found |
| 409 | Conflict (duplicate email/domain) |
| 422 | Request body validation error |
| 429 | Rate limited |

---

## 6. Complete Test Flow (Copy-Paste Script)

```bash
#!/bin/bash
# MaaS API Test Script
# Usage: bash test_api.sh

BASE="http://localhost:8000"

echo "=== 1. Health Check ==="
curl -s $BASE/health | python3 -m json.tool

echo -e "\n=== 2. Register ==="
REG=$(curl -s -X POST $BASE/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"testpass123","display_name":"Test User"}')
echo $REG | python3 -m json.tool
TOKEN=$(echo $REG | python3 -c "import sys,json; print(json.load(sys.stdin)['access_token'])")
echo "Token: ${TOKEN:0:20}..."

echo -e "\n=== 3. Login ==="
curl -s -X POST $BASE/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"testpass123"}' | python3 -m json.tool

echo -e "\n=== 4. Get Profile ==="
curl -s $BASE/users/me \
  -H "Authorization: Bearer $TOKEN" | python3 -m json.tool

echo -e "\n=== 5. Update Profile ==="
curl -s -X PATCH $BASE/users/me \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"display_name":"Updated Name"}' | python3 -m json.tool

echo -e "\n=== 6. List Domains ==="
curl -s $BASE/domains \
  -H "Authorization: Bearer $TOKEN" | python3 -m json.tool

echo -e "\n=== 7. Get Folders ==="
curl -s $BASE/mail/folders \
  -H "Authorization: Bearer $TOKEN" | python3 -m json.tool

echo -e "\n=== 8. Get Inbox ==="
curl -s "$BASE/mail/inbox?folder=INBOX&page=1&per_page=10" \
  -H "Authorization: Bearer $TOKEN" | python3 -m json.tool

echo -e "\n=== 9. Send Email ==="
curl -s -X POST $BASE/mail/send \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"to":"recipient@gmail.com","subject":"Test from MaaS","body":"Hello World!"}' | python3 -m json.tool

echo -e "\n=== All tests complete! ==="
```

---

## 7. Authentication Header Reference

All endpoints except `/auth/login`, `/auth/register`, `/`, and `/health` require:

```
Authorization: Bearer <jwt_token>
```

Tokens expire after **24 hours** (configurable via `JWT_EXPIRE_MINUTES`).

---

## 8. Endpoint Summary Table

| Method | Endpoint | Auth | Admin | Description |
|--------|----------|------|-------|-------------|
| GET | `/` | ❌ | ❌ | Health check |
| GET | `/health` | ❌ | ❌ | Detailed health |
| POST | `/auth/register` | ❌ | ❌ | Create account |
| POST | `/auth/login` | ❌ | ❌ | Sign in |
| GET | `/domains` | ✅ | ❌ | List domains |
| POST | `/domains` | ✅ | ✅ | Add domain |
| GET | `/domains/{id}/dns` | ✅ | ❌ | DNS records |
| DELETE | `/domains/{id}` | ✅ | ✅ | Delete domain |
| GET | `/users` | ✅ | ✅ | List users |
| POST | `/users` | ✅ | ✅ | Create user |
| GET | `/users/me` | ✅ | ❌ | Get profile |
| PATCH | `/users/me` | ✅ | ❌ | Update profile |
| DELETE | `/users/{id}` | ✅ | ✅ | Delete user |
| GET | `/mail/folders` | ✅ | ❌ | List folders |
| GET | `/mail/inbox` | ✅ | ❌ | Paginated inbox |
| GET | `/mail/{id}` | ✅ | ❌ | Read email |
| POST | `/mail/send` | ✅ | ❌ | Send email |
| POST | `/mail/{id}/action` | ✅ | ❌ | Mail action |
