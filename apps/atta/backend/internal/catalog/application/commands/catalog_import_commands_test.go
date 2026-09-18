package commands

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/atta/internal/catalog/application/ports"
	contractJobs "github.com/atta/internal/catalog/contracts/jobs"
	"github.com/atta/internal/catalog/domain"
	filesapi "github.com/atta/internal/files/api"
	appErrors "github.com/atta/internal/platform/errors"
	"github.com/atta/internal/platform/id"
	"github.com/atta/internal/platform/jobs"
	"github.com/atta/internal/platform/tenant"
)

type memCatalog struct {
	items   map[string]domain.Item
	imports map[string]domain.CatalogImport
	errors  []domain.ImportRowError
}

func newMemCatalog() *memCatalog {
	return &memCatalog{items: map[string]domain.Item{}, imports: map[string]domain.CatalogImport{}}
}

func (m *memCatalog) CreateItem(_ context.Context, item domain.Item) error {
	if m.items == nil {
		m.items = map[string]domain.Item{}
	}
	m.items[item.ID] = item
	return nil
}
func (m *memCatalog) UpdateItem(_ context.Context, item domain.Item) error {
	m.items[item.ID] = item
	return nil
}
func (m *memCatalog) GetItemByID(_ context.Context, id string) (*domain.Item, error) {
	item, ok := m.items[id]
	if !ok {
		return nil, nil
	}
	cp := item
	return &cp, nil
}
func (m *memCatalog) GetItemNames(context.Context, []string) (map[string]string, error) {
	return map[string]string{}, nil
}
func (m *memCatalog) GetItemsByIDs(context.Context, []string) ([]domain.Item, error) {
	return nil, nil
}
func (m *memCatalog) GetItemsByInternalCodes(_ context.Context, codes []string) ([]domain.Item, error) {
	want := map[string]struct{}{}
	for _, c := range codes {
		want[c] = struct{}{}
	}
	out := make([]domain.Item, 0)
	for _, item := range m.items {
		if _, ok := want[item.InternalCode]; ok {
			out = append(out, item)
		}
	}
	return out, nil
}
func (m *memCatalog) CreateItems(ctx context.Context, items []domain.Item) error {
	for _, item := range items {
		if err := m.CreateItem(ctx, item); err != nil {
			return err
		}
	}
	return nil
}
func (m *memCatalog) UpdateItems(ctx context.Context, items []domain.Item) error {
	for _, item := range items {
		if err := m.UpdateItem(ctx, item); err != nil {
			return err
		}
	}
	return nil
}
func (m *memCatalog) ListItems(context.Context, ports.ItemListFilter) (ports.ItemListPage, error) {
	return ports.ItemListPage{}, nil
}
func (m *memCatalog) FindByNormalizedDescription(context.Context, string) ([]domain.Item, error) {
	return nil, nil
}
func (m *memCatalog) CreateImport(_ context.Context, imp domain.CatalogImport) error {
	if m.imports == nil {
		m.imports = map[string]domain.CatalogImport{}
	}
	for _, existing := range m.imports {
		if existing.IsActive() {
			return appErrors.New(appErrors.CodeConflict, "an import is already in progress")
		}
	}
	m.imports[imp.ID] = imp
	return nil
}
func (m *memCatalog) UpdateImport(_ context.Context, imp domain.CatalogImport) error {
	m.imports[imp.ID] = imp
	return nil
}
func (m *memCatalog) GetImportByID(_ context.Context, id string) (*domain.CatalogImport, error) {
	imp, ok := m.imports[id]
	if !ok {
		return nil, nil
	}
	cp := imp
	return &cp, nil
}
func (m *memCatalog) GetActiveImport(context.Context) (*domain.CatalogImport, error) {
	for _, imp := range m.imports {
		if imp.IsActive() {
			cp := imp
			return &cp, nil
		}
	}
	return nil, nil
}
func (m *memCatalog) ListImports(context.Context, ports.ImportListFilter) (ports.ImportListPage, error) {
	return ports.ImportListPage{}, nil
}
func (m *memCatalog) InsertImportErrors(_ context.Context, rows []domain.ImportRowError) error {
	m.errors = append(m.errors, rows...)
	return nil
}
func (m *memCatalog) ListImportErrors(context.Context, ports.ImportErrorListFilter) (ports.ImportErrorListPage, error) {
	return ports.ImportErrorListPage{Items: m.errors, Total: int64(len(m.errors))}, nil
}
func (m *memCatalog) PurgeStaleImports(_ context.Context, before time.Time, _ int) (int64, error) {
	var n int64
	for id, imp := range m.imports {
		if imp.CreatedAt.Before(before) && !imp.IsActive() {
			delete(m.imports, id)
			n++
		}
	}
	kept := m.errors[:0]
	for _, row := range m.errors {
		if _, ok := m.imports[row.ImportID]; ok {
			kept = append(kept, row)
		}
	}
	m.errors = kept
	return n, nil
}
func (m *memCatalog) ApplyImportChunk(ctx context.Context, imp domain.CatalogImport, creates, updates []domain.Item, errs []domain.ImportRowError) error {
	current, ok := m.imports[imp.ID]
	if ok && current.IsTerminal() {
		return nil
	}
	if err := m.CreateItems(ctx, creates); err != nil {
		return err
	}
	if err := m.UpdateItems(ctx, updates); err != nil {
		return err
	}
	m.errors = append(m.errors, errs...)
	m.imports[imp.ID] = imp
	return nil
}

