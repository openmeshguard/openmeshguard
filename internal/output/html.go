package output

import (
	"fmt"
	"html/template"
	"io"
	"strings"
)

type summaryCell struct {
	Value string
	Tone  string
}

type summaryRow struct {
	Area     string
	Declared summaryCell
	Verified summaryCell
	Unknown  summaryCell
}

type verificationSummary struct {
	Corroborated      int
	Contradicted      int
	NoTrafficObserved int
	Unknown           int
	Unavailable       int
}

type htmlReportView struct {
	Report              report
	SummaryRows         []summaryRow
	Verification        verificationSummary
	OpenFindings        int
	UnknownFindings     []finding
	ExceptedFindings    int
	DeniedPermissions   int
	Unclassified        int
	UnverifiedWorkloads int
}

// WriteHTML renders a self-contained static report from canonical JSON.
func WriteHTML(writer io.Writer, reader io.Reader) error {
	canonical, err := readCanonicalReport(reader)
	if err != nil {
		return err
	}
	view := buildHTMLView(canonical)
	tmpl, err := template.New("report").Funcs(template.FuncMap{
		"join":           strings.Join,
		"percent":        formatPercent,
		"score":          formatScore,
		"statusClass":    statusClass,
		"verifiedClass":  verifiedClass,
		"resource":       formatResource,
		"optionalString": optionalString,
		"optionalBool":   optionalBool,
		"add":            addInts,
	}).Parse(htmlReportTemplate)
	if err != nil {
		return fmt.Errorf("parse HTML report template: %w", err)
	}
	if err := tmpl.Execute(writer, view); err != nil {
		return fmt.Errorf("render HTML report: %w", err)
	}
	return nil
}

func buildHTMLView(canonical report) htmlReportView {
	view := htmlReportView{Report: canonical}
	for _, item := range canonical.Findings {
		switch item.Status {
		case "open":
			view.OpenFindings++
		case "unknown":
			view.UnknownFindings = append(view.UnknownFindings, item)
		case "excepted":
			view.ExceptedFindings++
		}
	}
	for _, item := range canonical.PermissionSummary {
		if !item.Granted {
			view.DeniedPermissions++
		}
	}
	if canonical.Inventory.Classification != nil {
		view.Unclassified = canonical.Inventory.Classification.NamespacesUnclassified
	}
	for _, workload := range canonical.WorkloadPostures {
		if workload.DataPlaneMode != "not-applicable" && workload.Verified == nil {
			view.UnverifiedWorkloads++
		}
	}
	view.Verification = summarizeVerification(canonical.WorkloadPostures)
	view.SummaryRows = canonicalSummaryRows(canonical)
	return view
}

