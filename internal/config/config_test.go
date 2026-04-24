package config

import (
	"testing"
	"time"
)

func TestParseFlexibleDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		want  time.Duration
	}{
		{name: "hours", value: "6h", want: 6 * time.Hour},
		{name: "days", value: "30d", want: 30 * 24 * time.Hour},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseFlexibleDuration(tt.value)
			if err != nil {
				t.Fatalf("parseFlexibleDuration returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}
