package engine

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/ext"
	builtincontrols "github.com/openmeshguard/openmeshguard/controls"
	"gopkg.in/yaml.v3"
)

const namespaceCELVariable = "omg_nsctx"

var (
	builtinIDPattern = regexp.MustCompile(`^MG-[A-Z]+-[0-9]{3}$`)
	userIDPattern    = regexp.MustCompile(`^[A-Z]+-[A-Z]+-[0-9]{3}$`)
	requiresPattern  = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)+$`)
	celPathPattern   = regexp.MustCompile(`\b(?:workload|namespace|resource|inventory|params)(?:\.[A-Za-z_][A-Za-z0-9_]*)+`)
)

type validationIssue struct {
	file    string
	control string
	line    int
	column  int
	message string
}

func (i validationIssue) Error() string {
	return fmt.Sprintf("%s:%d:%d: control %s: %s", i.file, i.line, i.column, i.control, i.message)
}

type validationErrors []validationIssue

func (v validationErrors) Error() string {
	messages := make([]string, 0, len(v))
	for _, issue := range v {
		messages = append(messages, issue.Error())
	}
	return strings.Join(messages, "\n")
}

// LoadPacks loads all embedded built-ins plus repeatable user-supplied paths,
// validates the complete set, and rejects duplicate control IDs.
func LoadPacks(paths []string) ([]Pack, error) {
	packs, err := LoadBuiltins()
	if err != nil {
		return nil, err
	}
	for _, path := range paths {
		pack, loadErr := loadPath(path, SourceUser)
		if loadErr != nil {
			return nil, loadErr
		}
		packs = append(packs, pack)
	}
	if err := rejectDuplicateIDs(packs); err != nil {
		return nil, err
	}
	return packs, nil
}

// LoadBuiltins loads every embedded .yaml control pack.
func LoadBuiltins() ([]Pack, error) {
	entries, err := fs.ReadDir(builtincontrols.BuiltinFS, ".")
	if err != nil {
		return nil, fmt.Errorf("read embedded control packs: %w", err)
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".yaml" {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		return nil, fmt.Errorf("no embedded .yaml control packs found")
	}

	packs := make([]Pack, 0, len(names))
	for _, name := range names {
		data, readErr := fs.ReadFile(builtincontrols.BuiltinFS, name)
		if readErr != nil {
			return nil, fmt.Errorf("read embedded control pack %s: %w", name, readErr)
		}
		pack, decodeErr := decodeAndValidate(filepath.Join("controls", name), data, SourceBuiltin)
		if decodeErr != nil {
			return nil, decodeErr
		}
		packs = append(packs, pack)
	}
	if err := rejectDuplicateIDs(packs); err != nil {
		return nil, err
	}
	return packs, nil
}

// ValidateFile validates one user control pack without loading cluster state.
func ValidateFile(path string) error {
	pack, err := loadPath(path, SourceUser)
	if err != nil {
		return err
	}
	return rejectDuplicateIDs([]Pack{pack})
}

func loadPath(path string, source Source) (Pack, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Pack{}, fmt.Errorf("read control pack %s: %w", path, err)
	}
	return decodeAndValidate(path, data, source)
}

func decodeAndValidate(file string, data []byte, source Source) (Pack, error) {
	root, err := decodeSingleDocument(data)
	if err != nil {
		return Pack{}, fmt.Errorf("%s: control <pack>: decode YAML: %w", file, err)
	}

	issues := validateKnownFields(file, root)
	var pack Pack
	if err := root.Decode(&pack); err != nil {
		return Pack{}, fmt.Errorf("%s:%d:%d: control <pack>: decode fields: %w", file, root.Line, root.Column, err)
	}
	pack.File = file
	pack.Source = source
	issues = append(issues, validatePack(file, root, &pack)...)
	if len(issues) > 0 {
		return Pack{}, issues
	}
	return pack, nil
}

func decodeSingleDocument(data []byte) (*yaml.Node, error) {
	decoder := yaml.NewDecoder(strings.NewReader(string(data)))
	var document yaml.Node
	if err := decoder.Decode(&document); err != nil {
		return nil, err
	}
	if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("document must be a mapping")
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if len(extra.Content) != 0 {
		return nil, fmt.Errorf("multiple YAML documents are not supported")
	}
	return document.Content[0], nil
}

func validateKnownFields(file string, root *yaml.Node) validationErrors {
	var issues validationErrors
	issues = append(issues, unknownMappingFields(file, "<pack>", root, setOf(
		"apiVersion", "kind", "metadata", "params", "controls",
	))...)

	if metadata := mappingValue(root, "metadata"); metadata != nil {
		issues = append(issues, unknownMappingFields(file, "<pack>", metadata, setOf("name", "version"))...)
	}
	controls := mappingValue(root, "controls")
	if controls == nil || controls.Kind != yaml.SequenceNode {
		return issues
	}
	for _, node := range controls.Content {
		controlID := scalarValue(mappingValue(node, "id"))
		if controlID == "" {
			controlID = "<unknown>"
		}
		issues = append(issues, unknownMappingFields(file, controlID, node, setOf(
			"id", "title", "category", "severity", "evidenceType", "scope",
			"environments", "requires", "applicability", "expression", "message",
			"remediation", "frameworks", "match",
		))...)
		if remediation := mappingValue(node, "remediation"); remediation != nil {
			issues = append(issues, unknownMappingFields(
				file,
				controlID,
				remediation,
				setOf("guidance", "suggestedYAMLTemplate"),
			)...)
		}
		if match := mappingValue(node, "match"); match != nil {
			issues = append(issues, unknownMappingFields(file, controlID, match, setOf("kinds"))...)
		}
	}
	return issues
}

func unknownMappingFields(file, control string, node *yaml.Node, allowed map[string]struct{}) validationErrors {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	var issues validationErrors
	for index := 0; index+1 < len(node.Content); index += 2 {
		key := node.Content[index]
		if _, ok := allowed[key.Value]; ok {
			continue
		}
		issues = append(issues, validationIssue{
			file: file, control: control, line: key.Line, column: key.Column,
			message: fmt.Sprintf("unknown field %q", key.Value),
		})
	}
	return issues
}

func validatePack(file string, root *yaml.Node, pack *Pack) validationErrors {
	var issues validationErrors
	issues = append(issues, requireFields(file, "<pack>", root, "apiVersion", "kind", "metadata", "controls")...)
	if pack.APIVersion != "" && pack.APIVersion != APIVersion {
		issues = append(issues, issueAt(file, "<pack>", mappingValue(root, "apiVersion"), fmt.Sprintf("apiVersion must be %q", APIVersion)))
	}
	if pack.Kind != "" && pack.Kind != Kind {
		issues = append(issues, issueAt(file, "<pack>", mappingValue(root, "kind"), fmt.Sprintf("kind must be %q", Kind)))
	}
	metadata := mappingValue(root, "metadata")
	issues = append(issues, requireFields(file, "<pack>", metadata, "name", "version")...)
	if strings.TrimSpace(pack.Metadata.Name) == "" && mappingValue(metadata, "name") != nil {
		issues = append(issues, issueAt(file, "<pack>", mappingValue(metadata, "name"), "metadata.name must not be empty"))
	}
	if strings.TrimSpace(pack.Metadata.Version) == "" && mappingValue(metadata, "version") != nil {
		issues = append(issues, issueAt(file, "<pack>", mappingValue(metadata, "version"), "metadata.version must not be empty"))
	}

	controlsNode := mappingValue(root, "controls")
	if controlsNode == nil || controlsNode.Kind != yaml.SequenceNode {
		if controlsNode != nil {
			issues = append(issues, issueAt(file, "<pack>", controlsNode, "controls must be a sequence"))
		}
		return issues
	}
	for index := range pack.Controls {
		controlNode := controlsNode.Content[index]
		issues = append(issues, validateControl(file, controlNode, &pack.Controls[index], pack.Source)...)
	}
	return issues
}

func validateControl(file string, node *yaml.Node, control *Control, source Source) validationErrors {
	controlID := control.ID
	if controlID == "" {
		controlID = "<unknown>"
	}
	var issues validationErrors
	issues = append(issues, requireFields(
		file,
		controlID,
		node,
		"id", "title", "category", "severity", "evidenceType", "scope", "requires",
		"applicability", "expression", "message", "remediation",
	)...)

	pattern := userIDPattern
	if source == SourceBuiltin {
		pattern = builtinIDPattern
	}
	if control.ID != "" && !pattern.MatchString(control.ID) {
		issues = append(issues, issueAt(file, controlID, mappingValue(node, "id"), fmt.Sprintf("id must match %s", pattern.String())))
	}
	issues = append(issues, validateNonEmpty(file, controlID, node, map[string]string{
		"title": control.Title, "category": control.Category, "severity": control.Severity,
		"evidenceType": control.EvidenceType, "scope": control.Scope,
		"applicability": control.Applicability, "expression": control.Expression,
		"message": control.Message,
	})...)
	issues = append(issues, validateEnum(file, controlID, node, "category", control.Category, "mtls", "authz", "exposure", "governance", "lifecycle")...)
	issues = append(issues, validateEnum(file, controlID, node, "severity", control.Severity, "critical", "high", "medium", "low", "info")...)
	issues = append(issues, validateEnum(file, controlID, node, "evidenceType", control.EvidenceType, "config", "runtime", "context")...)
	issues = append(issues, validateEnum(file, controlID, node, "scope", control.Scope, "workload", "namespace", "resource")...)

	if requires := mappingValue(node, "requires"); requires != nil {
		if requires.Kind != yaml.SequenceNode || len(control.Requires) == 0 {
			issues = append(issues, issueAt(file, controlID, requires, "requires must contain at least one dotted path"))
		}
	}
	seenRequires := map[string]struct{}{}
	for index, required := range control.Requires {
		required = strings.TrimSpace(required)
		location := sequenceValue(mappingValue(node, "requires"), index)
		if !requiresPattern.MatchString(required) {
			issues = append(issues, issueAt(file, controlID, location, fmt.Sprintf("requires path %q must be a dotted path", required)))
		}
		if _, exists := seenRequires[required]; exists {
			issues = append(issues, issueAt(file, controlID, location, fmt.Sprintf("duplicate requires path %q", required)))
		}
		seenRequires[required] = struct{}{}
		control.Requires[index] = required
	}
	if control.EvidenceType == "runtime" && !requiresVerified(control.Requires) {
		issues = append(issues, issueAt(file, controlID, mappingValue(node, "requires"), "runtime evidenceType must require a verified.* field"))
	}
	issues = append(issues, validateExpressionRequires(file, controlID, node, *control)...)
	if control.Scope == "resource" && len(control.Match.Kinds) == 0 {
		issues = append(issues, issueAt(file, controlID, node, "resource scope requires match.kinds"))
	}
	if control.Scope != "resource" && len(control.Match.Kinds) > 0 {
		issues = append(issues, issueAt(file, controlID, mappingValue(node, "match"), "match.kinds is only valid for resource scope"))
	}
	remediation := mappingValue(node, "remediation")
	issues = append(issues, requireFields(file, controlID, remediation, "guidance")...)
	if remediation != nil && strings.TrimSpace(control.Remediation.Guidance) == "" && mappingValue(remediation, "guidance") != nil {
		issues = append(issues, issueAt(file, controlID, mappingValue(remediation, "guidance"), "remediation.guidance must not be empty"))
	}

	if strings.TrimSpace(control.Message) != "" {
		if _, err := template.New(control.ID).Option("missingkey=error").Parse(control.Message); err != nil {
			issues = append(issues, issueAt(file, controlID, mappingValue(node, "message"), fmt.Sprintf("message template is invalid: %v", err)))
		}
	}
	if validScope(control.Scope) {
		var compileIssues validationErrors
		control.applicabilityProgram, compileIssues = compileBoolean(file, controlID, "applicability", mappingValue(node, "applicability"), control.Scope, control.Applicability)
		issues = append(issues, compileIssues...)
		control.expressionProgram, compileIssues = compileBoolean(file, controlID, "expression", mappingValue(node, "expression"), control.Scope, control.Expression)
		issues = append(issues, compileIssues...)
	}
	return issues
}

func validateExpressionRequires(file, controlID string, node *yaml.Node, control Control) validationErrors {
	declared := make([]string, 0, len(control.Requires))
	for _, required := range control.Requires {
		declared = append(declared, absoluteRequiredPath(control.Scope, required))
	}
	seen := map[string]struct{}{}
	var issues validationErrors
	for _, path := range celPathPattern.FindAllString(control.Expression, -1) {
		if structurallyAvailablePath(path) || coveredByRequiredPath(path, declared) {
			continue
		}
		if _, exists := seen[path]; exists {
			continue
		}
		seen[path] = struct{}{}
		issues = append(issues, issueAt(
			file,
			controlID,
			mappingValue(node, "expression"),
			fmt.Sprintf("expression path %q must be declared in requires", path),
		))
	}
	return issues
}

func coveredByRequiredPath(path string, required []string) bool {
	for _, candidate := range required {
		if path == candidate || strings.HasPrefix(path, candidate+".") {
			return true
		}
	}
	return false
}

func structurallyAvailablePath(path string) bool {
	for _, prefix := range []string{
		"workload.workload",
		"workload.dataPlaneMode",
		"namespace.name",
		"namespace.labels",
		"resource.apiVersion",
		"resource.kind",
		"resource.namespace",
		"resource.name",
	} {
		if path == prefix || strings.HasPrefix(path, prefix+".") {
			return true
		}
	}
	return false
}

func compileBoolean(file, controlID, field string, node *yaml.Node, scope, expression string) (cel.Program, validationErrors) {
	if strings.TrimSpace(expression) == "" {
		return nil, nil
	}
	if position := rootIdentifierPosition(expression, namespaceCELVariable); position >= 0 {
		return nil, validationErrors{issueAt(
			file,
			controlID,
			node,
			fmt.Sprintf("%s CEL compile error at 1:%d: undeclared reference to %q", field, position+1, namespaceCELVariable),
		)}
	}
	environment, err := celEnvironment(scope)
	if err != nil {
		return nil, validationErrors{issueAt(file, controlID, node, fmt.Sprintf("create CEL environment: %v", err))}
	}
	ast, compileIssues := environment.Compile(rewriteNamespaceVariable(expression))
	if compileIssues != nil && compileIssues.Err() != nil {
		issues := make(validationErrors, 0, len(compileIssues.Errors()))
		for _, compileIssue := range compileIssues.Errors() {
			issues = append(issues, issueAt(
				file,
				controlID,
				node,
				fmt.Sprintf(
					"%s CEL compile error at %d:%d: %s",
					field,
					compileIssue.Location.Line(),
					compileIssue.Location.Column()+1,
					strings.ReplaceAll(compileIssue.Message, namespaceCELVariable, "namespace"),
				),
			))
		}
		return nil, issues
	}
	if ast.OutputType() != cel.BoolType && ast.OutputType() != cel.DynType {
		return nil, validationErrors{issueAt(file, controlID, node, fmt.Sprintf("%s CEL expression must return bool, got %s", field, ast.OutputType()))}
	}
	program, err := environment.Program(ast)
	if err != nil {
		return nil, validationErrors{issueAt(file, controlID, node, fmt.Sprintf("build %s CEL program: %v", field, err))}
	}
	return program, nil
}

func celEnvironment(scope string) (*cel.Env, error) {
	options := []cel.EnvOption{
		cel.Variable("inventory", cel.DynType),
		cel.Variable("params", cel.DynType),
		ext.Strings(),
	}
	switch scope {
	case "workload":
		options = append(options, cel.Variable("workload", cel.DynType), cel.Variable(namespaceCELVariable, cel.DynType))
	case "namespace":
		options = append(options, cel.Variable(namespaceCELVariable, cel.DynType))
	case "resource":
		options = append(options, cel.Variable("resource", cel.DynType))
	default:
		return nil, fmt.Errorf("unsupported scope %q", scope)
	}
	return cel.NewEnv(options...)
}

// CEL reserves the word "namespace", while the frozen control contract uses it
// as a root variable. Rewrite only root identifier tokens to an equal-length
// internal name so pack syntax and compile positions remain contract-accurate.
func rewriteNamespaceVariable(expression string) string {
	var rewritten strings.Builder
	rewritten.Grow(len(expression))
	var quote byte
	escaped := false
	for index := 0; index < len(expression); {
		current := expression[index]
		if quote != 0 {
			rewritten.WriteByte(current)
			index++
			if escaped {
				escaped = false
				continue
			}
			if current == '\\' {
				escaped = true
				continue
			}
			if current == quote {
				quote = 0
			}
			continue
		}
		if current == '\'' || current == '"' {
			quote = current
			rewritten.WriteByte(current)
			index++
			continue
		}
		if isIdentifierStart(current) {
			start := index
			index++
			for index < len(expression) && isIdentifierPart(expression[index]) {
				index++
			}
			token := expression[start:index]
			rootIdentifier := start == 0 || expression[start-1] != '.'
			if token == "namespace" && rootIdentifier {
				rewritten.WriteString(namespaceCELVariable)
			} else {
				rewritten.WriteString(token)
			}
			continue
		}
		rewritten.WriteByte(current)
		index++
	}
	return rewritten.String()
}

func isIdentifierStart(value byte) bool {
	return value == '_' || value >= 'A' && value <= 'Z' || value >= 'a' && value <= 'z'
}

func isIdentifierPart(value byte) bool {
	return isIdentifierStart(value) || value >= '0' && value <= '9'
}

func rootIdentifierPosition(expression, wanted string) int {
	var quote byte
	escaped := false
	for index := 0; index < len(expression); {
		current := expression[index]
		if quote != 0 {
			index++
			if escaped {
				escaped = false
				continue
			}
			if current == '\\' {
				escaped = true
				continue
			}
			if current == quote {
				quote = 0
			}
			continue
		}
		if current == '\'' || current == '"' {
			quote = current
			index++
			continue
		}
		if isIdentifierStart(current) {
			start := index
			index++
			for index < len(expression) && isIdentifierPart(expression[index]) {
				index++
			}
			rootIdentifier := start == 0 || expression[start-1] != '.'
			if rootIdentifier && expression[start:index] == wanted {
				return start
			}
			continue
		}
		index++
	}
	return -1
}

func rejectDuplicateIDs(packs []Pack) error {
	type firstSeen struct {
		file string
	}
	seen := map[string]firstSeen{}
	for _, pack := range packs {
		for _, control := range pack.Controls {
			if first, exists := seen[control.ID]; exists {
				return validationErrors{{
					file: pack.File, control: control.ID, line: 1, column: 1,
					message: fmt.Sprintf("duplicate control ID; first declared in %s", first.file),
				}}
			}
			seen[control.ID] = firstSeen{file: pack.File}
		}
	}
	return nil
}

func requireFields(file, control string, node *yaml.Node, fields ...string) validationErrors {
	if node == nil || node.Kind != yaml.MappingNode {
		if node == nil {
			node = &yaml.Node{Line: 1, Column: 1}
		}
		return validationErrors{issueAt(file, control, node, fmt.Sprintf("required mapping is missing; expected fields %s", strings.Join(fields, ", ")))}
	}
	var issues validationErrors
	for _, field := range fields {
		if mappingValue(node, field) == nil {
			issues = append(issues, issueAt(file, control, node, fmt.Sprintf("missing required field %q", field)))
		}
	}
	return issues
}

func validateNonEmpty(file, control string, node *yaml.Node, values map[string]string) validationErrors {
	var issues validationErrors
	for field, value := range values {
		fieldNode := mappingValue(node, field)
		if fieldNode != nil && strings.TrimSpace(value) == "" {
			issues = append(issues, issueAt(file, control, fieldNode, fmt.Sprintf("%s must not be empty", field)))
		}
	}
	return issues
}

func validateEnum(file, control string, node *yaml.Node, field, value string, allowed ...string) validationErrors {
	if value == "" {
		return nil
	}
	for _, candidate := range allowed {
		if value == candidate {
			return nil
		}
	}
	return validationErrors{issueAt(file, control, mappingValue(node, field), fmt.Sprintf("%s must be one of %s", field, strings.Join(allowed, ", ")))}
}

func requiresVerified(paths []string) bool {
	for _, path := range paths {
		if strings.HasPrefix(path, "verified.") || strings.HasPrefix(path, "workload.verified.") {
			return true
		}
	}
	return false
}

func validScope(scope string) bool {
	return scope == "workload" || scope == "namespace" || scope == "resource"
}

func issueAt(file, control string, node *yaml.Node, message string) validationIssue {
	line, column := 1, 1
	if node != nil {
		line, column = node.Line, node.Column
	}
	return validationIssue{file: file, control: control, line: line, column: column, message: message}
}

func mappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for index := 0; index+1 < len(node.Content); index += 2 {
		if node.Content[index].Value == key {
			return node.Content[index+1]
		}
	}
	return nil
}

func sequenceValue(node *yaml.Node, index int) *yaml.Node {
	if node == nil || node.Kind != yaml.SequenceNode || index < 0 || index >= len(node.Content) {
		return node
	}
	return node.Content[index]
}

func scalarValue(node *yaml.Node) string {
	if node == nil {
		return ""
	}
	return node.Value
}

func setOf(values ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}
