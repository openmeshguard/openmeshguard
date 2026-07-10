package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"text/template"

	"github.com/google/cel-go/cel"
	"github.com/openmeshguard/openmeshguard/internal/resolver"
)

const (
	statusOpen          = "open"
	statusUnknown       = "unknown"
	statusNotApplicable = "not-applicable"
)

type evaluationTarget struct {
	key           string
	cluster       string
	environment   string
	dataPlaneMode string
	activation    map[string]any
	availability  map[string]Availability
	resource      ResourceRef
	workload      *WorkloadInput
	evidence      []string
	templateData  messageData
}

type messageData struct {
	Workload  string
	Namespace string
	Resource  string
	Posture   postureData
	Inventory map[string]any
	Params    map[string]any
}

type postureData struct {
	Mtls          mtlsData
	Authorization authorizationData
}

type mtlsData struct {
	Effective              string
	ByPort                 map[int32]resolver.MTLSEffective
	ClientTLSContradiction bool
}

type authorizationData struct {
	Effective string
}

type categoryAccumulator struct {
	pass    int
	fail    int
	unknown int
}

// Evaluate applies every validated control to its scope targets. Applicability
// is always evaluated before requires so out-of-mesh workloads become
// not-applicable even when later posture evidence is unavailable. Expression
// evaluation is reachable only after every declared required path is known.
func Evaluate(packs []Pack, input Input) (Result, error) {
	if err := rejectDuplicateIDs(packs); err != nil {
		return Result{}, err
	}

	result := Result{Findings: []Finding{}}
	categories := map[string]*categoryAccumulator{}
	for _, pack := range packs {
		params := mergeMaps(pack.Params, input.Params)
		for _, control := range pack.Controls {
			if _, ok := categories[control.Category]; !ok {
				categories[control.Category] = &categoryAccumulator{}
			}
			targets := targetsFor(control, input, params)
			for _, target := range targets {
				if !matchesEnvironment(control.Environments, target.environment) {
					continue
				}
				finding, outcome, err := evaluateControl(pack, control, target)
				if err != nil {
					return Result{}, err
				}
				switch outcome {
				case "pass":
					categories[control.Category].pass++
				case statusOpen:
					categories[control.Category].fail++
				case statusUnknown:
					categories[control.Category].unknown++
				case statusNotApplicable:
					// Binding contract: not-applicable is excluded from pass rates.
				}
				if finding != nil {
					result.Findings = append(result.Findings, *finding)
				}
			}
		}
	}

	sort.Slice(result.Findings, func(i, j int) bool {
		if result.Findings[i].ControlID != result.Findings[j].ControlID {
			return result.Findings[i].ControlID < result.Findings[j].ControlID
		}
		return result.Findings[i].ID < result.Findings[j].ID
	})
	result.Scores = buildScores(categories)
	return result, nil
}

func evaluateControl(pack Pack, control Control, target evaluationTarget) (*Finding, string, error) {
	applicable, err := evaluateBool(control.applicabilityProgram, target.activation)
	if err != nil {
		return nil, "", fmt.Errorf(
			"%s: control %s: applicability CEL evaluation error for %s: %w",
			pack.File,
			control.ID,
			target.key,
			err,
		)
	}
	if !applicable {
		finding := assembleFinding(control, target, statusNotApplicable, "resolved")
		finding.Reasoning = fmt.Sprintf("Control %s does not apply to %s.", control.ID, target.key)
		return &finding, statusNotApplicable, nil
	}

	if unknownReason := unavailableReason(control, target); unknownReason != "" {
		finding := assembleFinding(control, target, statusUnknown, "unavailable")
		finding.UnknownReason = unknownReason
		finding.Reasoning = fmt.Sprintf("Control %s could not be evaluated for %s: %s.", control.ID, target.key, unknownReason)
		return &finding, statusUnknown, nil
	}

	passed, err := evaluateBool(control.expressionProgram, target.activation)
	if err != nil {
		return nil, "", fmt.Errorf(
			"%s: control %s: expression CEL evaluation error for %s: %w",
			pack.File,
			control.ID,
			target.key,
			err,
		)
	}
	if passed {
		return nil, "pass", nil
	}

	finding := assembleFinding(control, target, statusOpen, "resolved")
	reasoning, err := renderMessage(control, target.templateData)
	if err != nil {
		return nil, "", fmt.Errorf("%s: control %s: render message for %s: %w", pack.File, control.ID, target.key, err)
	}
	finding.Reasoning = reasoning
	return &finding, statusOpen, nil
}

