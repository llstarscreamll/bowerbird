# Product dictionary

Shared language for product and engineering.

| Term (UI / product) | Technical | Meaning                                                          |
| ------------------- | --------- | ---------------------------------------------------------------- |
| **Organization**    | `Tenant`  | Customer company. Isolates billing, operational data, and users. |
| **User**            | `User`    | Person with a role. Belongs to one or more organizations.        |

Avoid:

- **Account** for an organization (collides with ledger / AR / bank accounts).
- **Workspace** (too informal for finance).

Infrastructure, middleware, and interceptors keep the industry term `Tenant` (`X-Tenant-ID`, pools, CDK). User-facing copy and product docs use **Organization**.

Colombian e-invoicing domain terms stay in Spanish / official form: **DIAN**, **CUFE**, UBL 2.1 field names as defined by DIAN.
