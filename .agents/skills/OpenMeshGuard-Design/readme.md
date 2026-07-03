# OpenMeshGuard — Design System

> Move from assumed mesh security to **verified Istio posture**.

This repository is the OpenMeshGuard brand + product design system: tokens, foundations, reusable
React components, and a full UI kit for the flagship **Posture Scanner & Report Generator**.

---

## 1. Company & product context

**OpenMeshGuard** is an open-core **service-mesh governance platform**, starting with Istio. It helps
platform, security, and risk teams *continuously verify* mesh security posture across clusters:
mTLS enforcement, authorization coverage, exposure, ownership, exceptions, drift, lifecycle, and
compliance evidence.

Positioning — what it **is** and **is not**:
- **Is:** a governance and *evidence* layer that sits **above** the mesh. A read-only posture scanner
  and audit-ready report generator.
- **Is not:** another Kiali, not a mesh distribution, not a generic observability dashboard.

The product answers questions enterprises actually get audited on:
- Are production namespaces enforcing **strict mTLS**?
- Which mesh-enabled apps have **no AuthorizationPolicy**?
- Which Gateways / VirtualServices create **exposure risk**?
- Which Istio resources are **missing owner / app / env / repo** metadata?
- Which policies **drifted** in-cluster vs the GitOps source of truth?
- Which teams have **active / expired / missing exceptions**?
- Which workloads **block** an Istio or ambient mesh migration?
- Can teams **export audit-ready evidence** without spreadsheets?

**Audience:** platform engineers, security engineers, and GRC / risk teams. Technical, skeptical,
evidence-driven. They value clarity, correctness, and traceability over polish.

### Design north star
**Readability and utility first.** This is a compliance instrument, not a marketing surface. Data density,
legibility, and trustworthy status semantics beat decoration every time. No hype, no gradients-for-
gradients'-sake, no illustration flourish. Every pixel should help someone verify or prove something.

### Sources
This design system was created **from the written product brief only** — no codebase, Figma file, or
existing brand assets were provided. All visual decisions (logo, palette, type, components) are original
and should be treated as a **v1 proposal** open to iteration. If an existing brand kit or Istio-facing
product codebase exists, share it and this system will be reconciled against it.

---

## 2. Content fundamentals (voice & copy)

The voice is **precise, calm, and evidence-oriented** — a trusted auditor, not a salesperson.

- **Person:** Address the user as **you**; the product/team is **we** sparingly. Findings are stated
  impersonally and factually ("3 namespaces are not enforcing strict mTLS").
- **Tone:** Direct, declarative, unhurried. State the fact, then the risk, then the action. Never alarmist
  even when reporting critical findings — severity is carried by the status system, not by exclamation.
