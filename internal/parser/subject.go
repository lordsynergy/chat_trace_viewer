package parser

import (
	"fmt"
	"regexp"
	"strings"

	"chat-trace-viewer/internal/domain"
)

type SubjectInfo struct {
	Chat        domain.ChatIdentity
	From        string
	To          string
	Type        string
	Content     string
	OperatorID  string
	PatternName string
}

var chatKeyPattern = regexp.MustCompile(`([a-z0-9_-]+)\.([a-z0-9_-]+)\.([a-f0-9]{16,64})\.([a-z0-9_-]{6,})`)
var webSessionPattern = regexp.MustCompile(`web\.([a-z0-9_-]+)\.([a-f0-9]{16,64})\.([a-z0-9_-]{6,})`)

func ParseSubject(subject string) (SubjectInfo, bool) {
	subject = strings.TrimSpace(strings.Trim(subject, `"'`))
	if subject == "" {
		return SubjectInfo{}, false
	}

	if strings.HasPrefix(subject, "operators-chats.") {
		parts := strings.Split(subject, ".")
		if len(parts) >= 7 {
			return makeInfo(parts[1], parts[2], parts[3], parts[4], "operators-chats", "", "", parts[5], parts[6]), true
		}
	}

	if strings.HasPrefix(subject, "operator-assigned.") {
		parts := strings.Split(subject, ".")
		if len(parts) >= 7 {
			return makeInfo(parts[1], parts[2], parts[3], parts[4], "operator-assigned", "", parts[6], "assigned", parts[5]), true
		}
	}

	parts := strings.Split(subject, ".")
	if len(parts) >= 8 {
		return makeInfo(parts[0], parts[1], parts[2], parts[3], "chat-subject", parts[4], parts[5], parts[6], strings.Join(parts[7:], ".")), true
	}

	if len(parts) == 6 && parts[2] == "web" {
		return makeInfo("", parts[3], parts[4], parts[5], "legacy-routing", parts[1], parts[2], "legacy", parts[0]), true
	}

	if match := chatKeyPattern.FindStringSubmatch(subject); len(match) == 5 {
		return makeInfo(match[1], match[2], match[3], match[4], "chat-key", "", "", "", ""), true
	}

	if match := webSessionPattern.FindStringSubmatch(subject); len(match) == 4 {
		return makeInfo("", match[1], match[2], match[3], "web-session", "web", "", "", ""), true
	}

	return SubjectInfo{}, false
}

func FindIdentity(message string, subjects []string, payload map[string]any, query domain.TraceQuery) (domain.ChatIdentity, string, float64, string, bool) {
	if payload != nil {
		if msgData := nestedString(payload, "msg_data"); msgData != "" {
			if info := identityFromEmbeddedPayload(msgData); info.Chat.ChatKey != "" {
				return safeOverlayQuery(info.Chat, query), "payload.msg_data.subj", domain.DefaultConfidenceHigh, describeSubject(info), true
			}
		}
	}

	for _, key := range []string{"chat_subject", "subject", "msg_subject", "self_subject"} {
		if subject := nestedString(payload, key); subject != "" {
			if info, ok := ParseSubject(subject); ok {
				return safeOverlayQuery(info.Chat, query), "payload." + key, domain.DefaultConfidenceHigh, describeSubject(info), true
			}
		}
	}

	for _, subject := range subjects {
		if info, ok := ParseSubject(subject); ok {
			return safeOverlayQuery(info.Chat, query), "subject", domain.DefaultConfidenceMedium, describeSubject(info), true
		}
	}

	if match := webSessionPattern.FindStringSubmatch(message); len(match) == 4 {
		chat := safeOverlayQuery(domain.ChatIdentity{
			Project:   match[1],
			UserID:    match[2],
			SessionID: match[3],
			ChatKey:   buildChatKey(query.Client, match[1], match[2], match[3]),
		}, query)
		return chat, "_msg regex", domain.DefaultConfidenceLow, "regex:web-session", true
	}

	return safeOverlayQuery(domain.ChatIdentity{}, query), "none", 0, "identity not found", false
}