func unavailableReason(control Control, target evaluationTarget) string {
	var reasons []string
	for _, required := range control.Requires {
		path := absoluteRequiredPath(control.Scope, required)
		if override, ok := availabilityForPath(target.availability, path); ok {
			if override.Available {
				continue
			}
			reason := override.Reason
			if reason == "" {
				reason = "evidence unavailable"
			}
			reasons = append(reasons, fmt.Sprintf("%s unavailable: %s", required, reason))
			continue
		}
		value, available := lookupPath(target.activation, path)
		if available && !unknownValue(value) {
			continue
		}
		reasons = append(reasons, fmt.Sprintf("%s unavailable: required path has no known value", required))
	}
	return strings.Join(reasons, "; ")
}

func absoluteRequiredPath(scope, path string) string {
	path = strings.TrimSpace(path)
	for _, root := range []string{"workload.", "namespace.", "resource.", "inventory.", "params."} {
		if strings.HasPrefix(path, root) {
			return path
		}
	}
	switch scope {
	case "workload":
		return "workload." + path
	case "namespace":
		return "namespace." + path
	case "resource":
		return "resource." + path
	default:
		return path
	}
}

func evaluateBool(program cel.Program, activation map[string]any) (bool, error) {
	if program == nil {
		return false, fmt.Errorf("CEL program was not compiled")
	}
	value, _, err := program.Eval(activation)
	if err != nil {
		return false, err
	}
	boolean, ok := value.Value().(bool)
	if !ok {
		return false, fmt.Errorf("CEL result has type %T, want bool", value.Value())
	}
	return boolean, nil
}

func assembleFinding(control Control, target evaluationTarget, status, confidence string) Finding {
	return Finding{
		ID:              deterministicFindingID(control.ID, target),
		ControlID:       control.ID,
		Title:           control.Title,
		Severity:        control.Severity,
		EvidenceType:    control.EvidenceType,
		Status:          status,
		Confidence:      confidence,
		DataPlaneMode:   target.dataPlaneMode,
		EvidenceSources: findingEvidence(control, target.evidence),
		Resources:       []ResourceRef{target.resource},
		ResolutionChain: resolutionChain(control, target.workload),
		Remediation:     control.Remediation,
	}
}

func findingEvidence(control Control, targetEvidence []string) []string {
	switch control.EvidenceType {
	case "runtime":
		return []string{"prometheus"}
	case "context":
		return []string{"scan-config"}
	default:
		return append([]string(nil), targetEvidence...)
	}
}

func deterministicFindingID(controlID string, target evaluationTarget) string {
	identity := strings.Join([]string{
		controlID,
		target.cluster,
		target.resource.APIVersion,
		target.resource.Kind,
		target.resource.Namespace,
		target.resource.Name,
	}, "|")
	hash := sha256.Sum256([]byte(identity))
	return controlID + "-" + hex.EncodeToString(hash[:])[:12]
}

func resolutionChain(control Control, workload *WorkloadInput) []resolver.Step {
	if workload == nil {
		return nil
	}
	usesMTLS := strings.Contains(control.Expression, "workload.mtls")
	usesAuthz := strings.Contains(control.Expression, "workload.authorization")
	for _, path := range control.Requires {
		usesMTLS = usesMTLS || strings.HasPrefix(path, "mtls.") || strings.HasPrefix(path, "workload.mtls.")
		usesAuthz = usesAuthz || strings.HasPrefix(path, "authorization.") || strings.HasPrefix(path, "workload.authorization.")
	}
	var chain []resolver.Step
	if usesMTLS {
		chain = append(chain, workload.Posture.MTLS.Chain...)
	}
	if usesAuthz {
		chain = append(chain, workload.Posture.Authz.Chain...)
	}
	return append([]resolver.Step(nil), chain...)
}

func renderMessage(control Control, data messageData) (string, error) {
	tmpl, err := template.New(control.ID).Option("missingkey=error").Parse(control.Message)
	if err != nil {
		return "", err
	}
	var rendered strings.Builder
	if err := tmpl.Execute(&rendered, data); err != nil {
		return "", err
	}
	return strings.TrimSpace(rendered.String()), nil
}

