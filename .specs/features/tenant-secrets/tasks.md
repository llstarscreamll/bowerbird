# Tasks — tenant secrets vault

All implementation tasks are done.

- [x] Tenant migration: secrets tables, `secrets:*` permissions, admin grants, `user_roles` backfill
- [x] Fix `SeedOwner` to assign system `admin`
- [x] `internal/rbac` + permissions endpoint
- [x] `internal/secrets` + SSM encryption key
- [x] Invoice PDF/ZIP unlock with multi-password try
- [x] PWA PermissionsStore + secrets settings page
- [x] Wire API / SQS; unit tests

Operational (local/env, not code): re-init LocalStack SSM after `secrets.json` changes; run tenant migrations (`migrate:all`).
