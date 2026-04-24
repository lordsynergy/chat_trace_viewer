package normalizer

import (
	"fmt"
	"strings"

	"chat-trace-viewer/internal/domain"
	"chat-trace-viewer/internal/parser"
)

type Rule struct {
	Name      string
	EventType string
	Reason    string
	Outcome   string
	Stage     string
	Match     func(entry domain.RawLogEntry) bool
}

type Normalizer struct {
	rules []Rule
}

func New() *Normalizer {
	rules := []Rule{
		{Name: "finished-not-assigned", EventType: domain.EventTypeChatFinished, Reason: domain.ReasonNotAssignedChat, Outcome: "failed", Stage: "finish", Match: contains("Finished for not-assigned chat")},
		{Name: "session-timeout-spam", EventType: domain.EventTypeWarn, Reason: domain.ReasonTimeoutSpam, Outcome: "warn", Stage: "connector", Match: contains("Session terminated due to timeouts spam")},
		{Name: "throw-message-away", EventType: domain.EventTypeThrownAway, Reason: domain.ReasonMessageThrownAway, Outcome: "failed", Stage: "routing", Match: contains("Throw message away")},
		{Name: "skip-chat-event", EventType: domain.EventTypeSkipped, Reason: domain.ReasonChatEventSkipped, Outcome: "skipped", Stage: "filtering", Match: contains("Skip chat_event message")},
		{Name: "message-skipped", EventType: domain.EventTypeSkipped, Reason: domain.ReasonChatEventSkipped, Outcome: "skipped", Stage: "filtering", Match: contains("This message is skipped")},
		{Name: "timeout-removed", EventType: domain.EventTypeTimeoutRemoved, Reason: domain.ReasonRemovedTimeout, Outcome: "removed", Stage: "timeout", Match: contains("Removed timeout for chat key")},
		{Name: "timeout-processing", EventType: domain.EventTypeTimeoutSent, Outcome: "sent", Stage: "timeout", Match: contains("Processing due timeout")},
		{Name: "timeout-sent", EventType: domain.EventTypeTimeoutSent, Outcome: "sent", Stage: "timeout", Match: contains(" sent for web.")},
		{Name: "nlu-request", EventType: domain.EventTypeNLURequest, Outcome: "requested", Stage: "nlu", Match: contains("NLU requ <=")},
		{Name: "nlu-response", EventType: domain.EventTypeNLUResponse, Outcome: "received", Stage: "nlu", Match: contains("NLU resp =>")},
		{Name: "published", EventType: domain.EventTypePublished, Outcome: "published", Stage: "publish", Match: contains("Published message for subject")},
		{Name: "delivered", EventType: domain.EventTypeDelivered, Outcome: "delivered", Stage: "delivery", Match: contains("Delivered message from routing_key")},
		{Name: "started-processing", EventType: domain.EventTypeProcessingStarted, Outcome: "started", Stage: "processing", Match: contains("Started processing")},
		{Name: "transformed", EventType: domain.EventTypeTransformed, Outcome: "transformed", Stage: "transform", Match: contains("Transformer subject from")},
		{Name: "received", EventType: domain.EventTypeReceived, Outcome: "received", Stage: "ingress", Match: contains("Received message:")},
		{Name: "handle-finished", EventType: domain.EventTypeChatFinished, Outcome: "handled", Stage: "finish", Match: contains("Handle finished")},
		{Name: "operator-returned", EventType: domain.EventTypeOperatorReturned, Outcome: "returned", Stage: "operator", Match: contains("Returned chat to robot")},
		{Name: "operator-unassigned", EventType: domain.EventTypeOperatorUnassigned, Outcome: "unassigned", Stage: "operator", Match: contains("Chat unassigned")},
		{Name: "operator-processing", EventType: domain.EventTypeOperatorUnassigned, Outcome: "processed", Stage: "operator", Match: contains("as operator_unassigned")},
		{Name: "to-robot", EventType: domain.EventTypeOperatorReturned, Outcome: "published", Stage: "operator", Match: contains("transferred_back_to_robot")},
		{Name: "chat-finished-publish", EventType: domain.EventTypeChatFinished, Outcome: "published", Stage: "finish", Match: contains("chat_finished")},
		{Name: "session-cleared", EventType: domain.EventTypeInfo, Reason: domain.ReasonSessionCleared, Outcome: "cleared", Stage: "cleanup", Match: contains("Session cleared")},
	}
	return &Normalizer{rules: rules}
}

