# Governance context fixtures

These fixtures exercise the approved M6b governance input contracts:

- organization-specific environment and application-ID label keys;
- one application ID mapped to one owner across multiple namespaces;
- an explicitly unclassified and unowned mesh workload;
- active and expired annotation-referenced exception records;
- an exception whose owner does not match the annotated workload.

`scan-config.yaml` is passed to each fixture scan. The active and expired
exception goldens are review gates: the original MG-MTLS-001 finding must
remain present, with active records changing only its status to `excepted` and
expired records restoring `open` at the configured production severity.
The owner-mismatch golden proves that cross-owner reuse leaves MG-MTLS-001
open and raises MG-EXC-001.
