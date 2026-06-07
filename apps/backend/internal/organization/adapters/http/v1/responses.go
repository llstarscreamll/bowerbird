package v1

import "github.com/bowerbird/internal/organization/domain"

type organizationResponse struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Slug            string `json:"slug"`
	Status          string `json:"status"`
	CreatedAt       string `json:"created_at"`
	MembersCount    int    `json:"members_count,omitempty"`
	CurrentUserRole string `json:"current_user_role,omitempty"`
}

func newOrganizationResponse(org *domain.Organization) organizationResponse {
	return organizationResponse{
		ID:              org.ID,
		Name:            org.Name,
		Slug:            org.Slug,
		Status:          org.Status,
		CreatedAt:       org.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		MembersCount:    org.MembersCount,
		CurrentUserRole: org.CurrentUserRole,
	}
}
