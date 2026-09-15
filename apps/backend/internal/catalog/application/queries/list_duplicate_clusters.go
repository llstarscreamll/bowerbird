package queries

import (
	"context"
	"sort"
	"strings"

	"github.com/bowerbird/internal/catalog/application/ports"
	"github.com/bowerbird/internal/catalog/domain"
)

const (
	ClusterReasonDescription = "normalized_description"
	ClusterReasonSKU         = "cross_party_sku"
	ClusterReasonConflict    = "hard_conflict"
)

type DuplicateClusterItem struct {
	Item      domain.Item
	Aliases   []domain.Alias
	LineCount int
}

type DuplicateCluster struct {
	ID      string
	Reason  string
	Items   []DuplicateClusterItem
	ItemIDs []string
}

type ListDuplicateClustersQuery struct {
	items   ports.ItemRepository
	index   ports.DuplicateIndex
	aliases ports.AliasRepository
	pairs   ports.NotDuplicateRepository
	links   ports.ItemLinkSupport
}

func NewListDuplicateClustersQuery(
	items ports.ItemRepository,
	index ports.DuplicateIndex,
	aliases ports.AliasRepository,
	pairs ports.NotDuplicateRepository,
	links ports.ItemLinkSupport,
) *ListDuplicateClustersQuery {
	if items == nil {
		panic("item repository is required")
	}
	if index == nil {
		panic("duplicate index is required")
	}
	if aliases == nil {
		panic("alias repository is required")
	}
	if pairs == nil {
		panic("not-duplicate repository is required")
	}
	return &ListDuplicateClustersQuery{items: items, index: index, aliases: aliases, pairs: pairs, links: links}
}

func (q *ListDuplicateClustersQuery) BindLinks(links ports.ItemLinkSupport) {
	q.links = links
}

func (q *ListDuplicateClustersQuery) Execute(ctx context.Context) ([]DuplicateCluster, error) {
	excluded, err := q.pairs.ListNotDuplicatePairs(ctx)
	if err != nil {
		return nil, err
	}
	skip := map[string]struct{}{}
	for _, pair := range excluded {
		skip[pair.Left+"|"+pair.Right] = struct{}{}
	}

	uf := newUnionFind()
	reason := map[string]string{}
	addGroup := func(items []domain.Item, why string) {
		ids := make([]string, 0, len(items))
		for _, item := range items {
			if item.IsMerged() {
				continue
			}
			ids = append(ids, item.ID)
			uf.add(item.ID)
		}
		for i := 0; i < len(ids); i++ {
			for j := i + 1; j < len(ids); j++ {
				lo, hi := ids[i], ids[j]
				if lo > hi {
					lo, hi = hi, lo
				}
				if _, ok := skip[lo+"|"+hi]; ok {
					continue
				}
				uf.union(ids[i], ids[j])
				root := uf.find(ids[i])
				reason[root] = strongerReason(reason[root], why)
			}
		}
	}

	desc, err := q.index.DescriptionDuplicateGroups(ctx)
	if err != nil {
		return nil, err
	}
	for _, group := range desc {
		addGroup(group, ClusterReasonDescription)
	}
	sku, err := q.index.CrossPartySKUGroups(ctx)
	if err != nil {
		return nil, err
	}
	for _, group := range sku {
		addGroup(group, ClusterReasonSKU)
	}
	if q.links != nil {
		if pairs, err := q.links.HardConflictPairs(ctx); err == nil {
			for _, pair := range pairs {
				addGroup([]domain.Item{{ID: pair.Left}, {ID: pair.Right}}, ClusterReasonConflict)
			}
		}
	}

	groups := uf.groups()
	out := make([]DuplicateCluster, 0, len(groups))
	allIDs := make([]string, 0)
	for _, ids := range groups {
		if len(ids) < 2 {
			continue
		}
		allIDs = append(allIDs, ids...)
	}
	counts := map[string]int{}
	if q.links != nil && len(allIDs) > 0 {
		if n, err := q.links.CountLinks(ctx, allIDs); err == nil {
			counts = n
		}
	}

	for _, ids := range groups {
		if len(ids) < 2 {
			continue
		}
		sort.Strings(ids)
		loaded, err := q.items.GetItemsByIDs(ctx, ids)
		if err != nil {
			return nil, err
		}
		byID := map[string]domain.Item{}
		for _, item := range loaded {
			if item.IsMerged() {
				continue
			}
			byID[item.ID] = item
		}
		items := make([]DuplicateClusterItem, 0, len(ids))
		itemIDs := make([]string, 0, len(ids))
		for _, id := range ids {
			item, ok := byID[id]
			if !ok {
				continue
			}
			aliases, err := q.aliases.ListAliasesByItemID(ctx, id)
			if err != nil {
				return nil, err
			}
			items = append(items, DuplicateClusterItem{Item: item, Aliases: aliases, LineCount: counts[id]})
			itemIDs = append(itemIDs, id)
		}
		if len(items) < 2 {
			continue
		}
		root := uf.find(ids[0])
		why := reason[root]
		if why == "" {
			why = ClusterReasonDescription
		}
		out = append(out, DuplicateCluster{
			ID:      strings.Join(itemIDs, ":"),
			Reason:  why,
			Items:   items,
			ItemIDs: itemIDs,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func strongerReason(current, next string) string {
	rank := map[string]int{
		"":                       0,
		ClusterReasonDescription: 1,
		ClusterReasonSKU:         2,
		ClusterReasonConflict:    3,
	}
	if rank[next] > rank[current] {
		return next
	}
	return current
}

type unionFind struct {
	parent map[string]string
}

func newUnionFind() *unionFind {
	return &unionFind{parent: map[string]string{}}
}

func (u *unionFind) add(id string) {
	if _, ok := u.parent[id]; !ok {
		u.parent[id] = id
	}
}

func (u *unionFind) find(id string) string {
	u.add(id)
	if u.parent[id] != id {
		u.parent[id] = u.find(u.parent[id])
	}
	return u.parent[id]
}

func (u *unionFind) union(a, b string) {
	ra, rb := u.find(a), u.find(b)
	if ra != rb {
		u.parent[rb] = ra
	}
}

func (u *unionFind) groups() [][]string {
	bucket := map[string][]string{}
	for id := range u.parent {
		root := u.find(id)
		bucket[root] = append(bucket[root], id)
	}
	out := make([][]string, 0, len(bucket))
	for _, ids := range bucket {
		out = append(out, ids)
	}
	return out
}
