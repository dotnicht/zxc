# Database Structure

## Overview

Each tenant gets four separate PostgreSQL databases. A shared root database holds global state.

```
root        ← shared across all tenants
{name}      ← per-tenant, three schemas: main / deploy / account
{name}_jobs ← per-tenant, managed by go-workflows
```

---

## Root Database

Schema: `public`

### tenants

| Column | Type | Constraints |
|---|---|---|
| id | uuid | PK, default gen_random_uuid() |
| name | varchar(255) | NOT NULL, UNIQUE |
| owner_id | uuid | NOT NULL, FK → users (CASCADE update, RESTRICT delete) |
| main | text | connection string |
| deploy | text | connection string |
| account | text | connection string |
| jobs | text | connection string |
| storage | text | storage path |
| created_at | timestamptz | NOT NULL, default now() |
| updated_at | timestamptz | NOT NULL, default now() |
| deleted_at | timestamptz | nullable, indexed (soft delete) |

### users

| Column | Type | Constraints |
|---|---|---|
| id | uuid | PK, default gen_random_uuid() |
| name | varchar(255) | NOT NULL |
| created_at | timestamptz | NOT NULL, default now() |
| updated_at | timestamptz | NOT NULL, default now() |
| deleted_at | timestamptz | nullable, indexed (soft delete) |

---

## Per-Tenant: `main` Schema

Contains system configuration and the tenant's user roster.

### users

Same structure as root `users`.

### systems

| Column | Type | Constraints |
|---|---|---|
| id | uuid | PK, default gen_random_uuid() |
| name | varchar(255) | NOT NULL |
| sync | varchar(255) | NOT NULL, default `'generator'` — plugin binary name |
| created_at | timestamptz | NOT NULL, default now() |
| updated_at | timestamptz | NOT NULL, default now() |
| deleted_at | timestamptz | nullable, indexed (soft delete) |

A row named `default` with `sync = 'generator'` is seeded automatically on tenant creation.

---

## Per-Tenant: `deploy` Schema

Contains the release pipeline.

### targets

| Column | Type | Constraints |
|---|---|---|
| id | uuid | PK, default gen_random_uuid() |
| address | text | NOT NULL — SSH host |
| user | text | NOT NULL, default `''` — SSH user |
| key | text | NOT NULL, default `''` — SSH private key |
| status | varchar(20) | NOT NULL, default `'unknown'` — `unknown / online / offline` |
| deploying | bool | NOT NULL, default false — deploy lock |
| deploying_at | timestamptz | nullable, indexed — lock acquisition time |
| owner_id | uuid | NOT NULL |
| created_at | timestamptz | NOT NULL, default now() |
| updated_at | timestamptz | NOT NULL, default now() |
| deleted_at | timestamptz | nullable, indexed (soft delete) |

### payloads

| Column | Type | Constraints |
|---|---|---|
| id | uuid | PK, default gen_random_uuid() |
| path | text | NOT NULL — object storage path |
| system_id | uuid | nullable — which system profiles created from this payload belong to |
| owner_id | uuid | NOT NULL |
| config | text | NOT NULL, default `''` — config filename inside zip |
| start | text | NOT NULL, default `''` — start command |
| stop | text | NOT NULL, default `''` — stop command |
| created_at | timestamptz | NOT NULL, default now() |
| updated_at | timestamptz | NOT NULL, default now() |
| deleted_at | timestamptz | nullable, indexed (soft delete) |

### releases

| Column | Type | Constraints |
|---|---|---|
| id | uuid | PK, default gen_random_uuid() |
| status | varchar(20) | NOT NULL, default `'unknown'` — `unknown / wait / deployed / dead` |
| owner_id | uuid | NOT NULL |
| target_id | uuid | nullable |
| payload_id | uuid | nullable |
| changed_by_id | uuid | NOT NULL — user who triggered last status change |
| created_at | timestamptz | NOT NULL, default now() |
| updated_at | timestamptz | NOT NULL, default now() |
| deleted_at | timestamptz | nullable, indexed (soft delete) |

### requests

