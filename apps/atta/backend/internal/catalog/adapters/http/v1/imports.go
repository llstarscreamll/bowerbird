package v1

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/atta/internal/catalog/application/commands"
	"github.com/atta/internal/catalog/application/ports"
	"github.com/atta/internal/catalog/domain"
	"github.com/atta/internal/platform/auth"
	appErrors "github.com/atta/internal/platform/errors"
	"github.com/atta/internal/platform/http/api"
)

func actorFromRequest(r *http.Request) (domain.ImportActor, error) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		return domain.ImportActor{}, appErrors.New(appErrors.CodeUnauthorized, "unauthorized")
	}
	actor, err := domain.NewImportActor(claims.UserID, claims.Email, claims.FirstName, claims.LastName)
	if err != nil {
		return domain.ImportActor{}, appErrors.New(appErrors.CodeValidation, "authenticated user snapshot is required")
	}
	return actor, nil
}

func (c *Controller) DownloadImportTemplate(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="catalog-import-template.csv"`)
	_, err := w.Write([]byte(domain.ImportTemplateCSV()))
	return err
}

func (c *Controller) CreateImport(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorFromRequest(r)
	if err != nil {
		return err
	}
	var req struct {
		Data struct {
			Type       string `json:"type"`
			ID         string `json:"id"`
			Attributes struct {
				FileKey string `json:"file_key"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return appErrors.Wrap(err, appErrors.CodeValidation, "invalid request body")
	}
	if strings.TrimSpace(req.Data.Type) != "" && req.Data.Type != "catalog_imports" {
		return appErrors.New(appErrors.CodeValidation, "data.type must be catalog_imports")
	}
	imp, err := c.app.Commands.QueueCatalogImport.Execute(r.Context(), commands.QueueCatalogImportInput{
		ID:        req.Data.ID,
		FileKey:   req.Data.Attributes.FileKey,
		Requester: actor,
	})
	if err != nil {
		return err
	}
	return api.Success(w, http.StatusAccepted, map[string]any{"data": toImportResource(*imp)})
}

func (c *Controller) ListImports(w http.ResponseWriter, r *http.Request) error {
	limit := pageSize(r, 20, 50)
	afterCreated, afterID := decodeImportCursor(pageAfter(r))
	page, err := c.app.Queries.ListImports.Execute(r.Context(), ports.ImportListFilter{
		Limit:        limit,
		AfterCreated: afterCreated,
		AfterID:      afterID,
	})
	if err != nil {
		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to list catalog imports")
	}
	data := make([]importResource, 0, len(page.Items))
	for _, imp := range page.Items {
		data = append(data, toImportResource(imp))
	}
	cursor := ""
	if page.HasMore && len(page.Items) > 0 {
		last := page.Items[len(page.Items)-1]
		cursor = encodeCursor(last.CreatedAt.UTC().Format(time.RFC3339Nano), last.ID)
	}
	return api.Success(w, http.StatusOK, map[string]any{"data": data, "meta": pageMeta(page.HasMore, cursor, nil)})
}

func (c *Controller) GetImport(w http.ResponseWriter, r *http.Request) error {
	imp, err := c.app.Queries.GetImportByID.Execute(r.Context(), r.PathValue("id"))
	if err != nil {
		return err
	}
	return api.Success(w, http.StatusOK, map[string]any{"data": toImportResource(*imp)})
}

func (c *Controller) CancelImport(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorFromRequest(r)
	if err != nil {
		return err
	}
	imp, err := c.app.Commands.CancelCatalogImport.Execute(r.Context(), r.PathValue("id"), actor)
	if err != nil {
		return err
	}
	return api.Success(w, http.StatusOK, map[string]any{"data": toImportResource(*imp)})
}

