package cost

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/thissayantan/vitals/internal/claude"
)

func TestPriceFor(t *testing.T) {
	cases := []struct {
		model string
		in    float64
	}{
		{"claude-opus-4-8", 5},
		{"claude-opus-4-1", 15},
		{"claude-opus-3-opus", 15},
		{"claude-sonnet-4-6", 3},
		{"claude-haiku-4-5", 1},
		{"claude-haiku-3-legacy", 0.25}, // legacy rule keys on "haiku-3" (DESIGN §7)
		{"something-unknown", 3},        // default = sonnet
	}
	for _, c := range cases {
		if got := priceFor(c.model).In; got != c.in {
			t.Errorf("priceFor(%q).In = %v, want %v", c.model, got, c.in)
		}
	}
}

func TestEstimateFromTranscript(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "t.jsonl")
	// Two assistant messages with usage; one non-assistant line ignored.
	content := `{"type":"user","message":{}}
{"type":"assistant","message":{"model":"claude-opus-4-8","usage":{"input_tokens":1000000,"output_tokens":1000000,"cache_read_input_tokens":0,"cache_creation_input_tokens":0}}}
{"type":"assistant","message":{"model":"claude-sonnet-4-6","usage":{"input_tokens":1000000,"output_tokens":0,"cache_read_input_tokens":0,"cache_creation_input_tokens":0}}}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	// opus: 1M*5 + 1M*25 = 30; sonnet: 1M*3 = 3; total 33.
	got := estimateFromTranscript(path)
	if got != 33 {
		t.Errorf("estimate = %v, want 33", got)
	}
}

func TestGetSources(t *testing.T) {
	s := &claude.Session{Cost: claude.Cost{TotalCostUSD: 12.5}}

	if e := Get(s, nil, "cc"); e.Estimated || e.USD != 12.5 {
		t.Errorf("cc source = %+v", e)
	}
	if e := Get(s, nil, "auto"); e.Estimated || e.USD != 12.5 {
		t.Errorf("auto with real cost = %+v", e)
	}
	// auto with zero real cost ⇒ estimate path (no transcript ⇒ 0, Estimated true).
	s0 := &claude.Session{}
	if e := Get(s0, nil, "auto"); !e.Estimated {
		t.Errorf("auto with zero cost should be estimated, got %+v", e)
	}
}
