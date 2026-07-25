package output

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	nethtml "golang.org/x/net/html"
)

func TestWriteHTMLGoldenHasKeySectionsAndHonestUnknowns(t *testing.T) {
	golden := readOutputFixture(t, filepath.Join(
		"..",
		"..",
		"test",
		"fixtures",
		"sidecar-basic",
		"golden",
		"namespace-role-degraded.json",
	))

	var first bytes.Buffer
	if err := WriteHTML(&first, bytes.NewReader(golden)); err != nil {
		t.Fatalf("write HTML: %v", err)
	}
	var second bytes.Buffer
	if err := WriteHTML(&second, bytes.NewReader(golden)); err != nil {
		t.Fatalf("write HTML a second time: %v", err)
	}
	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Fatal("HTML projection is not deterministic for identical canonical JSON")
	}

	document, err := nethtml.Parse(bytes.NewReader(first.Bytes()))
	if err != nil {
		t.Fatalf("parse generated HTML headlessly: %v", err)
	}
	ids := map[string]bool{}
	var walk func(*nethtml.Node)
	walk = func(node *nethtml.Node) {
		if node.Type == nethtml.ElementNode && node.Data == "script" {
			t.Fatal("self-contained HTML must not contain script elements")
		}
		for _, attribute := range node.Attr {
			switch attribute.Key {
			case "id":
				ids[attribute.Val] = true
			case "src", "href":
				t.Fatalf("self-contained HTML contains external-capable %s=%q", attribute.Key, attribute.Val)
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(document)

	for _, id := range []string{
		"overview",
		"declared-verified-unknown",
		"unknowns",
		"category-grades",
		"classification-coverage",
		"evidence-summary",
		"findings",
	} {
		if !ids[id] {
			t.Errorf("generated HTML missing key section %q", id)
		}
	}
	rendered := first.String()
	for _, text := range []string{
		"Declared / Verified / Unknown",
		"Unknown means OpenMeshGuard could not establish pass or fail",
		"runtime verification unavailable",
		"Permission &amp; evidence summary",
		"Resolution chain",
	} {
		if !strings.Contains(rendered, text) {
			t.Errorf("generated HTML missing %q", text)
		}
	}

	canonical, err := readCanonicalReport(bytes.NewReader(golden))
	if err != nil {
		t.Fatalf("read canonical golden: %v", err)
	}
	for _, item := range canonical.Findings {
		if !ids["finding-"+item.ID] {
			t.Errorf("canonical finding %s is absent from HTML", item.ID)
		}
	}
}

func TestWriteHTMLRejectsNonCanonicalInput(t *testing.T) {
	var rendered bytes.Buffer
	err := WriteHTML(&rendered, strings.NewReader(`{"schemaVersion":"v1alpha1"}`))
	if err == nil || !strings.Contains(err.Error(), `required field "generatedAt"`) {
		t.Fatalf("WriteHTML error = %v, want missing canonical field", err)
	}
}

func readOutputFixture(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	return data
}
