# DevMatch API Reference

> For frontend developers / Claude: use this document to understand the full API contract before building any UI.

## Base

- Base URL: `http://localhost:8080`
- All API routes are prefixed with `/api/v1`
- Content-Type: `application/json`
- CORS: allows `http://localhost:5173` with credentials

---

## Authentication

Access is controlled via **JWT Bearer tokens**.

- Store `access_token` and `refresh_token` in memory (or `httpOnly` cookie if possible).
- Send with every protected request: `Authorization: Bearer <access_token>`
- Access token TTL: **15 minutes** — refresh proactively or on `401`.
- Refresh token TTL: **30 days**.
- On `401`, call `POST /api/v1/auth/refresh` with the refresh token. If that also fails, log the user out.

---

## Error Format

Every error response has this shape:

```json
{
  "error": {
    "code": "string",
    "message": "string"
  }
}
```

| HTTP | code | Meaning |
|------|------|---------|
| 400 | `validation_error` | Invalid input |
| 401 | `unauthorized` | Missing or invalid token |
| 403 | `forbidden` | Authenticated but not allowed |
| 404 | `not_found` | Resource not found |
| 409 | `conflict` | Duplicate or constraint violation |
| 500 | `internal_error` | Server error |

---

## Shared Types

### User
```ts
{
  id: number;
  email: string;
  username: string;
  bio: string;
  avatar_url: string;
  skills: Skill[];
  created_at: string; // ISO 8601
}
```

### Skill / Tag
```ts
{ id: number; name: string }
```

### Role
```ts
{
  id: number;
  name: string;
  description: string;
  required_skills: Skill[];
}
```

### ProjectRole
```ts
{
  id: number;
  project_id: number;
  role: Role;
  slots_total: number;
  slots_filled: number;
}
```

### Project
```ts
{
  id: number;
  owner_id: number;
  owner: User;
  title: string;
  description: string;
  status: "draft" | "recruiting" | "in_progress" | "completed" | "archived";
  tags: Tag[];
  roles: ProjectRole[]; // may be absent in list responses
  created_at: string;
}
```

### Application
```ts
{
  id: number;
  user_id: number;
  project_role_id: number;
  status: "pending" | "accepted" | "rejected" | "withdrawn";
  message: string;
  created_at: string;
}
```

### ApplicationWithApplicant (extends Application)
```ts
{
  // all Application fields, plus:
  applicant: User;
}
```

### TokenPair
```ts
{
  access_token: string;
  refresh_token: string;
  access_expires_at: string;
  refresh_expires_at: string;
}
```

### AuthResponse
```ts
{
  user: User;
  token: TokenPair;
}
```

---

## Endpoints

### Health

#### `GET /health`
Public. Returns DB connectivity status.

Response `200`:
```json
{ "status": "ok" }
```
Response `503`:
```json
{ "status": "down" }
```

---

### Auth

#### `POST /api/v1/auth/register`

Register a new user.

Request body:
```json
{
  "email": "user@example.com",    // required, valid email, max 255
  "username": "johndoe",          // required, 2-64 chars
  "password": "securepass123"     // required, 8-128 chars
}
```

Response `201`: `AuthResponse`

Errors: `400` validation, `409` email already taken

---

#### `POST /api/v1/auth/login`

Request body:
```json
{
  "email": "user@example.com",
  "password": "securepass123"
}
```

Response `200`: `AuthResponse`

Errors: `400` validation, `401` wrong credentials

---

#### `POST /api/v1/auth/refresh`

Exchange a refresh token for a new token pair.

Request body:
```json
{ "refresh_token": "..." }
```

Response `200`: `AuthResponse`

Errors: `400` validation, `401` invalid/expired refresh token

---

#### `POST /api/v1/auth/logout`

Revoke refresh token.

Request body:
```json
{ "refresh_token": "..." }
```

Response `204`: no body

Errors: `400` validation, `401` invalid refresh token

---

### Current User (Protected)

#### `GET /api/v1/me`

Get profile of the authenticated user.

Response `200`: `User`

---

#### `PATCH /api/v1/me`

Update authenticated user profile. All fields are optional.

