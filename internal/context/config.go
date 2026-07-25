package context

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	APIVersion          = "openmeshguard.io/v1alpha1"
	ScanConfigKind      = "ScanConfig"
	OwnershipImportKind = "OwnershipImport"
	ExceptionKind       = "Exception"
)

var (
	controlIDPattern = regexp.MustCompile(`^[A-Z]+-[A-Z]+-[0-9]{3}$`)
	recordIDPattern  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
)

type Metadata struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

type ScanConfig struct {
	APIVersion     string               `yaml:"apiVersion"`
	Kind           string               `yaml:"kind"`
	Metadata       Metadata             `yaml:"metadata"`
	Inputs         ContextInputs        `yaml:"inputs,omitempty"`
	Classification ClassificationConfig `yaml:"classification,omitempty"`
	Ownership      OwnershipConfig      `yaml:"ownership,omitempty"`
	Controls       ControlsConfig       `yaml:"controls,omitempty"`
	File           string               `yaml:"-"`
}

type ContextInputs struct {
	OwnershipImport string   `yaml:"ownershipImport,omitempty"`
	Exceptions      []string `yaml:"exceptions,omitempty"`
}

type ClassificationConfig struct {
	Sources []ClassificationSource `yaml:"sources,omitempty"`
}

type ClassificationSource struct {
	Type        string            `yaml:"type"`
	Mappings    map[string]string `yaml:"mappings,omitempty"`
	Keys        []string          `yaml:"keys,omitempty"`
	Rules       []NameRule        `yaml:"rules,omitempty"`
	Environment string            `yaml:"environment,omitempty"`
}

type NameRule struct {
	Pattern     string `yaml:"pattern"`
	Environment string `yaml:"environment"`
}

type OwnershipConfig struct {
	Labels       OwnershipLabels `yaml:"labels,omitempty"`
	Applications []Application   `yaml:"applications,omitempty"`
}

type OwnershipLabels struct {
	AppID []string `yaml:"appId,omitempty"`
	Owner []string `yaml:"owner,omitempty"`
}

type Application struct {
	AppID string `yaml:"appId"`
	Owner string `yaml:"owner"`
}

type ControlsConfig struct {
	Parameters ControlParameters `yaml:"parameters,omitempty"`
	Overrides  []ControlOverride `yaml:"overrides,omitempty"`
}

type ControlParameters struct {
	Defaults     map[string]any            `yaml:"defaults,omitempty"`
	Environments map[string]map[string]any `yaml:"environments,omitempty"`
}

type ControlOverride struct {
	ControlID             string            `yaml:"controlId"`
	Environments          *[]string         `yaml:"environments,omitempty"`
	SeverityByEnvironment map[string]string `yaml:"severityByEnvironment,omitempty"`
}

type OwnershipImport struct {
	APIVersion   string        `yaml:"apiVersion"`
	Kind         string        `yaml:"kind"`
	Metadata     Metadata      `yaml:"metadata"`
	Applications []Application `yaml:"applications"`
	File         string        `yaml:"-"`
}

type ExceptionRecord struct {
	APIVersion       string        `yaml:"apiVersion"`
	Kind             string        `yaml:"kind"`
	Metadata         Metadata      `yaml:"metadata"`
	Spec             ExceptionSpec `yaml:"spec"`
	File             string        `yaml:"-"`
	ValidationErrors []string      `yaml:"-"`
	ExpiresAt        time.Time     `yaml:"-"`
}

type ExceptionSpec struct {
	ControlIDs    []string `yaml:"controlIds"`
	Owner         string   `yaml:"owner"`
	Approver      string   `yaml:"approver"`
	Justification string   `yaml:"justification"`
	Ticket        string   `yaml:"ticket"`
	ExpiresAt     string   `yaml:"expiresAt"`
}

