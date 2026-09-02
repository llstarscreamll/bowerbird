package domain

import (
	"strings"
	"time"
)

type SyncCursorStatus string

const (
	SyncCursorStatusSyncing SyncCursorStatus = "syncing"
	SyncCursorStatusIdle    SyncCursorStatus = "idle"
	SyncCursorStatusError   SyncCursorStatus = "error"
)

type SyncCursor struct {
	connectionID string
	lastSyncedAt *time.Time
	historyID    *string
	lastError    *string
	status       SyncCursorStatus
}

type SyncCursorSnapshot struct {
	ConnectionID string
	LastSyncedAt *time.Time
	HistoryID    *string
	LastError    *string
	Status       SyncCursorStatus
}

func NewSyncCursor(connectionID string, initialSyncedAt *time.Time) (*SyncCursor, error) {
	if connectionID == "" {
		return nil, ErrSyncCursorConnectionIDRequired
	}

	cursor := &SyncCursor{
		connectionID: connectionID,
		status:       SyncCursorStatusIdle,
	}

	if initialSyncedAt != nil {
		t := initialSyncedAt.UTC()
		cursor.lastSyncedAt = &t
	}

	return cursor, nil
}

func RehydrateSyncCursor(snapshot SyncCursorSnapshot) *SyncCursor {
	return &SyncCursor{
		connectionID: snapshot.ConnectionID,
		lastSyncedAt: snapshot.LastSyncedAt,
		historyID:    snapshot.HistoryID,
		lastError:    snapshot.LastError,
		status:       snapshot.Status,
	}
}

func (c *SyncCursor) ConnectionID() string     { return c.connectionID }
func (c *SyncCursor) LastSyncedAt() *time.Time { return c.lastSyncedAt }
func (c *SyncCursor) HistoryID() string        { return derefString(c.historyID) }
func (c *SyncCursor) LastError() *string       { return c.lastError }
func (c *SyncCursor) Status() SyncCursorStatus { return c.status }

func (c *SyncCursor) Snapshot() SyncCursorSnapshot {
	return SyncCursorSnapshot{
		ConnectionID: c.connectionID,
		LastSyncedAt: c.lastSyncedAt,
		HistoryID:    c.historyID,
		LastError:    c.lastError,
		Status:       c.status,
	}
}

func (s SyncCursorStatus) String() string { return string(s) }

func (s SyncCursorStatus) IsValid() bool {
	switch s {
	case SyncCursorStatusSyncing, SyncCursorStatusIdle, SyncCursorStatusError:
		return true
	default:
		return false
	}
}

func (c *SyncCursor) MarkSyncing() {
	c.status = SyncCursorStatusSyncing
}

func (c *SyncCursor) MarkSyncFailed(failure string) {
	c.status = SyncCursorStatusError
	c.lastError = &failure
}

func (c *SyncCursor) MarkSyncSucceeded(at time.Time) {
	c.status = SyncCursorStatusIdle
	c.lastError = nil
	syncedAt := at.UTC()
	c.lastSyncedAt = &syncedAt
}

func (c *SyncCursor) AdvanceHistory(historyID string) error {
	if strings.TrimSpace(historyID) == "" {
		return ErrSyncCursorHistoryIDRequired
	}
	id := historyID
	c.historyID = &id
	return nil
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
