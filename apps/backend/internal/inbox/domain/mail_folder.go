package domain

type MailFolder string

const (
	MailFolderInbox   MailFolder = "inbox"
	MailFolderSent    MailFolder = "sent"
	MailFolderDrafts  MailFolder = "drafts"
	MailFolderTrash   MailFolder = "trash"
	MailFolderSpam    MailFolder = "spam"
	MailFolderArchive MailFolder = "archive"
	MailFolderStarred MailFolder = "starred"
)

func (f MailFolder) IsValid() bool {
	switch f {
	case MailFolderInbox, MailFolderSent, MailFolderDrafts, MailFolderTrash, MailFolderSpam, MailFolderArchive, MailFolderStarred:
		return true
	default:
		return false
	}
}

func (f MailFolder) String() string {
	if f == "" {
		return string(MailFolderInbox)
	}
	return string(f)
}

type MessageFlags struct {
	Folder    MailFolder
	IsRead    bool
	IsStarred bool
	IsDraft   bool
}

func FlagsFromProviderLabels(labelIDs []string) MessageFlags {
	has := make(map[string]struct{}, len(labelIDs))
	for _, id := range labelIDs {
		has[id] = struct{}{}
	}

	contains := func(id string) bool {
		_, ok := has[id]
		return ok
	}

	flags := MessageFlags{
		Folder:    MailFolderArchive,
		IsRead:    !contains("UNREAD"),
		IsStarred: contains("STARRED"),
		IsDraft:   contains("DRAFT"),
	}

	switch {
	case contains("TRASH"):
		flags.Folder = MailFolderTrash
	case contains("SPAM"):
		flags.Folder = MailFolderSpam
	case contains("DRAFT"):
		flags.Folder = MailFolderDrafts
	case contains("SENT") && !contains("INBOX"):
		flags.Folder = MailFolderSent
	case contains("INBOX"):
		flags.Folder = MailFolderInbox
	}

	return flags
}

func ParseAddressList(header string) []string {
	if header == "" {
		return nil
	}

	parts := splitAddressHeader(header)
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func splitAddressHeader(header string) []string {
	var parts []string
	current := make([]rune, 0, len(header))
	inQuotes := false
	for _, r := range header {
		switch {
		case r == '"':
			inQuotes = !inQuotes
			current = append(current, r)
		case r == ',' && !inQuotes:
			parts = append(parts, trimAddress(string(current)))
			current = current[:0]
		default:
			current = append(current, r)
		}
	}
	if len(current) > 0 {
		parts = append(parts, trimAddress(string(current)))
	}
	return parts
}

func trimAddress(value string) string {
	start := 0
	end := len(value)
	for start < end && (value[start] == ' ' || value[start] == '\t') {
		start++
	}
	for end > start && (value[end-1] == ' ' || value[end-1] == '\t') {
		end--
	}
	return value[start:end]
}
