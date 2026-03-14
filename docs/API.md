# API Reference (v1)

Base URL: `/api/v1`

---

## Health

**GET /api/v1/health**

Response: `200 OK`
```json
{ "status": "ok" }
```

---

## Auth (no auth required)

### Register

**POST /api/v1/auth/register**

Body:
```json
{ "email": "string", "password": "string", "name": "string" }
```

Response: `201 Created`
```json
{
  "accessToken": "string",
  "refreshToken": "string",
  "expiresIn": 3600,
  "user": { "id": "uuid", "email": "string", "name": "string", "createdAt": "ISO8601", "updatedAt": "ISO8601" }
}
```

### Login

**POST /api/v1/auth/login**

Body:
```json
{ "email": "string", "password": "string" }
```

Response: `200 OK`
```json
{
  "accessToken": "string",
  "refreshToken": "string",
  "expiresIn": 3600,
  "user": { "id": "uuid", "email": "string", "name": "string", "createdAt": "ISO8601", "updatedAt": "ISO8601" }
}
```

### Refresh tokens

**POST /api/v1/auth/refresh**

Body:
```json
{ "refreshToken": "string" }
```

Response: `200 OK`
```json
{
  "accessToken": "string",
  "refreshToken": "string",
  "expiresIn": 3600,
  "user": { "id": "uuid", "email": "string", "name": "string", "createdAt": "ISO8601", "updatedAt": "ISO8601" }
}
```

---

## Users (auth required)

Headers: `Authorization: Bearer <access_token>`

### Get current user

**GET /api/v1/users/me**

Response: `200 OK`
```json
{ "id": "uuid", "email": "string", "name": "string", "createdAt": "ISO8601", "updatedAt": "ISO8601" }
```

### Update current user

**PATCH /api/v1/users/me**

Body:
```json
{ "name": "string" }
```

Response: `200 OK`
```json
{ "id": "uuid", "email": "string", "name": "string", "createdAt": "ISO8601", "updatedAt": "ISO8601" }
```

---

## Projects (auth required)

Headers: `Authorization: Bearer <access_token>`

### List projects for current user

**GET /api/v1/projects**

Query (optional): `limit` (default 20), `offset` (default 0), `orderBy` (default `asc`), `orderByField` (default `id`)

Response: `200 OK`
```json
{
  "data": [
    { "id": "uuid", "ownerId": "uuid", "name": "string", "slug": "string", "description": "string", "createdAt": "ISO8601", "updatedAt": "ISO8601" }
  ],
  "page": { "limit": 20, "offset": 0, "totalCount": 0, "hasNextPage": false, "hasPrevPage": false }
}
```

### Create project

**POST /api/v1/projects**

Body:
```json
{ "name": "string", "slug": "string", "description": "string" }
```

Response: `201 Created`
```json
{ "id": "uuid", "ownerId": "uuid", "name": "string", "slug": "string", "description": "string", "createdAt": "ISO8601", "updatedAt": "ISO8601" }
```

### Get project by ID

**GET /api/v1/projects/:id**

Response: `200 OK`
```json
{ "id": "uuid", "ownerId": "uuid", "name": "string", "slug": "string", "description": "string", "createdAt": "ISO8601", "updatedAt": "ISO8601" }
```

### Get project by slug

**GET /api/v1/projects/slug/:slug**

Response: `200 OK`
```json
{ "id": "uuid", "ownerId": "uuid", "name": "string", "slug": "string", "description": "string", "createdAt": "ISO8601", "updatedAt": "ISO8601" }
```

### Update project

**PATCH /api/v1/projects/:id**

Body (all optional):
```json
{ "name": "string", "slug": "string", "description": "string" }
```

Response: `200 OK`
```json
{ "id": "uuid", "ownerId": "uuid", "name": "string", "slug": "string", "description": "string", "createdAt": "ISO8601", "updatedAt": "ISO8601" }
```

### Delete project

**DELETE /api/v1/projects/:id**

Response: `204 No Content`

---

## Environments (auth required)

Headers: `Authorization: Bearer <access_token>`

### Create environment

**POST /api/v1/environments**

Body:
```json
{ "projectId": "uuid", "name": "string", "slug": "string", "color": "string" }
```

`color` is optional (default `#6366f1`).

Response: `201 Created`
```json
{ "id": "uuid", "projectId": "uuid", "name": "string", "slug": "string", "color": "string", "createdAt": "ISO8601", "updatedAt": "ISO8601" }
```

Conflict when slug already exists in project: `409 Conflict`.

---

### List environments by project

**GET /api/v1/environments**

Query: `projectId` (required). Optional: `limit` (default 20), `offset` (default 0), `orderBy` (default `asc`), `orderByField` (default `created_at`).

Response: `200 OK`
```json
{
  "data": [
    { "id": "uuid", "projectId": "uuid", "name": "string", "slug": "string", "color": "string", "createdAt": "ISO8601", "updatedAt": "ISO8601" }
  ],
  "page": { "limit": 20, "offset": 0, "totalCount": 0, "hasNextPage": false, "hasPrevPage": false }
}
```

---

### Get environment by ID

**GET /api/v1/environments/:id**

Response: `200 OK`
```json
{ "id": "uuid", "projectId": "uuid", "name": "string", "slug": "string", "color": "string", "createdAt": "ISO8601", "updatedAt": "ISO8601" }
```

---

### Update environment

**PATCH /api/v1/environments/:id**

Body (all optional):
```json
{ "name": "string", "slug": "string", "color": "string" }
```

Response: `200 OK`
```json
{ "id": "uuid", "projectId": "uuid", "name": "string", "slug": "string", "color": "string", "createdAt": "ISO8601", "updatedAt": "ISO8601" }
```

---

### Delete environment

