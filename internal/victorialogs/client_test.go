package victorialogs

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"chat-trace-viewer/internal/domain"
)

func TestBuildQuery(t *testing.T) {
	t.Parallel()

	query := BuildQuery(domain.TraceQuery{
		UserID:    "61dfe3428df3572ee111ecb01c28c621",
		SessionID: "voazwkbpyvazvme4pq2ulj-sz7a",
		Project:   "csscat",
		Client:    "csquad",
	})

	for _, want := range []string{
		`_msg:~"voazwkbpyvazvme4pq2ulj-sz7a"`,
		`_msg:~"61dfe3428df3572ee111ecb01c28c621"`,
		`_msg:~"csscat"`,
		`_msg:~"csquad"`,
	} {
		if !strings.Contains(query, want) {
			t.Fatalf("expected query %q to contain %q", query, want)
		}
	}
	if strings.Count(query, " AND ") != 3 {
		t.Fatalf("expected query clauses to be joined with AND, got %q", query)
	}
}

func TestBuildQuerySkipsEmptyValues(t *testing.T) {
	t.Parallel()

	query := BuildQuery(domain.TraceQuery{SessionID: "voazwkbpyvazvme4pq2ulj-sz7a"})

	if strings.Contains(query, `_msg:~""`) {
		t.Fatalf("query must not contain an empty regexp clause: %q", query)
	}
	if got := strings.Count(query, `_msg:~`); got != 1 {
		t.Fatalf("expected exactly one regexp clause, got %d in %q", got, query)
	}
}

func TestSampleClientReadsJSONL(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "sample.jsonl")
	content := strings.Join([]string{
		`{"_msg":"first","_time":"2026-04-14T13:00:55Z"}`,
		`{"_msg":"second","_time":"2026-04-14T13:00:56Z"}`,
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write sample jsonl: %v", err)
	}

	client := &SampleClient{path: path}
	records, err := client.Query(context.Background(), domain.TraceQuery{})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0]["_msg"] != "first" {
		t.Fatalf("unexpected first message: %#v", records[0]["_msg"])
	}
}