func (n *Normalizer) Normalize(index int, entry domain.RawLogEntry, query domain.TraceQuery) domain.NormalizedEvent {
	chat, source, confidence, subjectInfo, identityVerified := parser.FindIdentity(entry.Message, entry.Subjects, entry.Payload, query)
	subject := firstSubject(entry.Subjects)
	routeInfo, hasRoute := parser.BestSubjectInfo(entry.Subjects, entry.Payload)
	event := domain.NormalizedEvent{
		ID:                 fmt.Sprintf("event-%04d", index+1),
		Timestamp:          entry.Timestamp,
		Service:            entry.Service,
		Component:          entry.Component,
		Level:              normalizeLevel(entry.Level),
		EventType:          domain.EventTypeUnknown,
		Outcome:            "observed",
		MessageKind:        parser.DetectMessageKind(entry),
		Chat:               chat,
		Subject:            subject,
		RoutingKey:         subject,
		PayloadPreview:     previewPayload(entry),
		RawRef:             fmt.Sprintf("raw-%04d", index+1),
		RawIndex:           index,
		Raw:                entry,
		Description:        entry.Message,
		MatchSource:        source,
		Confidence:         confidence,
		SubjectParseResult: subjectInfo,
		IdentityVerified:   identityVerified,
	}
	if hasRoute {
		event.From = routeInfo.From
		event.To = routeInfo.To
		if event.Subject == "" && routeInfo.Chat.ChatKey != "" {
			event.Subject = routeInfo.Chat.ChatKey
			event.RoutingKey = routeInfo.Chat.ChatKey
		}
	}

	for _, rule := range n.rules {
		if rule.Match(entry) {
			event.Rule = rule.Name
			event.EventType = rule.EventType
			event.Reason = rule.Reason
			event.Outcome = rule.Outcome
			event.Stage = rule.Stage
			return n.applyLevelDefaults(event)
		}
	}

	return n.applyLevelDefaults(event)
}

func (n *Normalizer) applyLevelDefaults(event domain.NormalizedEvent) domain.NormalizedEvent {
	if event.EventType != domain.EventTypeUnknown {
		return event
	}
	switch event.Level {
	case "error":
		event.EventType = domain.EventTypeError
		event.Outcome = "failed"
	case "warn":
		event.EventType = domain.EventTypeWarn
		event.Outcome = "warn"
	default:
		event.EventType = domain.EventTypeInfo
	}
	return event
}

func contains(fragment string) func(entry domain.RawLogEntry) bool {
	return func(entry domain.RawLogEntry) bool {
		targets := []string{entry.Message}
		for _, subject := range entry.Subjects {
			targets = append(targets, subject)
		}
		for _, target := range targets {
			if strings.Contains(strings.ToLower(target), strings.ToLower(fragment)) {
				return true
			}
		}
		if strings.Contains(strings.ToLower(previewPayload(entry)), strings.ToLower(fragment)) {
			return true
		}
		return false
	}
}

func firstSubject(subjects []string) string {
	if len(subjects) == 0 {
		return ""
	}
	return subjects[0]
}

func previewPayload(entry domain.RawLogEntry) string {
	if msg := parserString(entry.Payload, "msg_data"); msg != "" {
		return shrink(msg)
	}
	if payload := parserString(entry.Payload, "message_data"); payload != "" {
		return shrink(payload)
	}
	return shrink(entry.RawMessage)
}

func parserString(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	if value, ok := payload[key]; ok {
		return fmt.Sprintf("%v", value)
	}
	return ""
}

func shrink(text string) string {
	text = strings.TrimSpace(text)
	if len(text) <= 160 {
		return text
	}
	return text[:157] + "..."
}

func normalizeLevel(level string) string {
	level = strings.ToLower(strings.TrimSpace(level))
	switch level {
	case "":
		return "info"
	case "warning":
		return "warn"
	default:
		return level
	}
}
