package parser

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"chat-trace-viewer/internal/domain"
)

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func ParseRecord(record map[string]any) domain.RawLogEntry {
	entry := domain.RawLogEntry{
		Fields:   map[string]any{},
		Original: record,
		Payload:  map[string]any{},
	}

	entry.Stream = stringValue(record["_stream"])
	entry.Timestamp = parseTimestamp(record)

	rawMessage := stringValue(record["_msg"])
	entry.RawMessage = rawMessage

	decoded, ok := parseEmbeddedJSON(rawMessage)
	if ok {
		entry.Level = strings.ToLower(stringValue(decoded["level"]))
		entry.Service = normalizeService(stringValue(decoded["application"]), stringValue(record["kubernetes.container_name"]))
		entry.Component = stringValue(decoded["name"])
		entry.Message = stringValue(decoded["message"])
		entry.Fields = decoded
		entry.Payload = toStringAnyMap(decoded["payload"])
		entry.Subjects = extractSubjects(entry.Message, entry.Payload)
		return entry
	}

	clean := strings.TrimSpace(ansiPattern.ReplaceAllString(rawMessage, ""))
	entry.Message = clean
	entry.Level = detectTextLevel(clean)
	entry.Service = normalizeService(stringValue(record["kubernetes.container_name"]), stringValue(record["kubernetes.pod_name"]))
	entry.Component = detectTextComponent(clean)
	entry.Fields["text"] = clean
	entry.Subjects = extractSubjects(clean, nil)
	return entry
}

func parseEmbeddedJSON(raw string) (map[string]any, bool) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "{") {
		return nil, false
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil, false
	}
	return decoded, true
}

func parseTimestamp(record map[string]any) time.Time {
	candidates := []string{
		stringValue(record["_msg.timestamp"]),
		stringValue(record["_time"]),
	}
	if decoded, ok := parseEmbeddedJSON(stringValue(record["_msg"])); ok {
		candidates = append([]string{stringValue(decoded["timestamp"])}, candidates...)
	}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if ts, err := time.Parse(time.RFC3339Nano, candidate); err == nil {
			return ts
		}
	}
	return time.Time{}
}

func detectTextLevel(message string) string {
	switch {
	case strings.Contains(message, " WARNING "), strings.Contains(message, "| WARNING"):
		return "warn"
	case strings.Contains(message, " ERROR "), strings.Contains(message, "| ERROR"):
		return "error"
	case strings.Contains(message, " DEBUG "), strings.Contains(message, "| DEBUG"), strings.Contains(message, " D "):
		return "debug"
	default:
		return "info"
	}
}

func detectTextComponent(message string) string {
	if idx := strings.Index(message, " -- "); idx >= 0 {
		left := strings.TrimSpace(message[:idx])
		parts := strings.Fields(left)
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}
	return ""
}

func extractSubjects(message string, payload map[string]any) []string {
	set := map[string]struct{}{}
	add := func(v string) {
		v = strings.TrimSpace(v)
		v = strings.Trim(v, `"'`)
		if v != "" {
			set[v] = struct{}{}
		}
	}

	for _, key := range []string{"subject", "msg_subject", "chat_subject", "self_subject", "js_subject"} {
		add(nestedString(payload, key))
	}

	for _, rx := range []*regexp.Regexp{
		regexp.MustCompile(`subject:\s*([^,\s]+)`),
		regexp.MustCompile(`routing_key:\s*([^,\s]+)`),
		regexp.MustCompile(`session_id='([^']+)'`),
		regexp.MustCompile(`chat key ([^,\s]+)`),
		regexp.MustCompile(`from\s+'([^']+)'`),
		regexp.MustCompile(`to\s+'([^']+)'`),
	} {
		matches := rx.FindAllStringSubmatch(message, -1)
		for _, match := range matches {
			if len(match) > 1 {
				add(match[1])
			}
		}
	}

	if payload != nil {
		if msgData := nestedString(payload, "msg_data"); msgData != "" {
			var decoded map[string]any
			if err := json.Unmarshal([]byte(msgData), &decoded); err == nil {
				if subj := toStringAnyMap(decoded["subj"]); subj != nil {
					add(buildChatKey(
						stringValue(subj["client"]),
						stringValue(subj["project"]),
						stringValue(subj["user_id"]),
						stringValue(subj["session_id"]),
					))
				}
			}
		}
	}

	subjects := make([]string, 0, len(set))
	for subject := range set {
		subjects = append(subjects, subject)
	}
	sort.Strings(subjects)
	return subjects
}

func normalizeService(primary, fallback string) string {
	for _, candidate := range []string{primary, fallback} {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if idx := strings.Index(candidate, "/"); idx >= 0 {
			candidate = candidate[:idx]
		}
		return strings.ToLower(candidate)
	}
	return "unknown"
}

func toStringAnyMap(value any) map[string]any {
	if value == nil {
		return nil
	}
	switch typed := value.(type) {
	case map[string]any:
		return typed
	case map[string]string:
		out := make(map[string]any, len(typed))
		for k, v := range typed {
			out[k] = v
		}
		return out
	default:
		return nil
	}
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case json.Number:
		return typed.String()
	case int:
		return fmt.Sprintf("%d", typed)
	case int8:
		return fmt.Sprintf("%d", typed)
	case int16:
		return fmt.Sprintf("%d", typed)
	case int32:
		return fmt.Sprintf("%d", typed)
	case int64:
		return fmt.Sprintf("%d", typed)
	case uint:
		return fmt.Sprintf("%d", typed)
	case uint8:
		return fmt.Sprintf("%d", typed)
	case uint16:
		return fmt.Sprintf("%d", typed)
	case uint32:
		return fmt.Sprintf("%d", typed)
	case uint64:
		return fmt.Sprintf("%d", typed)
	case float32:
		return fmt.Sprintf("%v", typed)
	case float64:
		return fmt.Sprintf("%v", typed)
	case fmt.Stringer:
		return typed.String()
	default:
		return ""
	}
}

func nestedString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	return stringValue(m[key])
}