Request body:
```json
{
  "username": "newname",          // 2-64 chars
  "bio": "About me...",           // max 500 chars
  "avatar_url": "https://...",    // valid URL, max 512
  "skill_ids": [1, 2, 3]         // replaces current skills
}
```

Response `200`: `User`

---

### Projects

#### `GET /api/v1/projects`

Public. List/search projects.

Query params:
| Param | Type | Description |
|-------|------|-------------|
| `q` | string | Full-text search |
| `status` | string | Filter by status |
| `tag_ids` | string | Comma-separated tag IDs, e.g. `"1,2,3"` |
| `skill_ids` | string | Comma-separated skill IDs |
| `limit` | int | Default 20 |
| `offset` | int | Default 0 |

Response `200`:
```json
{ "items": Project[] }
```

---

#### `GET /api/v1/projects/:id`

Public. Get one project with full details.

Response `200`: `Project` (includes `roles`)

Errors: `400` invalid id, `404` not found

---

#### `POST /api/v1/projects` (Protected)

Create a project. Authenticated user becomes the owner.

Request body:
```json
{
  "title": "string",              // required, 3-120 chars
  "description": "string",        // required, 10-2000 chars
  "status": "draft",              // optional, default "draft"
  "tag_ids": [1, 2]               // optional
}
```

Response `201`: `Project`

---

#### `PATCH /api/v1/projects/:id` (Protected, owner only)

Update project. All fields optional.

Request body:
```json
{
  "title": "string",              // 3-120 chars
  "description": "string",        // 10-2000 chars
  "status": "recruiting",
  "tag_ids": [1, 2]
}
```

Response `200`: `Project`

Errors: `403` not owner, `404` not found

---

#### `DELETE /api/v1/projects/:id` (Protected, owner only)

Response `204`: no body

---

#### `POST /api/v1/projects/:id/roles` (Protected, owner only)

Add a role slot to a project.

Request body:
```json
{
  "role_id": 1,                   // required, must exist in dictionary
  "slots_total": 2                // required, 1-100
}
```

Response `201`: `ProjectRole`

Errors: `403` not owner, `404` project or role not found, `409` role already added

---

#### `GET /api/v1/projects/:id/applications` (Protected, owner only)

List all applications for all roles in this project.

Response `200`:
```json
{ "items": ApplicationWithApplicant[] }
```

---

### Applications

#### `POST /api/v1/project-roles/:id/applications` (Protected)

Apply to a specific project role.

Request body:
```json
{
  "message": "Why I want to join..." // required, 5-1000 chars
}
```

Response `201`: `Application`

Errors: `404` role not found, `409` no free slots or already applied

---

#### `PATCH /api/v1/applications/:id/status` (Protected, project owner only)

Accept or reject an application.

Request body:
```json
{
  "status": "accepted" // or "rejected" — only these two values allowed
}
```

Response `200`: `Application`

Errors: `403` not project owner, `404` application not found, `409` no free slots / invalid transition

---

### Dictionaries (Public)

#### `GET /api/v1/tags`

Response `200`:
```json
{ "items": Tag[] }
```

#### `GET /api/v1/skills`

Response `200`:
```json
{ "items": Skill[] }
```

---

## Frontend Flow Notes

### Auth flow
1. Register or login → save `access_token` + `refresh_token`
2. Attach `Authorization: Bearer <access_token>` to every protected request
3. On any `401`, try `POST /auth/refresh` → update stored tokens
4. If refresh fails → redirect to login

### Explore projects (unauthenticated)
- Call `GET /api/v1/projects` with filters
- Call `GET /api/v1/tags` and `GET /api/v1/skills` on load for filter options

### Apply to a project
1. `GET /api/v1/projects/:id` — view project + roles
2. `POST /api/v1/project-roles/:roleId/applications` with message

### Manage your project
1. Create: `POST /api/v1/projects`
2. Add roles: `POST /api/v1/projects/:id/roles`
3. View applicants: `GET /api/v1/projects/:id/applications`
4. Accept/reject: `PATCH /api/v1/applications/:id/status`

### Profile
- View: `GET /api/v1/me`
- Edit: `PATCH /api/v1/me` (skills are set by `skill_ids` array — full replace)
