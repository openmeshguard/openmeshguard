---
name: openmeshguard-design
description: Use this skill to generate well-branded interfaces and assets for OpenMeshGuard, either for production or throwaway prototypes/mocks/etc. Contains essential design guidelines, colors, type, fonts, assets, and UI kit components for prototyping.
user-invocable: true
---

Read the README.md file within this skill, and explore the other available files.
If creating visual artifacts (slides, mocks, throwaway prototypes, etc), copy assets out and create static HTML files for the user to view. If working on production code, you can copy assets and read the rules here to become an expert in designing with this brand.
If the user invokes this skill without any other guidance, ask them what they want to build or design, ask some questions, and act as an expert designer who outputs HTML artifacts _or_ production code, depending on the need.

## Quick map
- `readme.md` — full brand guide: context, voice, visual foundations, iconography, index.
- `styles.css` — single global entry point; link this to inherit every token and font.
- `tokens/` — colors, typography, spacing, effects, fonts (CSS custom properties).
- `components/` — reusable React primitives (Button, StatusBadge, DataTable, Card, forms, feedback…).
- `ui_kits/posture-scanner/` — full interactive product recreation to reference.
- `assets/` — logo mark + wordmarks.

## Brand in one breath
Serious, calm, evidence-grade governance console. Cool slate neutrals, guard-blue primary, emerald
"verified" accent, and a strict pass/warn/fail/info status system. IBM Plex Sans for UI, IBM Plex Mono
for every machine value (resource names, namespaces, YAML, counts). Readability and utility over
decoration — no gradients, no emoji, no illustration. Sentence case, "you" for the user, findings stated
as fact.
