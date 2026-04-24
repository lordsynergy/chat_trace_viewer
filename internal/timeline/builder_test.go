package timeline

import (
	"testing"
	"time"

	"chat-trace-viewer/internal/domain"
)

func TestBuildSkipsUnverifiedEvents(t *testing.T) {
	t.Parallel()

	b := New()
	now := time.Now()
	query := domain.TraceQuery{SessionID: "session-1"}

	events := []domain.NormalizedEvent{
		{
			ID:               "event-1",
			Timestamp:        now,
			Service:          "svc-a",
			Level:            "info",
			EventType:        domain.EventTypeInfo,
			Description:      "verified",
			IdentityVerified: true,
			Chat: domain.ChatIdentity{
				SessionID: "session-1",
				ChatKey:   "client.project.user.session-1",
			},
		},
		{
			ID:               "event-2",
			Timestamp:        now.Add(time.Second),
			Service:          "svc-b",
			Level:            "warn",
			EventType:        domain.EventTypeWarn,
			Description:      "unverified",
			IdentityVerified: false,
			Chat: domain.ChatIdentity{
				SessionID: "session-1",
			},
		},
	}

	timeline, anomalies, summary := b.Build(query, events)
	if len(timeline) != 1 {
		t.Fatalf("expected only verified event in timeline, got %d", len(timeline))
	}
	if len(anomalies) != 0 {
		t.Fatalf("expected no anomalies, got %d", len(anomalies))
	}
	if summary.EventsCount != 1 {
		t.Fatalf("expected summary to count only verified events, got %d", summary.EventsCount)
	}
}

func TestBuildOnlyAnomaliesKeepsSummaryFromFullTimeline(t *testing.T) {
	t.Parallel()

	b := New()
	now := time.Now()
	query := domain.TraceQuery{SessionID: "session-1", OnlyAnomalies: true}

	events := []domain.NormalizedEvent{
		{
			ID:               "event-1",
			Timestamp:        now,
			Service:          "svc-a",
			Level:            "info",
			EventType:        domain.EventTypeReceived,
			Description:      "received",
			IdentityVerified: true,
			Chat:             domain.ChatIdentity{SessionID: "session-1", ChatKey: "chat-key"},
		},
		{
			ID:               "event-2",
			Timestamp:        now.Add(time.Second),
			Service:          "svc-b",
			Level:            "warn",
			EventType:        domain.EventTypeWarn,
			Reason:           domain.ReasonTimeoutSpam,
			Description:      "warn",
			IdentityVerified: true,
			Chat:             domain.ChatIdentity{SessionID: "session-1", ChatKey: "chat-key"},
		},
	}

	timeline, anomalies, summary := b.Build(query, events)
	if len(timeline) != 1 {
		t.Fatalf("expected only anomalies in timeline, got %d", len(timeline))
	}
	if len(anomalies) != 1 {
		t.Fatalf("expected one anomaly, got %d", len(anomalies))
	}
	if summary.EventsCount != 2 {
		t.Fatalf("expected summary to be built from full verified timeline, got %d", summary.EventsCount)
	}
}
