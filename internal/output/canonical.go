package output

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

const maxCanonicalReportBytes = 64 << 20

const canonicalSchemaResource = "https://openmeshguard.io/schemas/report/v1alpha1.json"

//go:embed testdata/canonical-json-schema.json
var embeddedCanonicalSchema []byte

var compileCanonicalSchema = sync.OnceValues(func() (*jsonschema.Schema, error) {
	document, err := jsonschema.UnmarshalJSON(bytes.NewReader(embeddedCanonicalSchema))
	if err != nil {
		return nil, fmt.Errorf("decode embedded canonical schema: %w", err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(canonicalSchemaResource, document); err != nil {
		return nil, fmt.Errorf("register embedded canonical schema: %w", err)
	}
	schema, err := compiler.Compile(canonicalSchemaResource)
	if err != nil {
		return nil, fmt.Errorf("compile embedded canonical schema: %w", err)
	}
	return schema, nil
})

func readCanonicalReport(reader io.Reader) (report, error) {
	data, err := io.ReadAll(io.LimitReader(reader, maxCanonicalReportBytes+1))
	if err != nil {
		return report{}, fmt.Errorf("read canonical report: %w", err)
	}
	if len(data) > maxCanonicalReportBytes {
		return report{}, fmt.Errorf("read canonical report: input exceeds %d bytes", maxCanonicalReportBytes)
	}

	var topLevel map[string]json.RawMessage
	if err := json.Unmarshal(data, &topLevel); err != nil {
		return report{}, fmt.Errorf("decode canonical report: %w", err)
	}
	for _, field := range []string{
		"schemaVersion",
		"generatedAt",
		"scanner",
		"scan",
		"permissionSummary",
		"inventory",
		"workloadPostures",
		"findings",
		"scores",
	} {
		if _, exists := topLevel[field]; !exists {
			return report{}, fmt.Errorf("decode canonical report: required field %q is missing", field)
		}
	}

	schema, err := compileCanonicalSchema()
	if err != nil {
		return report{}, err
	}
	rawReport, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		return report{}, fmt.Errorf("decode canonical report for validation: %w", err)
	}
	if err := schema.Validate(rawReport); err != nil {
		return report{}, fmt.Errorf("validate canonical report: %w", err)
	}

	var decoded report
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&decoded); err != nil {
		return report{}, fmt.Errorf("decode canonical report: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return report{}, fmt.Errorf("decode canonical report: multiple JSON values are not allowed")
		}
		return report{}, fmt.Errorf("decode canonical report trailing data: %w", err)
	}
	if decoded.SchemaVersion != schemaVersion {
		return report{}, fmt.Errorf(
			"decode canonical report: schemaVersion %q is unsupported; want %q",
			decoded.SchemaVersion,
			schemaVersion,
		)
	}
	return decoded, nil
}
