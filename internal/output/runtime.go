package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/openmeshguard/openmeshguard/internal/engine"
	"github.com/openmeshguard/openmeshguard/internal/resolver"
)

// RuntimeInput supplies optional runtime evidence separately from the frozen
// ScanInput and resolver interfaces. Keys use namespace/name, within one cluster.
type RuntimeInput struct {
	Enabled    bool
	URL        string
	Lookback   string
	DegradedTo string
	Verified   map[string]map[string]any
}

// WriteScanJSONWithRuntime preserves the existing declared output assembly and
// adds only schema-defined runtime evidence. Credentials are never accepted here.
func WriteScanJSONWithRuntime(w io.Writer, input ScanInput, packs []engine.Pack, evaluated engine.Result, runtime RuntimeInput) error {
	report := buildReport(input, packs, evaluated)
	if runtime.Enabled {
		report.Scan.DataSources.Prometheus = prometheus{Enabled: true, URL: runtime.URL, Lookback: runtime.Lookback}
		if runtime.DegradedTo != "" {
			report.Scan.DataSources.Prometheus.DegradedTo = &runtime.DegradedTo
		}
		for i := range report.WorkloadPostures {
			posture := &report.WorkloadPostures[i]
			if posture.DataPlaneMode == resolver.ModeNotApplicable {
				continue
			}
			v := runtime.Verified[posture.Workload.Namespace+"/"+posture.Workload.Name]
			if v == nil {
				continue
			}
			status, _ := v["status"].(string)
			window, _ := v["window"].(string)
			verified := &verifiedPosture{Status: status, Window: window}
			if share, ok := v["mtlsTrafficShare"].(float64); ok {
				verified.MTLSTrafficShare = &share
			}
			if plain, ok := v["plaintextObserved"].(bool); ok {
				verified.PlaintextObserved = &plain
			}
			posture.Verified = verified
		}
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		return fmt.Errorf("encode canonical report: %w", err)
	}
	return nil
}
