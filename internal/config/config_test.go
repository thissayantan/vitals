package config

import (
	"encoding/json"
	"testing"
)

func TestDefaultsValidate(t *testing.T) {
	data, err := json.Marshal(Defaults())
	if err != nil {
		t.Fatal(err)
	}
	if err := Validate(data); err != nil {
		t.Fatalf("Defaults() should satisfy the schema: %v", err)
	}
}

func TestDefaultsLayout(t *testing.T) {
	d := Defaults()
	if len(d.Lines) != 2 {
		t.Fatalf("want 2 lines, got %d", len(d.Lines))
	}
	want := [][]string{
		{"model", "context", "block", "weekly", "duration", "cost"},
		{"directory", "worktree", "git", "runtime", "diff", "tasks"},
	}
	for i, line := range d.Lines {
		for j, sc := range line.Segments {
			if sc.Type != want[i][j] {
				t.Errorf("line %d seg %d = %q, want %q", i, j, sc.Type, want[i][j])
			}
		}
	}
}

func TestMergeScalarOverride(t *testing.T) {
	merged, err := Merge(Defaults(), []byte(`{"theme":"nord","separator":" • "}`))
	if err != nil {
		t.Fatal(err)
	}
	if merged.Theme != "nord" {
		t.Errorf("theme = %q, want nord", merged.Theme)
	}
	if merged.Separator != " • " {
		t.Errorf("separator = %q", merged.Separator)
	}
	// Untouched fields keep defaults.
	if merged.Charset != "auto" {
		t.Errorf("charset = %q, want auto", merged.Charset)
	}
	if len(merged.Lines) != 2 {
		t.Errorf("lines should be preserved, got %d", len(merged.Lines))
	}
}

func TestMergeLinesReplace(t *testing.T) {
	merged, err := Merge(Defaults(), []byte(`{"lines":[{"segments":[{"type":"model"}]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(merged.Lines) != 1 || len(merged.Lines[0].Segments) != 1 {
		t.Fatalf("user lines should replace defaults wholesale: %+v", merged.Lines)
	}
}

func TestEnabledDefaultsTrue(t *testing.T) {
	sc := SegmentConfig{Type: "model"}
	if !sc.IsEnabled() {
		t.Error("omitted enabled should default true")
	}
	no := false
	sc.Enabled = &no
	if sc.IsEnabled() {
		t.Error("enabled:false should disable")
	}
}

func TestValidateRejectsUnknownSegment(t *testing.T) {
	bad := []byte(`{"lines":[{"segments":[{"type":"bogus"}]}]}`)
	if err := Validate(bad); err == nil {
		t.Error("unknown segment type should fail schema validation")
	}
}
