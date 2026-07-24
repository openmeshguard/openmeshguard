// Package resolver — FROZEN CONTRACT DRAFT (v1alpha1).
//
// This file defines the interface and output types of the effective posture
// resolver. Move it to internal/resolver/ during M0 scaffolding; from then on,
// changes to exported types in this file require human approval.
//
// Resolver versions are a stable, comma-separated list of subsystem tags in
// the order mTLS, authorization (for example, mtls/v3,authz/v7). Bump only the
// tag whose precedence, interpretation, or unknown-propagation semantics
// change, and add or update table coverage for that behavior.
//
// INVARIANTS (enforced by tests and lint):
//  1. This package is PURE: no client-go imports, no I/O, no globals, no clock.
//     Inputs arrive fully normalized; outputs are deterministic.
//  2. Every non-unknown conclusion carries a non-empty resolution Chain.
//  3. Unknown is an explicit value, never a zero-value fallthrough.
package resolver

import "time"

// ---- Enumerations (string values are the canonical JSON values) ----

type MTLSEffective string

const (
	MTLSStrict      MTLSEffective = "strict"
	MTLSPermissive  MTLSEffective = "permissive"
	MTLSDisabled    MTLSEffective = "disabled"
	MTLSMixedByPort MTLSEffective = "mixed-by-port"
	MTLSNotInMesh   MTLSEffective = "not-in-mesh"
	MTLSUnknown     MTLSEffective = "unknown"
)

type AuthzEffective string

const (
	AuthzDefaultDenyExplicitAllow AuthzEffective = "default-deny-explicit-allow"
	AuthzAllowOnly                AuthzEffective = "allow-only"
	AuthzNoPolicy                 AuthzEffective = "no-policy"
	AuthzDenyPresent              AuthzEffective = "deny-present"
	AuthzWaypointUnenforced       AuthzEffective = "waypoint-policy-unenforced"
	AuthzNotInMesh                AuthzEffective = "not-in-mesh"
	AuthzUnknown                  AuthzEffective = "unknown"
)

type DataPlaneMode string

const (
	ModeSidecar       DataPlaneMode = "sidecar"
	ModeAmbient       DataPlaneMode = "ambient"
	ModeMixed         DataPlaneMode = "mixed"
	ModeUnknown       DataPlaneMode = "unknown"
	ModeNotApplicable DataPlaneMode = "not-applicable"
)

// ---- Evidence chain ----

// Step is one entry in the ordered chain of resources and rules that produced
// a resolved conclusion. Order starts at 1 (lowest-precedence input) and ends
// at the step that finalized the result.
type Step struct {
	Order     int    `json:"order"`
	Kind      string `json:"kind"` // PeerAuthentication, DestinationRule, AuthorizationPolicy, MeshConfigDefault, ...
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Field     string `json:"field,omitempty"` // e.g. spec.mtls.mode, spec.portLevelMtls["8080"]
	Effect    string `json:"effect"`          // human-readable contribution of this step
}

// ---- Inputs ----

// WorkloadInput is the normalized view of one workload plus every policy
// resource that could influence its posture. The normalizer is responsible
// for scoping (Sidecar resources, exportTo, revision/namespace selection);
// the resolver is responsible for precedence and semantics.
type WorkloadInput struct {
	Ref           WorkloadRef
	Labels        map[string]string
	Ports         []int32 // nil when unavailable; non-nil empty when observed with no declared ports
	DataPlaneMode DataPlaneMode
	Namespace     NamespaceInput
	MeshDefaults  MeshDefaults
	PeerAuthN     []PeerAuthenticationView // all PAs whose scope includes this workload
	DestRules     []DestinationRuleView    // DRs whose host selection targets this workload's services
	// DestinationRulesKnown distinguishes a completed collection with no
	// matching DestinationRules from unavailable DestinationRule evidence.
	DestinationRulesKnown bool
	AuthzPolicies         []AuthorizationPolicyView // nil when collection was unavailable
	// Waypoint is nil when waypoint evidence was collected and no waypoint
	// serves the workload. A non-nil view with Known=false records unavailable
	// Gateway API evidence without inventing a missing waypoint conclusion.
	Waypoint      *WaypointView
	ZtunnelOnNode Tristate // ambient: ztunnel health on the workload's node(s)
}

