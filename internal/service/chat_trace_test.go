package service

import (
	"context"
	"testing"

	"chat-trace-viewer/internal/config"
	"chat-trace-viewer/internal/domain"
	"chat-trace-viewer/internal/victorialogs"
)

func TestBuildChatTrace(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		DefaultLookback: 0,
		MaxLogLines:     500,
		MaxRawLines:     500,
	}
	svc := NewChatTraceService(cfg, victorialogs.NewSampleClient("../../testdata/sample_victorialogs.json"))
	trace, err := svc.BuildChatTrace(context.Background(), domain.TraceQuery{
		UserID:             "61dfe3428df3572ee111ecb01c28c621",
		SessionID:          "voazwkbpyvazvme4pq2ulj-sz7a",
		Project:            "csscat",
		Client:             "csquad",
		HideDebug:          true,
		CollapseDuplicates: true,
	})
	if err != nil {
		t.Fatalf("BuildChatTrace returned error: %v", err)
	}
	if len(trace.Timeline) == 0 {
		t.Fatalf("expected timeline entries")
	}
	if trace.Summary.ChatKey == "" {
		t.Fatalf("expected summary chat key")
	}
	if len(trace.Anomalies) == 0 {
		t.Fatalf("expected anomalies")
	}
}

func TestBuildChatTraceRequiresSessionID(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		DefaultLookback: 0,
		MaxLogLines:     500,
		MaxRawLines:     500,
	}
	svc := NewChatTraceService(cfg, victorialogs.NewSampleClient("../../testdata/sample_victorialogs.json"))

	_, err := svc.BuildChatTrace(context.Background(), domain.TraceQuery{
		Project:            "csscat",
		Client:             "csquad",
		HideDebug:          true,
		CollapseDuplicates: true,
	})
	if err == nil {
		t.Fatal("expected error when session_id is missing")
	}
	if err.Error() != "session_id is required for exact chat trace search" {
		t.Fatalf("unexpected error: %v", err)
	}
}
