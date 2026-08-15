# Spec — tenant secrets vault

## Goal

Provide an abstract, tenant-scoped secrets vault so organizations can store credentials used to access external income/outcome data. The first use case is multiple document passwords (cédula/NIT) to unlock DIAN-signed PDF invoices during extraction.

## Requirements

### Secrets vault

- Store multiple secrets per **purpose** (e.g. many passwords under `invoicing.document_password`).
- Encrypt values at rest (AES-GCM); never return plaintext via HTTP after write.
- Support create, rotate (update value), delete, and list/get **metadata** only.
- Append audit events for create / rotate / delete (never log values).
- Workers resolve candidates by purpose in-process (ordered by `last_used_at`, then `created_at`).

### ACL (tenant RBAC)

- Permissions: `secrets:read`, `secrets:write`, `secrets:delete`.
- Users without these permissions cannot list, mutate, or see the secrets UI.
- System `admin` role receives all secrets permissions.
- Org owners are assigned `admin` on provision; existing tenants get a one-time `user_roles` backfill.

### Invoice extraction

- Before Gemini PDF extraction (and for PDFs inside ZIPs), try tenant passwords for purpose `invoicing.document_password` until unlock succeeds.
- On success, update that secret’s `last_used_at`.
- Missing/wrong passwords: soft-fail (warn + skip document), do not put passwords on job payloads.

## Non-goals (v1)

- KMS envelope encryption (interface reserved via `key_id`)
- Reveal / plaintext GET API
- Custom roles UI / invite→role assignment UX
- Per-secret ACLs
