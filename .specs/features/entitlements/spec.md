# Product entitlements

## Problem statement

Bowerbird must enable and disable products and features per tenant, managed by platform operators (above any organization). First cut: Mail product and Send feature, without cutting invoice capture.

## Goals

- [ ] An operator enables or disables the Mail product for a tenant, with optional end date (trial).
- [ ] With Mail on, the operator can turn off Send; archive/label/read still work.
- [ ] With Mail off, Invoicing keeps syncing incoming mail.
- [ ] JWT carries neither roles nor features; operator and access resolve at runtime.

## Out of scope

| Feature                  | Reason                                            |
| ------------------------ | ------------------------------------------------- |
| Invoicing toggle in UI   | Catalog yes; UI in a later cut                    |
| Billing / Stripe / seats | Entitlements are the model; billing is another BC |
| UI to nominate operators | `PLATFORM_OPERATOR_EMAILS` is enough              |
| Deploy feature flags     | Separate problem                                  |

## Catalog

| Product     | Feature                        | Required | UI                                 |
| ----------- | ------------------------------ | -------- | ---------------------------------- |
| `invoicing` | `invoicing.workspace`          | yes      | Invoices                           |
| `invoicing` | `invoicing.capture_from_email` | yes      | (ingest; not Mail UI)              |
| `mail`      | `mail.inbox`                   | yes      | Mail: read mailbox                 |
| `mail`      | `mail.organize`                | yes      | Archive, label, read, star         |
| `mail`      | `mail.send`                    | no       | Send: compose/reply as the account |

Default pack (all tenants): invoicing.\* + mail.inbox + mail.organize. No mail.send.

## User stories

### P1: Operator manages access ⭐ MVP

**User story**: As an operator, I want to turn Mail and Send on/off per organization, and grant a trial with an end date.

**Acceptance criteria**:

1. WHEN the operator turns Mail off THEN the mail-client API SHALL return `ERR_FORBIDDEN` and the Mails nav SHALL hide.
2. WHEN Mail is off AND Invoicing is active THEN sync, connections, and `InboxMessageReceived` SHALL continue.
3. WHEN Mail is on AND Send is off THEN `POST /inbox/messages` SHALL be 403 AND modify/archive SHALL work.
4. WHEN an entitlement has `ends_at` in the past THEN it SHALL evaluate as off.
5. WHEN a user without `platform_role=operator` calls `/api/v1/platform/*` THEN they SHALL get 403 even with a valid JWT.
6. JWT SHALL NOT include `platform_role` or feature keys.

**Independent test**: Evaluate of expired trial excludes `mail.inbox`; ingest `invoicing.capture_from_email` stays true.