func LoadScanConfig(path string) (ScanConfig, error) {
	if strings.TrimSpace(path) == "" {
		return ScanConfig{}, nil
	}
	if err := validateScanConfigEnvironmentTypes(path); err != nil {
		return ScanConfig{}, err
	}
	var config ScanConfig
	if err := decodeSingleYAML(path, &config); err != nil {
		return ScanConfig{}, err
	}
	config.File = path
	if config.APIVersion != APIVersion {
		return ScanConfig{}, fmt.Errorf("%s: apiVersion must be %q", path, APIVersion)
	}
	if config.Kind != ScanConfigKind {
		return ScanConfig{}, fmt.Errorf("%s: kind must be %q", path, ScanConfigKind)
	}
	if strings.TrimSpace(config.Metadata.Name) == "" || strings.TrimSpace(config.Metadata.Version) == "" {
		return ScanConfig{}, fmt.Errorf("%s: metadata.name and metadata.version must not be empty", path)
	}
	if err := validateClassification(config.Classification); err != nil {
		return ScanConfig{}, fmt.Errorf("%s: classification: %w", path, err)
	}
	if err := validateOwnership(config.Ownership); err != nil {
		return ScanConfig{}, fmt.Errorf("%s: ownership: %w", path, err)
	}
	if err := validateControls(config.Controls); err != nil {
		return ScanConfig{}, fmt.Errorf("%s: controls: %w", path, err)
	}
	base := filepath.Dir(path)
	if config.Inputs.OwnershipImport != "" {
		config.Inputs.OwnershipImport = resolveRelative(base, config.Inputs.OwnershipImport)
	}
	for index, exceptionPath := range config.Inputs.Exceptions {
		config.Inputs.Exceptions[index] = resolveRelative(base, exceptionPath)
	}
	return config, nil
}

func LoadOwnershipImport(path string) (OwnershipImport, error) {
	if strings.TrimSpace(path) == "" {
		return OwnershipImport{}, nil
	}
	if strings.EqualFold(filepath.Ext(path), ".csv") {
		applications, err := loadOwnershipCSV(path)
		if err != nil {
			return OwnershipImport{}, err
		}
		return OwnershipImport{
			APIVersion:   APIVersion,
			Kind:         OwnershipImportKind,
			Metadata:     Metadata{Name: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)), Version: "csv"},
			Applications: applications,
			File:         path,
		}, nil
	}
	var ownership OwnershipImport
	if err := decodeSingleYAML(path, &ownership); err != nil {
		return OwnershipImport{}, err
	}
	ownership.File = path
	if ownership.APIVersion != APIVersion {
		return OwnershipImport{}, fmt.Errorf("%s: apiVersion must be %q", path, APIVersion)
	}
	if ownership.Kind != OwnershipImportKind {
		return OwnershipImport{}, fmt.Errorf("%s: kind must be %q", path, OwnershipImportKind)
	}
	if strings.TrimSpace(ownership.Metadata.Name) == "" || strings.TrimSpace(ownership.Metadata.Version) == "" {
		return OwnershipImport{}, fmt.Errorf("%s: metadata.name and metadata.version must not be empty", path)
	}
	if err := validateApplications(ownership.Applications); err != nil {
		return OwnershipImport{}, fmt.Errorf("%s: applications: %w", path, err)
	}
	return ownership, nil
}

func LoadExceptions(paths []string) ([]ExceptionRecord, error) {
	var files []string
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			return nil, fmt.Errorf("exception path must not be empty")
		}
		info, err := os.Lstat(path)
		if err != nil {
			return nil, fmt.Errorf("inspect exception path %s: %w", path, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("%s: exception path must not be a symlink", path)
		}
		if !info.IsDir() {
			files = append(files, path)
			continue
		}
		entries, readErr := os.ReadDir(path)
		if readErr != nil {
			return nil, fmt.Errorf("read exception directory %s: %w", path, readErr)
		}
		for _, entry := range entries {
			extension := strings.ToLower(filepath.Ext(entry.Name()))
			if extension != ".yaml" && extension != ".yml" {
				continue
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return nil, fmt.Errorf("%s: exception entry must not be a symlink", filepath.Join(path, entry.Name()))
			}
			if !entry.Type().IsRegular() {
				continue
			}
			files = append(files, filepath.Join(path, entry.Name()))
		}
	}
	sort.Strings(files)

	seen := map[string]string{}
	var records []ExceptionRecord
	for _, path := range files {
		decoded, err := decodeExceptionDocuments(path)
		if err != nil {
			return nil, err
		}
		for _, record := range decoded {
			if previous, exists := seen[record.Metadata.Name]; exists && record.Metadata.Name != "" {
				return nil, fmt.Errorf("%s: duplicate exception ID %q also declared in %s", path, record.Metadata.Name, previous)
			}
			if record.Metadata.Name != "" {
				seen[record.Metadata.Name] = path
			}
			records = append(records, record)
		}
	}
	return records, nil
}

