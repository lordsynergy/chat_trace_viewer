package normalizer

import (
	"testing"
	"time"

	"chat-trace-viewer/internal/domain"
)

func TestNormalizeRules(t *testing.T) {
	t.Parallel()

	n := New()
	query := domain.TraceQuery{
		UserID:    "61dfe3428df3572ee111ecb01c28c621",
		SessionID: "voazwkbpyvazvme4pq2ulj-sz7a",
		Project:   "csscat",
		Client:    "csquad",
	}

	entry := domain.RawLogEntry{
		Timestamp: time.Now(),
		Level:     "warn",
		Service:   "connector",
		Message:   "Session terminated due to timeouts spam web.csscat.61dfe3428df3572ee111ecb01c28c621.voazwkbpyvazvme4pq2ulj-sz7a",
	}

	event := n.Normalize(0, entry, query)
	if event.EventType != domain.EventTypeWarn {
		t.Fatalf("expected warn event, got %q", event.EventType)
	}
	if event.Reason != domain.ReasonTimeoutSpam {
		t.Fatalf("expected timeout spam reason, got %q", event.Reason)
	}
	if event.MessageKind != domain.MessageKindSystem {
		t.Fatalf("expected system message kind, got %q", event.MessageKind)
	}
}

func TestNormalizeContentMessageKind(t *testing.T) {
	t.Parallel()

	n := New()
	query := domain.TraceQuery{
		UserID:    "61dfe3428df3572ee111ecb01c28c621",
		SessionID: "voazwkbpyvazvme4pq2ulj-sz7a",
		Project:   "csscat",
		Client:    "csquad",
	}

	entry := domain.RawLogEntry{
		Timestamp:  time.Now(),
		Level:      "debug",
		Service:    "transformer",
		Message:    `Received message: msg.user.web.csscat.61dfe3428df3572ee111ecb01c28c621.voazwkbpyvazvme4pq2ulj-sz7a => {"id":"1","side":"robot","text":"Привет"}`,
		Subjects:   []string{"msg.user.web.csscat.61dfe3428df3572ee111ecb01c28c621.voazwkbpyvazvme4pq2ulj-sz7a"},
		RawMessage: `{"id":"1","side":"robot","text":"Привет"}`,
	}

	event := n.Normalize(0, entry, query)
	if event.MessageKind != domain.MessageKindContent {
		t.Fatalf("expected content message kind, got %q", event.MessageKind)
	}
}

func TestNormalizeCommandMessageKindAndRoute(t *testing.T) {
	t.Parallel()

	n := New()
	query := domain.TraceQuery{
		UserID:    "f6fc7a1bcd7da6de6d302822a58af9d9",
		SessionID: "prbwcmnsonfyxa3zeyke5p-l6aq",
		Project:   "csscat",
		Client:    "csquad",
	}

	entry := domain.RawLogEntry{
		Timestamp: time.Now(),
		Level:     "debug",
		Service:   "transformer",
		Message:   `Published message for subject: msg.user.web.csscat.f6fc7a1bcd7da6de6d302822a58af9d9.prbwcmnsonfyxa3zeyke5p-l6aq, message: #<NATS::Msg:0x00007f77d8b940b8>`,
		Subjects:  []string{"msg.user.web.csscat.f6fc7a1bcd7da6de6d302822a58af9d9.prbwcmnsonfyxa3zeyke5p-l6aq"},
	}

	event := n.Normalize(0, entry, query)
	if event.MessageKind != domain.MessageKindCommand {
		t.Fatalf("expected command message kind, got %q", event.MessageKind)
	}
	if event.From != "user" || event.To != "web" {
		t.Fatalf("expected route user -> web, got %q -> %q", event.From, event.To)
	}
}

func TestNormalizeTimeoutCommandIsSystemMessage(t *testing.T) {
	t.Parallel()

	n := New()
	query := domain.TraceQuery{
		UserID:    "61dfe3428df3572ee111ecb01c28c621",
		SessionID: "voazwkbpyvazvme4pq2ulj-sz7a",
		Project:   "csscat",
		Client:    "csquad",
	}

	entry := domain.RawLogEntry{
		Timestamp:  time.Now(),
		Level:      "debug",
		Service:    "transformer",
		Message:    `Received message: timeout.nlu.web.csscat.61dfe3428df3572ee111ecb01c28c621.voazwkbpyvazvme4pq2ulj-sz7a => {"id":"1","side":"robot","timeout_id":"abc"}`,
		Subjects:   []string{"timeout.nlu.web.csscat.61dfe3428df3572ee111ecb01c28c621.voazwkbpyvazvme4pq2ulj-sz7a"},
		RawMessage: `{"id":"1","side":"robot","timeout_id":"abc"}`,
	}

	event := n.Normalize(0, entry, query)
	if event.MessageKind != domain.MessageKindSystem {
		t.Fatalf("expected system message kind, got %q", event.MessageKind)
	}
}
