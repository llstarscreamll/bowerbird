package domain

import "time"

type SyncCursorStatus string

const (
	SyncCursorStatusSyncing SyncCursorStatus = "syncing"
	SyncCursorStatusIdle    SyncCursorStatus = "idle"
	SyncCursorStatusError   SyncCursorStatus = "error"
)

type SyncCursor struct {
	ConnectionID string
	LastSyncedAt *time.Time
	LastError    *string
	Status       SyncCursorStatus
}

func NewSyncCursor(connectionID string, initialSyncedAt *time.Time) (*SyncCursor, error) {
	if connectionID == "" {
		return nil, ErrSyncCursorConnectionIDRequired
	}

	cursor := &SyncCursor{
		ConnectionID: connectionID,
		Status:       SyncCursorStatusIdle,
	}

	if initialSyncedAt != nil {
		t := initialSyncedAt.UTC()
		cursor.LastSyncedAt = &t
	}

	return cursor, nil
}

func (s SyncCursorStatus) String() string {
	return string(s)
}

func (s SyncCursorStatus) IsValid() bool {
	switch s {
	case SyncCursorStatusSyncing, SyncCursorStatusIdle, SyncCursorStatusError:
		return true
	default:
		return false
	}
}

func (c *SyncCursor) MarkSyncing() {
	c.Status = SyncCursorStatusSyncing
}

func (c *SyncCursor) MarkSyncFailed(failure string) {
	c.Status = SyncCursorStatusError
	c.LastError = &failure
}

func (c *SyncCursor) MarkSyncSucceeded(at time.Time) {
	c.Status = SyncCursorStatusIdle
	c.LastError = nil
	syncedAt := at.UTC()
	c.LastSyncedAt = &syncedAt
}