func validateClassification(config ClassificationConfig) error {
	for index, source := range config.Sources {
		prefix := fmt.Sprintf("sources[%d]", index)
		switch source.Type {
		case "namespace-mapping":
			if len(source.Mappings) == 0 || len(source.Keys) > 0 || len(source.Rules) > 0 || source.Environment != "" {
				return fmt.Errorf("%s: namespace-mapping requires only non-empty mappings", prefix)
			}
		case "namespace-label":
			if len(source.Keys) == 0 || len(source.Mappings) > 0 || len(source.Rules) > 0 || source.Environment != "" {
				return fmt.Errorf("%s: namespace-label requires only non-empty keys", prefix)
			}
			if err := validateNonEmptyUnique(source.Keys, prefix+".keys"); err != nil {
				return err
			}
		case "namespace-name":
			if len(source.Rules) == 0 || len(source.Mappings) > 0 || len(source.Keys) > 0 || source.Environment != "" {
				return fmt.Errorf("%s: namespace-name requires only non-empty rules", prefix)
			}
			for ruleIndex, rule := range source.Rules {
				if _, err := regexp.Compile(rule.Pattern); err != nil {
					return fmt.Errorf("%s.rules[%d].pattern: %w", prefix, ruleIndex, err)
				}
				if err := validateEnvironmentName(rule.Environment, fmt.Sprintf("%s.rules[%d].environment", prefix, ruleIndex)); err != nil {
					return err
				}
			}
		case "cluster":
			if len(source.Mappings) > 0 || len(source.Keys) > 0 || len(source.Rules) > 0 {
				return fmt.Errorf("%s: cluster requires only environment", prefix)
			}
			if err := validateEnvironmentName(source.Environment, prefix+".environment"); err != nil {
				return err
			}
		case "cluster-context":
			if len(source.Mappings) == 0 || len(source.Keys) > 0 || len(source.Rules) > 0 || source.Environment != "" {
				return fmt.Errorf("%s: cluster-context requires only non-empty mappings", prefix)
			}
		default:
			return fmt.Errorf("%s.type %q is unsupported", prefix, source.Type)
		}
		for key, environment := range source.Mappings {
			if strings.TrimSpace(key) == "" {
				return fmt.Errorf("%s.mappings keys must not be empty", prefix)
			}
			if err := validateEnvironmentName(environment, prefix+".mappings["+key+"]"); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateOwnership(config OwnershipConfig) error {
	if err := validateNonEmptyUnique(config.Labels.AppID, "labels.appId"); err != nil {
		return err
	}
	if err := validateNonEmptyUnique(config.Labels.Owner, "labels.owner"); err != nil {
		return err
	}
	return validateApplications(config.Applications)
}

func validateApplications(applications []Application) error {
	seen := map[string]struct{}{}
	for index, application := range applications {
		application.AppID = strings.TrimSpace(application.AppID)
		application.Owner = strings.TrimSpace(application.Owner)
		if application.AppID == "" || application.Owner == "" {
			return fmt.Errorf("[%d]: appId and owner must not be empty", index)
		}
		if _, exists := seen[application.AppID]; exists {
			return fmt.Errorf("[%d]: duplicate appId %q", index, application.AppID)
		}
		seen[application.AppID] = struct{}{}
	}
	return nil
}

func validateControls(config ControlsConfig) error {
	seen := map[string]struct{}{}
	validSeverity := map[string]struct{}{"critical": {}, "high": {}, "medium": {}, "low": {}, "info": {}}
	for environment := range config.Parameters.Environments {
		if err := validateEnvironmentName(environment, "parameters.environments"); err != nil {
			return err
		}
	}
	for index, override := range config.Overrides {
		if !controlIDPattern.MatchString(override.ControlID) {
			return fmt.Errorf("overrides[%d].controlId %q is invalid", index, override.ControlID)
		}
		if _, exists := seen[override.ControlID]; exists {
			return fmt.Errorf("overrides[%d]: duplicate controlId %q", index, override.ControlID)
		}
		seen[override.ControlID] = struct{}{}
		if override.Environments != nil {
			if mandatoryGovernanceControl(override.ControlID) {
				return fmt.Errorf(
					"overrides[%d]: cannot override environments for mandatory governance control %s",
					index,
					override.ControlID,
				)
			}
			if err := validateNonEmptyUnique(*override.Environments, fmt.Sprintf("overrides[%d].environments", index)); err != nil {
				return err
			}
			for environmentIndex, environment := range *override.Environments {
				if err := validateEnvironmentName(
					environment,
					fmt.Sprintf("overrides[%d].environments[%d]", index, environmentIndex),
				); err != nil {
					return err
				}
			}
		}
		for environment, severity := range override.SeverityByEnvironment {
			if err := validateEnvironmentName(
				environment,
				fmt.Sprintf("overrides[%d].severityByEnvironment", index),
			); err != nil {
				return err
			}
			if _, ok := validSeverity[severity]; !ok {
				return fmt.Errorf("overrides[%d].severityByEnvironment[%q] %q is invalid", index, environment, severity)
			}
		}
	}
	return nil
}

func mandatoryGovernanceControl(controlID string) bool {
	switch controlID {
	case "MG-ENV-001", "MG-OWN-001", "MG-EXC-001", "MG-EXC-002":
		return true
	default:
		return false
	}
}

func validateEnvironmentName(value, field string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fmt.Errorf("%s must not be empty", field)
	}
	if trimmed != value {
		return fmt.Errorf("%s must not contain surrounding whitespace", field)
	}
	return nil
}

func validateNonEmptyUnique(values []string, field string) error {
	seen := map[string]struct{}{}
	for index, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return fmt.Errorf("%s[%d] must not be empty", field, index)
		}
		if _, exists := seen[value]; exists {
			return fmt.Errorf("%s[%d] duplicates %q", field, index, value)
		}
		seen[value] = struct{}{}
	}
	return nil
}

