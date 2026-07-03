# Posture Scanner — UI kit

Interactive click-through recreation of OpenMeshGuard's flagship product: a **read-only Istio posture
scanner and evidence generator**. Composes the design-system primitives (`window.OpenMeshGuardDesignSystem_65348c`);
it does not re-implement them.

Open `index.html` for the live app.

## Screens & flow
- **Overview** (`Overview.jsx`) — KPI metrics, control-posture coverage list, findings-by-severity, recent critical findings.
- **Findings** (`Findings.jsx`) — filterable findings table (by control), rows drill into detail.
- **Workloads** (`Workloads.jsx`) — mesh-enabled workload inventory with mTLS / AuthZ / GitOps status.
- **Drift** & **Exceptions** (`app.jsx`) — GitOps drift log and exception lifecycle tables.
- **Resource detail** (`ResourceDetail.jsx`) — a finding's control checklist, ownership/metadata, and raw YAML evidence.
- **Evidence** (`Evidence.jsx`) — audit-ready report builder (scope, framework, sections, format).
- **Settings** (`app.jsx`) — scanning toggles and integrations.

Try: click **Run scan** (toast), switch nav, open a finding row, build an evidence report.

## Files
- `index.html` — entry; loads React + Babel + the DS bundle, then the screen scripts.
- `app.jsx` — routing/state, plus Drift / Exceptions / Settings screens.
- `AppShell.jsx` — sidebar + topbar chrome.
- `Icons.jsx` — Lucide-style outline icon set (`<Icon name=… />`), exposed on `window.Icon`.
- `data.jsx` — mock cluster/finding/workload data on `window.OMG_DATA`.

## Notes
- Data is mock. Behavior is cosmetic (no real cluster calls).
- Icons are a Lucide substitution — see the ICONOGRAPHY section in the root `readme.md`.
