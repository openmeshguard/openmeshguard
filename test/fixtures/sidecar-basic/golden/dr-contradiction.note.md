# Golden note: DestinationRule contradiction

`dr-contradiction.json` must retain the server-side STRICT result with
`clientTLSContradiction` absent. Absence is the frozen canonical representation
of unavailable DestinationRule evidence. `plan/M3-rule-engine.md` defers that
collector, likely to M5; M4 does not infer the pure resolver's contradiction.