func canonicalSummaryRows(canonical report) []summaryRow {
	meshWorkloads := 0
	strictWorkloads := 0
	mtlsUnknown := 0
	authzCovered := 0
	authzUnknown := 0
	owned := 0
	namespaceAuthz := map[string][]string{}

	for _, workload := range canonical.WorkloadPostures {
		if workload.DataPlaneMode == "not-applicable" {
			continue
		}
		meshWorkloads++
		switch workload.MTLS.Effective {
		case "strict":
			strictWorkloads++
		case "unknown":
			mtlsUnknown++
		}
		switch workload.Authorization.Effective {
		case "unknown":
			authzUnknown++
		case "default-deny-explicit-allow", "allow-only", "deny-present":
			authzCovered++
		case "no-policy", "waypoint-policy-unenforced", "not-in-mesh":
			// Known but not enforced authorization is not coverage.
		default:
			// The frozen schema rejects new values before projection. Keep a
			// conservative fallback if this switch is extended incorrectly.
			authzUnknown++
		}
		namespaceAuthz[workload.Workload.Namespace] = append(
			namespaceAuthz[workload.Workload.Namespace],
			string(workload.Authorization.Effective),
		)
		if workload.Owner != nil && *workload.Owner != "" {
			owned++
		}
	}

	defaultDenyNamespaces := 0
	unknownAuthzNamespaces := 0
	for _, postures := range namespaceAuthz {
		allDefaultDeny := len(postures) > 0
		hasUnknown := false
		for _, posture := range postures {
			allDefaultDeny = allDefaultDeny && posture == "default-deny-explicit-allow"
			hasUnknown = hasUnknown || posture == "unknown"
		}
		if allDefaultDeny {
			defaultDenyNamespaces++
		}
		if hasUnknown {
			unknownAuthzNamespaces++
		}
	}

	classified := 0
	unclassified := 0
	if canonical.Inventory.Classification != nil {
		classified = canonical.Inventory.Classification.NamespacesClassified
		unclassified = canonical.Inventory.Classification.NamespacesUnclassified
	}
	classificationTotal := classified + unclassified
	verifiedCell := summaryCell{
		Value: fmt.Sprintf("Unknown — no telemetry access; runtime verification unavailable for %d workload(s)", meshWorkloads),
		Tone:  "unknown",
	}
	verification := summarizeVerification(canonical.WorkloadPostures)
	explicitVerification := verification.Corroborated +
		verification.Contradicted +
		verification.NoTrafficObserved +
		verification.Unknown
	if canonical.Scan.DataSources.Prometheus.Enabled || explicitVerification > 0 {
		verifiedCell = verificationSummaryCell(verification)
	}

	return []summaryRow{
		{
			Area:     "Strict mTLS (effective, per workload)",
			Declared: coverageCell(strictWorkloads, meshWorkloads, "workload(s) strict"),
			Verified: verifiedCell,
			Unknown:  countCell(mtlsUnknown, "workload posture(s) unknown"),
		},
		{
			Area:     "Explicit authorization coverage",
			Declared: coverageCell(authzCovered, meshWorkloads, "workload(s) covered"),
			Verified: summaryCell{Value: "— config-only control", Tone: "muted"},
			Unknown:  countCell(authzUnknown, "workload posture(s) unknown"),
		},
		{
			Area:     "Default-deny posture",
			Declared: coverageCell(defaultDenyNamespaces, len(namespaceAuthz), "namespace(s) default deny"),
			Verified: summaryCell{Value: "— config-only control", Tone: "muted"},
			Unknown:  countCell(unknownAuthzNamespaces, "namespace posture(s) unknown"),
		},
		{
			Area:     "Environment classification coverage",
			Declared: coverageCell(classified, classificationTotal, "namespace(s) classified"),
			Verified: summaryCell{Value: "— context metric", Tone: "muted"},
			Unknown:  countCell(unclassified, "namespace(s) unclassified"),
		},
		{
			Area:     "Ownership metadata coverage",
			Declared: coverageCell(owned, meshWorkloads, "mesh workload(s) owned"),
			Verified: summaryCell{Value: "— context metric", Tone: "muted"},
			Unknown:  countCell(meshWorkloads-owned, "workload owner(s) unknown"),
		},
	}
}

func summarizeVerification(workloads []canonicalWorkloadPosture) verificationSummary {
	var summary verificationSummary
	for _, workload := range workloads {
		if workload.DataPlaneMode == "not-applicable" {
			continue
		}
		if workload.Verified == nil {
			summary.Unavailable++
			continue
		}
		switch workload.Verified.Status {
		case "corroborated":
			summary.Corroborated++
		case "contradicted":
			summary.Contradicted++
		case "no-traffic-observed":
			summary.NoTrafficObserved++
		case "unknown":
			summary.Unknown++
		default:
			// Complete canonical validation rejects this case. Treat it as
			// unavailable if the schema and this projection ever drift.
			summary.Unknown++
		}
	}
	return summary
}

