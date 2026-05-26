package database

import (
	"context"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/tenant"
)

// Registry manages database connection pools for multiple tenants.
type Registry struct {
	mu            sync.RWMutex
	pools         map[string]*pgxpool.Pool
	controlDB     *pgxpool.Pool
	baseConfigURL string // Base URL format, e.g., postgres://user:pass@host:5432/%s
}

// NewRegistry initializes a new Registry with the control plane database pool.
func NewRegistry(controlDB *pgxpool.Pool, baseConfigURL string) *Registry {
	return &Registry{
		pools:         make(map[string]*pgxpool.Pool),
		controlDB:     controlDB,
		baseConfigURL: baseConfigURL,
	}
}

// GetPool returns the connection pool for the tenant slug in the context.
// If the pool doesn't exist, it resolves the database name and creates a new one.
func (r *Registry) GetPool(ctx context.Context) (*pgxpool.Pool, error) {
	tenantSlug, err := tenant.TenantSlugFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Fast path: read lock
	r.mu.RLock()
	pool, exists := r.pools[tenantSlug]
	r.mu.RUnlock()
	if exists {
		return pool, nil
	}

	// Slow path: resolve from control plane and initialize
	r.mu.Lock()
	defer r.mu.Unlock()

	// Double check
	if pool, exists := r.pools[tenantSlug]; exists {
		return pool, nil
	}

	dbName, err := r.resolveTenantDatabase(ctx, tenantSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve tenant db: %w", err)
	}

	dbURL := fmt.Sprintf(r.baseConfigURL, dbName)
	newPool, err := Connect(ctx, dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to tenant db %s: %w", tenantSlug, err)
	}

	r.pools[tenantSlug] = newPool
	return newPool, nil
}

// GetPoolByDBName allows direct access to a database pool using the physical db_name.
// This is useful during tenant creation or explicit cross-tenant operations where
// the tenant context is not set.
func (r *Registry) GetPoolByDBName(ctx context.Context, dbName string) (*pgxpool.Pool, error) {
	// Fast path: read lock
	r.mu.RLock()
	pool, exists := r.pools[dbName] // we use dbName as key here for this specific path
	r.mu.RUnlock()
	if exists {
		return pool, nil
	}

	// Slow path
	r.mu.Lock()
	defer r.mu.Unlock()

	// Double check
	if pool, exists := r.pools[dbName]; exists {
		return pool, nil
	}

	dbURL := fmt.Sprintf(r.baseConfigURL, dbName)
	newPool, err := Connect(ctx, dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to tenant db by name %s: %w", dbName, err)
	}

	r.pools[dbName] = newPool
	return newPool, nil
}

// resolveTenantDatabase looks up the database name for a given tenant identifier (ID or slug) in the control plane.
func (r *Registry) resolveTenantDatabase(ctx context.Context, identifier string) (string, error) {
	var dbName string
	// Allow resolving by either ID or slug, as X-Tenant-ID may contain the ULID
	query := `SELECT db_name FROM tenants WHERE (id = $1 OR slug = $1) AND status = 'active'`
	err := r.controlDB.QueryRow(ctx, query, identifier).Scan(&dbName)
	if err != nil {
		return "", err
	}
	return dbName, nil
}

// CloseAll closes all connection pools.
func (r *Registry) CloseAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, pool := range r.pools {
		pool.Close()
	}
	r.controlDB.Close()
}
