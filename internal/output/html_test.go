package output

import (
	"bytes"
	"encoding/json"
	"fmt"
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
		"runtime-verification",
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
	if strings.Contains(rendered, "gradient") {
		t.Error("generated HTML contains a decorative gradient contrary to the product design system")
	}
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

func TestWriteHTMLDoesNotCallUnenforcedWaypointPolicyCovered(t *testing.T) {
	golden := readOutputFixture(t, filepath.Join(
		"..",
		"..",
		"test",
		"fixtures",
		"ambient-basic",
		"golden",
		"ambient-missing.json",
	))
	var rendered bytes.Buffer
	if err := WriteHTML(&rendered, bytes.NewReader(golden)); err != nil {
		t.Fatalf("write HTML: %v", err)
	}
	if !strings.Contains(rendered.String(), "0% — 0/1 workload(s) covered") {
		t.Fatalf("unenforced waypoint policy was not projected as uncovered:\n%s", rendered.String())
	}
	if strings.Contains(rendered.String(), "100% — 1/1 workload(s) covered") {
		t.Fatal("unenforced waypoint policy was misrepresented as authorization coverage")
	}
}

func TestWriteHTMLProjectsEveryVerifiedStatusWithoutBlending(t *testing.T) {
	golden := readOutputFixture(t, filepath.Join(
		"..",
		"..",
		"test",
		"fixtures",
		"sidecar-basic",
		"golden",
		"strict.json",
	))
	var canonical report
	if err := json.Unmarshal(golden, &canonical); err != nil {
		t.Fatalf("decode golden: %v", err)
	}
	canonical.Scan.DataSources.Prometheus.Enabled = true
	base := canonical.WorkloadPostures[0]
	plaintext := true
	share := 0.5
	statuses := []verifiedPosture{
		{Status: "corroborated", Window: "168h", MTLSTrafficShare: floatPointer(1)},
		{
			Status:            "contradicted",
			Window:            "168h",
			MTLSTrafficShare:  &share,
			PlaintextObserved: &plaintext,
			PlaintextSources:  []string{"payments/client"},
		},
		{Status: "no-traffic-observed", Window: "168h"},
		{Status: "unknown", Window: "168h"},
	}
	canonical.WorkloadPostures = make([]canonicalWorkloadPosture, 0, len(statuses))
	for index, verified := range statuses {
		workload := base
		workload.Workload.Name = fmt.Sprintf("workload-%d", index)
		workload.Verified = &verified
		canonical.WorkloadPostures = append(canonical.WorkloadPostures, workload)
	}
	data, err := json.Marshal(canonical)
	if err != nil {
		t.Fatalf("encode canonical report: %v", err)
	}

	var rendered bytes.Buffer
	if err := WriteHTML(&rendered, bytes.NewReader(data)); err != nil {
		t.Fatalf("write HTML: %v", err)
	}
	for _, want := range []string{
		"1 corroborated; 1 contradicted; 1 no traffic observed; 1 unknown or unavailable",
		">corroborated<",
		">contradicted<",
		">no-traffic-observed<",
		">unknown<",
		"50.0%",
		"payments/client",
	} {
		if !strings.Contains(rendered.String(), want) {
			t.Errorf("verified projection missing %q", want)
		}
	}
	if strings.Contains(rendered.String(), "4 workload(s) verified") {
		t.Fatal("non-corroborated runtime states were blended into verified")
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
