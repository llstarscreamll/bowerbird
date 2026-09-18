package v1

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/atta/internal/catalog/application/commands"
	"github.com/atta/internal/catalog/application/queries"
	appErrors "github.com/atta/internal/platform/errors"
	"github.com/atta/internal/platform/http/api"
)

type clusterMemberAttributes struct {
	Name           string            `json:"name"`
	Kind           string            `json:"kind"`
	Status         string            `json:"status"`
	CreationSource string            `json:"creation_source"`
	InternalCode   *string           `json:"internal_code"`
	CreatedAt      string            `json:"created_at"`
	UpdatedAt      string            `json:"updated_at"`
	LineCount      int               `json:"line_count"`
	Aliases        []aliasAttributes `json:"aliases,omitempty"`
}

type clusterMemberResource struct {
	Type       string                  `json:"type"`
	ID         string                  `json:"id"`
	Attributes clusterMemberAttributes `json:"attributes"`
}

type clusterAttributes struct {
	Reason  string                  `json:"reason"`
	ItemIDs []string                `json:"item_ids"`
	Items   []clusterMemberResource `json:"items"`
}

type clusterResource struct {
	Type       string            `json:"type"`
	ID         string            `json:"id"`
	Attributes clusterAttributes `json:"attributes"`
}

func (c *Controller) ListDuplicateClusters(w http.ResponseWriter, r *http.Request) error {
	clusters, err := c.app.Queries.ListDuplicateClusters.Execute(r.Context())
	if err != nil {
		return err
	}
	data := make([]clusterResource, 0, len(clusters))
	for _, cluster := range clusters {
		data = append(data, toClusterResource(cluster))
	}
	return api.Success(w, http.StatusOK, map[string]any{"data": data})
}

func (c *Controller) MergeItems(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Data struct {
			Type       string `json:"type"`
			Attributes struct {
				SurvivorID   string   `json:"survivor_id"`
				SourceIDs    []string `json:"source_ids"`
				Name         *string  `json:"name"`
				Kind         *string  `json:"kind"`
				InternalCode *string  `json:"internal_code"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return appErrors.Wrap(err, appErrors.CodeValidation, "invalid request body")
	}
	if strings.TrimSpace(req.Data.Type) != "" && req.Data.Type != "catalog_item_merges" {
		return appErrors.New(appErrors.CodeValidation, "data.type must be catalog_item_merges")
	}
	if err := c.app.Commands.MergeItems.Execute(r.Context(), commands.MergeItemsInput{
		SurvivorID:   req.Data.Attributes.SurvivorID,
		SourceIDs:    req.Data.Attributes.SourceIDs,
		Name:         req.Data.Attributes.Name,
		Kind:         req.Data.Attributes.Kind,
		InternalCode: req.Data.Attributes.InternalCode,
	}); err != nil {
		return err
	}
	detail, err := c.app.Queries.GetItemByID.Execute(r.Context(), req.Data.Attributes.SurvivorID)
	if err != nil {
		return err
	}
	return api.Success(w, http.StatusOK, map[string]any{"data": toItemDetailResource(*detail)})
}

func (c *Controller) MarkNotDuplicates(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Data struct {
			Type       string `json:"type"`
			Attributes struct {
				ItemIDs []string `json:"item_ids"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return appErrors.Wrap(err, appErrors.CodeValidation, "invalid request body")
	}
	if strings.TrimSpace(req.Data.Type) != "" && req.Data.Type != "catalog_item_not_duplicates" {
		return appErrors.New(appErrors.CodeValidation, "data.type must be catalog_item_not_duplicates")
	}
	if err := c.app.Commands.MarkNotDuplicates.Execute(r.Context(), req.Data.Attributes.ItemIDs); err != nil {
		return err
	}
	return api.Success(w, http.StatusNoContent, nil)
}

func toClusterResource(cluster queries.DuplicateCluster) clusterResource {
	items := make([]clusterMemberResource, 0, len(cluster.Items))
	for _, member := range cluster.Items {
		items = append(items, toClusterMemberResource(member))
	}
	return clusterResource{
		Type: "catalog_duplicate_clusters",
		ID:   cluster.ID,
		Attributes: clusterAttributes{
			Reason:  cluster.Reason,
			ItemIDs: cluster.ItemIDs,
			Items:   items,
		},
	}
}

func toClusterMemberResource(member queries.DuplicateClusterItem) clusterMemberResource {
	item := member.Item
	var code *string
	if parsed, ok := item.ParsedInternalCode(); ok {
		s := parsed.String()
		code = &s
	}
	aliases := make([]aliasAttributes, 0, len(member.Aliases))
	for _, alias := range member.Aliases {
		aliases = append(aliases, toAliasAttributes(alias))
	}
	attrs := clusterMemberAttributes{
		Name:           item.Name,
		Kind:           item.Kind,
		Status:         item.Status,
		CreationSource: item.CreationSource,
		InternalCode:   code,
		CreatedAt:      item.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:      item.UpdatedAt.UTC().Format(time.RFC3339),
		LineCount:      member.LineCount,
	}
	if len(aliases) > 0 {
		attrs.Aliases = aliases
	}
	return clusterMemberResource{
		Type:       "catalog_items",
		ID:         item.ID,
		Attributes: attrs,
	}
}
