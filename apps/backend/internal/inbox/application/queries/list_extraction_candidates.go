package queries

import (
	"context"

	inboxapi "github.com/bowerbird/internal/inbox/api"
)

type ExtractionCandidateLister interface {
	ListExtractionCandidates(ctx context.Context, cursor string, limit int) (inboxapi.ExtractionCandidatePage, error)
}

type ListExtractionCandidatesQuery struct {
	repo ExtractionCandidateLister
}

func NewListExtractionCandidatesQuery(repo ExtractionCandidateLister) *ListExtractionCandidatesQuery {
	if repo == nil {
		panic("extraction candidate lister is required")
	}
	return &ListExtractionCandidatesQuery{repo: repo}
}

func (q *ListExtractionCandidatesQuery) Execute(ctx context.Context, cursor string, limit int) (inboxapi.ExtractionCandidatePage, error) {
	return q.repo.ListExtractionCandidates(ctx, cursor, limit)
}