func BestSubjectInfo(subjects []string, payload map[string]any) (SubjectInfo, bool) {
	if payload != nil {
		if msgData := nestedString(payload, "msg_data"); msgData != "" {
			if info := identityFromEmbeddedPayload(msgData); isUsefulSubjectInfo(info) {
				return info, true
			}
		}
	}

	candidates := make([]SubjectInfo, 0, len(subjects)+5)
	for _, key := range []string{"chat_subject", "subject", "msg_subject", "self_subject", "js_subject"} {
		if subject := nestedString(payload, key); subject != "" {
			if info, ok := ParseSubject(subject); ok {
				candidates = append(candidates, info)
			}
		}
	}

	for _, subject := range subjects {
		if info, ok := ParseSubject(subject); ok {
			candidates = append(candidates, info)
		}
	}

	var (
		best      SubjectInfo
		bestScore int
		found     bool
	)
	for _, info := range candidates {
		if !isUsefulSubjectInfo(info) {
			continue
		}
		score := scoreSubjectInfo(info)
		if !found || score > bestScore {
			best = info
			bestScore = score
			found = true
		}
	}

	return best, found
}

func identityFromEmbeddedPayload(msgData string) SubjectInfo {
	decoded, ok := parseEmbeddedJSON(msgData)
	if !ok {
		return SubjectInfo{}
	}
	subj := toStringAnyMap(decoded["subj"])
	if subj == nil {
		return SubjectInfo{}
	}
	return makeInfo(
		stringValue(subj["client"]),
		stringValue(subj["project"]),
		stringValue(subj["user_id"]),
		stringValue(subj["session_id"]),
		"payload-subj",
		stringValue(subj["from"]),
		stringValue(subj["to"]),
		stringValue(subj["type"]),
		stringValue(subj["content"]),
	)
}

func safeOverlayQuery(chat domain.ChatIdentity, query domain.TraceQuery) domain.ChatIdentity {
	if chat.Client == "" {
		chat.Client = query.Client
	}
	if chat.Project == "" {
		chat.Project = query.Project
	}
	chat.ChatKey = buildChatKey(chat.Client, chat.Project, chat.UserID, chat.SessionID)
	return chat
}

func makeInfo(client, project, userID, sessionID, pattern, from, to, msgType, content string) SubjectInfo {
	return SubjectInfo{
		Chat: domain.ChatIdentity{
			Client:    client,
			Project:   project,
			UserID:    userID,
			SessionID: sessionID,
			ChatKey:   buildChatKey(client, project, userID, sessionID),
		},
		From:        from,
		To:          to,
		Type:        msgType,
		Content:     content,
		PatternName: pattern,
	}
}

func buildChatKey(client, project, userID, sessionID string) string {
	parts := []string{}
	for _, part := range []string{client, project, userID, sessionID} {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, ".")
}

func describeSubject(info SubjectInfo) string {
	return fmt.Sprintf("%s client=%s project=%s user=%s session=%s from=%s to=%s type=%s content=%s",
		info.PatternName,
		info.Chat.Client,
		info.Chat.Project,
		info.Chat.UserID,
		info.Chat.SessionID,
		info.From,
		info.To,
		info.Type,
		info.Content,
	)
}

func isUsefulSubjectInfo(info SubjectInfo) bool {
	return info.Chat.ChatKey != "" || info.From != "" || info.To != "" || info.Type != "" || info.Content != ""
}

func scoreSubjectInfo(info SubjectInfo) int {
	score := 0
	if info.Chat.ChatKey != "" {
		score += 5
	}
	if info.Chat.Client != "" {
		score += 2
	}
	if info.From != "" {
		score += 4
	}
	if info.To != "" {
		score += 4
	}
	if info.Type != "" {
		score += 2
	}
	if info.Content != "" {
		score += 2
	}

	switch info.PatternName {
	case "payload-subj":
		score += 10
	case "chat-subject":
		score += 8
	case "operators-chats", "operator-assigned":
		score += 6
	case "legacy-routing":
		score += 4
	case "chat-key", "web-session":
		score += 1
	}

	return score
}