type memFiles struct {
	files   map[string][]byte
	offsets []int64
}

func (f *memFiles) Open(_ context.Context, _ string, key string, offset int64) (*filesapi.Object, error) {
	data, ok := f.files[key]
	if !ok {
		return nil, errors.New("not found")
	}
	if offset < 0 {
		offset = 0
	}
	if offset > int64(len(data)) {
		offset = int64(len(data))
	}
	f.offsets = append(f.offsets, offset)
	return &filesapi.Object{Body: io.NopCloser(bytes.NewReader(data[offset:])), SizeBytes: int64(len(data))}, nil
}

type spyQueue struct {
	jobs []jobs.Job
}

func (q *spyQueue) Enqueue(_ context.Context, job jobs.Job) error {
	q.jobs = append(q.jobs, job)
	return nil
}

func testActor() domain.ImportActor {
	actor, _ := domain.NewImportActor("u1", "a@b.co", "Ana", "Pérez")
	return actor
}

func TestQueueCatalogImport(t *testing.T) {
	t.Parallel()
	store := newMemCatalog()
	files := &memFiles{files: map[string][]byte{"k.csv": []byte("internal_code,name,kind\nA,Item,bien\n")}}
	queue := &spyQueue{}
	cmd := NewQueueCatalogImportCommand(store, files, queue)
	imp, err := cmd.Execute(context.Background(), QueueCatalogImportInput{ID: id.NewULID(), FileKey: "k.csv", Requester: testActor()})
	if err != nil {
		t.Fatalf("queue: %v", err)
	}
	if imp.Status != domain.ImportStatusQueued {
		t.Fatalf("status=%s", imp.Status)
	}
	if len(queue.jobs) != 1 {
		t.Fatalf("jobs=%d", len(queue.jobs))
	}
	_, err = cmd.Execute(context.Background(), QueueCatalogImportInput{ID: id.NewULID(), FileKey: "k.csv", Requester: testActor()})
	var appErr *appErrors.AppError
	if err == nil || !errors.As(err, &appErr) || appErr.Code != appErrors.CodeConflict {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestQueueCatalogImportReleasesSlotWhenEnqueueFails(t *testing.T) {
	t.Parallel()
	store := newMemCatalog()
	files := &memFiles{files: map[string][]byte{"k.csv": []byte("internal_code,name,kind\nA,Item,bien\n")}}
	cmd := NewQueueCatalogImportCommand(store, files, failQueue{})
	_, err := cmd.Execute(context.Background(), QueueCatalogImportInput{ID: id.NewULID(), FileKey: "k.csv", Requester: testActor()})
	if err == nil {
		t.Fatal("expected enqueue error")
	}
	active, err := store.GetActiveImport(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if active != nil {
		t.Fatalf("active import still held: %+v", active)
	}
	queue := &spyQueue{}
	okCmd := NewQueueCatalogImportCommand(store, files, queue)
	if _, err := okCmd.Execute(context.Background(), QueueCatalogImportInput{ID: id.NewULID(), FileKey: "k.csv", Requester: testActor()}); err != nil {
		t.Fatalf("retry queue: %v", err)
	}
	if len(queue.jobs) != 1 {
		t.Fatalf("jobs=%d", len(queue.jobs))
	}
}

type failQueue struct{}

func (failQueue) Enqueue(context.Context, jobs.Job) error {
	return errors.New("outbox down")
}

func TestProcessCatalogImportCreatesAndUpdates(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	store := newMemCatalog()
	existing, err := domain.NewManualItem(id.NewULID(), "Old", mustKind(domain.KindGoods), mustCode("SKU-1"), now)
	if err != nil {
		t.Fatal(err)
	}
	store.items[existing.ID] = existing
	csvBody := "internal_code,name,kind\nSKU-1,New Name,servicio\nSKU-2,Imported,bien\n"
	files := &memFiles{files: map[string][]byte{"k.csv": []byte(csvBody)}}
	queue := &spyQueue{}
	imp := mustImport(store, "k.csv")
	cmd := NewProcessCatalogImportCommand(store, store, files, queue)
	cmd.now = func() time.Time { return now }
	if err := cmd.Execute(context.Background(), contractJobs.CatalogImportRequestedJob{ImportID: imp.ID}); err != nil {
		t.Fatalf("process: %v", err)
	}
	got := store.imports[imp.ID]
	if got.Status != domain.ImportStatusCompleted {
		t.Fatalf("status=%s reason=%s", got.Status, got.FailureReason)
	}
	if got.CreatedCount != 1 || got.UpdatedCount != 1 {
		t.Fatalf("created=%d updated=%d", got.CreatedCount, got.UpdatedCount)
	}
	updated := store.items[existing.ID]
	if updated.Name != "New Name" || updated.Kind != domain.KindService {
		t.Fatalf("updated=%+v", updated)
	}
	if updated.CreationSource != domain.CreationSourceManual || updated.InternalCode != "SKU-1" {
		t.Fatalf("identity mutated: %+v", updated)
	}
	var created domain.Item
	for _, item := range store.items {
		if item.InternalCode == "SKU-2" {
			created = item
		}
	}
	if created.CreationSource != domain.CreationSourceImport || created.Status != domain.StatusConfirmed {
		t.Fatalf("created=%+v", created)
	}
}

func TestProcessCatalogImportRowErrorsAndLastWriteWins(t *testing.T) {
	t.Parallel()
	store := newMemCatalog()
	csvBody := "internal_code,name,kind\n,Missing,bien\nSKU-DUP,First,bien\nSKU-DUP,Second,servicio\nSKU-BAD,Name,xyz\n"
	files := &memFiles{files: map[string][]byte{"k.csv": []byte(csvBody)}}
	imp := mustImport(store, "k.csv")
	cmd := NewProcessCatalogImportCommand(store, store, files, &spyQueue{})
	if err := cmd.Execute(context.Background(), contractJobs.CatalogImportRequestedJob{ImportID: imp.ID}); err != nil {
		t.Fatalf("process: %v", err)
	}
	got := store.imports[imp.ID]
	if got.FailedCount != 2 || got.CreatedCount != 1 {
		t.Fatalf("failed=%d created=%d errors=%d", got.FailedCount, got.CreatedCount, len(store.errors))
	}
	var item domain.Item
	for _, it := range store.items {
		if it.InternalCode == "SKU-DUP" {
			item = it
		}
	}
	if item.Name != "Second" || item.Kind != domain.KindService {
		t.Fatalf("last write: %+v", item)
	}
}

func TestProcessCatalogImportResumeAndCancel(t *testing.T) {
	t.Parallel()
	store := newMemCatalog()
	csvBody := "internal_code,name,kind\nA,One,bien\nB,Two,bien\nC,Three,bien\n"
	files := &memFiles{files: map[string][]byte{"k.csv": []byte(csvBody)}}
	queue := &spyQueue{}
	imp := mustImport(store, "k.csv")
	cmd := NewProcessCatalogImportCommand(store, store, files, queue)
	cmd.chunkRows = 1
	if err := cmd.Execute(context.Background(), contractJobs.CatalogImportRequestedJob{ImportID: imp.ID}); err != nil {
		t.Fatalf("chunk1: %v", err)
	}
	if store.imports[imp.ID].Status != domain.ImportStatusProcessing {
		t.Fatalf("status=%s", store.imports[imp.ID].Status)
	}
	if len(queue.jobs) != 1 {
		t.Fatalf("expected re-enqueue, jobs=%d", len(queue.jobs))
	}
	canceller, _ := domain.NewImportActor("u2", "c@b.co", "Cata", "Lina")
	cancel := NewCancelCatalogImportCommand(store)
	if _, err := cancel.Execute(context.Background(), imp.ID, canceller); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	queue.jobs = nil
	if err := cmd.Execute(context.Background(), contractJobs.CatalogImportRequestedJob{ImportID: imp.ID}); err != nil {
		t.Fatalf("after cancel: %v", err)
	}
	if len(queue.jobs) != 0 {
		t.Fatalf("must not re-enqueue after cancel")
	}
	if store.imports[imp.ID].Status != domain.ImportStatusCancelled {
		t.Fatalf("status=%s", store.imports[imp.ID].Status)
	}
}

func TestProcessCatalogImportResumesFromByteOffset(t *testing.T) {
	t.Parallel()
	store := newMemCatalog()
	csvBody := "internal_code,name,kind\nA,One,bien\nB,Two,bien\nC,Three,bien\n"
	files := &memFiles{files: map[string][]byte{"k.csv": []byte(csvBody)}}
	queue := &spyQueue{}
	imp := mustImport(store, "k.csv")
	cmd := NewProcessCatalogImportCommand(store, store, files, queue)
	cmd.chunkRows = 1
	if err := cmd.Execute(context.Background(), contractJobs.CatalogImportRequestedJob{ImportID: imp.ID}); err != nil {
		t.Fatalf("chunk1: %v", err)
	}
	offset := store.imports[imp.ID].ByteOffset
	if offset <= 0 {
		t.Fatalf("byte_offset=%d", offset)
	}
	files.offsets = nil
	if err := cmd.Execute(context.Background(), contractJobs.CatalogImportRequestedJob{ImportID: imp.ID}); err != nil {
		t.Fatalf("chunk2: %v", err)
	}
	foundRange := false
	for _, off := range files.offsets {
		if off == offset {
			foundRange = true
			break
		}
	}
	if !foundRange {
		t.Fatalf("expected Open at byte_offset=%d, opens=%v", offset, files.offsets)
	}
	if err := cmd.Execute(context.Background(), contractJobs.CatalogImportRequestedJob{ImportID: imp.ID}); err != nil {
		t.Fatalf("chunk3: %v", err)
	}
	got := store.imports[imp.ID]
	if got.Status != domain.ImportStatusCompleted || got.CreatedCount != 3 {
		t.Fatalf("status=%s created=%d", got.Status, got.CreatedCount)
	}
}

func TestProcessCatalogImportBadHeader(t *testing.T) {
	t.Parallel()
	store := newMemCatalog()
	files := &memFiles{files: map[string][]byte{"k.csv": []byte("foo,bar\n1,2\n")}}
	imp := mustImport(store, "k.csv")
	cmd := NewProcessCatalogImportCommand(store, store, files, &spyQueue{})
	if err := cmd.Execute(context.Background(), contractJobs.CatalogImportRequestedJob{ImportID: imp.ID}); err != nil {
		t.Fatalf("process: %v", err)
	}
	got := store.imports[imp.ID]
	if got.Status != domain.ImportStatusFailed || !strings.Contains(got.FailureReason, "encabezado") {
		t.Fatalf("got %+v", got)
	}
}

func TestProcessCatalogImportConfirmsProvisional(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	store := newMemCatalog()
	prov, err := domain.NewProvisionalItem(id.NewULID(), "Widget", "SKU-P", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := prov.AssignInternalCode(mustCode("SKU-P"), now); err != nil {
		t.Fatal(err)
	}
	store.items[prov.ID] = prov
	files := &memFiles{files: map[string][]byte{"k.csv": []byte("internal_code,name,kind\nSKU-P,Confirmed,bien\n")}}
	imp := mustImport(store, "k.csv")
	cmd := NewProcessCatalogImportCommand(store, store, files, &spyQueue{})
	cmd.now = func() time.Time { return now }
	if err := cmd.Execute(context.Background(), contractJobs.CatalogImportRequestedJob{ImportID: imp.ID}); err != nil {
		t.Fatalf("process: %v", err)
	}
	got := store.items[prov.ID]
	if got.Status != domain.StatusConfirmed || got.Name != "Confirmed" {
		t.Fatalf("provisional: %+v", got)
	}
	if got.CreationSource != domain.CreationSourceInvoice {
		t.Fatalf("creation_source mutated: %s", got.CreationSource)
	}
}

func TestCancelCompletedImportConflicts(t *testing.T) {
	t.Parallel()
	store := newMemCatalog()
	imp, err := domain.NewCatalogImport(id.NewULID(), "k.csv", 1, testActor(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	_ = imp.Start(time.Now())
	_ = imp.Complete(1, time.Now())
	store.imports[imp.ID] = imp
	_, err = NewCancelCatalogImportCommand(store).Execute(context.Background(), imp.ID, testActor())
	var appErr *appErrors.AppError
	if err == nil || !errors.As(err, &appErr) || appErr.Code != appErrors.CodeConflict {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestPurgeStaleCatalogImports(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 15, 5, 0, 0, 0, time.UTC)
	store := newMemCatalog()
	oldImp, _ := domain.NewCatalogImport(id.NewULID(), "old.csv", 1, testActor(), now.AddDate(-2, 0, 0))
	_ = oldImp.Complete(1, now.AddDate(-2, 0, 0))
	recent, _ := domain.NewCatalogImport(id.NewULID(), "new.csv", 1, testActor(), now.AddDate(0, -1, 0))
	_ = recent.Complete(1, now)
	active, _ := domain.NewCatalogImport(id.NewULID(), "run.csv", 1, testActor(), now.AddDate(-2, 0, 0))
	store.imports[oldImp.ID] = oldImp
	store.imports[recent.ID] = recent
	store.imports[active.ID] = active
	store.errors = []domain.ImportRowError{{ID: "e1", ImportID: oldImp.ID}, {ID: "e2", ImportID: recent.ID}}
	cmd := NewPurgeStaleCatalogImportsCommand(store)
	cmd.now = func() time.Time { return now }
	deleted, err := cmd.Execute(context.Background())
	if err != nil {
		t.Fatalf("purge: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("deleted=%d", deleted)
	}
	if _, ok := store.imports[oldImp.ID]; ok {
		t.Fatal("old import should be gone")
	}
	if _, ok := store.imports[recent.ID]; !ok {
		t.Fatal("recent import should remain")
	}
	if _, ok := store.imports[active.ID]; !ok {
		t.Fatal("active import should remain")
	}
	if len(store.errors) != 1 || store.errors[0].ImportID != recent.ID {
		t.Fatalf("errors=%+v", store.errors)
	}
}

func TestPlatformPurgeStaleCatalogImportsFansOut(t *testing.T) {
	t.Parallel()
	store := newMemCatalog()
	cmd := NewPlatformPurgeStaleCatalogImportsCommand(NewPurgeStaleCatalogImportsCommand(store), fakeLister{slugs: []string{"acme", "beta"}})
	if err := cmd.Execute(context.Background()); err != nil {
		t.Fatalf("platform purge: %v", err)
	}
}

type fakeLister struct {
	slugs []string
}

func (f fakeLister) ListActiveTenantSlugs(context.Context) ([]string, error) {
	return f.slugs, nil
}

func TestPlatformPurgeUsesTenantContext(t *testing.T) {
	t.Parallel()
	seen := map[string]struct{}{}
	cmd := NewPlatformPurgeStaleCatalogImportsCommand(
		&PurgeStaleCatalogImportsCommand{
			imports: tenantSpy{seen: seen},
			now:     time.Now,
		},
		fakeLister{slugs: []string{"acme"}},
	)
	if err := cmd.Execute(context.Background()); err != nil {
		t.Fatalf("purge: %v", err)
	}
	if _, ok := seen["acme"]; !ok {
		t.Fatal("expected tenant context")
	}
}

type tenantSpy struct {
	seen map[string]struct{}
}

func (s tenantSpy) CreateImport(context.Context, domain.CatalogImport) error { return nil }
func (s tenantSpy) UpdateImport(context.Context, domain.CatalogImport) error { return nil }
func (s tenantSpy) GetImportByID(context.Context, string) (*domain.CatalogImport, error) {
	return nil, nil
}
func (s tenantSpy) GetActiveImport(context.Context) (*domain.CatalogImport, error) { return nil, nil }
func (s tenantSpy) ListImports(context.Context, ports.ImportListFilter) (ports.ImportListPage, error) {
	return ports.ImportListPage{}, nil
}
func (s tenantSpy) InsertImportErrors(context.Context, []domain.ImportRowError) error { return nil }
func (s tenantSpy) ListImportErrors(context.Context, ports.ImportErrorListFilter) (ports.ImportErrorListPage, error) {
	return ports.ImportErrorListPage{}, nil
}
func (s tenantSpy) ApplyImportChunk(context.Context, domain.CatalogImport, []domain.Item, []domain.Item, []domain.ImportRowError) error {
	return nil
}
func (s tenantSpy) PurgeStaleImports(ctx context.Context, _ time.Time, _ int) (int64, error) {
	slug, _ := tenant.TenantIDFromContext(ctx)
	s.seen[slug] = struct{}{}
	return 0, nil
}

func mustImport(store *memCatalog, key string) domain.CatalogImport {
	imp, err := domain.NewCatalogImport(id.NewULID(), key, 10, testActor(), time.Now().UTC())
	if err != nil {
		panic(err)
	}
	store.imports[imp.ID] = imp
	return imp
}

func mustKind(v string) domain.ItemKind {
	k, err := domain.ParseItemKind(v)
	if err != nil {
		panic(err)
	}
	return k
}

func mustCode(v string) domain.InternalCode {
	c, err := domain.ParseInternalCode(v)
	if err != nil {
		panic(err)
	}
	return c
}