| Column | Type | Constraints |
|---|---|---|
| id | uuid | PK, default gen_random_uuid() |
| release_id | uuid | NOT NULL |
| data | jsonb | NOT NULL — raw webhook POST body |
| created_at | timestamptz | NOT NULL, default now() |
| updated_at | timestamptz | NOT NULL, default now() |
| deleted_at | timestamptz | nullable, indexed (soft delete) |

---

## Per-Tenant: `account` Schema

Contains profiles and their generated content.

### profiles

| Column | Type | Constraints |
|---|---|---|
| id | uuid | PK, default gen_random_uuid() |
| system_id | uuid | nullable — FK to `main.systems.id` (app-level only) |
| name | varchar(255) | NOT NULL, UNIQUE — derived from webhook node_name |
| status | varchar(20) | NOT NULL, default `'unknown'` — `unknown / active / disabled` |
| created_at | timestamptz | NOT NULL, default now() |
| updated_at | timestamptz | NOT NULL, default now() |
| deleted_at | timestamptz | nullable, indexed (soft delete) |

### sessions

| Column | Type | Constraints |
|---|---|---|
| id | uuid | PK, default gen_random_uuid() |
| profile_id | uuid | NOT NULL, indexed |
| status | varchar(20) | NOT NULL, default `'offline'` — `online / offline / sync` |
| created_at | timestamptz | NOT NULL, default now() |
| updated_at | timestamptz | NOT NULL, default now() |
| deleted_at | timestamptz | nullable, indexed (soft delete) |

### talks

| Column | Type | Constraints |
|---|---|---|
| id | uuid | PK, default gen_random_uuid() |
| profile_id | uuid | NOT NULL |
| created_at | timestamptz | NOT NULL, default now() |
| updated_at | timestamptz | NOT NULL, default now() |
| deleted_at | timestamptz | nullable, indexed (soft delete) |

### contacts

| Column | Type | Constraints |
|---|---|---|
| id | uuid | PK, default gen_random_uuid() |
| profile_id | uuid | NOT NULL, indexed |
| name | varchar(255) | NOT NULL |
| created_at | timestamptz | NOT NULL, default now() |
| updated_at | timestamptz | NOT NULL, default now() |
| deleted_at | timestamptz | nullable, indexed (soft delete) |

### posts

| Column | Type | Constraints |
|---|---|---|
| id | uuid | PK, default gen_random_uuid() |
| talk_id | uuid | NOT NULL |
| profile_id | uuid | NOT NULL |
| contact_id | uuid | NOT NULL |
| text | text | NOT NULL, default `''` — generated content; sync-produced posts prefixed `zxc-gen:` |
| created_at | timestamptz | NOT NULL, default now() |
| updated_at | timestamptz | NOT NULL, default now() |
| deleted_at | timestamptz | nullable, indexed (soft delete) |

### files

| Column | Type | Constraints |
|---|---|---|
| id | uuid | PK, default gen_random_uuid() |
| talk_id | uuid | NOT NULL |
| profile_id | uuid | NOT NULL |
| contact_id | uuid | NOT NULL |
| name | varchar(255) | NOT NULL, default `''` |
| created_at | timestamptz | NOT NULL, default now() |
| updated_at | timestamptz | NOT NULL, default now() |
| deleted_at | timestamptz | nullable, indexed (soft delete) |

---

## Per-Tenant: Jobs Database (`{name}_jobs`)

Managed entirely by `github.com/cschleiden/go-workflows`. Schema applied automatically on first connection. Not touched by the application migrator.

---

## Foreign Keys

Database-level constraint: `tenants.owner_id → root.users.id` (CASCADE update, RESTRICT delete).

All other relationships (profile→system, post→talk, post→contact, release→target, etc.) are enforced at the application level only — no DB-level foreign key constraints across schemas or databases.

---

## Migrations

Applied by the `migrator` binary via GORM `AutoMigrate` at startup:

| Run | Models |
|---|---|
| Root | Tenant, User |
| Main | User, System |
| Deploy | Target, Payload, Release, Request |
| Account | Profile, Session, Talk, File, Contact, Post |
