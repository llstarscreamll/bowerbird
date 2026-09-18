package domain

import "testing"

func TestFlagsFromProviderLabels(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		labels []string
		want   MessageFlags
	}{
		{
			name:   "unread inbox",
			labels: []string{"INBOX", "UNREAD"},
			want:   MessageFlags{Folder: MailFolderInbox, IsRead: false},
		},
		{
			name:   "starred inbox",
			labels: []string{"INBOX", "STARRED"},
			want:   MessageFlags{Folder: MailFolderInbox, IsRead: true, IsStarred: true},
		},
		{
			name:   "trash wins over inbox",
			labels: []string{"INBOX", "TRASH"},
			want:   MessageFlags{Folder: MailFolderTrash, IsRead: true},
		},
		{
			name:   "sent",
			labels: []string{"SENT"},
			want:   MessageFlags{Folder: MailFolderSent, IsRead: true},
		},
		{
			name:   "archive",
			labels: []string{"IMPORTANT"},
			want:   MessageFlags{Folder: MailFolderArchive, IsRead: true},
		},
		{
			name:   "draft",
			labels: []string{"DRAFT"},
			want:   MessageFlags{Folder: MailFolderDrafts, IsRead: true, IsDraft: true},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := FlagsFromProviderLabels(tc.labels)
			if got != tc.want {
				t.Fatalf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestParseAddressList(t *testing.T) {
	t.Parallel()

	got := ParseAddressList(`Ada Lovelace <ada@example.com>, "Babbage, Charles" <charles@example.com>`)
	if len(got) != 2 {
		t.Fatalf("expected 2 addresses, got %#v", got)
	}
	if got[0] != "Ada Lovelace <ada@example.com>" {
		t.Fatalf("unexpected first address %q", got[0])
	}
}