func targetsFor(control Control, input Input, params map[string]any) []evaluationTarget {
	switch control.Scope {
	case "workload":
		return workloadTargets(input, params)
	case "namespace":
		return namespaceTargets(input, params)
	case "resource":
		return resourceTargets(control, input, params)
	default:
		return nil
	}
}

func workloadTargets(input Input, params map[string]any) []evaluationTarget {
	targets := make([]evaluationTarget, 0, len(input.Workloads))
	for index := range input.Workloads {
		workload := &input.Workloads[index]
		namespace := workload.Namespace
		if namespace.Name == "" {
			namespace.Name = workload.Posture.Ref.Namespace
		}
		environment := workload.Environment
		if environment == "" {
			environment = namespace.Environment
		}
		availability := defaultWorkloadAvailability(*workload, namespace)
		activation := map[string]any{
			"workload":           workloadValue(*workload, availability),
			"namespace":          namespaceValue(namespace),
			namespaceCELVariable: namespaceValue(namespace),
			"inventory":          nonNilMap(input.Inventory),
			"params":             params,
		}
		name := workload.Posture.Ref.Namespace + "/" + workload.Posture.Ref.Name
		targets = append(targets, evaluationTarget{
			key: name, cluster: workload.Posture.Ref.Cluster, environment: environment,
			dataPlaneMode: string(workload.Posture.Mode), activation: activation,
			availability: availability,
			resource:     ResourceRef{Kind: workload.Posture.Ref.Kind, Namespace: workload.Posture.Ref.Namespace, Name: workload.Posture.Ref.Name},
			workload:     workload,
			evidence:     []string{"kubernetes-api", "istio-crd"},
			templateData: messageData{
				Workload:  name,
				Namespace: namespace.Name,
				Posture: postureData{
					Mtls:          mtlsData{Effective: string(workload.Posture.MTLS.Effective), ByPort: workload.Posture.MTLS.ByPort, ClientTLSContradiction: workload.Posture.MTLS.ClientTLSContradiction},
					Authorization: authorizationData{Effective: string(workload.Posture.Authz.Effective)},
				},
				Inventory: nonNilMap(input.Inventory), Params: params,
			},
		})
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].key < targets[j].key })
	return targets
}

func namespaceTargets(input Input, params map[string]any) []evaluationTarget {
	namespaces := append([]NamespaceInput(nil), input.Namespaces...)
	seen := map[string]struct{}{}
	for _, namespace := range namespaces {
		seen[namespace.Name] = struct{}{}
	}
	for _, workload := range input.Workloads {
		namespace := workload.Namespace
		if namespace.Name == "" {
			namespace.Name = workload.Posture.Ref.Namespace
		}
		if _, exists := seen[namespace.Name]; exists {
			continue
		}
		seen[namespace.Name] = struct{}{}
		namespaces = append(namespaces, namespace)
	}
	targets := make([]evaluationTarget, 0, len(namespaces))
	for _, namespace := range namespaces {
		availability := normalizeAvailability("namespace", namespace.Availability)
		activation := map[string]any{
			"namespace":          namespaceValue(namespace),
			namespaceCELVariable: namespaceValue(namespace),
			"inventory":          nonNilMap(input.Inventory),
			"params":             params,
		}
		targets = append(targets, evaluationTarget{
			key: namespace.Name, environment: namespace.Environment, activation: activation,
			availability: availability,
			resource:     ResourceRef{Kind: "Namespace", Name: namespace.Name},
			evidence:     []string{"kubernetes-api"},
			templateData: messageData{Namespace: namespace.Name, Inventory: nonNilMap(input.Inventory), Params: params},
		})
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].key < targets[j].key })
	return targets
}