func (c *Controller) ListImportErrors(w http.ResponseWriter, r *http.Request) error {
	limit := pageSize(r, 50, 100)
	afterRow, afterID := decodeErrorCursor(pageAfter(r))
	page, err := c.app.Queries.ListImportErrors.Execute(r.Context(), ports.ImportErrorListFilter{
		ImportID:     r.PathValue("id"),
		Limit:        limit,
		AfterFileRow: afterRow,
		AfterID:      afterID,
	})
	if err != nil {
		return err
	}
	data := make([]importErrorResource, 0, len(page.Items))
	for _, row := range page.Items {
		data = append(data, toImportErrorResource(row))
	}
	cursor := ""
	if page.HasMore && len(page.Items) > 0 {
		last := page.Items[len(page.Items)-1]
		cursor = encodeCursor(strconv.Itoa(last.FileRow), last.ID)
	}
	return api.Success(w, http.StatusOK, map[string]any{
		"data": data,
		"meta": pageMeta(page.HasMore, cursor, map[string]any{"total": page.Total}),
	})
}

type importActorAttrs struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
}

type importAttributes struct {
	FileKey       string            `json:"file_key"`
	FileSizeBytes int64             `json:"file_size_bytes"`
	Status        string            `json:"status"`
	TotalRows     int64             `json:"total_rows"`
	CreatedCount  int64             `json:"created_count"`
	UpdatedCount  int64             `json:"updated_count"`
	FailedCount   int64             `json:"failed_count"`
	Processed     int64             `json:"processed"`
	ByteOffset    int64             `json:"byte_offset"`
	FailureReason string            `json:"failure_reason,omitempty"`
	RequestedBy   importActorAttrs  `json:"requested_by"`
	CancelledBy   *importActorAttrs `json:"cancelled_by"`
	CreatedAt     string            `json:"created_at"`
	UpdatedAt     string            `json:"updated_at"`
	CompletedAt   *string           `json:"completed_at"`
	CancelledAt   *string           `json:"cancelled_at"`
}

type importResource struct {
	Type       string           `json:"type"`
	ID         string           `json:"id"`
	Attributes importAttributes `json:"attributes"`
}

type importErrorAttributes struct {
	FileRow      int    `json:"file_row"`
	Column       string `json:"column,omitempty"`
	InternalCode string `json:"internal_code,omitempty"`
	Name         string `json:"name,omitempty"`
	Kind         string `json:"kind,omitempty"`
	Code         string `json:"code"`
	Message      string `json:"message"`
}

type importErrorResource struct {
	Type       string                `json:"type"`
	ID         string                `json:"id"`
	Attributes importErrorAttributes `json:"attributes"`
}

func toImportResource(imp domain.CatalogImport) importResource {
	var cancelled *importActorAttrs
	if imp.CancelledBy != nil {
		cancelled = &importActorAttrs{UserID: imp.CancelledBy.UserID, Email: imp.CancelledBy.Email, Name: imp.CancelledBy.Name}
	}
	return importResource{
		Type: "catalog_imports",
		ID:   imp.ID,
		Attributes: importAttributes{
			FileKey:       imp.FileKey,
			FileSizeBytes: imp.FileSizeBytes,
			Status:        imp.Status,
			TotalRows:     imp.TotalRows,
			CreatedCount:  imp.CreatedCount,
			UpdatedCount:  imp.UpdatedCount,
			FailedCount:   imp.FailedCount,
			Processed:     imp.Processed(),
			ByteOffset:    imp.ByteOffset,
			FailureReason: imp.FailureReason,
			RequestedBy:   importActorAttrs{UserID: imp.RequestedBy.UserID, Email: imp.RequestedBy.Email, Name: imp.RequestedBy.Name},
			CancelledBy:   cancelled,
			CreatedAt:     imp.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt:     imp.UpdatedAt.UTC().Format(time.RFC3339),
			CompletedAt:   formatTimePtr(imp.CompletedAt),
			CancelledAt:   formatTimePtr(imp.CancelledAt),
		},
	}
}

func toImportErrorResource(row domain.ImportRowError) importErrorResource {
	return importErrorResource{
		Type: "catalog_import_errors",
		ID:   row.ID,
		Attributes: importErrorAttributes{
			FileRow:      row.FileRow,
			Column:       row.Column,
			InternalCode: row.InternalCode,
			Name:         row.Name,
			Kind:         row.Kind,
			Code:         row.Code,
			Message:      row.Message,
		},
	}
}

func formatTimePtr(t time.Time) *string {
	if t.IsZero() {
		return nil
	}
	s := t.UTC().Format(time.RFC3339)
	return &s
}