**DELETE /api/v1/environments/:id**

Response: `204 No Content`

---

## Flags (auth required)

Headers: `Authorization: Bearer <access_token>`

### Create flag

**POST /api/v1/flags**

Body:
```json
{ "projectId": "uuid", "key": "string", "name": "string", "description": "string" }
```

Response: `201 Created`
```json
{ "id": "uuid", "projectId": "uuid", "key": "string", "name": "string", "description": "string", "createdAt": "ISO8601", "updatedAt": "ISO8601" }
```

### List flags by project

**GET /api/v1/flags**

Query: `projectId` (required). Optional: `limit` (default 20), `offset` (default 0), `orderBy` (default `asc`), `orderByField` (default `id`)

Response: `200 OK`
```json
{
  "data": [
    { "id": "uuid", "projectId": "uuid", "key": "string", "name": "string", "description": "string", "createdAt": "ISO8601", "updatedAt": "ISO8601" }
  ],
  "page": { "limit": 20, "offset": 0, "totalCount": 0, "hasNextPage": false, "hasPrevPage": false }
}
```

### Get flag by ID

**GET /api/v1/flags/:id**

Response: `200 OK`
```json
{ "id": "uuid", "projectId": "uuid", "key": "string", "name": "string", "description": "string", "createdAt": "ISO8601", "updatedAt": "ISO8601" }
```

### Get flag by key

**GET /api/v1/flags/key/:key**

Response: `200 OK`
```json
{ "id": "uuid", "projectId": "uuid", "key": "string", "name": "string", "description": "string", "createdAt": "ISO8601", "updatedAt": "ISO8601" }
```

### Update flag

**PATCH /api/v1/flags/:id**

Body (all optional):
```json
{ "key": "string", "name": "string", "description": "string" }
```

Response: `200 OK`
```json
{ "id": "uuid", "projectId": "uuid", "key": "string", "name": "string", "description": "string", "createdAt": "ISO8601", "updatedAt": "ISO8601" }
```

### Delete flag

**DELETE /api/v1/flags/:id**

Response: `204 No Content`

---

## Flag rules (auth required)

Headers: `Authorization: Bearer <access_token>`

### Create flag rule

**POST /api/v1/flag-rules**

Body:
```json
{
  "flagId": "uuid",
  "environmentId": "uuid",
  "enabled": true,
  "rolloutPct": 0,
  "allowList": ["string"],
  "denyList": ["string"],
  "conditions": { "operator": "AND", "conditions": [{ "attribute": "string", "operator": "eq", "value": null }] }
}
```

Response: `201 Created`
```json
{
  "id": "uuid",
  "flagId": "uuid",
  "environmentId": "uuid",
  "enabled": true,
  "rolloutPct": 0,
  "allowList": [],
  "denyList": [],
  "conditions": { "operator": "AND", "conditions": [] },
  "updatedBy": "uuid",
  "createdAt": "ISO8601",
  "updatedAt": "ISO8601"
}
```

### List flag rules by flag

**GET /api/v1/flag-rules**

Query: `flagId` (required). Optional: `limit` (default 20), `offset` (default 0), `orderBy` (default `asc`), `orderByField` (default `id`)

Response: `200 OK`
```json
{
  "data": [
    { "id": "uuid", "flagId": "uuid", "environmentId": "uuid", "enabled": true, "rolloutPct": 0, "allowList": [], "denyList": [], "conditions": {}, "updatedBy": "uuid", "createdAt": "ISO8601", "updatedAt": "ISO8601" }
  ],
  "page": { "limit": 20, "offset": 0, "totalCount": 0, "hasNextPage": false, "hasPrevPage": false }
}
```

### Get flag rule by ID

**GET /api/v1/flag-rules/:id**

Response: `200 OK`
```json
{
  "id": "uuid",
  "flagId": "uuid",
  "environmentId": "uuid",
  "enabled": true,
  "rolloutPct": 0,
  "allowList": [],
  "denyList": [],
  "conditions": {},
  "updatedBy": "uuid",
  "createdAt": "ISO8601",
  "updatedAt": "ISO8601"
}
```

### Update flag rule

**PATCH /api/v1/flag-rules/:id**

Body (all optional):
```json
{
  "enabled": true,
  "rolloutPct": 0,
  "allowList": ["string"],
  "denyList": ["string"],
  "conditions": { "operator": "AND", "conditions": [] }
}
```

Response: `200 OK` (full flag rule object)

### Delete flag rule

**DELETE /api/v1/flag-rules/:id**

Response: `204 No Content`

---

## Evaluation (auth required)

Headers: `Authorization: Bearer <access_token>`

### Evaluate single flag

**POST /api/v1/evaluation**

Body:
```json
{
  "envId": "uuid",
  "flagKey": "string",
  "userId": "string",
  "attributes": { "key": "value" }
}
```

Response: `200 OK`
```json
{ "flagKey": "string", "enabled": true, "reason": "flag_enabled" }
```

Reasons: `flag_disabled`, `deny_list`, `allow_list`, `conditions_not_met`, `rollout`, `flag_enabled`.

### Evaluate multiple flags (batch)

**POST /api/v1/evaluation/batch**

Body:
```json
{
  "envId": "uuid",
  "flagKeys": ["string"],
  "userId": "string",
  "attributes": { "key": "value" }
}
```

Response: `200 OK`
```json
[
  { "flagKey": "string", "enabled": true, "reason": "flag_enabled" },
  { "flagKey": "string", "enabled": false, "reason": "flag_disabled" }
]
```

---

## Error responses

Errors return JSON with `status` (HTTP status) and `message` (string), e.g.:

```json
{ "status": 400, "message": "email is required" }
```

Common statuses: `400` Bad Request, `401` Unauthorized, `404` Not Found, `409` Conflict.