func resourceTargets(control Control, input Input, params map[string]any) []evaluationTarget {
	kinds := setOf(control.Match.Kinds...)
	var targets []evaluationTarget
	for _, resource := range input.Resources {
		if _, matches := kinds[resource.Kind]; !matches {
			continue
		}
		value := copyMap(resource.Fields)
		value["apiVersion"] = resource.APIVersion
		value["kind"] = resource.Kind
		value["namespace"] = resource.Namespace
		value["name"] = resource.Name
		availability := normalizeAvailability("resource", resource.Availability)
		key := resource.Namespace + "/" + resource.Name
		if resource.Namespace == "" {
			key = resource.Name
		}
		evidence := append([]string(nil), resource.EvidenceSources...)
		if len(evidence) == 0 {
			evidence = []string{"kubernetes-api", "istio-crd"}
		}
		targets = append(targets, evaluationTarget{
			key: key, environment: resource.Environment,
			activation:   map[string]any{"resource": value, "inventory": nonNilMap(input.Inventory), "params": params},
			availability: availability,
			resource:     ResourceRef{APIVersion: resource.APIVersion, Kind: resource.Kind, Namespace: resource.Namespace, Name: resource.Name},
			evidence:     evidence,
			templateData: messageData{Resource: key, Inventory: nonNilMap(input.Inventory), Params: params},
		})
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].key < targets[j].key })
	return targets
}

func defaultWorkloadAvailability(workload WorkloadInput, namespace NamespaceInput) map[string]Availability {
	availability := normalizeAvailability("workload", workload.Availability)
	if workload.Posture.MTLS.Effective == resolver.MTLSUnknown {
		reason := workload.Posture.MTLS.UnknownReason
		if reason == "" {
			reason = "effective mTLS posture unavailable"
		}
		setDefaultAvailability(availability, "workload.mtls.effective", Availability{Reason: reason})
	}
	if workload.Posture.MTLS.ByPort == nil {
		setDefaultAvailability(availability, "workload.mtls.byPort", Availability{Reason: "workload ports unavailable"})
	}
	setDefaultAvailability(availability, "workload.mtls.clientTLSContradiction", Availability{Reason: "DestinationRule collection unavailable"})
	if workload.Posture.Authz.Effective == resolver.AuthzUnknown {
		reason := workload.Posture.Authz.UnknownReason
		if reason == "" {
			reason = "effective authorization posture unavailable"
		}
		setDefaultAvailability(availability, "workload.authorization.effective", Availability{Reason: reason})
	}
	if workload.Verified == nil {
		setDefaultAvailability(availability, "workload.verified", Availability{Reason: "runtime verification unavailable"})
	}
	if workload.Environment == "" && namespace.Environment == "" {
		setDefaultAvailability(availability, "workload.environment", Availability{Reason: "environment classification unavailable"})
	}
	if workload.Owner == "" {
		setDefaultAvailability(availability, "workload.owner", Availability{Reason: "ownership unavailable"})
	}
	return availability
}

func workloadValue(workload WorkloadInput, availability map[string]Availability) map[string]any {
	value := map[string]any{
		"workload": map[string]any{
			"cluster":   workload.Posture.Ref.Cluster,
			"namespace": workload.Posture.Ref.Namespace,
			"name":      workload.Posture.Ref.Name,
			"kind":      workload.Posture.Ref.Kind,
		},
		"dataPlaneMode": string(workload.Posture.Mode),
	}
	mtls := map[string]any{
		"chain": workload.Posture.MTLS.Chain,
	}
	if available(availability, "workload.mtls.effective") {
		mtls["effective"] = string(workload.Posture.MTLS.Effective)
	}
	if available(availability, "workload.mtls.byPort") {
		byPort := map[string]any{}
		for port, mode := range workload.Posture.MTLS.ByPort {
			byPort[fmt.Sprint(port)] = string(mode)
		}
		mtls["byPort"] = byPort
	}
	if available(availability, "workload.mtls.clientTLSContradiction") {
		mtls["clientTLSContradiction"] = workload.Posture.MTLS.ClientTLSContradiction
	}
	value["mtls"] = mtls

	authz := map[string]any{"chain": workload.Posture.Authz.Chain}
	if available(availability, "workload.authorization.effective") {
		authz["effective"] = string(workload.Posture.Authz.Effective)
	}
	value["authorization"] = authz
	if workload.Verified != nil {
		value["verified"] = copyMap(workload.Verified)
	}
	if workload.Environment != "" {
		value["environment"] = workload.Environment
	} else if workload.Namespace.Environment != "" {
		value["environment"] = workload.Namespace.Environment
	}
	if workload.Owner != "" {
		value["owner"] = workload.Owner
	}
	if workload.AppID != "" {
		value["appId"] = workload.AppID
	}
	return value
}

