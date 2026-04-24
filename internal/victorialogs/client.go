package victorialogs

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"chat-trace-viewer/internal/config"
	"chat-trace-viewer/internal/domain"
)

type Client interface {
	Query(ctx context.Context, query domain.TraceQuery) ([]map[string]any, error)
}

type HTTPClient struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
	cfg        config.Config
}

type DisabledClient struct{}

type SampleClient struct {
	path string
}

func New(cfg config.Config) Client {
	if strings.TrimSpace(cfg.VictoriaLogsBaseURL) == "" {
		return DisabledClient{}
	}
	return &HTTPClient{
		baseURL:  strings.TrimRight(cfg.VictoriaLogsBaseURL, "/"),
		username: cfg.VictoriaLogsUsername,
		password: cfg.VictoriaLogsPassword,
		httpClient: &http.Client{
			Timeout: cfg.VictoriaLogsTimeout,
		},
		cfg: cfg,
	}
}

func NewSampleClient(path string) Client {
	return &SampleClient{path: path}
}

func (DisabledClient) Query(context.Context, domain.TraceQuery) ([]map[string]any, error) {
	return nil, fmt.Errorf("victorialogs is not configured")
}

func (c *HTTPClient) Query(ctx context.Context, query domain.TraceQuery) ([]map[string]any, error) {
	values := url.Values{}
	values.Set("query", BuildQuery(query))
	if query.From != nil {
		values.Set("start", query.From.UTC().Format(time.RFC3339Nano))
	}
	if query.To != nil {
		values.Set("end", query.To.UTC().Format(time.RFC3339Nano))
	}
	values.Set("limit", fmt.Sprintf("%d", c.cfg.MaxLogLines))
	values.Set("timeout", c.cfg.VictoriaLogsTimeout.String())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/select/logsql/query", strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if c.cfg.VictoriaLogsAccountID != "" {
		req.Header.Set("AccountID", c.cfg.VictoriaLogsAccountID)
	}
	if c.cfg.VictoriaLogsProjectID != "" {
		req.Header.Set("ProjectID", c.cfg.VictoriaLogsProjectID)
	}
	if c.username != "" || c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		return nil, fmt.Errorf("victorialogs query failed: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)

	var out []map[string]any
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			return nil, fmt.Errorf("decode victorialogs line: %w", err)
		}
		out = append(out, record)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *SampleClient) Query(ctx context.Context, _ domain.TraceQuery) ([]map[string]any, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if strings.EqualFold(filepath.Ext(c.path), ".jsonl") {
		return readJSONLFile(c.path)
	}

	data, err := os.ReadFile(c.path)
	if err != nil {
		return nil, err
	}
	var records []map[string]any
	if err := json.Unmarshal(data, &records); err == nil {
		return records, nil
	}
	return readJSONLFile(c.path)
}

func BuildQuery(query domain.TraceQuery) string {
	parts := make([]string, 0, 4)

	for _, value := range []string{query.SessionID, query.UserID, query.Project, query.Client} {
		if clause, ok := regexpClause("_msg", value); ok {
			parts = append(parts, clause)
		}
	}

	return strings.Join(parts, " AND ")
}

func regexpClause(field, value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}

	escaped := regexp.QuoteMeta(value)
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return fmt.Sprintf(`%s:~"%s"`, field, replacer.Replace(escaped)), true
}

func readJSONLFile(path string) ([]map[string]any, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)

	records := make([]map[string]any, 0, 256)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			return nil, fmt.Errorf("decode jsonl line: %w", err)
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return records, nil
}
