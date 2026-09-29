# Security policy

OpenMeshGuard scans security posture using read-only Kubernetes access. A defect
that produces misleading conclusions, suppresses findings, exposes credentials,
or accesses unauthorized resources can be a security issue.

## Report a vulnerability privately

Use [GitHub private vulnerability reporting](https://github.com/openmeshguard/openmeshguard/security/advisories/new).
Do not put exploit details, credentials, kubeconfigs, or sensitive cluster
reports in public issues. Include the affected version, a minimal reproduction,
expected and actual behavior, and the security impact. Redact private resource
names and tokens before attaching evidence.

Maintainers will investigate and coordinate disclosure with the reporter.
This small project does not promise a fixed response time. Ordinary bugs and
feature requests belong in public GitHub issues after sensitive data is removed.

## Supported versions

Security fixes target the latest published release. During the v0 series,
interfaces remain experimental and users should upgrade to the latest patch
release. Release archives and checksums have Sigstore bundles; see
[release verification](docs/releases/verification.md).
