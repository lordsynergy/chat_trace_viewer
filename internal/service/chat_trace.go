package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"chat-trace-viewer/internal/config"
	"chat-trace-viewer/internal/domain"
	"chat-trace-viewer/internal/normalizer"
	"chat-trace-viewer/internal/parser"
	"chat-trace-viewer/internal/timeline"
	"chat-trace-viewer/internal/victorialogs"
)

type ChatTraceService struct {
	client     victorialogs.Client
	normalizer *normalizer.Normalizer
	timeline   *timeline.Builder
	cfg        config.Config
}

func NewChatTraceService(cfg config.Config, client victorialogs.Client) *ChatTraceService {
	return &ChatTraceService{
		client:     client,
		normalizer: normalizer.New(),
		timeline:   timeline.New(),
		cfg:        cfg,
	}
}

func (s *ChatTraceService) BuildChatTrace(ctx context.Context, query domain.TraceQuery) (domain.ChatTraceResponse, error) {
	if !hasIdentityFilters(query) {
		return domain.ChatTraceResponse{}, errors.New("session_id is required for exact chat trace search")
	}
	if query.From == nil || query.To == nil {
		now := time.Now().UTC()
		if query.To == nil {
			query.To = &now
		}
		if query.From == nil {
			from := query.To.Add(-s.cfg.DefaultLookback)
			query.From = &from
		}
	}

	records, err := s.client.Query(ctx, query)
	if err != nil {
		return domain.ChatTraceResponse{}, fmt.Errorf("query logs: %w", err)
	}

	stats := domain.ParseStats{TotalLines: len(records)}
	rawEntries := make([]domain.RawLogEntry, 0, len(records))
	events := make([]domain.NormalizedEvent, 0, len(records))

	for index, record := range records {
		raw := parser.ParseRecord(record)
		rawEntries = append(rawEntries, raw)
		stats.ParsedLines++
		event := s.normalizer.Normalize(index, raw, query)
		events = append(events, event)
		stats.NormalizedLines++
		if event.EventType == domain.EventTypeUnknown || event.EventType == domain.EventTypeInfo {
			stats.UnclassifiedLines++
		}
	}

	timelineEvents, anomalies, summary := s.timeline.Build(query, events)
	summary.LimitReached = len(records) >= s.cfg.MaxLogLines
	if len(timelineEvents) > s.cfg.MaxRawLines {
		timelineEvents = timelineEvents[:s.cfg.MaxRawLines]
	}
	if len(anomalies) > s.cfg.MaxRawLines {
		anomalies = anomalies[:s.cfg.MaxRawLines]
	}

	return domain.ChatTraceResponse{
		Query:     query,
		Summary:   summary,
		Timeline:  timelineEvents,
		Anomalies: anomalies,
		RawCount:  len(rawEntries),
		Stats:     stats,
	}, nil
}

func hasIdentityFilters(query domain.TraceQuery) bool {
	return query.SessionID != ""
}
