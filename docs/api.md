# API Structure

## Transport

| Component | Protocol | Port | Auth |
|---|---|---|---|
| gRPC server | gRPC + mTLS | 50051 | metadata headers |
| Webhook | HTTP | 8080 | `Authorization: Bearer <JWT>` |

---

## Authentication (gRPC)

Every request must carry gRPC metadata headers:

| Header | Required | Description |
|---|---|---|
| `x-user-id` | always | UUID of the calling user |
| `x-tenant-id` | non-root only | UUID of the tenant |

The interceptor resolves the user from the root DB. For non-root requests it also loads the tenant and injects three per-request DB connections into the context: `mainDB`, `deployDB`, `accountDB`.

**Root user** (configured UUID, `00000000-0000-0000-0000-000000000001` in dev): can call tenant-level endpoints without `x-tenant-id`.

---

## Common Patterns

- All list RPCs: `page` (default 1), `page_size` (default 10, max 100), ordered by `created_at DESC`
- Soft deletes: all queries filter `deleted_at IS NULL`
- UUIDs: 16-byte arrays in proto, formatted as standard UUID strings in client output
- Timestamps: RFC3339 strings in responses

---

## Services

### TenantService

Operates on the root database.

#### `Create`
```
CreateRequest  { name string }
CreateResponse { tenant Tenant }
```
- `name`: 1–63 chars, must start with lowercase letter, only `[a-z0-9_]`
- Creates per-tenant databases, runs migrations, seeds default system and owner user, starts Sync workflow
- Errors: `InvalidArgument` (validation), `AlreadyExists` (duplicate name), `Internal`

#### `Get`
```
GetRequest  { id bytes }
GetResponse { tenant Tenant }
```
- Errors: `NotFound`, `Internal`

#### `List`
```
ListRequest  { page int32, page_size int32 }
ListResponse { tenants []Tenant, total int32 }
```

**Tenant message**
```
id, name, owner_id, database, deploy, account, jobs, storage, created_at, updated_at
```

---

### UserService

Operates on `main` schema (tenant-scoped).

#### `Create`
```
CreateRequest  { name string }
CreateResponse { user User }
```
- `name` required
- Errors: `InvalidArgument`, `Internal`

#### `Get` / `Delete`
```
{GetRequest,DeleteRequest}  { id bytes }
{GetResponse}               { user User }
{DeleteResponse}            { success bool }
```

#### `Update`
```
UpdateRequest  { id bytes, name string }
UpdateResponse { user User }
```
- `name` required

#### `List`
```
ListRequest  { page int32, page_size int32 }
ListResponse { users []User, total int32 }
```

**User message**
```
id, name, created_at, updated_at
```

---

### SystemService

Operates on `main` schema (tenant-scoped).

#### `Create`
```
CreateRequest  { name string, sync string }
CreateResponse { system System }
```
- `sync`: plugin binary name (e.g. `generator`)

#### `Get` / `Delete`
```
{GetRequest,DeleteRequest}  { id bytes }
{GetResponse}               { system System }
{DeleteResponse}            { success bool }
```

#### `Update`
```
UpdateRequest  { id bytes, name string, sync string }
UpdateResponse { system System }
```
- `name` required

#### `List`
```
ListRequest  { page int32, page_size int32 }
ListResponse { systems []System, total int32 }
```

**System message**
```
id, name, sync, created_at, updated_at
```

---

### TargetService

Operates on `deploy` schema.

#### `Create`
```
CreateRequest  { owner_id bytes, address string, user string, key string }
CreateResponse { target Target }
```
- `address` required (SSH host)
- Side effect: starts a `Probe` workflow (30s loop, SSH connectivity check)
- Errors: `InvalidArgument`, `Internal`

#### `Get` / `Delete`
```
{GetRequest,DeleteRequest}  { id bytes }
{GetResponse}               { target Target }
{DeleteResponse}            { success bool }
```

#### `Update`
```
UpdateRequest  { id bytes, address string, user string, key string }
UpdateResponse { target Target }
```
- Side effect: re-enqueues Probe workflow

#### `List`
```
ListRequest  { page int32, page_size int32 }
ListResponse { targets []Target, total int32 }
```

**Target message**
```
id, address, user, status, owner_id, created_at, updated_at
```
Status values: `unknown`, `online`, `offline`

---

### PayloadService

Operates on `deploy` schema + object storage.