func verificationSummaryCell(summary verificationSummary) summaryCell {
	tone := "known"
	if summary.Contradicted > 0 {
		tone = "risk"
	} else if summary.NoTrafficObserved+summary.Unknown+summary.Unavailable > 0 {
		tone = "unknown"
	} else if summary.Corroborated == 0 {
		tone = "muted"
	}
	return summaryCell{
		Value: fmt.Sprintf(
			"%d corroborated; %d contradicted; %d no traffic observed; %d unknown or unavailable",
			summary.Corroborated,
			summary.Contradicted,
			summary.NoTrafficObserved,
			summary.Unknown+summary.Unavailable,
		),
		Tone: tone,
	}
}

func coverageCell(value, total int, label string) summaryCell {
	if total == 0 {
		return summaryCell{Value: "Unknown — no evaluable targets", Tone: "unknown"}
	}
	return summaryCell{
		Value: fmt.Sprintf("%.0f%% — %d/%d %s", float64(value)/float64(total)*100, value, total, label),
		Tone:  "declared",
	}
}

func countCell(value int, label string) summaryCell {
	return summaryCell{Value: fmt.Sprintf("%d %s", value, label), Tone: toneForUnknown(value)}
}

func toneForUnknown(value int) string {
	if value > 0 {
		return "unknown"
	}
	return "known"
}

func formatPercent(value *float64) string {
	if value == nil {
		return "unknown"
	}
	return fmt.Sprintf("%.1f%%", *value*100)
}

func formatScore(value *float64) string {
	if value == nil {
		return "unknown"
	}
	return fmt.Sprintf("%.1f", *value)
}

func statusClass(status string) string {
	switch status {
	case "open":
		return "risk"
	case "unknown":
		return "unknown"
	case "excepted":
		return "excepted"
	default:
		return "muted"
	}
}

func verifiedClass(status string) string {
	switch status {
	case "corroborated":
		return "known"
	case "contradicted":
		return "risk"
	case "no-traffic-observed", "unknown":
		return "unknown"
	default:
		return "muted"
	}
}

func formatResource(value resourceRef) string {
	parts := make([]string, 0, 3)
	if value.Namespace != "" {
		parts = append(parts, value.Namespace)
	}
	parts = append(parts, value.Kind, value.Name)
	return strings.Join(parts, " / ")
}

func optionalString(value *string) string {
	if value == nil || *value == "" {
		return "unknown"
	}
	return *value
}

func optionalBool(value *bool) string {
	if value == nil {
		return "unknown"
	}
	return fmt.Sprintf("%t", *value)
}

func addInts(left, right int) int {
	return left + right
}

