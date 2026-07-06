package config

import (
	"encoding/json"
	"testing"
)

func TestPresets(t *testing.T) {
	for _, name := range PresetNames() {
		p := Preset(name)
		if p == nil {
			t.Fatalf("preset %q returned nil", name)
		}
		if len(p.Lines) == 0 {
			t.Errorf("preset %q has no lines", name)
		}
		// Every preset must validate against the schema.
		data, err := json.Marshal(p)
		if err != nil {
			t.Fatalf("marshal %q: %v", name, err)
		}
		if err := Validate(data); err != nil {
			t.Errorf("preset %q fails schema validation: %v", name, err)
		}
	}
	if Preset("nope") != nil {
		t.Error("unknown preset should return nil")
	}
}
