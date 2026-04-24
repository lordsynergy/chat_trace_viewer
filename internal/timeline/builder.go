package timeline

import (
	"sort"
	"strings"
	"time"

	"chat-trace-viewer/internal/domain"
)

type Builder struct{}

func New() *Builder {
	return &Builder{}
}

func (b *Builder) Build(query domain.TraceQuery, events []domain.NormalizedEvent) ([]domain.NormalizedEvent, []domain.NormalizedEvent, domain.TraceSummary) {
	filtered := make([]domain.NormalizedEvent, 0, len(events))
	for _, event := range events {
		if !event.IdentityVerified {
			continue
		}
		if !matchesQuery(event.Chat, query) {
			continue
		}
		if query.HideDebug && event.Level == "debug" && !isAnomaly(event) {
			continue
		}
		filtered = append(filtered, event)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Timestamp.Before(filtered[j].Timestamp)
	})

	if query.CollapseDuplicates {
		filtered = collapse(filtered)
	}

	anomalies := make([]domain.NormalizedEvent, 0)
	for _, event := range filtered {
		if isAnomaly(event) {
			anomalies = append(anomalies, event)
		}
	}

	summaryBase := filtered
	if query.OnlyAnomalies {
		filtered = anomalies
	}

	return filtered, anomalies, summarize(summaryBase, anomalies)
}

func matchesQuery(chat domain.ChatIdentity, query domain.TraceQuery) bool {
	if query.UserID != "" && chat.UserID != query.UserID {
		return false
	}
	if query.SessionID != "" && chat.SessionID != query.SessionID {
		return false
	}
	if query.Project != "" && chat.Project != query.Project {
		return false
	}
	if query.Client != "" && chat.Client != query.Client {
		return false
	}
	return true
}

func collapse(events []domain.NormalizedEvent) []domain.NormalizedEvent {
	if len(events) < 2 {
		return events
	}
	collapsed := []domain.NormalizedEvent{events[0]}
	for _, event := range events[1:] {
		prev := collapsed[len(collapsed)-1]
		if duplicate(prev, event) {
			continue
		}
		collapsed = append(collapsed, event)
	}
	return collapsed
}

func duplicate(left, right domain.NormalizedEvent) bool {
	if left.Service != right.Service || left.EventType != right.EventType || left.Subject != right.Subject || left.Reason != right.Reason {
		return false
	}
	if left.Description != right.Description {
		return false
	}
	delta := right.Timestamp.Sub(left.Timestamp)
	return delta >= 0 && delta <= 3*time.Second
}

func summarize(events, anomalies []domain.NormalizedEvent) domain.TraceSummary {
	summary := domain.TraceSummary{
		Services: []string{},
	}
	services := map[string]struct{}{}
	for _, event := range events {
		if summary.StartedAt == nil || event.Timestamp.Before(*summary.StartedAt) {
			ts := event.Timestamp
			summary.StartedAt = &ts
		}
		if summary.FinishedAt == nil || event.Timestamp.After(*summary.FinishedAt) {
			ts := event.Timestamp
			summary.FinishedAt = &ts
		}
		if summary.ChatKey == "" {
			summary.ChatKey = event.Chat.ChatKey
		}
		if event.Service != "" {
			services[event.Service] = struct{}{}
		}
		summary.EventsCount++
		summary.LastEventType = event.EventType
		switch event.EventType {
		case domain.EventTypeError, domain.EventTypeThrownAway:
			summary.ErrorCount++
			summary.HasErrors = true
			summary.SuspectedFailurePoint = event.Service
		case domain.EventTypeWarn:
			summary.WarnCount++
			summary.HasWarnings = true
			if summary.SuspectedFailurePoint == "" {
				summary.SuspectedFailurePoint = event.Service
			}
		case domain.EventTypeSkipped:
			summary.SkipCount++
		}
	}
	for service := range services {
		summary.Services = append(summary.Services, service)
	}
	sort.Strings(summary.Services)

	if len(events) > 0 {
		last := events[len(events)-1]
		summary.FinalState = finalState(last)
	}
	if len(anomalies) == 0 && summary.FinalState == "" {
		summary.FinalState = "no_data"
	}
	return summary
}

func finalState(event domain.NormalizedEvent) string {
	if event.Reason != "" {
		return strings.TrimSpace(event.EventType + ":" + event.Reason)
	}
	return event.EventType
}

func isAnomaly(event domain.NormalizedEvent) bool {
	switch event.EventType {
	case domain.EventTypeError, domain.EventTypeWarn, domain.EventTypeSkipped, domain.EventTypeThrownAway:
		return true
	}
	switch event.Level {
	case "error", "warn":
		return true
	}
	return event.Reason != ""
}
