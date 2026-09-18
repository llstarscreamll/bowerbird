# Shared Go (`pkg/`)

Mature Go libraries shared by more than one Canopy product. Import path
shape: `canopy/pkg/<name>` (for example `httpx`, `rbac`).

Atta's backend still owns platform code under
`apps/atta/backend/internal/platform`. Extract a module here only when
a second product needs it.
