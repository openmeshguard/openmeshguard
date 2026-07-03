package resolver

import "sort"

const (
	notImplementedM2Reason = "not yet implemented (M2)"
)

const provisionalVersion = "resolver-m1-provisional"

// ProvisionalVersion returns the version string used in scan output.
func ProvisionalVersion() string {
	return provisionalVersion
}

// ProvisionalResolver implements the narrow M1 mTLS path. It is replaced by
// full Istio mTLS semantics in M2.
type ProvisionalResolver struct{}

// NewProvisional returns the M1 resolver implementation.
func NewProvisional() ProvisionalResolver {
	return ProvisionalResolver{}
}

func (ProvisionalResolver) Version() string {
	return ProvisionalVersion()
}

func (ProvisionalResolver) ResolveMTLS(in WorkloadInput) MTLSResult {
	if !in.MeshDefaults.Known {
		return MTLSResult{
			Effective:              MTLSUnknown,
			ClientTLSContradiction: false,
			Chain:                  []Step{},
			UnknownReason:          "PeerAuthentication resources unavailable",
		}
	}
	if len(in.DestRules) > 0 ||
		hasM2PeerAuthenticationInputs(in.PeerAuthN) ||
		hasSameScopePeerAuthenticationConflict(in) {
		return MTLSResult{
			Effective:              MTLSUnknown,
			ClientTLSContradiction: false,
			Chain:                  []Step{},
			UnknownReason:          notImplementedM2Reason,
		}
	}

	rootNamespace := in.MeshDefaults.RootNamespace
	if rootNamespace == "" {
		rootNamespace = "istio-system"
	}

	chain := []Step{{
		Order:  1,
		Kind:   "MeshConfigDefault",
		Field:  "defaultPeerAuthenticationMode",
		Effect: "defaults destination workloads to PERMISSIVE when no PeerAuthentication mode overrides it",
	}}
	effective := MTLSPermissive

	for _, peerAuthentication := range sortedPeerAuthentications(in.PeerAuthN) {
		if peerAuthentication.Namespace != rootNamespace || peerAuthentication.SelectorMatch {
			continue
		}
		next, ok := mtlsModeToEffective(peerAuthentication.Mode)
		if !ok {
			return MTLSResult{
				Effective:              MTLSUnknown,
				ClientTLSContradiction: false,
				Chain:                  []Step{},
				UnknownReason:          notImplementedM2Reason,
			}
		}
		if next == "" {
			continue
		}
		effective = next
		chain = append(chain, Step{
			Order:     len(chain) + 1,
			Kind:      "PeerAuthentication",
			Name:      peerAuthentication.Name,
			Namespace: peerAuthentication.Namespace,
			Field:     "spec.mtls.mode",
			Effect:    "sets mesh-wide mTLS mode to " + peerAuthentication.Mode,
		})
	}

	for _, peerAuthentication := range sortedPeerAuthentications(in.PeerAuthN) {
		if peerAuthentication.Namespace != in.Ref.Namespace || peerAuthentication.SelectorMatch {
			continue
		}
		next, ok := mtlsModeToEffective(peerAuthentication.Mode)
		if !ok {
			return MTLSResult{
				Effective:              MTLSUnknown,
				ClientTLSContradiction: false,
				Chain:                  []Step{},
				UnknownReason:          notImplementedM2Reason,
			}
		}
		if next == "" {
			continue
		}
		effective = next
		chain = append(chain, Step{
			Order:     len(chain) + 1,
			Kind:      "PeerAuthentication",
			Name:      peerAuthentication.Name,
			Namespace: peerAuthentication.Namespace,
			Field:     "spec.mtls.mode",
			Effect:    "sets namespace mTLS mode to " + peerAuthentication.Mode,
		})
	}

	return MTLSResult{
		Effective:              effective,
		ClientTLSContradiction: false,
		Chain:                  chain,
	}
}

func (ProvisionalResolver) ResolveAuthz(WorkloadInput) AuthzResult {
	return AuthzResult{
		Effective:     AuthzUnknown,
		Chain:         []Step{},
		UnknownReason: notImplementedM2Reason,
	}
}

func hasM2PeerAuthenticationInputs(peerAuthentications []PeerAuthenticationView) bool {
	for _, peerAuthentication := range peerAuthentications {
		if peerAuthentication.SelectorMatch || len(peerAuthentication.PortLevelModes) > 0 {
			return true
		}
	}
	return false
}

func hasSameScopePeerAuthenticationConflict(in WorkloadInput) bool {
	rootNamespace := in.MeshDefaults.RootNamespace
	if rootNamespace == "" {
		rootNamespace = "istio-system"
	}
	counts := map[string]int{}
	for _, peerAuthentication := range in.PeerAuthN {
		if peerAuthentication.SelectorMatch {
			continue
		}
		if peerAuthentication.Namespace != rootNamespace && peerAuthentication.Namespace != in.Ref.Namespace {
			continue
		}
		counts[peerAuthentication.Namespace]++
		if counts[peerAuthentication.Namespace] > 1 {
			return true
		}
	}
	return false
}

func sortedPeerAuthentications(peerAuthentications []PeerAuthenticationView) []PeerAuthenticationView {
	out := append([]PeerAuthenticationView(nil), peerAuthentications...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Namespace != out[j].Namespace {
			return out[i].Namespace < out[j].Namespace
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func mtlsModeToEffective(mode string) (MTLSEffective, bool) {
	switch mode {
	case "", "UNSET":
		return "", true
	case "STRICT":
		return MTLSStrict, true
	case "PERMISSIVE":
		return MTLSPermissive, true
	case "DISABLE":
		return MTLSDisabled, true
	default:
		return MTLSUnknown, false
	}
}