#### `Create`
```
CreateRequest  {
  owner_id  bytes,
  content   bytes,   // zip file, max 50 MB
  config    string,  // filename inside zip
  name      string,  // defaults to "payload"
  start     string,  // start command
  stop      string,  // stop command
  system_id bytes    // system to assign to created profiles
}
CreateResponse { payload Payload }
```
- `content` and `config` required
- Config file inside zip must contain `{ZXC_URL}` and `{ZXC_AUTH}` placeholders
- Zip uploaded to object storage at `payloads/<id>/<name>`
- Errors: `InvalidArgument` (missing fields, invalid zip, missing placeholders), `Internal`

#### `Get` / `Delete`
```
{GetRequest,DeleteRequest}  { id bytes }
{GetResponse}               { payload Payload }
{DeleteResponse}            { success bool }
```

#### `Update`
```
UpdateRequest  { id bytes, path string, config string, start string, stop string }
UpdateResponse { payload Payload }
```

#### `List`
```
ListRequest  { page int32, page_size int32 }
ListResponse { payloads []Payload, total int32 }
```

**Payload message**
```
id, path, name, owner_id, system_id, config, start, stop, created_at, updated_at
```

---

### ReleaseService

Operates on `deploy` schema.

#### `Create`
```
CreateRequest  { owner_id bytes, target_id bytes, payload_id bytes }
CreateResponse { release Release }
```
- Both `target_id` and `payload_id` must exist
- Initial status: `unknown`
- Errors: `NotFound`, `Internal`

#### `Get`
```
GetRequest  { id bytes }
GetResponse { release Release }
```

#### `Deploy`
```
DeployRequest  { id bytes, user_id bytes }
DeployResponse { release Release }
```
- Precondition: `status == "unknown"`
- Transition: `unknown → wait`
- Starts Deploy workflow (instance ID `deploy:<releaseID>`)
- Errors: `NotFound`, `Internal`

#### `List`
```
ListRequest  { page int32, page_size int32 }
ListResponse { releases []Release, total int32 }
```

**Release message**
```
id, status, owner_id, target_id, payload_id, changed_by_id, created_at, updated_at
```
Status values: `unknown`, `wait`, `deployed`, `dead`

---

### AccountService

Operates on `account` schema. Profiles are created automatically by the Account workflow — not via this API.

#### `Get`
```
GetRequest  { id bytes }
GetResponse { account Account }
```

#### `List`
```
ListRequest  { page int32, page_size int32 }
ListResponse { accounts []Account, total int32 }
```

#### `Disable`
```
DisableRequest  { id bytes }
DisableResponse { account Account }
```
- Sets `status = "disabled"`

#### `GetTalks`
```
GetTalksRequest  { profile_id bytes }
GetTalksResponse { talks []Talk }
```
- Returns all talks for the profile with their posts and files, sorted by `created_at`
- Each `TalkItem` is a `oneof { Post, File }`

**Account message**
```
id, system_id, name, status, created_at, updated_at
```
Status values: `unknown`, `active`, `disabled`

---

### SessionService

Operates on `account` schema.

#### `Get`
```
GetRequest  { id bytes }
GetResponse { session Session }
```

#### `Start` / `Stop`
```
{StartRequest,StopRequest}   { id bytes }
{StartResponse,StopResponse} { session Session }
```
- `Start` sets `status = "online"`
- `Stop` sets `status = "offline"`

#### `List`
```
ListRequest  { page int32, page_size int32 }
ListResponse { sessions []Session, total int32 }
```

**Session message**
```
id, profile_id, status, created_at, updated_at
```
Status values: `online`, `offline`, `sync`

---

## HTTP Webhook

```
POST /webhooks
Authorization: Bearer <HS256-signed JWT>
Content-Type: application/json
```

**JWT claims**
```json
{ "release_id": "<uuid>", "tenant_id": "<uuid>" }
```

**Response codes**
| Code | Meaning |
|---|---|
| 201 | Stored, Account workflow enqueued |
| 400 | Missing/invalid auth header, invalid JWT, malformed JSON |
| 404 | Tenant not found |
| 500 | DB or workflow error |

---

## Error Codes

| gRPC code | Value | Condition |
|---|---|---|
| `InvalidArgument` | 3 | Validation failure, missing required field |
| `NotFound` | 5 | Entity does not exist |
| `AlreadyExists` | 6 | Duplicate name (tenant, profile) |
| `PermissionDenied` | 7 | Non-root caller missing `x-tenant-id` |
| `Internal` | 13 | DB, storage, or workflow error |
| `Unauthenticated` | 16 | Missing or unresolvable `x-user-id` |
