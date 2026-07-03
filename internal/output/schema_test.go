package output

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestMinimalReportMatchesSchema(t *testing.T) {
	root := filepath.Join("..", "..")
	schemaPath := filepath.Join(root, "docs", "contracts", "canonical-json-schema.json")
	fixturePath := filepath.Join(root, "test", "fixtures", "reports", "minimal-report.json")

	compiler := jsonschema.NewCompiler()
	schema, err := compiler.Compile(schemaPath)
	if err != nil {
		t.Fatalf("compile canonical schema: %v", err)
	}

	file, err := os.Open(fixturePath)
	if err != nil {
		t.Fatalf("open minimal report fixture: %v", err)
	}
	defer file.Close()

	report, err := jsonschema.UnmarshalJSON(file)
	if err != nil {
		t.Fatalf("decode minimal report fixture: %v", err)
	}

	if err := schema.Validate(report); err != nil {
		t.Fatalf("minimal report does not match canonical schema: %v", err)
	}
}