func namespaceValue(namespace NamespaceInput) map[string]any {
	value := map[string]any{
		"name":   namespace.Name,
		"labels": copyStringMap(namespace.Labels),
	}
	if namespace.Environment != "" {
		value["environment"] = namespace.Environment
	}
	if namespace.MeshEnrollment != "" {
		value["meshEnrollment"] = namespace.MeshEnrollment
	}
	return value
}

func normalizeAvailability(defaultRoot string, input map[string]Availability) map[string]Availability {
	out := make(map[string]Availability, len(input))
	for path, availability := range input {
		path = strings.TrimSpace(path)
		if !strings.Contains(path, ".") || (!strings.HasPrefix(path, "workload.") && !strings.HasPrefix(path, "namespace.") && !strings.HasPrefix(path, "resource.") && !strings.HasPrefix(path, "inventory.") && !strings.HasPrefix(path, "params.")) {
			path = defaultRoot + "." + path
		}
		out[path] = availability
	}
	return out
}

func setDefaultAvailability(values map[string]Availability, path string, value Availability) {
	if _, exists := values[path]; !exists {
		values[path] = value
	}
}

func available(values map[string]Availability, path string) bool {
	value, exists := values[path]
	return !exists || value.Available
}

func lookupPath(activation map[string]any, path string) (any, bool) {
	parts := strings.Split(path, ".")
	var current any = activation
	for _, part := range parts {
		var ok bool
		current, ok = lookupMapKey(current, part)
		if !ok || current == nil {
			return nil, false
		}
	}
	return current, true
}

func lookupMapKey(value any, key string) (any, bool) {
	switch mapping := value.(type) {
	case map[string]any:
		result, ok := mapping[key]
		return result, ok
	case map[string]string:
		result, ok := mapping[key]
		return result, ok
	case map[string]int:
		result, ok := mapping[key]
		return result, ok
	}
	reflected := reflect.ValueOf(value)
	if !reflected.IsValid() || reflected.Kind() != reflect.Map || reflected.Type().Key().Kind() != reflect.String {
		return nil, false
	}
	result := reflected.MapIndex(reflect.ValueOf(key).Convert(reflected.Type().Key()))
	if !result.IsValid() {
		return nil, false
	}
	return result.Interface(), true
}

func availabilityForPath(values map[string]Availability, path string) (Availability, bool) {
	for candidate := path; candidate != ""; {
		if value, exists := values[candidate]; exists {
			return value, true
		}
		separator := strings.LastIndex(candidate, ".")
		if separator < 0 {
			break
		}
		candidate = candidate[:separator]
	}
	return Availability{}, false
}

func unknownValue(value any) bool {
	text, ok := value.(string)
	return ok && strings.EqualFold(text, "unknown")
}

func matchesEnvironment(environments []string, environment string) bool {
	if len(environments) == 0 {
		return true
	}
	if environment == "" {
		return false
	}
	for _, candidate := range environments {
		if candidate == environment {
			return true
		}
	}
	return false
}

func buildScores(categories map[string]*categoryAccumulator) []CategoryScore {
	names := make([]string, 0, len(categories))
	for name := range categories {
		names = append(names, name)
	}
	sort.Strings(names)
	scores := make([]CategoryScore, 0, len(names))
	for _, name := range names {
		category := categories[name]
		evaluated := category.pass + category.fail
		score := CategoryScore{Category: name, Grade: "unknown", Evaluated: evaluated, Unknown: category.unknown}
		if evaluated > 0 {
			passRate := float64(category.pass) / float64(evaluated)
			score.PassRate = &passRate
			score.Grade = letterGrade(passRate)
		}
		scores = append(scores, score)
	}
	return scores
}

func letterGrade(passRate float64) string {
	switch {
	case passRate >= 0.9:
		return "A"
	case passRate >= 0.8:
		return "B"
	case passRate >= 0.7:
		return "C"
	case passRate >= 0.6:
		return "D"
	default:
		return "F"
	}
}

func mergeMaps(defaults, overrides map[string]any) map[string]any {
	out := copyMap(defaults)
	for key, value := range overrides {
		out[key] = value
	}
	return out
}

func nonNilMap(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	return input
}

func copyMap(input map[string]any) map[string]any {
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func copyStringMap(input map[string]string) map[string]string {
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