func decodeSingleYAML(path string, target any) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()
	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("%s: exactly one YAML document is required", path)
		}
		return fmt.Errorf("decode trailing document in %s: %w", path, err)
	}
	return nil
}

func decodeExceptionDocuments(path string) ([]ExceptionRecord, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()
	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)
	var records []ExceptionRecord
	for document := 1; ; document++ {
		var record ExceptionRecord
		if err := decoder.Decode(&record); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, fmt.Errorf("decode %s document %d: %w", path, document, err)
		}
		if exceptionRecordEmpty(record) {
			continue
		}
		if record.APIVersion != APIVersion {
			return nil, fmt.Errorf("%s document %d: apiVersion must be %q", path, document, APIVersion)
		}
		if record.Kind != ExceptionKind {
			return nil, fmt.Errorf("%s document %d: kind must be %q", path, document, ExceptionKind)
		}
		record.File = path
		validateExceptionRecord(&record)
		records = append(records, record)
	}
	return records, nil
}

func exceptionRecordEmpty(record ExceptionRecord) bool {
	return record.APIVersion == "" &&
		record.Kind == "" &&
		record.Metadata.Name == "" &&
		record.Metadata.Version == "" &&
		len(record.Spec.ControlIDs) == 0 &&
		record.Spec.Owner == "" &&
		record.Spec.Approver == "" &&
		record.Spec.Justification == "" &&
		record.Spec.Ticket == "" &&
		record.Spec.ExpiresAt == ""
}

func validateScanConfigEnvironmentTypes(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()
	var document yaml.Node
	if err := yaml.NewDecoder(file).Decode(&document); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	if len(document.Content) == 0 {
		return nil
	}
	root := document.Content[0]
	classification := yamlMappingValue(root, "classification")
	sources := yamlMappingValue(classification, "sources")
	if sources != nil && sources.Kind == yaml.SequenceNode {
		for index, source := range sources.Content {
			prefix := fmt.Sprintf("classification.sources[%d]", index)
			if err := requireYAMLString(yamlMappingValue(source, "environment"), prefix+".environment"); err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}
			mappings := yamlMappingValue(source, "mappings")
			if mappings != nil && mappings.Kind == yaml.MappingNode {
				for mappingIndex := 0; mappingIndex+1 < len(mappings.Content); mappingIndex += 2 {
					if err := requireYAMLString(
						mappings.Content[mappingIndex+1],
						prefix+".mappings",
					); err != nil {
						return fmt.Errorf("%s: %w", path, err)
					}
				}
			}
			rules := yamlMappingValue(source, "rules")
			if rules != nil && rules.Kind == yaml.SequenceNode {
				for ruleIndex, rule := range rules.Content {
					if err := requireYAMLString(
						yamlMappingValue(rule, "environment"),
						fmt.Sprintf("%s.rules[%d].environment", prefix, ruleIndex),
					); err != nil {
						return fmt.Errorf("%s: %w", path, err)
					}
				}
			}
		}
	}
	controls := yamlMappingValue(root, "controls")
	parameters := yamlMappingValue(controls, "parameters")
	parameterEnvironments := yamlMappingValue(parameters, "environments")
	if parameterEnvironments != nil && parameterEnvironments.Kind == yaml.MappingNode {
		for index := 0; index+1 < len(parameterEnvironments.Content); index += 2 {
			if err := requireYAMLString(parameterEnvironments.Content[index], "controls.parameters.environments"); err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}
		}
	}
	overrides := yamlMappingValue(controls, "overrides")
	if overrides != nil && overrides.Kind == yaml.SequenceNode {
		for overrideIndex, override := range overrides.Content {
			environments := yamlMappingValue(override, "environments")
			if environments != nil && environments.Kind == yaml.SequenceNode {
				for environmentIndex, environment := range environments.Content {
					if err := requireYAMLString(
						environment,
						fmt.Sprintf(
							"controls.overrides[%d].environments[%d]",
							overrideIndex,
							environmentIndex,
						),
					); err != nil {
						return fmt.Errorf("%s: %w", path, err)
					}
				}
			}
			severities := yamlMappingValue(override, "severityByEnvironment")
			if severities != nil && severities.Kind == yaml.MappingNode {
				for index := 0; index+1 < len(severities.Content); index += 2 {
					if err := requireYAMLString(
						severities.Content[index],
						fmt.Sprintf("controls.overrides[%d].severityByEnvironment", overrideIndex),
					); err != nil {
						return fmt.Errorf("%s: %w", path, err)
					}
				}
			}
		}
	}
	return nil
}

