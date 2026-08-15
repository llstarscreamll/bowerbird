# Design — tenant secrets vault

## Bounded contexts

| Module              | Owns                                                                   | Does not                   |
| ------------------- | ---------------------------------------------------------------------- | -------------------------- |
| `internal/rbac`     | Effective permissions, `RequirePermission`, `GET /rbac/me/permissions` | Secret storage             |
| `internal/secrets`  | Encrypted CRUD, audit, `ResolveByPurpose`                              | PDF unlock, invoice domain |
| `internal/invoices` | Try password candidates before LLM                                     | Secret UI / ACL            |

## Domain

```text
Secret
  id, purpose, kind, label
  ciphertext, version, key_id
  last_used_at
  created_by, updated_by, timestamps
```

- Purpose `invoicing.document_password` groups DIAN document passwords (many rows).
- Also: `integrations.api_key`, `integrations.idp_credential`, `generic.credential`.
- Unique `(purpose, label)` per tenant.
- Kind: `document_password` | `generic_string`.

## Security

- Dedicated SSM key `tenant_secrets_encryption_key`.
- Write-only HTTP API; worker resolve bypasses user ACL under tenant context.
- Audit table `secret_audit_events`.

## HTTP

- `GET /api/v1/secrets?purpose=`
- `GET /api/v1/secrets/{id}`
- `POST /api/v1/secrets`
- `PUT /api/v1/secrets/{id}`
- `DELETE /api/v1/secrets/{id}`
- `GET /api/v1/rbac/me/permissions`

## Extraction flow

1. Detect encrypted PDF (or passworded ZIP).
2. `ResolveByPurpose(invoicing.document_password)`.
3. Try each password until unlock works → mark used → Gemini on plaintext.
4. Else warn + skip.
