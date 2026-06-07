package ports

import (
	"context"
	"time"
)

type MessageListView struct {
	ID               string
	Provider         string
	AccountID        string
	AccountEmail     string
	Subject          string
	Sender           string
	Snippet          string
	ReceivedAt       time.Time
	ProcessingStatus string
	HasXML           bool
	HasPDF           bool
}

type MessageDetailView struct {
	ID               string
	Provider         string
	AccountID        string
	AccountEmail     string
	Subject          string
	Sender           string
	Snippet          string
	BodyText         string
	RawData          []byte
	ReceivedAt       time.Time
	ProcessingStatus string
	HasXML           bool
	HasPDF           bool
}

type MessageQueryRepository interface {
	ListMessageViews(ctx context.Context) ([]MessageListView, error)
	GetMessageViewByID(ctx context.Context, messageID string) (*MessageDetailView, error)
}