func yamlMappingValue(node *yaml.Node, key string) *yaml.Node {
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

func requireYAMLString(node *yaml.Node, field string) error {
	if node == nil {
		return nil
	}
	if node.Kind != yaml.ScalarNode || node.Tag != "!!str" {
		return fmt.Errorf("%s must be a string", field)
	}
	return nil
}

func validateExceptionRecord(record *ExceptionRecord) {
	if !recordIDPattern.MatchString(record.Metadata.Name) {
		record.ValidationErrors = append(record.ValidationErrors, "metadata.name must be a non-empty stable ID")
	}
	if len(record.Spec.ControlIDs) == 0 {
		record.ValidationErrors = append(record.ValidationErrors, "spec.controlIds must not be empty")
	}
	seen := map[string]struct{}{}
	for _, controlID := range record.Spec.ControlIDs {
		if !controlIDPattern.MatchString(controlID) {
			record.ValidationErrors = append(record.ValidationErrors, fmt.Sprintf("spec.controlIds contains invalid ID %q", controlID))
		}
		if controlID == "MG-EXC-001" || controlID == "MG-EXC-002" {
			record.ValidationErrors = append(
				record.ValidationErrors,
				fmt.Sprintf("spec.controlIds cannot except exception hygiene control %q", controlID),
			)
		}
		if _, exists := seen[controlID]; exists {
			record.ValidationErrors = append(record.ValidationErrors, fmt.Sprintf("spec.controlIds duplicates %q", controlID))
		}
		seen[controlID] = struct{}{}
	}
	for field, value := range map[string]string{
		"spec.owner":         record.Spec.Owner,
		"spec.approver":      record.Spec.Approver,
		"spec.justification": record.Spec.Justification,
		"spec.ticket":        record.Spec.Ticket,
		"spec.expiresAt":     record.Spec.ExpiresAt,
	} {
		if strings.TrimSpace(value) == "" {
			record.ValidationErrors = append(record.ValidationErrors, field+" must not be empty")
		}
	}
	if ticket, err := url.Parse(record.Spec.Ticket); err != nil || ticket.Scheme != "https" || ticket.Host == "" {
		record.ValidationErrors = append(record.ValidationErrors, "spec.ticket must be an absolute HTTPS URL")
	}
	expiresAt, err := time.Parse(time.RFC3339, record.Spec.ExpiresAt)
	if err != nil {
		record.ValidationErrors = append(record.ValidationErrors, "spec.expiresAt must be an RFC3339 timestamp")
	} else {
		record.ExpiresAt = expiresAt
	}
	sort.Strings(record.ValidationErrors)
}

func loadOwnershipCSV(path string) ([]Application, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()
	reader := csv.NewReader(file)
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	if len(rows) == 0 || len(rows[0]) != 2 || rows[0][0] != "appId" || rows[0][1] != "owner" {
		return nil, fmt.Errorf("%s: CSV header must be exactly appId,owner", path)
	}
	applications := make([]Application, 0, len(rows)-1)
	for index, row := range rows[1:] {
		if len(row) != 2 {
			return nil, fmt.Errorf("%s: row %d must contain exactly appId and owner", path, index+2)
		}
		applications = append(applications, Application{AppID: strings.TrimSpace(row[0]), Owner: strings.TrimSpace(row[1])})
	}
	if err := validateApplications(applications); err != nil {
		return nil, fmt.Errorf("%s: applications: %w", path, err)
	}
	return applications, nil
}

func resolveRelative(base, path string) string {
	path = strings.TrimSpace(path)
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Clean(filepath.Join(base, path))
}
