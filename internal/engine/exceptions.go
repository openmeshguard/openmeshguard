package engine

import "strings"

// ApplyExceptions annotates already-evaluated findings. It never removes a
// finding, changes its severity, or exposes exception state to control CEL.
func ApplyExceptions(result Result, exceptions []ExceptionInput, bindings []ExceptionBinding) Result {
	result.Findings = append([]Finding(nil), result.Findings...)
	records := make(map[string]ExceptionInput, len(exceptions))
	for _, exception := range exceptions {
		records[exception.ID] = exception
	}
	bound := make(map[string]string, len(bindings))
	for _, binding := range bindings {
		bound[resourceIdentity(binding.Resource)] = binding.ExceptionID
	}
	for index := range result.Findings {
		finding := &result.Findings[index]
		if finding.Status != statusOpen ||
			finding.ControlID == "MG-EXC-001" ||
			finding.ControlID == "MG-EXC-002" ||
			len(finding.Resources) == 0 {
			continue
		}
		exceptionID := strings.TrimSpace(bound[resourceIdentity(finding.Resources[0])])
		if exceptionID == "" {
			continue
		}
		exception, exists := records[exceptionID]
		if !exists || !exception.Valid || !exceptionCoversControl(exception.ControlIDs, finding.ControlID) {
			continue
		}
		finding.Exception = &ExceptionEvidence{
			ID:        exception.ID,
			Expired:   exception.Expired,
			ExpiresAt: exception.ExpiresAt,
			Approver:  exception.Approver,
			Ticket:    exception.Ticket,
		}
		finding.EvidenceSources = uniqueStrings(append(finding.EvidenceSources, "exception-record"))
		if exception.Expired {
			finding.Status = statusOpen
			continue
		}
		finding.Status = "excepted"
	}
	return result
}

func resourceIdentity(resource ResourceRef) string {
	return strings.Join([]string{
		apiGroupFromAPIVersion(resource.APIVersion),
		resource.Kind,
		resource.Namespace,
		resource.Name,
	}, "|")
}

func exceptionCoversControl(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}
