# End-to-End Scenario

## Overview

The full flow from tenant creation to generated content in the account database:

```
tenant create
    └── seed: default system + owner user
    └── start: Sync workflow (60s loop)
          │
target create → Probe workflow (30s loop, SSH check)
          │
payload create (zip + system_id)
          │
release create → deploy
          │
          └── Deploy workflow
                  ├── lock target
                  ├── inject config ({ZXC_URL}, {ZXC_AUTH} → JWT)
                  ├── SSH: stop + start on target
                  └── status: wait → deployed
                          │
                          └── target starts calling webhook
                                  │
                          POST /webhooks (JWT auth)
                                  │
                          Account workflow
                                  ├── extract node_name from request data
                                  ├── follow release → payload → system_id
                                  └── create Profile in account DB
                                          │
                          Sync workflow (every 60s)
                                  ├── load default system from main DB
                                  ├── load plugin binary (system.sync)
                                  ├── load profiles WHERE system_id = system.id
                                  └── for each profile:
                                          ├── get/create Talk
                                          ├── get/create Contact
                                          ├── call plugin.Post(profile.name)
                                          └── create Post (text: "zxc-gen: ...")
```

---

## Step-by-Step

### 1. Tenant Creation

```
client tenant add --name <name>
```

- Creates three PostgreSQL databases: `<name>` (main/deploy/account schemas), `<name>_jobs`
- Runs AutoMigrate for each schema
- Seeds `main.users` with the owner user
- Seeds `main.systems` with `{name: "default", sync: "generator"}`
- Starts a `Sync` workflow instance (instance ID `sync:<tenantID>`) — loops every 60 seconds

### 2. Target Setup

```
client --tenant <name> target add --address <host> --user deploy --key <path>
```

- Creates a `deploy.targets` row with `status = "unknown"`
- Starts a `Probe` workflow (instance ID `probe:<targetID>`) — loops every 30 seconds, SSH-checks the target and updates status to `online` or `offline`

### 3. Payload Upload

```
client --tenant <name> payload add \
  --file payload.zip \
  --config script.conf \
  --start "bash ~/script.sh" \
  --stop "true" \
  --system <systemID>
```

- The zip must contain the file named by `--config`
- That config file must contain `{ZXC_URL}` and `{ZXC_AUTH}` placeholders
- Zip is uploaded to object storage at `payloads/<payloadID>/<name>`
- `system_id` is stored on the payload row — this is how profiles created from this payload are assigned to a system

### 4. Release Creation

```
client --tenant <name> release add --target <targetID> --payload <payloadID>
```

- Creates a `deploy.releases` row with `status = "unknown"`

### 5. Deploy

```
client --tenant <name> release deploy --id <releaseID>
```

- Status: `unknown → wait`
- Starts a `Deploy` workflow

#### Deploy Workflow internals

1. Acquires a deploy lock on the target (`deploying = true`, `deploying_at = now()`). Stale locks older than 15 minutes are forcibly released.
2. Downloads the payload zip from object storage.
3. Injects configuration into the config file inside the zip:
   - `{ZXC_URL}` → webhook URL from config
   - `{ZXC_AUTH}` → HS256-signed JWT containing `release_id` and `tenant_id`
4. Uploads the injected zip to `releases/<releaseID>.zip`.
5. SSHes into the target:
   - Extracts the zip
   - Runs the `stop` command
   - Runs the `start` command
6. On success: `status → deployed`, target `status → online`
7. On failure: `status → dead`

### 6. Webhook Reception

The deployed script calls `POST /webhooks` with the injected JWT.

```
POST /webhooks
Authorization: Bearer <jwt>
Content-Type: application/json

{"node_name": "node-abc123", ...}
```

- JWT is verified (HS256, secret from config); `release_id` and `tenant_id` extracted from claims
- Request body stored as-is in `deploy.requests.data` (jsonb)
- `Account` workflow enqueued (instance ID `account:<requestID>`)

### 7. Account Workflow

Extracts the node name from the request data. Checked fields in order:

1. `data.node_name`
2. `data.nodeName`
3. `data.node` (string)
4. `data.node.name` (nested object)

Follows `Request → Release → Payload` to get `system_id`, then creates a `Profile` in the account database:

```
profiles.name      = extracted node_name
profiles.status    = "unknown"
profiles.system_id = payload.system_id
```

Duplicate names are silently ignored (idempotent).

### 8. Sync Job

Runs every 60 seconds for each tenant.

1. Loads the `default` system from `main.systems`
2. Loads (and caches) the plugin binary from `plugins/<system.sync>`
3. Queries `account.profiles WHERE system_id = <system.id> AND deleted_at IS NULL`
4. For each profile:
   - Picks a random existing `Talk`, or creates a new one
   - Picks a random existing `Contact`, or creates a new one named `"bot"`
   - Calls `plugin.Post(ctx, profile.name)` — returns a text string
   - Creates a `Post` with that text

The `generator` plugin prefixes every generated text with `zxc-gen:` — used by tests to identify sync-produced posts.

---

## Test Coverage

### TestE2E (`test/e2e_test.go`)

Runs the full scenario end-to-end using a real Docker stack.

| Assertion | Timeout |
|---|---|
| Release reaches `deployed` | 90s |
| Webhook received ≥ 2 requests | 60s |
| ≥ 1 profile created in account DB | 60s |
| Profile name matches `node_name` in request | — |
| Post with `zxc-gen:` prefix exists in account DB | 90s |

### TestGenerateJob (`test/generate_test.go`)

Bypasses the webhook flow — inserts a profile directly and verifies that the Sync job picks it up.

| Assertion | Timeout |
|---|---|
| Post created with correct `talk_id` and `contact_id` | 120s |

---

## Timeouts Reference

| Stage | Timeout |
|---|---|
| Migrator container completion | 120s |
| gRPC endpoint readiness | 60s |
| Worker container running | 60s |
| Shared fixture: first profile appears | 90s |
| Release reaches `deployed` | 90s |
| Webhook requests + profile | 60s |
| Generated post (`zxc-gen:`) | 90s |
| TestGenerateJob post | 120s |
| Deploy lock stale cleanup | 15 min |
| Sync loop interval | 60s |
| Probe loop interval | 30s |
