package normalize

import "github.com/openmeshguard/openmeshguard/internal/resolver"

// Inventory is the normalized inventory summary needed for canonical output.
type Inventory struct {
	Counts        map[string]int
	DataPlaneMode resolver.DataPlaneMode
	MultiCluster  MultiCluster
}

// MultiCluster captures non-secret signals that a cluster participates in a
// multi-network mesh. OSS v1 detects but does not evaluate cross-cluster posture.
type MultiCluster struct {
	ParticipationDetected bool
	Signals               []string
	MeshNetworks          []string
}

// Result is the normalized M1 model passed to the resolver and output writer.
type Result struct {
	Inventory Inventory
	Workloads []resolver.WorkloadInput
}
