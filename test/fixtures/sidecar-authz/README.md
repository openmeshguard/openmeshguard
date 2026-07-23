# Sidecar authorization acceptance fixtures

Each cases.tsv row isolates one live AuthorizationPolicy semantic boundary:
root-plus-namespace additive merge, empty policy versus `rules: [{}]`, DENY
ahead of ALLOW, ALLOW-only posture, and selector mismatch. Every workload is a
sidecar-injected Deployment with one Service-bound port so mTLS controls remain
fully evaluable while the authorization result is under test.