- **Casing:** Sentence case for all UI text, headings, and buttons ("Export evidence", not "Export
  Evidence"). Uppercase is reserved for small **eyebrow labels** and table column headers, with letter-
  spacing. Istio resource kinds keep their real casing: `AuthorizationPolicy`, `PeerAuthentication`,
  `VirtualService`, `Gateway`.
- **Numbers & identifiers:** Always monospace for counts-in-context, resource names, namespaces, hashes,
  and versions (`payments-api`, `prod-eu-1`, `sha:9f2c…`). Percentages and coverage read as "82% mTLS
  coverage".
- **Terminology (house terms):** *posture*, *finding*, *coverage*, *exposure*, *drift*, *exception*,
  *evidence*, *enforced / permissive / disabled*, *owned / unowned*, *scan*, *control*. Prefer "finding"
  over "issue/alert"; "evidence" over "report/proof"; "exception" over "waiver".
- **Status language:** `Pass` / `Warn` / `Fail` for control outcomes; `Enforced` / `Permissive` /
  `Disabled` for mTLS mode; `Active` / `Expiring` / `Expired` for exceptions; `In sync` / `Drifted` for
  GitOps.
- **No emoji, ever.** No exclamation marks in product copy. No jokes in findings. Microcopy can be warm
  ("Nothing to prove yet — run your first scan") but stays understated.
- **Examples:**
  - Empty state: "No findings in this view. Adjust filters or run a new scan."
  - Critical finding title: "Ingress Gateway exposes an internal service to the public"
  - CTA: "Export evidence (PDF)" · "Run scan" · "Assign owner" · "Request exception"

---

## 3. Visual foundations

**Overall feel:** a serious, screen-first governance console. Cool, neutral, calm — with color used almost
exclusively to communicate *status*, never for decoration.

### Color
- **Neutrals** are cool **slate** (`--slate-*`), from near-white page (`#f8fafc`) to deep ink (`#0b1220`).
  The majority of every screen is slate + white.
- **Brand** is **guard blue** (`--brand-500 #3560d1`) — used for primary actions, links, active nav, and
  the logo. Confident, institutional, not neon.
- **Verified accent** is **emerald** (`--emerald-*`) — the "verified / pass / enforced" signal and the
  check in the logo.
- **Status semantics** are the workhorse: **pass** = emerald, **warn / expiring** = amber, **fail /
  exposed / critical** = red, **info** = a distinct blue (`--info-*`, kept separate from brand so "info"
  and "primary action" never collide), **none / not-scanned** = slate. Each status has an `-fg`, `-bg`,
  `-border`, and `-solid` token so badges, rows, and pills stay consistent.
- Color is **low-saturation on surfaces** (soft `-bg` tints behind text) and **saturated only for small
  solid indicators** (dots, solid badges, bars).

### Typography
- **IBM Plex Sans** for all UI and display — a humanist grotesque with technical credibility.
- **IBM Plex Mono** for every machine value: resource names, namespaces, YAML, IDs, counts-in-context.
  This mono/sans split is a core brand signature and makes evidence scannable.
- Dense, utilitarian scale: default body **14px**, tables down to 12–13px, display 30px. Headings are
  semibold (600), never black. Uppercase eyebrow labels at 11px with `0.06em` tracking.
- **Substitution note:** IBM Plex is loaded from the **Google Fonts CDN**. To self-host, drop the
  `.woff2` files into `assets/fonts/` and swap the `@import` in `tokens/fonts.css` for local `@font-face`
  rules. Flagging this — provide licensed font files if self-hosting is required.

### Spacing & layout
- **4px base grid.** Dense, table-friendly rhythm (`--space-*`). Fixed app chrome: 248px sidebar, 56px
  topbar, 1360px content max. Generous vertical whitespace between sections, tight within data rows.

### Shape, border, shadow
- **Modest radii** — 4–8px on controls and cards, 12px on larger panels, pill only for badges/tags.
  Nothing heavily rounded; this is an instrument.
- **Borders do the structural work:** 1px `--border-subtle`/`--border-default` hairlines define cards,
  tables, and inputs. Cards are **white with a 1px border + `--shadow-xs`** — flat and calm, not floaty.
  Elevated surfaces (dropdowns, dialogs, toasts) step up to `--shadow-md`/`--shadow-lg`.
- Cool slate-tinted shadows only; no colored glows except the focus ring (`--shadow-focus`, brand blue).

### Motion & states
- **Short, precise, no bounce.** 120–260ms, ease-out. Fades and small position shifts only — trust is
  conveyed by restraint. No infinite decorative loops.
- **Hover:** surfaces go to `--surface-hover` (one slate step); solid buttons darken one brand step;
  links underline. **Press:** darken a further step (buttons `--action-bg-active`); no scale/shrink.
  **Focus:** always a visible `--shadow-focus` ring — accessibility is non-negotiable for this audience.
- **Disabled:** reduced opacity + `not-allowed`, never removed entirely (auditors need to see what's off).

### Backgrounds & imagery
- No photography, no illustration, no gradients-as-decor. Backgrounds are flat slate/white. The only
  recurring "texture" is data itself: tables, status pills, coverage bars, and small mesh-node motifs
  derived from the logo. If a hero/empty-state needs visual interest, use the **mesh-node line motif**
  in low-opacity slate — never stock imagery.

---

## 4. Iconography

- **Icon set: [Lucide](https://lucide.dev)** — loaded from CDN. Chosen for its clean **1.5px stroke,
  rounded joins, outline (not filled)** style, which matches the calm, technical, hairline-driven UI.
  Consistent 16px / 18px / 20px sizes; `currentColor` so icons inherit text color and status color.
  - **Substitution flag:** No source icon set existed, so Lucide is a proposed default. If OpenMeshGuard
    has or adopts a specific icon library, swap the CDN link and re-document here.
- **Status is never carried by icon alone** — always icon + color + text label, for colorblind safety and
  audit clarity. Common status glyphs: `shield-check` (verified/pass), `shield-alert` (fail),
  `triangle-alert` (warn), `circle-help` (unknown), `git-branch` / `git-compare` (drift/GitOps),
  `file-check` (evidence), `user-round` (ownership), `clock` (exception lifecycle).
- **No emoji. No Unicode dingbats as icons.** The logo's mesh-node motif may be used decoratively.
- Load Lucide in any card/kit: `<script src="https://unpkg.com/lucide@latest"></script>` then
  `lucide.createIcons()`; in React components icons are passed as inline SVG or a small wrapper.

---

## 5. Index / manifest

**Root**
- `styles.css` — global entry point (consumers link this). `@import`s only.
- `readme.md` — this file.
- `SKILL.md` — Agent-compatible skill wrapper for this design bundle.

**`tokens/`** — foundations (all `@import`ed by `styles.css`)
- `fonts.css` · `colors.css` · `typography.css` · `spacing.css` · `effects.css` · `base.css`

**`assets/`**
- `logo-mark.svg` · `logo-wordmark.svg` · `logo-wordmark-dark.svg`

**`guidelines/`** — foundation specimen cards (Design System tab): color, type, spacing, brand.

**`components/core/`** — reusable primitives: Button, IconButton, Badge, StatusBadge, Tag, Card,
Input, Select, Checkbox, Switch, Avatar, Tabs, ProgressBar, Tooltip, Toast, MetricStat, DataTable.

**`ui_kits/posture-scanner/`** — full click-through recreation of the flagship product: overview
dashboard, findings table, resource detail, evidence/report export.

---

*v1 — created from the product brief. Treat colors, logo, and type as proposals; iterate freely.*
