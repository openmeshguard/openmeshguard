package engine

import (
	"testing"
	"time"
)

func TestApplyExceptionsNeverRemovesFindings(t *testing.T) {
	expiresAt := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	base := Result{Findings: []Finding{{
		ID: "finding", ControlID: "MG-MTLS-001", Severity: "high", Status: statusOpen,
		EvidenceSources: []string{"kubernetes-api"},
		Resources:       []ResourceRef{{Kind: "Deployment", Namespace: "payments", Name: "api"}},
	}}}
	bindings := []ExceptionBinding{{
		Resource:    ResourceRef{Kind: "Deployment", Namespace: "payments", Name: "api"},
		ExceptionID: "EXC-42",
		Owner:       "payments-team",
		OwnerKnown:  true,
	}}
	tests := []struct {
		name       string
		exception  ExceptionInput
		wantStatus string
		wantAttach bool
	}{
		{
			name: "active exception marks finding without changing severity",
			exception: ExceptionInput{
				ID: "EXC-42", ControlIDs: []string{"MG-MTLS-001"}, Valid: true,
				Owner: "payments-team", ExpiresAt: expiresAt, Approver: "security", Ticket: "https://tickets.example/42",
			},
			wantStatus: "excepted", wantAttach: true,
		},
		{
			name: "expired exception restores open finding",
			exception: ExceptionInput{
				ID: "EXC-42", ControlIDs: []string{"MG-MTLS-001"}, Valid: true, Expired: true,
				Owner: "payments-team", ExpiresAt: expiresAt, Approver: "security", Ticket: "https://tickets.example/42",
			},
			wantStatus: statusOpen, wantAttach: true,
		},
		{name: "invalid exception cannot alter finding", exception: ExceptionInput{ID: "EXC-42", ControlIDs: []string{"MG-MTLS-001"}}, wantStatus: statusOpen},
		{name: "wrong control cannot alter finding", exception: ExceptionInput{ID: "EXC-42", ControlIDs: []string{"MG-AUTHZ-001"}, Valid: true}, wantStatus: statusOpen},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ApplyExceptions(base, []ExceptionInput{tt.exception}, bindings)
			if len(got.Findings) != 1 {
				t.Fatalf("findings = %d, want original finding retained", len(got.Findings))
			}
			finding := got.Findings[0]
			if finding.Status != tt.wantStatus || finding.Severity != "high" {
				t.Fatalf("finding status/severity = %q/%q, want %q/high", finding.Status, finding.Severity, tt.wantStatus)
			}
			if (finding.Exception != nil) != tt.wantAttach {
				t.Fatalf("finding exception = %#v, want attached %v", finding.Exception, tt.wantAttach)
			}
		})
	}
}

func TestApplyExceptionsRequiresKnownMatchingOwner(t *testing.T) {
	base := Result{Findings: []Finding{{
		ID: "finding", ControlID: "MG-MTLS-001", Severity: "high", Status: statusOpen,
		Resources: []ResourceRef{{Kind: "Deployment", Namespace: "payments", Name: "api"}},
	}}}
	tests := []struct {
		name    string
		binding ExceptionBinding
	}{
		{
			name: "mismatched owner",
			binding: ExceptionBinding{
				Resource: ResourceRef{Kind: "Deployment", Namespace: "payments", Name: "api"},
				Owner:    "other-team", OwnerKnown: true,
			},
		},
		{
			name: "unknown owner",
			binding: ExceptionBinding{
				Resource: ResourceRef{Kind: "Deployment", Namespace: "payments", Name: "api"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.binding.ExceptionID = "EXC-42"
			got := ApplyExceptions(
				base,
				[]ExceptionInput{{
					ID: "EXC-42", Owner: "payments-team",
					ControlIDs: []string{"MG-MTLS-001"}, Valid: true,
				}},
				[]ExceptionBinding{tt.binding},
			)
			if got.Findings[0].Status != statusOpen || got.Findings[0].Exception != nil {
				t.Fatalf("finding = %#v, want unmatched exception to leave it open", got.Findings[0])
			}
		})
	}
}

func TestApplyExceptionsNeverExceptsExceptionHygieneControls(t *testing.T) {
	result := Result{Findings: []Finding{{
		ControlID: "MG-EXC-002", Severity: "high", Status: statusOpen,
		Resources: []ResourceRef{{APIVersion: "openmeshguard.io/v1alpha1", Kind: "Exception", Name: "EXC-42"}},
	}}}
	got := ApplyExceptions(
		result,
		[]ExceptionInput{{ID: "EXC-42", ControlIDs: []string{"MG-EXC-002"}, Valid: true}},
		[]ExceptionBinding{{Resource: ResourceRef{APIVersion: "openmeshguard.io/v1alpha1", Kind: "Exception", Name: "EXC-42"}, ExceptionID: "EXC-42"}},
	)
	if got.Findings[0].Status != statusOpen || got.Findings[0].Exception != nil {
		t.Fatalf("exception hygiene finding was excepted: %#v", got.Findings[0])
	}
}

func TestApplyExceptionsPreservesUnknownAndNotApplicable(t *testing.T) {
	tests := []struct {
		name   string
		status string
	}{
		{name: "unknown evidence remains unknown", status: statusUnknown},
		{name: "not applicable remains not applicable", status: statusNotApplicable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Result{Findings: []Finding{{
				ControlID: "MG-MTLS-001",
				Status:    tt.status,
				Severity:  "high",
				Resources: []ResourceRef{{Kind: "Deployment", Namespace: "payments", Name: "api"}},
			}}}
			got := ApplyExceptions(
				result,
				[]ExceptionInput{{ID: "EXC-42", ControlIDs: []string{"MG-MTLS-001"}, Valid: true}},
				[]ExceptionBinding{{
					Resource:    ResourceRef{Kind: "Deployment", Namespace: "payments", Name: "api"},
					ExceptionID: "EXC-42",
				}},
			)
			if len(got.Findings) != 1 ||
				got.Findings[0].Status != tt.status ||
				got.Findings[0].Exception != nil {
				t.Fatalf("finding = %#v, want unchanged %s without exception evidence", got.Findings, tt.status)
			}
		})
	}
}
