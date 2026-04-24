package parser

import (
	"testing"

	"chat-trace-viewer/internal/domain"
)

func TestParseSubject(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		subject string
		client  string
		project string
		userID  string
		session string
	}{
		{
			name:    "chat subject",
			subject: "csquad.csscat.61dfe3428df3572ee111ecb01c28c621.voazwkbpyvazvme4pq2ulj-sz7a.robot.widget.command.finished",
			client:  "csquad",
			project: "csscat",
			userID:  "61dfe3428df3572ee111ecb01c28c621",
			session: "voazwkbpyvazvme4pq2ulj-sz7a",
		},
		{
			name:    "legacy",
			subject: "msg.user.web.csscat.61dfe3428df3572ee111ecb01c28c621.voazwkbpyvazvme4pq2ulj-sz7a",
			project: "csscat",
			userID:  "61dfe3428df3572ee111ecb01c28c621",
			session: "voazwkbpyvazvme4pq2ulj-sz7a",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			info, ok := ParseSubject(tc.subject)
			if !ok {
				t.Fatalf("expected subject to parse")
			}
			if info.Chat.Client != tc.client || info.Chat.Project != tc.project || info.Chat.UserID != tc.userID || info.Chat.SessionID != tc.session {
				t.Fatalf("unexpected identity: %#v", info.Chat)
			}
		})
	}
}

func TestFindIdentityDoesNotFallbackToQuery(t *testing.T) {
	t.Parallel()

	chat, source, confidence, description, verified := FindIdentity(
		"plain log line without chat identity",
		nil,
		nil,
		domain.TraceQuery{
			UserID:    "61dfe3428df3572ee111ecb01c28c621",
			SessionID: "voazwkbpyvazvme4pq2ulj-sz7a",
			Project:   "csscat",
			Client:    "csquad",
		},
	)

	if verified {
		t.Fatal("expected identity to be unverified")
	}
	if source != "none" {
		t.Fatalf("unexpected source: %q", source)
	}
	if confidence != 0 {
		t.Fatalf("unexpected confidence: %v", confidence)
	}
	if description != "identity not found" {
		t.Fatalf("unexpected description: %q", description)
	}
	if chat.UserID != "" || chat.SessionID != "" {
		t.Fatalf("identity must not be populated from query: %#v", chat)
	}
	if chat.Client != "csquad" || chat.Project != "csscat" {
		t.Fatalf("expected only non-critical query fields to be preserved, got %#v", chat)
	}
}