type WorkloadRef struct {
	Cluster   string `json:"cluster,omitempty"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
}

type NamespaceInput struct {
	Name            string
	Labels          map[string]string
	AmbientEnrolled Tristate
}

type MeshDefaults struct {
	RootNamespace string // istio root config namespace (default istio-system)
	TrustDomain   string
	Known         bool // false when control-plane config was unreadable -> unknown propagation
}

// Tristate models values that can be genuinely unobservable.
type Tristate int

const (
	Unobserved Tristate = iota
	False
	True
)

// PeerAuthenticationView, DestinationRuleView, AuthorizationPolicyView, and
// WaypointView are normalized projections of the corresponding Istio resources
// containing only the fields the resolver consumes. Defined in M2/M5 alongside
// the normalizer; they must remain client-go-free (plain structs).
type PeerAuthenticationView struct {
	Name, Namespace   string
	SelectorMatch     bool      // whether it selected this workload specifically
	CreationTimestamp time.Time // Kubernetes metadata.creationTimestamp for oldest-policy tie-breaks
	Mode              string    // UNSET | DISABLE | PERMISSIVE | STRICT
	PortLevelModes    map[int32]string
}

type DestinationRuleView struct {
	Name, Namespace string
	Host            string
	TLSMode         string // DISABLE | SIMPLE | MUTUAL | ISTIO_MUTUAL | "" (unset)
	PortTLSModes    map[int32]string
}

type AuthorizationPolicyView struct {
	Name, Namespace string
	Action          string // ALLOW | DENY | CUSTOM | AUDIT
	HasSelector     bool
	SelectorMatch   bool
	TargetsWaypoint bool // attached via targetRefs to a waypoint/Gateway
	TargetRefKind   string
	TargetRefName   string
	TargetWaypoint  *WaypointView // waypoint selected for this exact targetRef attachment
	RequiresL7      bool          // rule uses L7-only attributes (methods, paths, headers, request principals, ...)
	HasRules        bool          // distinguishes spec: {} from rules: [{}]
	BroadAllow      bool          // unrestricted rule or wildcard source hint; ignored when HasRules is false
	IdentityScoped  bool          // every matching ALLOW rule is constrained to explicit workload identity
	RootNamespace   bool          // lives in the mesh root namespace
}

type WaypointView struct {
	Name, Namespace string
	Known           bool // false when Gateway API evidence was unavailable
	Ready           bool
	Scope           string // namespace | service | workload
}

// ---- Outputs ----

type MTLSResult struct {
	Effective              MTLSEffective           `json:"effective"`
	ByPort                 map[int32]MTLSEffective `json:"byPort,omitempty"`
	ClientTLSContradiction *bool                   `json:"clientTLSContradiction,omitempty"`
	Chain                  []Step                  `json:"chain"`
	UnknownReason          string                  `json:"unknownReason,omitempty"` // required when Effective == unknown
}

type AuthzResult struct {
	Effective          AuthzEffective `json:"effective"`
	BroadAllow         *bool          `json:"broadAllow,omitempty"`
	IdentityScoped     *bool          `json:"identityScoped,omitempty"`
	PoliciesInScope    []string       `json:"policiesInScope,omitempty"` // "namespace/name"
	WaypointUnenforced []string       `json:"waypointUnenforced,omitempty"`
	Chain              []Step         `json:"chain"`
	UnknownReason      string         `json:"unknownReason,omitempty"`
}

type WorkloadResult struct {
	Ref   WorkloadRef   `json:"workload"`
	Mode  DataPlaneMode `json:"dataPlaneMode"`
	MTLS  MTLSResult    `json:"mtls"`
	Authz AuthzResult   `json:"authorization"`
}

// ---- Interface ----

// Resolver computes effective posture. Implementations must be deterministic
// and side-effect free. Version() identifies the semantics version recorded in
// report output as scanner.resolverVersion; it changes whenever precedence or
// interpretation logic changes, with a corresponding entry in the test tables.
type Resolver interface {
	Version() string
	ResolveMTLS(in WorkloadInput) MTLSResult
	ResolveAuthz(in WorkloadInput) AuthzResult
}
