package domain

import "time"

const (
	EventTypeUnknown            = "unknown"
	EventTypeReceived           = "received"
	EventTypeProcessingStarted  = "processing_started"
	EventTypeTransformed        = "transformed"
	EventTypePublished          = "published"
	EventTypeDelivered          = "delivered"
	EventTypeSkipped            = "skipped"
	EventTypeThrownAway         = "thrown_away"
	EventTypeTimeoutSent        = "timeout_sent"
	EventTypeTimeoutRemoved     = "timeout_removed"
	EventTypeNLURequest         = "nlu_request"
	EventTypeNLUResponse        = "nlu_response"
	EventTypeChatFinished       = "chat_finished"
	EventTypeOperatorAssigned   = "operator_assigned"
	EventTypeOperatorUnassigned = "operator_unassigned"
	EventTypeOperatorReturned   = "operator_returned"
	EventTypeWarn               = "warn"
	EventTypeError              = "error"
	EventTypeInfo               = "info"
	ReasonNotAssignedChat       = "not_assigned_chat"
	ReasonChatEventSkipped      = "chat_event_skipped"
	ReasonTimeoutSpam           = "timeout_spam"
	ReasonUnsupportedSubject    = "unsupported_subject"
	ReasonInvalidSubjectFormat  = "invalid_subject_format"
	ReasonInvalidJSON           = "invalid_json"
	ReasonRemovedTimeout        = "removed_timeout"
	ReasonMessageThrownAway     = "message_thrown_away"
	ReasonSessionCleared        = "session_cleared"
	DefaultConfidenceHigh       = 0.95
	DefaultConfidenceMedium     = 0.75
	DefaultConfidenceLow        = 0.45
	MessageKindSystem           = "system"
	MessageKindCommand          = "command"
	MessageKindContent          = "content"
)

type TraceQuery struct {
	UserID             string     `json:"user_id"`
	SessionID          string     `json:"session_id"`
	Project            string     `json:"project,omitempty"`
	Client             string     `json:"client,omitempty"`
	From               *time.Time `json:"from,omitempty"`
	To                 *time.Time `json:"to,omitempty"`
	HideDebug          bool       `json:"hide_debug"`
	OnlyAnomalies      bool       `json:"only_anomalies"`
	CollapseDuplicates bool       `json:"collapse_duplicates"`
}

type ChatIdentity struct {
	Client    string `json:"client,omitempty"`
	Project   string `json:"project,omitempty"`
	UserID    string `json:"user_id,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	ChatKey   string `json:"chat_key,omitempty"`
}

type RawLogEntry struct {
	Timestamp  time.Time      `json:"timestamp"`
	Level      string         `json:"level"`
	Service    string         `json:"service"`
	Component  string         `json:"component,omitempty"`
	Message    string         `json:"message"`
	Fields     map[string]any `json:"fields,omitempty"`
	Original   map[string]any `json:"original,omitempty"`
	Subjects   []string       `json:"subjects,omitempty"`
	Payload    map[string]any `json:"payload,omitempty"`
	Stream     string         `json:"stream,omitempty"`
	RawMessage string         `json:"raw_message,omitempty"`
}

type NormalizedEvent struct {
	ID                 string       `json:"id"`
	Timestamp          time.Time    `json:"timestamp"`
	Service            string       `json:"service"`
	Component          string       `json:"component,omitempty"`
	Level              string       `json:"level"`
	EventType          string       `json:"event_type"`
	Stage              string       `json:"stage,omitempty"`
	Outcome            string       `json:"outcome,omitempty"`
	Reason             string       `json:"reason,omitempty"`
	Rule               string       `json:"rule,omitempty"`
	MatchSource        string       `json:"match_source,omitempty"`
	Confidence         float64      `json:"confidence"`
	Description        string       `json:"description"`
	MessageKind        string       `json:"message_kind,omitempty"`
	From               string       `json:"from,omitempty"`
	To                 string       `json:"to,omitempty"`
	Chat               ChatIdentity `json:"chat"`
	Subject            string       `json:"subject,omitempty"`
	RoutingKey         string       `json:"routing_key,omitempty"`
	PayloadPreview     string       `json:"payload_preview,omitempty"`
	RawRef             string       `json:"raw_ref"`
	RawIndex           int          `json:"raw_index"`
	Raw                RawLogEntry  `json:"raw"`
	SubjectParseResult string       `json:"subject_parse_result,omitempty"`
	IdentityVerified   bool         `json:"identity_verified"`
}

type ParseStats struct {
	TotalLines        int `json:"total_lines"`
	ParsedLines       int `json:"parsed_lines"`
	NormalizedLines   int `json:"normalized_lines"`
	UnclassifiedLines int `json:"unclassified_lines"`
}

type TraceSummary struct {
	ChatKey               string     `json:"chat_key,omitempty"`
	StartedAt             *time.Time `json:"started_at,omitempty"`
	FinishedAt            *time.Time `json:"finished_at,omitempty"`
	Services              []string   `json:"services"`
	EventsCount           int        `json:"events_count"`
	ErrorCount            int        `json:"error_count"`
	WarnCount             int        `json:"warn_count"`
	SkipCount             int        `json:"skip_count"`
	HasErrors             bool       `json:"has_errors"`
	HasWarnings           bool       `json:"has_warnings"`
	LastEventType         string     `json:"last_event_type,omitempty"`
	FinalState            string     `json:"final_state,omitempty"`
	SuspectedFailurePoint string     `json:"suspected_failure_point,omitempty"`
	LimitReached          bool       `json:"limit_reached"`
}

type ChatTraceResponse struct {
	Query     TraceQuery        `json:"query"`
	Summary   TraceSummary      `json:"summary"`
	Timeline  []NormalizedEvent `json:"timeline"`
	Anomalies []NormalizedEvent `json:"anomalies"`
	RawCount  int               `json:"raw_count"`
	Stats     ParseStats        `json:"stats"`
}