const htmlReportTemplate = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'unsafe-inline'; img-src data:">
  <title>OpenMeshGuard — {{.Report.Scan.ClusterContext}}</title>
  <style>
    :root { --ink:#131c2b; --body:#334155; --muted:#64748b; --page:#f8fafc; --panel:#ffffff; --sunken:#f1f5f9; --line:#e5eaf1; --line-strong:#cbd5e1; --brand:#3560d1; --risk:#b91c1c; --risk-bg:#fef2f2; --risk-line:#fecaca; --unknown:#b45309; --unknown-bg:#fffbeb; --unknown-line:#fde68a; --ok:#047857; --ok-bg:#ecfdf5; --ok-line:#a7f3d0; --violet:#1d4ed8; }
    * { box-sizing: border-box; }
    body { margin:0; background:var(--page); color:var(--body); font:14px/1.5 "IBM Plex Sans",-apple-system,BlinkMacSystemFont,"Segoe UI",system-ui,sans-serif; }
    main { width:min(1180px,calc(100% - 32px)); margin:0 auto; padding:40px 0 72px; }
    h1,h2,h3,p { margin-top:0; } h1 { color:var(--ink); font-size:30px; font-weight:600; line-height:1.2; letter-spacing:-.01em; max-width:800px; }
    h2 { color:var(--ink); margin-top:48px; font-size:20px; font-weight:600; letter-spacing:-.01em; } h3 { color:var(--ink); font-size:16px; font-weight:600; }
    .eyebrow { color:var(--brand); font-weight:600; letter-spacing:.06em; text-transform:uppercase; font-size:11px; }
    .lede { color:var(--muted); max-width:760px; font-size:15px; }
    .grid { display:grid; grid-template-columns:repeat(4,minmax(0,1fr)); gap:12px; margin:28px 0; }
    .card,.panel { background:var(--panel); border:1px solid var(--line); border-radius:8px; box-shadow:0 1px 2px rgba(11,18,32,.06); padding:18px; }
    .card strong { color:var(--ink); display:block; font:600 30px/1.2 "IBM Plex Mono",ui-monospace,"SFMono-Regular",Menlo,monospace; margin:.45rem 0; }
    .card span,.meta,.muted { color:var(--muted); }
    .unknown-card { background:var(--unknown-bg); border-color:var(--unknown-line); }
    .risk-card { background:var(--risk-bg); border-color:var(--risk-line); }
    table { width:100%; border-collapse:collapse; background:var(--panel); border:1px solid var(--line); }
    th,td { padding:12px 14px; text-align:left; border-bottom:1px solid var(--line); vertical-align:top; }
    th { color:var(--muted); font-size:.78rem; text-transform:uppercase; letter-spacing:.08em; }
    tr:last-child td { border-bottom:0; } .unknown { color:var(--unknown); } .risk { color:var(--risk); } .known,.declared { color:var(--ok); } .excepted { color:var(--violet); }
    .pill { display:inline-flex; border:1px solid currentColor; border-radius:999px; padding:2px 8px; font-size:11px; font-weight:600; text-transform:uppercase; letter-spacing:.06em; }
    .grade { color:var(--ink); font:600 20px/1.2 "IBM Plex Mono",ui-monospace,"SFMono-Regular",Menlo,monospace; }
    details { background:var(--panel); border:1px solid var(--line); border-radius:8px; margin:10px 0; overflow:hidden; }
    summary { cursor:pointer; list-style:none; padding:14px 16px; display:flex; gap:10px; align-items:center; flex-wrap:wrap; }
    summary:focus-visible { outline:3px solid rgba(53,96,209,.25); outline-offset:2px; }
    summary::-webkit-details-marker { display:none; } summary::before { content:"+"; color:var(--ok); font-weight:900; }
    details[open] summary::before { content:"−"; } .detail-body { border-top:1px solid var(--line); padding:16px; }
    .chain { margin:12px 0 0; padding:0; list-style:none; counter-reset:chain; }
    .chain li { position:relative; margin-left:14px; padding:0 0 16px 24px; border-left:1px solid var(--line); }
    .chain li::before { counter-increment:chain; content:counter(chain); position:absolute; left:-13px; width:24px; height:24px; border-radius:50%; background:var(--ok-bg); border:1px solid var(--ok-line); color:var(--ok); display:grid; place-items:center; font-size:11px; font-weight:700; }
    code,pre,.machine { font-family:"IBM Plex Mono",ui-monospace,"SFMono-Regular",Menlo,monospace; } pre { white-space:pre-wrap; overflow-wrap:anywhere; background:var(--sunken); border:1px solid var(--line); padding:12px; border-radius:6px; }
    footer { margin-top:48px; padding-top:18px; border-top:1px solid var(--line); color:var(--muted); }
    @media (max-width:800px) { .grid { grid-template-columns:repeat(2,minmax(0,1fr)); } .wide { overflow-x:auto; } }
    @media (max-width:520px) { main { width:min(100% - 20px,1180px); padding-top:24px; } .grid { grid-template-columns:1fr; } th,td { padding:10px; } }
  </style>
</head>
<body>
<main>
  <header id="overview">
    <p class="eyebrow">OpenMeshGuard / effective posture</p>
    <h1>Are you actually protected by the mesh?</h1>
    <p class="lede">A canonical-JSON projection for <strong>{{.Report.Scan.ClusterContext}}</strong>. Declared posture, runtime verification, and unavailable evidence remain separate throughout this report.</p>
    <p class="meta">Generated {{.Report.GeneratedAt}} · scanner {{.Report.Scanner.Version}} · resolver {{.Report.Scanner.ResolverVersion}}</p>
  </header>

  <section class="grid" aria-label="Report status">
    <article class="card risk-card"><span>Open findings</span><strong>{{.OpenFindings}}</strong><small>Confirmed control failures</small></article>
    <article class="card unknown-card"><span>Unknown findings</span><strong>{{len .UnknownFindings}}</strong><small>Never counted as pass or fail</small></article>
    <article class="card unknown-card"><span>Evidence gaps</span><strong>{{.DeniedPermissions}}</strong><small>Denied permission paths</small></article>
    <article class="card"><span>Overall score</span><strong>{{score .Report.Scores.Overall}}</strong><small>Weighted from evaluable categories</small></article>
  </section>

  <section id="declared-verified-unknown">
    <h2>Declared / Verified / Unknown</h2>
    <p class="muted">Each value is projected from canonical workload posture, runtime verification, and classification fields. Missing telemetry stays visible.</p>
    <div class="wide"><table>
      <thead><tr><th>Control area</th><th>Declared</th><th>Verified</th><th>Unknown</th></tr></thead>
      <tbody>{{range .SummaryRows}}<tr><td><strong>{{.Area}}</strong></td><td class="{{.Declared.Tone}}">{{.Declared.Value}}</td><td class="{{.Verified.Tone}}">{{.Verified.Value}}</td><td class="{{.Unknown.Tone}}">{{.Unknown.Value}}</td></tr>{{end}}</tbody>
    </table></div>
  </section>

  <section id="runtime-verification">
    <h2>Runtime verification</h2>
    <p class="muted">Canonical telemetry states remain distinct. No traffic observed and unavailable evidence are not corroboration.</p>
    <div class="grid">
      <article class="card"><span>Corroborated</span><strong class="known">{{.Verification.Corroborated}}</strong></article>
      <article class="card risk-card"><span>Contradicted</span><strong class="risk">{{.Verification.Contradicted}}</strong></article>
      <article class="card unknown-card"><span>No traffic observed</span><strong class="unknown">{{.Verification.NoTrafficObserved}}</strong></article>
      <article class="card unknown-card"><span>Unknown or unavailable</span><strong class="unknown">{{add .Verification.Unknown .Verification.Unavailable}}</strong></article>
    </div>
    <div class="wide"><table>
      <thead><tr><th>Workload</th><th>Status</th><th>Window</th><th>mTLS traffic share</th><th>Plaintext observed</th><th>Plaintext sources</th></tr></thead>
      <tbody>{{range .Report.WorkloadPostures}}<tr>
        <td class="machine">{{.Workload.Namespace}} / {{.Workload.Kind}} / {{.Workload.Name}}</td>
        {{with .Verified}}
        <td><span class="pill {{verifiedClass .Status}}">{{.Status}}</span></td><td class="machine">{{.Window}}</td><td class="machine">{{percent .MTLSTrafficShare}}</td><td class="machine">{{optionalBool .PlaintextObserved}}</td><td class="machine">{{join .PlaintextSources ", "}}</td>
        {{else}}
        <td><span class="pill unknown">unavailable</span></td><td class="unknown">unknown</td><td class="unknown">unknown</td><td class="unknown">unknown</td><td class="unknown">unknown</td>
        {{end}}
      </tr>{{else}}<tr><td colspan="6" class="unknown">No workload posture is present in the canonical report.</td></tr>{{end}}</tbody>
    </table></div>
  </section>

  <section id="workload-postures">
    <h2>Declared workload posture &amp; resolution chains</h2>
    <p class="muted">Effective mTLS and authorization conclusions are shown with the complete ordered canonical chains that produced them.</p>
    {{range .Report.WorkloadPostures}}
    <details>
      <summary>
        <strong class="machine">{{.Workload.Namespace}} / {{.Workload.Kind}} / {{.Workload.Name}}</strong>
        <span class="pill">mTLS {{.MTLS.Effective}}</span>
        <span class="pill">authorization {{.Authorization.Effective}}</span>
        <span class="pill">{{.DataPlaneMode}}</span>
      </summary>
      <div class="detail-body">
        <p><strong>Environment:</strong> {{optionalString .Environment}} · <strong>Owner:</strong> {{optionalString .Owner}}</p>
        <h3>mTLS resolution chain</h3>
        {{if .MTLS.Chain}}<ol class="chain">{{range .MTLS.Chain}}<li><strong>{{.Kind}}{{if .Name}} / {{.Name}}{{end}}</strong>{{if .Namespace}} in {{.Namespace}}{{end}}{{if .Field}} · <code>{{.Field}}</code>{{end}}<br><span class="muted">{{.Effect}}</span></li>{{end}}</ol>{{else}}<p class="unknown">No mTLS resolution chain is present in canonical JSON.</p>{{end}}
        <h3>Authorization resolution chain</h3>
        {{if .Authorization.Chain}}<ol class="chain">{{range .Authorization.Chain}}<li><strong>{{.Kind}}{{if .Name}} / {{.Name}}{{end}}</strong>{{if .Namespace}} in {{.Namespace}}{{end}}{{if .Field}} · <code>{{.Field}}</code>{{end}}<br><span class="muted">{{.Effect}}</span></li>{{end}}</ol>{{else}}<p class="unknown">No authorization resolution chain is present in canonical JSON.</p>{{end}}
      </div>
    </details>
    {{else}}<div class="panel unknown-card"><strong>No workload posture is present in the canonical report.</strong></div>{{end}}
  </section>

  <section id="unknowns">
    <h2>Unknown posture &amp; missing evidence</h2>
    <div class="panel unknown-card">
      <p><strong>{{len .UnknownFindings}} unknown finding(s), {{.DeniedPermissions}} denied permission(s), {{.Unclassified}} unclassified namespace(s), and {{.UnverifiedWorkloads}} workload(s) without runtime verification.</strong></p>
      <p class="muted">Unknown means OpenMeshGuard could not establish pass or fail. It is not a clean bill of health.</p>
    </div>
    {{range .UnknownFindings}}
    <details>
      <summary><span class="pill unknown">unknown</span><strong>{{.ControlID}}</strong> {{.Title}}</summary>
      <div class="detail-body"><p>{{.Reasoning}}</p><p class="unknown"><strong>Missing:</strong> {{.UnknownReason}}</p></div>
    </details>
    {{else}}<p class="muted">No finding is marked unknown in this report.</p>{{end}}
  </section>

  <section id="category-grades">
    <h2>Category grades</h2>
    <div class="wide"><table>
      <thead><tr><th>Category</th><th>Grade</th><th>Pass rate</th><th>Evaluated</th><th>Unknown</th></tr></thead>
      <tbody>{{range .Report.Scores.Categories}}<tr><td>{{.Category}}</td><td class="grade">{{.Grade}}</td><td>{{percent .PassRate}}</td><td>{{.Evaluated}}</td><td class="{{if .Unknown}}unknown{{end}}">{{.Unknown}}</td></tr>{{else}}<tr><td colspan="5" class="unknown">No category had evaluable controls.</td></tr>{{end}}</tbody>
    </table></div>
  </section>

  <section id="classification-coverage">
    <h2>Classification coverage</h2>
    {{with .Report.Inventory.Classification}}
    <div class="grid">
      <article class="card"><span>Classified namespaces</span><strong>{{.NamespacesClassified}}</strong></article>
      <article class="card unknown-card"><span>Unclassified namespaces</span><strong>{{.NamespacesUnclassified}}</strong></article>
    </div>
    <div class="panel"><h3>By environment</h3>{{range $name,$count := .ByEnvironment}}<p><strong>{{$name}}</strong> — {{$count}}</p>{{end}}</div>
    {{else}}<div class="panel unknown-card"><strong>Unknown — classification summary is absent from the canonical report.</strong></div>{{end}}
  </section>

  <section id="evidence-summary">
    <h2>Permission &amp; evidence summary</h2>
    <p>Kubernetes API: <strong>{{if .Report.Scan.DataSources.KubernetesAPI}}enabled{{else}}unavailable{{end}}</strong> · Prometheus: <strong class="{{if not .Report.Scan.DataSources.Prometheus.Enabled}}unknown{{end}}">{{if .Report.Scan.DataSources.Prometheus.Enabled}}enabled{{else}}not enabled{{end}}</strong></p>
    {{if .Report.Scan.DataSources.Prometheus.DegradedTo}}<p class="unknown">Telemetry query cost reduced verification to {{.Report.Scan.DataSources.Prometheus.DegradedTo}}. Requested lookback: {{.Report.Scan.DataSources.Prometheus.Lookback}}.</p>{{end}}
    <div class="wide"><table>
      <thead><tr><th>API group / resource</th><th>Verbs</th><th>Granted</th><th>Impact</th><th>Affected controls</th></tr></thead>
      <tbody>{{range .Report.PermissionSummary}}<tr><td>{{.APIGroup}} / {{.Resource}}</td><td>{{join .Verbs ", "}}</td><td><span class="pill {{if .Granted}}known{{else}}unknown{{end}}">{{if .Granted}}yes{{else}}no{{end}}</span></td><td>{{.Impact}}</td><td>{{join .AffectedControls ", "}}</td></tr>{{else}}<tr><td colspan="5" class="muted">No permission attempts were recorded.</td></tr>{{end}}</tbody>
    </table></div>
  </section>

  <section id="findings">
    <h2>All canonical findings</h2>
    <p class="muted">{{len .Report.Findings}} total · {{.OpenFindings}} open · {{len .UnknownFindings}} unknown · {{.ExceptedFindings}} excepted. Not-applicable findings remain visible for projection fidelity.</p>
    {{range .Report.Findings}}
    <details id="finding-{{.ID}}">
      <summary><span class="pill {{statusClass .Status}}">{{.Status}}</span><span class="pill {{statusClass .Status}}">{{.Severity}}</span><strong>{{.ControlID}}</strong> {{.Title}}</summary>
      <div class="detail-body">
        <p>{{.Reasoning}}</p>
        {{if .UnknownReason}}<p class="unknown"><strong>Unknown reason:</strong> {{.UnknownReason}}</p>{{end}}
        <p><strong>Evidence:</strong> {{.EvidenceType}} / {{.Confidence}}{{if .EvidenceSources}} · {{join .EvidenceSources ", "}}{{end}}</p>
        <p><strong>Resources:</strong> {{range $index,$item := .Resources}}{{if $index}}; {{end}}{{resource $item}}{{end}}</p>
        {{with .Exception}}<p class="excepted"><strong>Exception {{.ID}}</strong> · expires {{.ExpiresAt}} · approver {{.Approver}} · ticket {{.Ticket}}{{if .Expired}} · expired{{end}}</p>{{end}}
        {{if .ResolutionChain}}<h3>Resolution chain</h3><ol class="chain">{{range .ResolutionChain}}<li><strong>{{.Kind}}{{if .Name}} / {{.Name}}{{end}}</strong>{{if .Namespace}} in {{.Namespace}}{{end}}{{if .Field}} · <code>{{.Field}}</code>{{end}}<br><span class="muted">{{.Effect}}</span></li>{{end}}</ol>{{end}}
        {{with .Remediation}}<h3>Remediation</h3><p>{{.Guidance}}</p>{{if .SuggestedYAML}}<pre>{{.SuggestedYAML}}</pre>{{end}}{{end}}
      </div>
    </details>
    {{else}}<div class="panel"><strong>No findings are present in the canonical report.</strong></div>{{end}}
  </section>

  <footer>Generated locally from OpenMeshGuard canonical JSON. This file contains inline CSS only and performs no network requests.</footer>
</main>
</body>
</html>
`
