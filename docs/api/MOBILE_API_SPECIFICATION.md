# Staff Mobile API Specification & Integration Guide

> **Base Path:** `/api/v1/mobile`  
> **Protocol:** HTTPS / JSON REST  
> **Auth Scheme:** `Authorization: Bearer <access_token>`

> **Audience:** This is the HydroGo admin/staff mobile surface. Login accepts a
> core `auth.User` (admin, editor, or viewer), not a public `members.Member`.
> A customer/member mobile application must use a separately approved,
> member-native API contract; it must not reuse this endpoint or token table.

---

## 1. Endpoints & Operations

### 1.1 Login & Device Registration
`POST /api/v1/mobile/auth/login`

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "secret_password",
  "device_id": "9B8F8B60-705E-4B0A-B39C-7F187CE3FA9E",
  "device_name": "iPhone 16 Pro",
  "platform": "iOS"
}
```

**Response (HTTP 200 OK):**
```json
{
  "access_token": "mat_3489fe0b...",
  "refresh_token": "mrt_a98f12c4...",
  "token_type": "Bearer",
  "expires_in": 900,
  "expires_at": "2026-09-19T01:15:00Z"
}
```

---

### 1.2 Token Refresh (Rotation)
`POST /api/v1/mobile/auth/refresh`

**Request Body:**
```json
{
  "refresh_token": "mrt_a98f12c4..."
}
```

**Response (HTTP 200 OK):**
```json
{
  "access_token": "mat_728b9d31...",
  "refresh_token": "mrt_91f4c78a...",
  "token_type": "Bearer",
  "expires_in": 900,
  "expires_at": "2026-09-19T01:30:00Z"
}
```

---

### 1.3 Active Device List
`GET /api/v1/mobile/auth/devices`

**Response (HTTP 200 OK):**
```json
{
  "devices": [
    {
      "device_id": "9B8F8B60-705E-4B0A-B39C-7F187CE3FA9E",
      "device_name": "iPhone 16 Pro",
      "platform": "iOS",
      "created_at": "2026-09-18T10:00:00Z",
      "last_used_at": "2026-09-19T01:00:00Z",
      "is_current": true
    }
  ]
}
```

---

### 1.4 Single Device Logout / Revocation
`POST /api/v1/mobile/auth/revoke`

**Request Body:**
```json
{
  "device_id": "9B8F8B60-705E-4B0A-B39C-7F187CE3FA9E"
}
```

---

### 1.5 Global Logout
`POST /api/v1/mobile/auth/revoke-all`

---

## 2. Standard Mobile Error Envelope

All mobile errors return a standard JSON structure with HTTP 4xx/5xx status codes:

```json
{
  "error": {
    "code": "DEVICE_REVOKED",
    "message": "The session for this device has been revoked.",
    "request_id": "req-98f1-c42e"
  }
}
```

## 3. Idempotency & Offline Sync

- Mutations (e.g., checkout, form submissions) require the `Idempotency-Key: <UUID>` header.
- Cached content payloads include `ETag` and support conditional `If-None-Match` requests to minimize mobile battery and data usage.
