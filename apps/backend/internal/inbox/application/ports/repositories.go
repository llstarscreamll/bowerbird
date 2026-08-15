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
	ToEmails         []string
	ThreadID         string
	Folder           string
	IsRead           bool
	IsStarred        bool
	IsDraft          bool
	ReceivedAt       time.Time
	ProcessingStatus string
	HasXML           bool
	HasPDF           bool
}

type AttachmentView struct {
	ID        string
	Filename  string
	MimeType  string
	SizeBytes int64
}

type MessageDetailView struct {
	ID               string
	Provider         string
	AccountID        string
	AccountEmail     string
	Subject          string
	Sender           string
	Snippet          string
	ToEmails         []string
	CcEmails         []string
	BccEmails        []string
	ThreadID         string
	Folder           string
	IsRead           bool
	IsStarred        bool
	IsDraft          bool
	BodyText         string
	RawData          []byte
	ReceivedAt       time.Time
	ProcessingStatus string
	HasXML           bool
	HasPDF           bool
	Attachments      []AttachmentView
}

type ListMessagesFilter struct {
	ViewerUserID string
	AccountID    string
	Folder       string
	Query        string
	Limit        int
	Offset       int
	OnlyInvoices bool
}

type MessageListPage struct {
	Items  []MessageListView
	Total  int
	Limit  int
	Offset int
}

type MessageQueryRepository interface {
	ListMessageViews(ctx context.Context, filter ListMessagesFilter) (MessageListPage, error)
	GetMessageViewByID(ctx context.Context, messageID, viewerUserID string) (*MessageDetailView, error)
}
