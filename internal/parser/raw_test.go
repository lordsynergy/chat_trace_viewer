package parser

import "testing"

func TestParseRecordExtractsTransformerSubjects(t *testing.T) {
	t.Parallel()

	record := map[string]any{
		"_msg": `{"application":"Transformer/1.0.1","timestamp":"2026-04-02T14:46:36.825993Z","level":"debug","name":"NatsApp::MessageProcessor","message":"Transformer subject from 'msg.user.web.csscat.f6fc7a1bcd7da6de6d302822a58af9d9.prbwcmnsonfyxa3zeyke5p-l6aq' to 'csquad.csscat.f6fc7a1bcd7da6de6d302822a58af9d9.prbwcmnsonfyxa3zeyke5p-l6aq.operator.widget.msg.msg'"}`,
	}

	entry := ParseRecord(record)
	if len(entry.Subjects) != 2 {
		t.Fatalf("expected 2 extracted subjects, got %d: %#v", len(entry.Subjects), entry.Subjects)
	}

	want := map[string]bool{
		"msg.user.web.csscat.f6fc7a1bcd7da6de6d302822a58af9d9.prbwcmnsonfyxa3zeyke5p-l6aq":                   false,
		"csquad.csscat.f6fc7a1bcd7da6de6d302822a58af9d9.prbwcmnsonfyxa3zeyke5p-l6aq.operator.widget.msg.msg": false,
	}
	for _, subject := range entry.Subjects {
		if _, ok := want[subject]; ok {
			want[subject] = true
		}
	}
	for subject, matched := range want {
		if !matched {
			t.Fatalf("expected subject %q to be extracted, got %#v", subject, entry.Subjects)
		}
	}
}
