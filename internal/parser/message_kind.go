package parser

import (
	"encoding/json"
	"strings"

	"chat-trace-viewer/internal/domain"
)

func DetectMessageKind(entry domain.RawLogEntry) string {
	if hasContentPayload(entry.Payload) || hasContentText(entry.Message) || hasContentText(entry.RawMessage) {
		return domain.MessageKindContent
	}

	if info, ok := BestSubjectInfo(entry.Subjects, entry.Payload); ok && isCommandSubject(info) {
		return domain.MessageKindCommand
	}

	for _, subject := range collectSubjects(entry) {
		info, ok := ParseSubject(subject)
		if !ok {
			continue
		}
		if isCommandSubject(info) {
			return domain.MessageKindCommand
		}
	}

	return domain.MessageKindSystem
}

func collectSubjects(entry domain.RawLogEntry) []string {
	subjects := make([]string, 0, len(entry.Subjects)+5)
	subjects = append(subjects, entry.Subjects...)
	for _, key := range []string{"subject", "msg_subject", "chat_subject", "self_subject", "js_subject"} {
		if subject := nestedString(entry.Payload, key); subject != "" {
			subjects = append(subjects, subject)
		}
	}
	return subjects
}

func hasContentPayload(payload map[string]any) bool {
	if payload == nil {
		return false
	}

	for _, key := range []string{"text", "message_text"} {
		if nestedString(payload, key) != "" {
			return true
		}
	}

	msgData := nestedString(payload, "msg_data")
	if msgData == "" {
		return false
	}

	decoded, ok := parseEmbeddedJSON(msgData)
	if !ok {
		return false
	}

	if stringValue(decoded["text"]) != "" {
		return true
	}
	return false
}

func isCommandSubject(info SubjectInfo) bool {
	if info.Content == "msg" {
		return true
	}
	if info.PatternName == "legacy-routing" && info.Content == "msg" {
		return true
	}
	return false
}

func hasContentText(text string) bool {
	text = strings.ToLower(strings.TrimSpace(text))
	if text == "" {
		return false
	}
	if strings.Contains(text, "user_output=") {
		return true
	}
	if strings.Contains(text, `"text":"`) || strings.Contains(text, `"text": "`) {
		return true
	}
	if strings.Contains(text, "chatmessage(") && strings.Contains(text, "text=") {
		return true
	}

	if json.Valid([]byte(text)) {
		var decoded map[string]any
		if err := json.Unmarshal([]byte(text), &decoded); err == nil {
			if stringValue(decoded["text"]) != "" {
				return true
			}
		}
	}

	return false
}
