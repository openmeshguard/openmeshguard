package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

const maxCanonicalReportBytes = 64 << 20

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
