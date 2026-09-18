package domain

import (
	"errors"
	"time"
)

var (
	ErrEntitlementNotFound = errors.New("entitlement not found")
	ErrUnknownFeature      = errors.New("unknown feature")
	ErrUnknownProduct      = errors.New("unknown product")
)

type Entitlement struct {
	ID         string
	TenantID   string
	FeatureKey string
	Status     string
	Source     string
	StartsAt   time.Time
	EndsAt     *time.Time
	CreatedBy  string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (e Entitlement) IsEffective(at time.Time) bool {
	if e.Status != StatusActive && e.Status != StatusTrial {
		return false
	}
	if at.Before(e.StartsAt) {
		return false
	}
	if e.EndsAt != nil && !at.Before(*e.EndsAt) {
		return false
	}
	return true
}
