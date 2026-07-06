package config

import (
	"bytes"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v6"

	vitals "github.com/sayantan/vitals"
)

// Validate checks raw config JSON against the embedded JSON Schema. It returns a
// descriptive error on failure. This is intentionally off the render hot path —
// `vitals doctor` and the TUI use it, the renderer does not.
func Validate(data []byte) error {
	schemaDoc, err := jsonschema.UnmarshalJSON(bytes.NewReader(mustSchema()))
	if err != nil {
		return fmt.Errorf("load schema: %w", err)
	}
	c := jsonschema.NewCompiler()
	if err := c.AddResource("vitals.schema.json", schemaDoc); err != nil {
		return fmt.Errorf("add schema: %w", err)
	}
	sch, err := c.Compile("vitals.schema.json")
	if err != nil {
		return fmt.Errorf("compile schema: %w", err)
	}
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("parse config: %w", err)
	}
	if err := sch.Validate(inst); err != nil {
		return err
	}
	return nil
}

func mustSchema() []byte {
	data, err := vitals.Schema.ReadFile("schema/vitals.schema.json")
	if err != nil {
		// Embedded; should never fail.
		return []byte(`{}`)
	}
	return data
}
